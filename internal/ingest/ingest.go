package ingest

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"nlscada-cloud/internal/db/influxdb"
	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/pkg/channel"

	"github.com/google/uuid"
)

// GatewayMessage là cấu trúc JSON payload từ Gateway cho data
type GatewayMessage struct {
	Timestamp int64  `json:"timestamp"`
	DeviceID  string `json:"device_id"`
	Tags      []struct {
		Name    string      `json:"name"`
		Value   interface{} `json:"value"`
		Quality string      `json:"quality"`
	} `json:"tags"`
}

// GatewayEvent là cấu trúc JSON payload từ Gateway cho event
type GatewayEvent struct {
	EventType     string `json:"event_type"`
	SeverityLevel string `json:"severity"`
	Message       string `json:"message"`
	Timestamp     int64  `json:"timestamp"`
}

type Service struct {
	pg     *postgres.Store
	influx *influxdb.Writer
}

func NewService(pg *postgres.Store, influx *influxdb.Writer) *Service {
	return &Service{pg: pg, influx: influx}
}

// HandleData xử lý message MQTT từ Gateway (data và heartbeat)
func (s *Service) HandleData(topic string, payload []byte) {
	parts := strings.Split(topic, "/")
	if len(parts) < 5 || parts[0] != "site" {
		log.Printf("Ingest: invalid topic format: %s", topic)
		return
	}
	siteID, err := uuid.Parse(parts[1])
	if err != nil {
		log.Printf("Ingest: invalid site_id in topic: %s", parts[1])
		return
	}
	deviceID, err := uuid.Parse(parts[3])
	if err != nil {
		log.Printf("Ingest: invalid device_id in topic: %s", parts[3])
		return
	}
	msgType := parts[4]

	switch msgType {
	case "data":
		s.handleDataMessage(siteID, deviceID, payload)
	case "heartbeat":
		s.handleHeartbeat(siteID, deviceID)
	default:
		log.Printf("Ingest: unknown message type: %s", msgType)
	}
}

// HandleEvent xử lý event từ Gateway
func (s *Service) HandleEvent(topic string, payload []byte) {
	parts := strings.Split(topic, "/")
	if len(parts) < 5 || parts[0] != "site" {
		log.Printf("Ingest: invalid event topic: %s", topic)
		return
	}
	siteID, err := uuid.Parse(parts[1])
	if err != nil {
		return
	}
	deviceID, err := uuid.Parse(parts[3])
	if err != nil {
		return
	}

	var event GatewayEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("Ingest: invalid event payload: %v", err)
		return
	}

	// Ghi system_event_logs
	_, err = s.pg.Pool.Exec(context.Background(),
		`INSERT INTO system_event_logs (site_id, device_id, event_type, severity_level, message)
		 VALUES ($1, $2, $3, $4, $5)`,
		siteID, deviceID, event.EventType, event.SeverityLevel, event.Message)
	if err != nil {
		log.Printf("Ingest: failed to insert system event: %v", err)
	}
}

func (s *Service) handleDataMessage(siteID, deviceID uuid.UUID, payload []byte) {
	var msg GatewayMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		log.Printf("Ingest: invalid data payload: %v", err)
		return
	}

	// Kiểm tra device tồn tại
	var exists bool
	err := s.pg.Pool.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM devices WHERE id = $1 AND site_id = $2)", deviceID, siteID).Scan(&exists)
	if err != nil || !exists {
		log.Printf("Ingest: device %s not found in site %s, ignoring", deviceID, siteID)
		return
	}

	ts := time.Unix(msg.Timestamp, 0)

	// Ghi InfluxDB và đánh giá cảnh báo
	tagMap := make(map[string]interface{}, len(msg.Tags))
	for _, tag := range msg.Tags {
		// Ghi InfluxDB
		fields := map[string]interface{}{
			"value":   tag.Value,
			"quality": tag.Quality,
		}
		tags := map[string]string{
			"device_id": deviceID.String(),
			"tag_name":  tag.Name,
		}
		if err := s.influx.WritePoint("metrics", tags, fields, ts); err != nil {
			log.Printf("Ingest: write InfluxDB error: %v", err)
		}

		tagMap[tag.Name] = tag.Value

		// Đánh giá cảnh báo
		s.evaluateAlert(siteID, deviceID, tag.Name, tag.Value)
	}

	// Đẩy realtime update
	channel.RealTimeDataChan <- channel.RealTimeUpdate{
		Type:      "tag_update",
		SiteID:    siteID,
		DeviceID:  deviceID,
		Timestamp: msg.Timestamp,
		Tags:      tagMap,
	}
}

func (s *Service) handleHeartbeat(siteID, deviceID uuid.UUID) {
	_, err := s.pg.Pool.Exec(context.Background(),
		"UPDATE devices SET last_heartbeat = NOW(), status = 'online' WHERE id = $1 AND site_id = $2",
		deviceID, siteID)
	if err != nil {
		log.Printf("Ingest: update heartbeat error: %v", err)
	}
}

func (s *Service) evaluateAlert(siteID, deviceID uuid.UUID, tagName string, value interface{}) {
	// Lấy alert_rules cho tag này
	rows, err := s.pg.Pool.Query(context.Background(),
		`SELECT id, name, min_value, max_value, severity, message_template
		 FROM alert_rules
		 WHERE site_id = $1 AND tag_id IN (
		     SELECT id FROM tags WHERE device_id = $2 AND name = $3
		 ) AND is_enabled = true`, siteID, deviceID, tagName)
	if err != nil {
		log.Printf("Ingest: query alert_rules error: %v", err)
		return
	}
	defer rows.Close()

	floatVal, ok := toFloat64(value)
	if !ok {
		return // không thể so sánh nếu không phải số
	}

	for rows.Next() {
		var rule models.AlertRule
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.MinValue, &rule.MaxValue, &rule.Severity, &rule.MessageTemplate); err != nil {
			continue
		}

		triggered := false
		if rule.MinValue != nil && floatVal < *rule.MinValue {
			triggered = true
		}
		if rule.MaxValue != nil && floatVal > *rule.MaxValue {
			triggered = true
		}

		if triggered {
			// Tạo alert_log
			var alertID uuid.UUID
			msg := rule.Name
			if rule.MessageTemplate != nil {
				msg = *rule.MessageTemplate
			}
			threshold := 0.0
			if rule.MaxValue != nil {
				threshold = *rule.MaxValue
			} else if rule.MinValue != nil {
				threshold = *rule.MinValue
			}
			err := s.pg.Pool.QueryRow(context.Background(),
				`INSERT INTO alert_logs (site_id, device_id, tag_name, triggered_value, threshold_value, severity, message)
				 VALUES ($1, $2, $3, $4, $5, $6, $7)
				 RETURNING id`,
				siteID, deviceID, tagName, floatVal, threshold, rule.Severity, msg).Scan(&alertID)
			if err != nil {
				log.Printf("Ingest: insert alert_log error: %v", err)
				continue
			}

			// Gửi alert notification
			channel.AlertNotificationChan <- channel.AlertNotification{
				AlertID:  alertID.String(),
				SiteID:   siteID,
				Severity: rule.Severity,
				Message:  msg,
				DeviceID: deviceID.String(),
				TagName:  tagName,
			}
		}
	}
}

func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

// StartOfflineChecker định kỳ chuyển trạng thái offline
func (s *Service) StartOfflineChecker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			_, err := s.pg.Pool.Exec(context.Background(),
				"UPDATE devices SET status = 'offline' WHERE last_heartbeat < NOW() - INTERVAL '90 seconds' AND status = 'online'")
			if err != nil {
				log.Printf("Ingest: offline check error: %v", err)
			}
		}
	}()
}

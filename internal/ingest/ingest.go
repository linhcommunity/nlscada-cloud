package ingest

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"nlscada-cloud/internal/db/influxdb"
	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/pkg/channel"

	"github.com/google/uuid"
)

// GatewayMessage là cấu trúc JSON payload từ Gateway
type GatewayMessage struct {
	Timestamp int64  `json:"timestamp"`
	DeviceID  string `json:"device_id"`
	Tags      []struct {
		Name    string      `json:"name"`
		Value   interface{} `json:"value"`
		Quality string      `json:"quality"`
	} `json:"tags"`
}

// Service xử lý ingest dữ liệu
type Service struct {
	pg     *postgres.Store
	influx *influxdb.Writer
}

// NewService tạo ingest service mới
func NewService(pg *postgres.Store, influx *influxdb.Writer) *Service {
	return &Service{pg: pg, influx: influx}
}

// HandleData xử lý message MQTT từ Gateway (data và heartbeat)
func (s *Service) HandleData(topic string, payload []byte) {
	// Parse site_id và device_id từ topic: "site/{site_id}/device/{device_id}/data" hoặc "/heartbeat"
	parts := strings.Split(topic, "/")
	if len(parts) < 5 {
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
	msgType := parts[4] // "data" hoặc "heartbeat"

	// Kiểm tra device có tồn tại trong DB không (Lớp bảo mật 4)
	var exists bool
	err = s.pg.Pool.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM devices WHERE id = $1 AND site_id = $2)", deviceID, siteID).Scan(&exists)
	if err != nil || !exists {
		log.Printf("Ingest: device %s not found in site %s, ignoring", deviceID, siteID)
		return
	}

	switch msgType {
	case "data":
		var msg GatewayMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			log.Printf("Ingest: invalid data payload: %v", err)
			return
		}

		// Ghi vào InfluxDB
		ts := time.Unix(msg.Timestamp, 0)
		for _, tag := range msg.Tags {
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
		}

		// Đẩy real-time update vào channel
		tagMap := make(map[string]interface{}, len(msg.Tags))
		for _, tag := range msg.Tags {
			tagMap[tag.Name] = tag.Value
		}
		channel.RealTimeDataChan <- channel.RealTimeUpdate{
			Type:      "tag_update",
			SiteID:    siteID, // thay TenantID
			DeviceID:  deviceID,
			Timestamp: msg.Timestamp,
			Tags:      tagMap,
		}

	case "heartbeat":
		// Cập nhật last_heartbeat và status = online
		_, err := s.pg.Pool.Exec(context.Background(),
			"UPDATE devices SET last_heartbeat = NOW(), status = 'online' WHERE id = $1 AND site_id = $2",
			deviceID, siteID)
		if err != nil {
			log.Printf("Ingest: update heartbeat error: %v", err)
		}
	}
}

// StartOfflineChecker định kỳ chuyển trạng thái offline nếu quá 90s không có heartbeat
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

package ingest

import (
	"encoding/json"
	"log"
	"time"

	"nlscada-cloud/internal/db/influxdb"
	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/pkg/channel"

	"github.com/google/uuid"
)

// GatewayMessage là payload MQTT từ Gateway
type GatewayMessage struct {
	Timestamp int64  `json:"timestamp"`
	DeviceID  string `json:"device_id"`
	Tags      []struct {
		Name    string      `json:"name"`
		Value   interface{} `json:"value"`
		Quality string      `json:"quality"`
	} `json:"tags"`
}

type Service struct {
	pg     *postgres.Store
	influx *influxdb.Writer
}

func NewService(pg *postgres.Store, influx *influxdb.Writer) *Service {
	return &Service{pg: pg, influx: influx}
}

// HandleData xử lý message MQTT từ Gateway
func (s *Service) HandleData(topic string, payload []byte) {
	var msg GatewayMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		log.Printf("Ingest: invalid payload: %v", err)
		return
	}

	deviceID, err := uuid.Parse(msg.DeviceID)
	if err != nil {
		log.Printf("Ingest: invalid device_id: %v", err)
		return
	}

	// TODO: Lấy tenant_id từ device (query PostgreSQL hoặc parse từ topic)
	// Tạm thời hardcode tenant demo
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Ghi từng tag vào InfluxDB
	for _, tag := range msg.Tags {
		fields := map[string]interface{}{
			"value":   tag.Value,
			"quality": tag.Quality,
		}
		tags := map[string]string{
			"device_id": msg.DeviceID,
			"tag_name":  tag.Name,
		}
		if err := s.influx.WritePoint("metrics", tags, fields, time.Unix(msg.Timestamp, 0)); err != nil {
			log.Printf("Ingest: write InfluxDB error: %v", err)
		}
	}

	// TODO: Cập nhật last_heartbeat, status nếu là heartbeat topic

	// Đẩy realtime update vào channel
	channel.RealTimeDataChan <- channel.RealTimeUpdate{
		Type:      "tag_update",
		TenantID:  tenantID,
		DeviceID:  deviceID,
		Timestamp: msg.Timestamp,
		Tags:      convertTags(msg.Tags),
	}
}

func convertTags(tags []struct {
	Name    string      `json:"name"`
	Value   interface{} `json:"value"`
	Quality string      `json:"quality"`
}) map[string]interface{} {
	result := make(map[string]interface{}, len(tags))
	for _, t := range tags {
		result[t.Name] = t.Value
	}
	return result
}

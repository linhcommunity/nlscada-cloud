package control

import (
	"context"
	"fmt"
	"log"
	"time"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/mqtt"

	"github.com/google/uuid"
)

type Service struct {
	store      *postgres.Store
	mqttClient *mqtt.Client
}

func NewService(store *postgres.Store, mqttClient *mqtt.Client) *Service {
	return &Service{store: store, mqttClient: mqttClient}
}

// SendControl thực hiện kiểm tra, tạo control log và publish MQTT command.
func (s *Service) SendControl(ctx context.Context, siteID, deviceID, userID uuid.UUID, tagName, value string) (*models.ControlLog, error) {
	// 1. Kiểm tra device thuộc site
	var deviceSiteID uuid.UUID
	err := s.store.Pool.QueryRow(ctx,
		"SELECT site_id FROM devices WHERE id = $1", deviceID).Scan(&deviceSiteID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}
	if deviceSiteID != siteID {
		return nil, fmt.Errorf("device does not belong to site")
	}

	// 2. Kiểm tra tag tồn tại (có thể bỏ qua hoặc kiểm tra thêm)
	var tagID uuid.UUID
	err = s.store.Pool.QueryRow(ctx,
		"SELECT id FROM tags WHERE device_id = $1 AND name = $2", deviceID, tagName).Scan(&tagID)
	if err != nil {
		return nil, fmt.Errorf("tag not found: %w", err)
	}

	// 3. Tạo control log
	var logEntry models.ControlLog
	err = s.store.Pool.QueryRow(ctx,
		`INSERT INTO control_logs (site_id, device_id, user_id, tag_name, requested_value, status)
         VALUES ($1, $2, $3, $4, $5, 'PENDING')
         RETURNING id, site_id, device_id, user_id, tag_name, requested_value, previous_value, status, created_at`,
		siteID, deviceID, userID, tagName, value,
	).Scan(&logEntry.ID, &logEntry.SiteID, &logEntry.DeviceID, &logEntry.UserID,
		&logEntry.TagName, &logEntry.RequestedValue, &logEntry.PreviousValue, &logEntry.Status, &logEntry.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create control log: %w", err)
	}

	// 4. Publish MQTT command
	if s.mqttClient != nil {
		topic := fmt.Sprintf("site/%s/device/%s/cmd", siteID.String(), deviceID.String())
		payload := fmt.Sprintf(`{"log_id":"%s","tag_name":"%s","value":"%s","timestamp":%d}`,
			logEntry.ID.String(), tagName, value, time.Now().Unix())
		s.mqttClient.Publish(topic, 1, payload)
		log.Printf("Published MQTT command: topic=%s payload=%s", topic, payload)
		// Cập nhật trạng thái SENT (có thể đợi ACK từ Gateway để chuyển SUCCESS/FAILED)
		s.store.Pool.Exec(ctx,
			"UPDATE control_logs SET status = 'SENT', sent_at = NOW() WHERE id = $1", logEntry.ID)
		logEntry.Status = "SENT"
		logEntry.SentAt = timePtr(time.Now())
	}

	return &logEntry, nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}

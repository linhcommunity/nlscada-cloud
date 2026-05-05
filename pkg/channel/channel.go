package channel

import (
	"github.com/google/uuid"
)

// RealTimeUpdate là dữ liệu thời gian thực gửi từ Ingest đến WebSocket Hub
type RealTimeUpdate struct {
	Type      string                 `json:"type"` // "tag_update"
	TenantID  uuid.UUID              `json:"tenant_id"`
	DeviceID  uuid.UUID              `json:"device_id"`
	Timestamp int64                  `json:"timestamp"`
	Tags      map[string]interface{} `json:"tags"`
}

// MetadataEvent là sự kiện thay đổi metadata (tạo/xóa device, tag)
type MetadataEvent struct {
	Action   string      `json:"action"` // "device_created", "device_deleted", "tag_created", "tag_deleted"
	TenantID uuid.UUID   `json:"tenant_id"`
	DeviceID uuid.UUID   `json:"device_id"`
	TagID    *uuid.UUID  `json:"tag_id,omitempty"`
	Payload  interface{} `json:"payload,omitempty"`
}

// RealTimeDataChan là kênh giao tiếp Ingest -> WS Hub
var RealTimeDataChan = make(chan RealTimeUpdate, 1000)

// MetadataEventChan là kênh giao tiếp API -> Ingest/WS Hub (tùy chọn)
var MetadataEventChan = make(chan MetadataEvent, 100)

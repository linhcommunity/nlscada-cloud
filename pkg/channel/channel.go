package channel

import "github.com/google/uuid"

type RealTimeUpdate struct {
	Type      string                 `json:"type"`
	SiteID    uuid.UUID              `json:"site_id"`
	DeviceID  uuid.UUID              `json:"device_id"`
	Timestamp int64                  `json:"timestamp"`
	Tags      map[string]interface{} `json:"tags"`
}

type AlertNotification struct {
	AlertID  string    `json:"alert_id"`
	SiteID   uuid.UUID `json:"site_id"`
	Severity string    `json:"severity"`
	Message  string    `json:"message"`
	DeviceID string    `json:"device_id,omitempty"`
	TagName  string    `json:"tag_name,omitempty"`
}

type SystemEventNotification struct {
	EventID   string    `json:"event_id"`
	SiteID    uuid.UUID `json:"site_id"`
	DeviceID  string    `json:"device_id,omitempty"`
	EventType string    `json:"event_type"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Timestamp int64     `json:"timestamp"`
}

var RealTimeDataChan = make(chan RealTimeUpdate, 1000)
var AlertNotificationChan = make(chan AlertNotification, 500)
var SystemEventNotificationChan = make(chan SystemEventNotification, 500)

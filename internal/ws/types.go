package ws

import "encoding/json"

type WSMessage struct {
	Event     string          `json:"event"`
	Timestamp string          `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// Client → Server payloads
type SubscribePayload struct {
	SiteID   string `json:"site_id"`
	DeviceID string `json:"device_id"`
}

type ControlPayload struct {
	DeviceID string `json:"device_id"`
	TagName  string `json:"tag_name"`
	Value    string `json:"value"`
}

// Server → Client payloads
type SubOkPayload struct {
	SiteID   string `json:"site_id"`
	DeviceID string `json:"device_id"`
}

type TagUpdatePayload struct {
	DeviceID  string                 `json:"device_id"`
	Tags      map[string]interface{} `json:"tags"`
	Timestamp int64                  `json:"timestamp"`
}

type AlertNewPayload struct {
	AlertID  string `json:"alert_id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	DeviceID string `json:"device_id,omitempty"`
	TagName  string `json:"tag_name,omitempty"`
}

type ControlAckPayload struct {
	LogID  string `json:"log_id"`
	Status string `json:"status"`
}

type ErrorPayload struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

package ws

// Message gửi từ client qua WebSocket
type ClientMessage struct {
	Action   string `json:"action"`              // "auth", "subscribe", "unsubscribe"
	Token    string `json:"token,omitempty"`     // JWT cho action "auth"
	DeviceID string `json:"device_id,omitempty"` // cho subscribe/unsubscribe
}

// Message gửi từ server đến client
type ServerMessage struct {
	Type      string                 `json:"type"`                // "auth_ok", "auth_error", "tag_update", "error"
	Message   string                 `json:"message,omitempty"`   // mô tả lỗi
	DeviceID  string                 `json:"device_id,omitempty"` // khi có tag_update
	Timestamp int64                  `json:"timestamp,omitempty"`
	Tags      map[string]interface{} `json:"tags,omitempty"`
}

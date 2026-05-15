package ws

// ClientMessage là message client gửi lên
type ClientMessage struct {
	Action   string `json:"action"`              // "auth" (không cần nếu token qua query), "sub", "unsub"
	SiteID   string `json:"site_id,omitempty"`   // UUID của site, dùng với sub/unsub
	DeviceID string `json:"device_id,omitempty"` // UUID của device, dùng với sub/unsub
}

// ServerMessage là message server gửi về client
type ServerMessage struct {
	Type      string                 `json:"type"`              // "auth_ok", "auth_error", "sub_ok", "error", "tag_update"
	Message   string                 `json:"message,omitempty"` // mô tả lỗi
	SiteID    string                 `json:"site_id,omitempty"`
	DeviceID  string                 `json:"device_id,omitempty"`
	Timestamp int64                  `json:"timestamp,omitempty"`
	Tags      map[string]interface{} `json:"tags,omitempty"`
}

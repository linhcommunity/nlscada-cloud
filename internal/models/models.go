package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Site đại diện cho một site (trước đây là tenant)
type Site struct {
	ID          uuid.UUID `json:"id"         db:"id"`
	Name        string    `json:"name"       db:"name"`
	Description *string   `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// User đại diện cho người dùng toàn cục (không gắn trực tiếp site)
type User struct {
	ID           uuid.UUID `json:"id"            db:"id"`
	Email        string    `json:"email"         db:"email"`
	Name         string    `json:"name"          db:"name"`
	PasswordHash string    `json:"-"             db:"password_hash"` // không serialize ra JSON
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
}

// Membership liên kết User với Site kèm vai trò trong site đó
type Membership struct {
	ID        uuid.UUID `json:"id"         db:"id"`
	UserID    uuid.UUID `json:"user_id"    db:"user_id"`
	SiteID    uuid.UUID `json:"site_id"    db:"site_id"`
	Role      string    `json:"role"       db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Device thuộc về một Site
type Device struct {
	ID            uuid.UUID  `json:"id"              db:"id"`
	SiteID        uuid.UUID  `json:"site_id"         db:"site_id"`
	Name          string     `json:"name"            db:"name"`
	DeviceType    string     `json:"device_type"     db:"device_type"`
	MqttClientID  *string    `json:"mqtt_client_id"  db:"mqtt_client_id"`
	Status        string     `json:"status"          db:"status"`
	LastHeartbeat *time.Time `json:"last_heartbeat"  db:"last_heartbeat"`
	CreatedAt     time.Time  `json:"created_at"      db:"created_at"`
}

// Tag là điểm dữ liệu của một Device
type Tag struct {
	ID          uuid.UUID `json:"id"          db:"id"`
	DeviceID    uuid.UUID `json:"device_id"   db:"device_id"`
	Name        string    `json:"name"        db:"name"`
	DataType    string    `json:"data_type"   db:"data_type"`
	Unit        *string   `json:"unit"        db:"unit"`
	Description *string   `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at"  db:"created_at"`
}

// ============================================
// Các models mới cho v1.1+
// ============================================
// AlertRule định nghĩa quy tắc cảnh báo cho một tag
type AlertRule struct {
	ID              uuid.UUID  `json:"id"               db:"id"`
	SiteID          uuid.UUID  `json:"site_id"          db:"site_id"`
	TagID           uuid.UUID  `json:"tag_id"           db:"tag_id"`
	Name            string     `json:"name"             db:"name"`
	Description     *string    `json:"description"      db:"description"`
	MinValue        *float64   `json:"min_value"        db:"min_value"`
	MaxValue        *float64   `json:"max_value"        db:"max_value"`
	Severity        string     `json:"severity"         db:"severity"` // INFO, WARNING, CRITICAL
	MessageTemplate *string    `json:"message_template" db:"message_template"`
	IsEnabled       bool       `json:"is_enabled"       db:"is_enabled"`
	CreatedAt       time.Time  `json:"created_at"       db:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"       db:"updated_at"`
}

// ControlConfig định nghĩa cấu hình điều khiển cho một tag
type ControlConfig struct {
	ID            uuid.UUID       `json:"id"             db:"id"`
	SiteID        uuid.UUID       `json:"site_id"        db:"site_id"`
	TagID         uuid.UUID       `json:"tag_id"         db:"tag_id"`
	ControlType   string          `json:"control_type"   db:"control_type"`   // ON_OFF, SET_VALUE, OPEN_CLOSE
	AllowedValues json.RawMessage `json:"allowed_values" db:"allowed_values"` // JSONB
	IsEnabled     bool            `json:"is_enabled"     db:"is_enabled"`
	CreatedAt     time.Time       `json:"created_at"     db:"created_at"`
	UpdatedAt     *time.Time      `json:"updated_at"     db:"updated_at"`
}

// ControlLog ghi nhận lệnh điều khiển thiết bị
type ControlLog struct {
	ID             uuid.UUID  `json:"id"               db:"id"`
	SiteID         uuid.UUID  `json:"site_id"          db:"site_id"`
	DeviceID       uuid.UUID  `json:"device_id"        db:"device_id"`
	UserID         uuid.UUID  `json:"user_id"          db:"user_id"`
	TagName        string     `json:"tag_name"         db:"tag_name"`
	RequestedValue string     `json:"requested_value"  db:"requested_value"`
	PreviousValue  *string    `json:"previous_value"   db:"previous_value"`
	Status         string     `json:"status"           db:"status"` // PENDING, SENT, SUCCESS, FAILED, TIMEOUT
	ErrorMessage   *string    `json:"error_message"    db:"error_message"`
	CreatedAt      time.Time  `json:"created_at"       db:"created_at"`
	SentAt         *time.Time `json:"sent_at"          db:"sent_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at"  db:"acknowledged_at"`
}

// AlertLog ghi nhận sự kiện cảnh báo
type AlertLog struct {
	ID             uuid.UUID  `json:"id"                db:"id"`
	SiteID         uuid.UUID  `json:"site_id"           db:"site_id"`
	DeviceID       *uuid.UUID `json:"device_id"         db:"device_id"` // nullable cho cảnh báo cấp site
	TagName        string     `json:"tag_name"          db:"tag_name"`
	TriggeredValue float64    `json:"triggered_value"   db:"triggered_value"`
	ThresholdValue float64    `json:"threshold_value"   db:"threshold_value"`
	Severity       string     `json:"severity"          db:"severity"` // INFO, WARNING, CRITICAL
	Message        string     `json:"message"           db:"message"`
	Status         string     `json:"status"            db:"status"` // ACTIVE, ACKNOWLEDGED, RESOLVED
	CreatedAt      time.Time  `json:"created_at"        db:"created_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at"   db:"acknowledged_at"`
	AcknowledgedBy *uuid.UUID `json:"acknowledged_by"   db:"acknowledged_by"`
	ResolvedAt     *time.Time `json:"resolved_at"       db:"resolved_at"`
}

// PidDiagram lưu sơ đồ công nghệ
type PidDiagram struct {
	ID            uuid.UUID  `json:"id"             db:"id"`
	SiteID        uuid.UUID  `json:"site_id"        db:"site_id"`
	Name          string     `json:"name"           db:"name"`
	BackgroundURL string     `json:"background_url" db:"background_url"`
	CreatedAt     time.Time  `json:"created_at"     db:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"     db:"updated_at"`
}

// PidWidget lưu widget trên sơ đồ
type PidWidget struct {
	ID         uuid.UUID `json:"id"          db:"id"`
	DiagramID  uuid.UUID `json:"diagram_id"  db:"diagram_id"`
	DeviceID   uuid.UUID `json:"device_id"   db:"device_id"`
	TagName    string    `json:"tag_name"    db:"tag_name"`
	PositionX  float64   `json:"position_x"  db:"position_x"`
	PositionY  float64   `json:"position_y"  db:"position_y"`
	WidgetType string    `json:"widget_type" db:"widget_type"` // TEXT, PUMP, VALVE
}

// AuditLog ghi nhận hành động người dùng
type AuditLog struct {
	ID             uuid.UUID       `json:"id"              db:"id"`
	SiteID         uuid.UUID       `json:"site_id"         db:"site_id"`
	UserID         uuid.UUID       `json:"user_id"         db:"user_id"`
	ActionType     string          `json:"action_type"     db:"action_type"`     // LOGIN, CREATE_DEVICE, UPDATE_THRESHOLD, DELETE_MEMBER
	ResourceTarget string          `json:"resource_target" db:"resource_target"` // tên bảng
	TargetID       *uuid.UUID      `json:"target_id"       db:"target_id"`
	Description    string          `json:"description"     db:"description"`
	OldValues      json.RawMessage `json:"old_values"      db:"old_values"` // JSONB
	NewValues      json.RawMessage `json:"new_values"      db:"new_values"` // JSONB
	IPAddress      string          `json:"ip_address"      db:"ip_address"`
	CreatedAt      time.Time       `json:"created_at"      db:"created_at"`
}

// SystemEventLog ghi nhận sự kiện hệ thống tự động
type SystemEventLog struct {
	ID            uuid.UUID  `json:"id"             db:"id"`
	SiteID        uuid.UUID  `json:"site_id"        db:"site_id"`
	DeviceID      *uuid.UUID `json:"device_id"      db:"device_id"`      // nullable nếu là lỗi toàn hệ thống
	EventType     string     `json:"event_type"     db:"event_type"`     // DEVICE_OFFLINE, GATEWAY_CONNECTED, FIRMWARE_UPDATE_START
	SeverityLevel string     `json:"severity_level" db:"severity_level"` // INFO, WARNING, ERROR
	Message       string     `json:"message"        db:"message"`
	CreatedAt     time.Time  `json:"created_at"     db:"created_at"`
}

// SiteRetentionPolicy lưu chính sách xóa dữ liệu
type SiteRetentionPolicy struct {
	ID                  uuid.UUID  `json:"id"                       db:"id"`
	SiteID              uuid.UUID  `json:"site_id"                  db:"site_id"`
	AuditLogsDays       int        `json:"audit_logs_days"          db:"audit_logs_days"`
	SystemEventLogsDays int        `json:"system_event_logs_days"   db:"system_event_logs_days"`
	AlertLogsDays       int        `json:"alert_logs_days"          db:"alert_logs_days"`
	TelemetryInfluxDays int        `json:"telemetry_influx_days"    db:"telemetry_influx_days"`
	UpdatedAt           *time.Time `json:"updated_at"               db:"updated_at"`
	UpdatedBy           *uuid.UUID `json:"updated_by"               db:"updated_by"`
}

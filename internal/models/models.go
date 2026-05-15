package models

import (
	"time"

	"github.com/google/uuid"
)

// Site đại diện cho một site (trước đây là tenant)
type Site struct {
	ID        uuid.UUID `json:"id"         db:"id"`
	Name      string    `json:"name"       db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
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

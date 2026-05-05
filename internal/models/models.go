package models

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID        uuid.UUID `json:"id"         db:"id"`
	Name      string    `json:"name"       db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type User struct {
	ID           uuid.UUID `json:"id"            db:"id"`
	TenantID     uuid.UUID `json:"tenant_id"     db:"tenant_id"`
	Email        string    `json:"email"         db:"email"`
	PasswordHash string    `json:"-"             db:"password_hash"`
	Role         string    `json:"role"          db:"role"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
}

type Device struct {
	ID            uuid.UUID  `json:"id"              db:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"       db:"tenant_id"`
	Name          string     `json:"name"            db:"name"`
	DeviceType    string     `json:"device_type"     db:"device_type"`
	MqttClientID  *string    `json:"mqtt_client_id"  db:"mqtt_client_id"`
	Status        string     `json:"status"          db:"status"`
	LastHeartbeat *time.Time `json:"last_heartbeat"  db:"last_heartbeat"`
	CreatedAt     time.Time  `json:"created_at"      db:"created_at"`
}

type Tag struct {
	ID          uuid.UUID `json:"id"          db:"id"`
	DeviceID    uuid.UUID `json:"device_id"   db:"device_id"`
	Name        string    `json:"name"        db:"name"`
	DataType    string    `json:"data_type"   db:"data_type"`
	Unit        *string   `json:"unit"        db:"unit"`
	Description *string   `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at"  db:"created_at"`
}

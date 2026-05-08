package api

import (
	"encoding/json"
	"net/http"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type DeviceHandler struct {
	store *postgres.Store
}

func NewDeviceHandler(store *postgres.Store) *DeviceHandler {
	return &DeviceHandler{store: store}
}

// ListDevices trả về tất cả devices thuộc site của user
func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT id, site_id, name, device_type, mqtt_client_id, status, last_heartbeat, created_at 
		 FROM devices WHERE site_id = $1`, membership.SiteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var d models.Device
		if err := rows.Scan(&d.ID, &d.SiteID, &d.Name, &d.DeviceType, &d.MqttClientID, &d.Status, &d.LastHeartbeat, &d.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		devices = append(devices, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

// Get trả về chi tiết một device
func (h *DeviceHandler) Get(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	siteID := membership.SiteID

	var d models.Device
	err = h.store.Pool.QueryRow(r.Context(),
		`SELECT id, site_id, name, device_type, mqtt_client_id, status, last_heartbeat, created_at 
		 FROM devices WHERE id = $1 AND site_id = $2`, id, siteID).
		Scan(&d.ID, &d.SiteID, &d.Name, &d.DeviceType, &d.MqttClientID, &d.Status, &d.LastHeartbeat, &d.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}

// Create tạo device mới
func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var input struct {
		Name         string `json:"name"`
		DeviceType   string `json:"device_type"`
		MqttClientID string `json:"mqtt_client_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	siteID := membership.SiteID

	var d models.Device
	err := h.store.Pool.QueryRow(r.Context(),
		`INSERT INTO devices (site_id, name, device_type, mqtt_client_id) 
		 VALUES ($1, $2, $3, NULLIF($4, '')) 
		 RETURNING id, site_id, name, device_type, mqtt_client_id, status, last_heartbeat, created_at`,
		siteID, input.Name, input.DeviceType, input.MqttClientID).
		Scan(&d.ID, &d.SiteID, &d.Name, &d.DeviceType, &d.MqttClientID, &d.Status, &d.LastHeartbeat, &d.CreatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(d)
}

// Update cập nhật device
func (h *DeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	siteID := membership.SiteID

	var input struct {
		Name       string `json:"name"`
		DeviceType string `json:"device_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	var d models.Device
	err = h.store.Pool.QueryRow(r.Context(),
		`UPDATE devices SET name=$1, device_type=$2 
		 WHERE id=$3 AND site_id=$4 
		 RETURNING id, site_id, name, device_type, mqtt_client_id, status, last_heartbeat, created_at`,
		input.Name, input.DeviceType, id, siteID).
		Scan(&d.ID, &d.SiteID, &d.Name, &d.DeviceType, &d.MqttClientID, &d.Status, &d.LastHeartbeat, &d.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"not found or update failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}

// Delete xóa device
func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	siteID := membership.SiteID

	_, err = h.store.Pool.Exec(r.Context(),
		"DELETE FROM devices WHERE id=$1 AND site_id=$2", id, siteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

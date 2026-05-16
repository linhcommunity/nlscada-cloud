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

type CreateDeviceRequest struct {
	Name         string `json:"name"`
	DeviceType   string `json:"device_type"`
	MqttClientID string `json:"mqtt_client_id,omitempty"`
}

type UpdateDeviceRequest struct {
	Name       string `json:"name"`
	DeviceType string `json:"device_type"`
}

// @Summary Danh sách thiết bị
// @Tags Devices
// @Produce json
// @Param siteID path string true "Site UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=[]models.Device} "Danh sách thiết bị"
// @Router /sites/{siteID}/devices [get]
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

// @Summary Chi tiết thiết bị
// @Tags Devices
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param deviceID path string true "Device UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.Device} "Chi tiết thiết bị"
// @Failure 404 {object} response.ErrorResponse "Không tìm thấy"
// @Router /sites/{siteID}/devices/{deviceID} [get]
func (h *DeviceHandler) Get(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "deviceID"))
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

// @Summary Tạo thiết bị
// @Tags Devices
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param request body CreateDeviceRequest true "Thông tin thiết bị"
// @Security BearerAuth
// @Success 201 {object} response.SuccessResponse{data=models.Device} "Thiết bị đã tạo"
// @Router /sites/{siteID}/devices [post]
func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var input CreateDeviceRequest
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

// @Summary Cập nhật thiết bị
// @Tags Devices
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param deviceID path string true "Device UUID"
// @Param request body UpdateDeviceRequest true "Thông tin cập nhật"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.Device} "Thiết bị đã cập nhật"
// @Failure 404 {object} response.ErrorResponse "Không tìm thấy"
// @Router /sites/{siteID}/devices/{deviceID} [put]
func (h *DeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "deviceID"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	siteID := membership.SiteID

	var input UpdateDeviceRequest
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

// @Summary Xóa thiết bị
// @Tags Devices
// @Param siteID path string true "Site UUID"
// @Param deviceID path string true "Device UUID"
// @Security BearerAuth
// @Success 204 "Đã xóa"
// @Failure 404 {object} response.ErrorResponse "Không tìm thấy"
// @Router /sites/{siteID}/devices/{deviceID} [delete]
func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "deviceID"))
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

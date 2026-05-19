package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/response"

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
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT id, site_id, name, device_type, mqtt_client_id, status, last_heartbeat, created_at 
		 FROM devices WHERE site_id = $1`, membership.SiteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", "Truy vấn thất bại")
		return
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var d models.Device
		if err := rows.Scan(&d.ID, &d.SiteID, &d.Name, &d.DeviceType, &d.MqttClientID, &d.Status, &d.LastHeartbeat, &d.CreatedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Tạo thiết bị thất bại")
			return
		}
		devices = append(devices, d)
	}

	response.ListJSON(w, http.StatusOK, devices, 1, len(devices), int64(len(devices)))
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
	log.Println("Get device called, URL:", r.URL.Path)
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "deviceID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_DEVICE_ID", "ID thiết bị không hợp lệ")
		return
	}
	siteID := membership.SiteID
	log.Printf("SiteID: %s, DeviceID: %s", id, siteID)
	var d models.Device
	err = h.store.Pool.QueryRow(r.Context(),
		`SELECT id, site_id, name, device_type, mqtt_client_id, status, last_heartbeat, created_at 
		 FROM devices WHERE id = $1 AND site_id = $2`, id, siteID).
		Scan(&d.ID, &d.SiteID, &d.Name, &d.DeviceType, &d.MqttClientID, &d.Status, &d.LastHeartbeat, &d.CreatedAt)
	if err != nil {
		response.Error(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Thiết bị không tồn tại")
		return
	}

	response.JSON(w, http.StatusOK, d)
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
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	var input CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu yêu cầu không hợp lệ")
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
		response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", "Truy vấn thất bại")
		return
	}

	response.JSON(w, http.StatusCreated, d)
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
// @Failure 401 {object} response.ErrorResponse "Bạn không có quyền truy cập tài nguyên này"
// @Failure 400 {object} response.ErrorResponse "ID thiết bị không hợp lệ"
// @Failure 500 {object} response.ErrorResponse "Cập nhật thiết bị thất bại"
// @Router /sites/{siteID}/devices/{deviceID} [put]
func (h *DeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		fmt.Print("Lỗi 1")
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này") // code : 401
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "deviceID"))
	if err != nil {
		fmt.Print("Lỗi 2")
		response.Error(w, http.StatusBadRequest, "INVALID_DEVICE_ID", "ID thiết bị không hợp lệ")
		return
	}

	siteID := membership.SiteID

	var input UpdateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		fmt.Print("Lỗi 3")
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu yêu cầu không hợp lệ") // code : 400
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
		fmt.Print("Lỗi 4: ", err)
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Cập nhật thiết bị thất bại") // code : 500
		return
	}

	response.JSON(w, http.StatusOK, d)
}

// @Summary Xóa thiết bị
// @Tags Devices
// @Param siteID path string true "Site UUID"
// @Param deviceID path string true "Device UUID"
// @Security BearerAuth
// @Success 204 "Đã xóa"
// @Failure 404 {object} response.ErrorResponse "Không tìm thấy"
// @Failure 500 {object} response.ErrorResponse "Xóa thiết bị thất bại"
// @Router /sites/{siteID}/devices/{deviceID} [delete]
func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này") // code : 401
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "deviceID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_DEVICE_ID", "ID thiết bị không hợp lệ") // code : 400
		return
	}

	siteID := membership.SiteID

	_, err = h.store.Pool.Exec(r.Context(),
		"DELETE FROM devices WHERE id=$1 AND site_id=$2", id, siteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "SCAN_FAILED", "Quét dữ liệu thất bại") // code : 500
		return
	}

	response.JSON(w, http.StatusOK, nil)
}

// @Summary Test API
// @Tags Devices
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=string} "Test thành công"
// @Router /sites/{siteID}/devices/test [get]
func (h *DeviceHandler) Test(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"message": "Test thành công"})
}

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"nlscada-cloud/internal/control"
	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/mqtt"
	"nlscada-cloud/internal/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ControlHandler xử lý các endpoint điều khiển thiết bị
type ControlHandler struct {
	store          *postgres.Store
	mqttClient     *mqtt.Client
	controlService *control.Service
}

// ControlRequest là body gửi lệnh điều khiển
type ControlRequest struct {
	TagName string `json:"tag_name"`
	Value   string `json:"value"`
}

type controlRequest = ControlRequest

// NewControlHandler tạo instance mới
func NewControlHandler(store *postgres.Store, mqttClient *mqtt.Client, controlService *control.Service) *ControlHandler {
	return &ControlHandler{store: store, mqttClient: mqttClient}
}

// @Summary Gửi lệnh điều khiển
// @Tags Control
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param deviceID path string true "Device UUID"
// @Param request body ControlRequest true "Lệnh điều khiển"
// @Security BearerAuth
// @Success 201 {object} response.SuccessResponse{data=models.ControlLog} "Lệnh đã gửi"
// @Failure 403 {object} response.ErrorResponse "Không có quyền"
// @Router /sites/{siteID}/devices/{deviceID}/control [post]
func (h *ControlHandler) Send(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	deviceID, err := uuid.Parse(chi.URLParam(r, "deviceID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_DEVICE_ID", "ID thiết bị không hợp lệ")
		return
	}

	// Kiểm tra device thuộc site
	var deviceSiteID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(),
		"SELECT site_id FROM devices WHERE id = $1", deviceID).Scan(&deviceSiteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "INVALID_DEVICE_ID", "Thiết bị không tồn tại")
		return
	}
	if deviceSiteID != membership.SiteID {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Thiết bị không thuộc site của bạn")
		return
	}

	// Parse request body
	var input controlRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Dữ liệu yêu cầu không hợp lệ")
		return
	}
	if input.TagName == "" || input.Value == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Tên tag và giá trị là bắt buộc")
		return
	}

	// Kiểm tra tag có tồn tại và thuộc device không
	var tagID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(),
		"SELECT id FROM tags WHERE device_id = $1 AND name = $2", deviceID, input.TagName).Scan(&tagID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "INVALID_TAG_NAME", "Tag không tồn tại trên thiết bị này")
		return
	}

	// Kiểm tra control_config (nếu có bảng) - tạm thời bỏ qua nếu chưa có dữ liệu
	// TODO: Kiểm tra control_config cho tag này

	// Tạo control log
	var logEntry models.ControlLog
	err = h.store.Pool.QueryRow(r.Context(),
		`INSERT INTO control_logs (site_id, device_id, user_id, tag_name, requested_value, status)
		 VALUES ($1, $2, $3, $4, $5, 'PENDING')
		 RETURNING id, site_id, device_id, user_id, tag_name, requested_value, status, created_at`,
		membership.SiteID, deviceID, GetClaims(r).UserID, input.TagName, input.Value,
	).Scan(&logEntry.ID, &logEntry.SiteID, &logEntry.DeviceID, &logEntry.UserID,
		&logEntry.TagName, &logEntry.RequestedValue, &logEntry.Status, &logEntry.CreatedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "FAILED_TO_CREATE_CONTROL_LOG", "Không thể tạo nhật ký điều khiển")
		return
	}
	// Sau khi tạo control_log thành công:
	logEntryID := logEntry.ID.String()

	// Publish MQTT command trong goroutine
	if h.mqttClient != nil {
		go func() {
			topic := fmt.Sprintf("site/%s/device/%s/cmd", membership.SiteID.String(), deviceID.String())
			payload := fmt.Sprintf(`{"log_id":"%s","tag_name":"%s","value":"%s","timestamp":%d}`,
				logEntryID, input.TagName, input.Value, time.Now().Unix())
			h.mqttClient.Publish(topic, 1, payload)
			log.Printf("Published MQTT command: topic=%s payload=%s", topic, payload)

			// Cập nhật trạng thái SENT
			_, err := h.store.Pool.Exec(context.Background(),
				"UPDATE control_logs SET status = 'SENT', sent_at = NOW() WHERE id = $1", logEntry.ID)
			if err != nil {
				log.Printf("Failed to update control_log status to SENT: %v", err)
			}
		}()
	}

	response.JSON(w, http.StatusCreated, logEntry)
}

// @Summary Lịch sử lệnh điều khiển
// @Tags Control
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param deviceID path string true "Device UUID"
// @Param page query int false "Số trang"
// @Param limit query int false "Số bản ghi/trang"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=[]models.ControlLog} "Danh sách lệnh"
// @Router /sites/{siteID}/devices/{deviceID}/control/logs [get]
func (h *ControlHandler) Logs(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	deviceID, err := uuid.Parse(chi.URLParam(r, "deviceID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_DEVICE_ID", "ID thiết bị không hợp lệ")
		return
	}

	// Parse query params
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT id, site_id, device_id, user_id, tag_name, requested_value, previous_value, 
		        status, error_message, created_at, sent_at, acknowledged_at
		 FROM control_logs
		 WHERE site_id = $1 AND device_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3 OFFSET $4`, membership.SiteID, deviceID, limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", "Truy vấn thất bại")
		return
	}
	defer rows.Close()

	var logs []models.ControlLog
	for rows.Next() {
		var l models.ControlLog
		if err := rows.Scan(&l.ID, &l.SiteID, &l.DeviceID, &l.UserID, &l.TagName,
			&l.RequestedValue, &l.PreviousValue, &l.Status, &l.ErrorMessage,
			&l.CreatedAt, &l.SentAt, &l.AcknowledgedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "SCAN_FAILED", "Scan dữ liệu thất bại")
			return
		}
		logs = append(logs, l)
	}

	response.JSONWithPagination(w, http.StatusOK, logs, page, limit, int64(len(logs))) // total count có thể thêm sau
}

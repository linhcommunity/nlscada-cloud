package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AlertHandler xử lý các endpoint cảnh báo
type AlertHandler struct {
	store *postgres.Store
}

// NewAlertHandler tạo instance mới
func NewAlertHandler(store *postgres.Store) *AlertHandler {
	return &AlertHandler{store: store}
}

// @Summary Danh sách cảnh báo
// @Tags Alerts
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param status query string false "ACTIVE, ACKNOWLEDGED, RESOLVED"
// @Param severity query string false "INFO, WARNING, CRITICAL"
// @Param device_id query string false "Lọc theo device UUID"
// @Param page query int false "Số trang"
// @Param limit query int false "Số bản ghi/trang"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=[]models.AlertLog} "Danh sách cảnh báo"
// @Router /sites/{siteID}/alerts [get]
func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
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

	statusFilter := r.URL.Query().Get("status")
	severityFilter := r.URL.Query().Get("severity")
	deviceIDFilter := r.URL.Query().Get("device_id")

	// Build query động
	query := `SELECT id, site_id, device_id, tag_name, triggered_value, threshold_value, 
	          severity, message, status, created_at, acknowledged_at, acknowledged_by, resolved_at
	          FROM alert_logs WHERE site_id = $1`
	args := []interface{}{membership.SiteID}
	argIdx := 2

	if statusFilter != "" {
		query += ` AND status = $` + strconv.Itoa(argIdx)
		args = append(args, statusFilter)
		argIdx++
	}
	if severityFilter != "" {
		query += ` AND severity = $` + strconv.Itoa(argIdx)
		args = append(args, severityFilter)
		argIdx++
	}
	if deviceIDFilter != "" {
		query += ` AND device_id = $` + strconv.Itoa(argIdx)
		args = append(args, deviceIDFilter)
		argIdx++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, limit, offset)

	rows, err := h.store.Pool.Query(r.Context(), query, args...)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", "Truy vấn thất bại")
		return
	}
	defer rows.Close()

	var alerts []models.AlertLog
	for rows.Next() {
		var a models.AlertLog
		if err := rows.Scan(&a.ID, &a.SiteID, &a.DeviceID, &a.TagName, &a.TriggeredValue,
			&a.ThresholdValue, &a.Severity, &a.Message, &a.Status, &a.CreatedAt,
			&a.AcknowledgedAt, &a.AcknowledgedBy, &a.ResolvedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "SCAN_FAILED", "Quét dữ liệu thất bại")
			return
		}
		alerts = append(alerts, a)
	}

	response.JSONWithPagination(w, http.StatusOK, alerts, page, limit, 0)
}

type CreateAlertRequest struct {
	DeviceID       *string `json:"device_id"` // optional
	TagName        string  `json:"tag_name"`
	TriggeredValue float64 `json:"triggered_value"`
	ThresholdValue float64 `json:"threshold_value"`
	Severity       string  `json:"severity"`
	Message        string  `json:"message"`
}

// @Summary Tạo cảnh báo thủ công
// @Tags Alerts
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param request body CreateAlertRequest true "Thông tin cảnh báo"
// @Security BearerAuth
// @Success 201 {object} response.SuccessResponse{data=models.AlertLog} "Cảnh báo đã tạo"
// @Failure 400 {object} response.ErrorResponse "Dữ liệu không hợp lệ"
// @Router /sites/{siteID}/alerts [post]
func (h *AlertHandler) Create(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var input CreateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Thêm yêu cầu không hợp lệ")
		return
	}

	// Validate severity
	if input.Severity != "INFO" && input.Severity != "WARNING" && input.Severity != "CRITICAL" {
		response.Error(w, http.StatusBadRequest, "INVALID_SEVERITY", "Mức độ nghiêm trọng không hợp lệ")
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Thêm yêu cầu không hợp lệ")
		return
	}

	// Validate severity
	if input.Severity != "INFO" && input.Severity != "WARNING" && input.Severity != "CRITICAL" {
		response.Error(w, http.StatusBadRequest, "INVALID_SEVERITY", "Mức độ nghiêm trọng không hợp lệ")
		return
	}

	var deviceID *uuid.UUID
	if input.DeviceID != nil && *input.DeviceID != "" {
		id, err := uuid.Parse(*input.DeviceID)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_DEVICE_ID", "ID thiết bị không hợp lệ")
			return
		}
		deviceID = &id
	}

	var a models.AlertLog
	err := h.store.Pool.QueryRow(r.Context(),
		`INSERT INTO alert_logs (site_id, device_id, tag_name, triggered_value, threshold_value, severity, message, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'ACTIVE')
		 RETURNING id, site_id, device_id, tag_name, triggered_value, threshold_value, severity, message, status, created_at`,
		membership.SiteID, deviceID, input.TagName, input.TriggeredValue, input.ThresholdValue,
		input.Severity, input.Message,
	).Scan(&a.ID, &a.SiteID, &a.DeviceID, &a.TagName, &a.TriggeredValue, &a.ThresholdValue,
		&a.Severity, &a.Message, &a.Status, &a.CreatedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Tạo cảnh báo thất bại")
		return
	}

	response.JSON(w, http.StatusCreated, a)
}

type UpdateAlertRequest struct {
	Message  *string `json:"message,omitempty"`
	Severity *string `json:"severity,omitempty"`
}

// @Summary Cập nhật cảnh báo
// @Tags Alerts
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param alertID path string true "Alert UUID"
// @Param request body UpdateAlertRequest true "Các trường cần cập nhật"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.AlertLog} "Cảnh báo đã cập nhật"
// @Router /sites/{siteID}/alerts/{alertID} [put]
func (h *AlertHandler) Update(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	alertID, err := uuid.Parse(chi.URLParam(r, "alertID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ALERT_ID", "ID cảnh báo không hợp lệ")
		return
	}

	// Kiểm tra alert thuộc site
	var siteID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(),
		"SELECT site_id FROM alert_logs WHERE id = $1", alertID).Scan(&siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "ALERT_NOT_FOUND", "Cảnh báo không tồn tại")
		return
	}
	if siteID != membership.SiteID {
		response.Error(w, http.StatusForbidden, "ALERT_NOT_FOUND", "Cảnh báo không thuộc về nhà máy của bạn")
		return
	}

	var input UpdateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Thân yêu cầu không hợp lệ")
		return
	}

	// Build update query động
	query := "UPDATE alert_logs SET "
	args := []interface{}{}
	argIdx := 1

	if input.Message != nil {
		query += `message = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.Message)
		argIdx++
	}
	if input.Severity != nil {
		query += `severity = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.Severity)
		argIdx++
	}
	// Không có trường nào để update
	if len(args) == 0 {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Không có trường nào để cập nhật")
		return
	}
	// Bỏ dấu phẩy cuối
	query = query[:len(query)-2]
	query += ` WHERE id = $` + strconv.Itoa(argIdx) + ` RETURNING id, site_id, device_id, tag_name, triggered_value, threshold_value, severity, message, status, created_at, acknowledged_at, acknowledged_by, resolved_at`
	args = append(args, alertID)

	var a models.AlertLog
	err = h.store.Pool.QueryRow(r.Context(), query, args...).Scan(
		&a.ID, &a.SiteID, &a.DeviceID, &a.TagName, &a.TriggeredValue, &a.ThresholdValue,
		&a.Severity, &a.Message, &a.Status, &a.CreatedAt, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.ResolvedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Cập nhật cảnh báo thất bại")
		return
	}

	response.JSON(w, http.StatusOK, a)
}

// @Summary Xóa cảnh báo
// @Tags Alerts
// @Param siteID path string true "Site UUID"
// @Param alertID path string true "Alert UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse "Cảnh báo đã xóa"
// @Router /sites/{siteID}/alerts/{alertID} [delete]
func (h *AlertHandler) Delete(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	alertID, err := uuid.Parse(chi.URLParam(r, "alertID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ALERT_ID", "ID cảnh báo không hợp lệ")
		return
	}

	// Kiểm tra alert thuộc site
	var siteID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(),
		"SELECT site_id FROM alert_logs WHERE id = $1", alertID).Scan(&siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "ALERT_NOT_FOUND", "Cảnh báo không tồn tại")
		return
	}
	if siteID != membership.SiteID {
		response.Error(w, http.StatusForbidden, "ALERT_NOT_FOUND", "Cảnh báo không thuộc về nhà máy của bạn")
		return
	}

	_, err = h.store.Pool.Exec(r.Context(), "DELETE FROM alert_logs WHERE id = $1", alertID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", "Xóa cảnh báo thất bại")
		return
	}

	response.JSON(w, http.StatusOK, nil)
}

// @Summary Xác nhận cảnh báo
// @Tags Alerts
// @Param siteID path string true "Site UUID"
// @Param alertID path string true "Alert UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse "Đã xác nhận"
// @Router /sites/{siteID}/alerts/{alertID}/acknowledge [post]
func (h *AlertHandler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	alertID, err := uuid.Parse(chi.URLParam(r, "alertID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ALERT_ID", "ID cảnh báo không hợp lệ")
		return
	}

	// Kiểm tra alert thuộc site
	var siteID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(),
		"SELECT site_id FROM alert_logs WHERE id = $1", alertID).Scan(&siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "ALERT_NOT_FOUND", "Cảnh báo không tồn tại")
		return
	}
	if siteID != membership.SiteID {
		response.Error(w, http.StatusForbidden, "ALERT_NOT_FOUND", "Cảnh báo không thuộc về nhà máy của bạn")
		return
	}

	now := time.Now()
	userID := GetClaims(r).UserID

	_, err = h.store.Pool.Exec(r.Context(),
		`UPDATE alert_logs SET status = 'ACKNOWLEDGED', acknowledged_at = $1, acknowledged_by = $2
		 WHERE id = $3 AND status = 'ACTIVE'`, now, userID, alertID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "ACKNOWLEDGE_FAILED", "Xác nhận cảnh báo thất bại")
		return
	}

	response.JSON(w, http.StatusOK, nil)
}

// @Summary Đánh dấu đã giải quyết
// @Tags Alerts
// @Param siteID path string true "Site UUID"
// @Param alertID path string true "Alert UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse "Đã giải quyết"
// @Router /sites/{siteID}/alerts/{alertID}/resolve [post]
func (h *AlertHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	alertID, err := uuid.Parse(chi.URLParam(r, "alertID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ALERT_ID", "ID cảnh báo không hợp lệ")
		return
	}

	// Kiểm tra alert thuộc site
	var siteID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(),
		"SELECT site_id FROM alert_logs WHERE id = $1", alertID).Scan(&siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "ALERT_NOT_FOUND", "Cảnh báo không tồn tại")
		return
	}
	if siteID != membership.SiteID {
		response.Error(w, http.StatusForbidden, "ALERT_NOT_FOUND", "Cảnh báo không thuộc về nhà máy của bạn")
		return
	}

	now := time.Now()
	_, err = h.store.Pool.Exec(r.Context(),
		`UPDATE alert_logs SET status = 'RESOLVED', resolved_at = $1
		 WHERE id = $2 AND status IN ('ACTIVE', 'ACKNOWLEDGED')`, now, alertID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "RESOLVE_FAILED", "Đánh dấu cảnh báo đã giải quyết thất bại")
		return
	}

	response.JSON(w, http.StatusOK, nil)
}

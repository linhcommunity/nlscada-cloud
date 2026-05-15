package api

import (
	"net/http"
	"strconv"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// SystemEventHandler xử lý các endpoint nhật ký sự kiện hệ thống
type SystemEventHandler struct {
	store *postgres.Store
}

// NewSystemEventHandler tạo instance mới
func NewSystemEventHandler(store *postgres.Store) *SystemEventHandler {
	return &SystemEventHandler{store: store}
}

// List trả về danh sách sự kiện hệ thống của site
func (h *SystemEventHandler) List(w http.ResponseWriter, r *http.Request) {
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

	deviceIDFilter := r.URL.Query().Get("device_id")
	eventTypeFilter := r.URL.Query().Get("event_type")
	severityFilter := r.URL.Query().Get("severity")

	// Build query động
	query := `SELECT id, site_id, device_id, event_type, severity_level, message, created_at
	          FROM system_event_logs WHERE site_id = $1`
	args := []interface{}{membership.SiteID}
	argIdx := 2

	if deviceIDFilter != "" {
		query += ` AND device_id = $` + strconv.Itoa(argIdx)
		args = append(args, deviceIDFilter)
		argIdx++
	}
	if eventTypeFilter != "" {
		query += ` AND event_type = $` + strconv.Itoa(argIdx)
		args = append(args, eventTypeFilter)
		argIdx++
	}
	if severityFilter != "" {
		query += ` AND severity_level = $` + strconv.Itoa(argIdx)
		args = append(args, severityFilter)
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

	var events []models.SystemEventLog
	for rows.Next() {
		var e models.SystemEventLog
		if err := rows.Scan(&e.ID, &e.SiteID, &e.DeviceID, &e.EventType, &e.SeverityLevel, &e.Message, &e.CreatedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "SCAN_FAILED", "Quét dữ liệu thất bại")
			return
		}
		events = append(events, e)
	}

	response.JSONWithPagination(w, http.StatusOK, events, page, limit, 0)
}

// Get trả về chi tiết một sự kiện hệ thống
func (h *SystemEventHandler) Get(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_EVENT_ID", "ID sự kiện không hợp lệ")
		return
	}

	var e models.SystemEventLog
	err = h.store.Pool.QueryRow(r.Context(),
		`SELECT id, site_id, device_id, event_type, severity_level, message, created_at
		 FROM system_event_logs WHERE id = $1 AND site_id = $2`, eventID, membership.SiteID,
	).Scan(&e.ID, &e.SiteID, &e.DeviceID, &e.EventType, &e.SeverityLevel, &e.Message, &e.CreatedAt)
	if err != nil {
		response.Error(w, http.StatusNotFound, "SYSTEM_EVENT_NOT_FOUND", "Sự kiện hệ thống không tìm thấy")
		return
	}

	response.JSON(w, http.StatusOK, e)
}

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

// AuditLogHandler xử lý các endpoint nhật ký hoạt động người dùng
type AuditLogHandler struct {
	store *postgres.Store
}

// NewAuditLogHandler tạo instance mới
func NewAuditLogHandler(store *postgres.Store) *AuditLogHandler {
	return &AuditLogHandler{store: store}
}

// List trả về danh sách audit logs của site (có phân trang và filter)
func (h *AuditLogHandler) List(w http.ResponseWriter, r *http.Request) {
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

	userIDFilter := r.URL.Query().Get("user_id")
	actionTypeFilter := r.URL.Query().Get("action_type")

	// Build query động
	query := `SELECT id, site_id, user_id, action_type, resource_target, target_id, 
	          description, old_values, new_values, ip_address, created_at
	          FROM audit_logs WHERE site_id = $1`
	args := []interface{}{membership.SiteID}
	argIdx := 2

	if userIDFilter != "" {
		query += ` AND user_id = $` + strconv.Itoa(argIdx)
		args = append(args, userIDFilter)
		argIdx++
	}
	if actionTypeFilter != "" {
		query += ` AND action_type = $` + strconv.Itoa(argIdx)
		args = append(args, actionTypeFilter)
		argIdx++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, limit, offset)

	rows, err := h.store.Pool.Query(r.Context(), query, args...)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", "Truy vấn nhật ký thất bại")
		return
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		if err := rows.Scan(&l.ID, &l.SiteID, &l.UserID, &l.ActionType, &l.ResourceTarget,
			&l.TargetID, &l.Description, &l.OldValues, &l.NewValues, &l.IPAddress, &l.CreatedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "SCAN_FAILED", "Scan nhật ký thất bại")
			return
		}
		logs = append(logs, l)
	}

	response.JSONWithPagination(w, http.StatusOK, logs, page, limit, 0)
}

// Get trả về chi tiết một audit log
func (h *AuditLogHandler) Get(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	logID, err := uuid.Parse(chi.URLParam(r, "logID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_LOG_ID", "ID nhật ký không hợp lệ")
		return
	}

	var l models.AuditLog
	err = h.store.Pool.QueryRow(r.Context(),
		`SELECT id, site_id, user_id, action_type, resource_target, target_id, 
		        description, old_values, new_values, ip_address, created_at
		 FROM audit_logs WHERE id = $1 AND site_id = $2`, logID, membership.SiteID,
	).Scan(&l.ID, &l.SiteID, &l.UserID, &l.ActionType, &l.ResourceTarget,
		&l.TargetID, &l.Description, &l.OldValues, &l.NewValues, &l.IPAddress, &l.CreatedAt)
	if err != nil {
		response.Error(w, http.StatusNotFound, "AUDIT_LOG_NOT_FOUND", "Nhật ký hoạt động không tồn tại")
		return
	}

	response.JSON(w, http.StatusOK, l)
}

package api

import (
	"encoding/json"
	"net/http"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/response"
)

// RetentionPolicyHandler xử lý các endpoint cấu hình tự động xóa dữ liệu
type RetentionPolicyHandler struct {
	store *postgres.Store
}

// NewRetentionPolicyHandler tạo instance mới
func NewRetentionPolicyHandler(store *postgres.Store) *RetentionPolicyHandler {
	return &RetentionPolicyHandler{store: store}
}

// Get trả về cấu hình retention policy của site
func (h *RetentionPolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var policy models.SiteRetentionPolicy
	err := h.store.Pool.QueryRow(r.Context(),
		`SELECT id, site_id, audit_logs_days, system_event_logs_days, alert_logs_days, 
		        telemetry_influx_days, updated_at, updated_by
		 FROM site_retention_policies WHERE site_id = $1`, membership.SiteID,
	).Scan(&policy.ID, &policy.SiteID, &policy.AuditLogsDays, &policy.SystemEventLogsDays,
		&policy.AlertLogsDays, &policy.TelemetryInfluxDays, &policy.UpdatedAt, &policy.UpdatedBy)
	if err != nil {
		// Nếu chưa có cấu hình, trả về giá trị mặc định
		policy = models.SiteRetentionPolicy{
			SiteID:              membership.SiteID,
			AuditLogsDays:       90,
			SystemEventLogsDays: 90,
			AlertLogsDays:       180,
			TelemetryInfluxDays: 30,
		}
	}

	response.JSON(w, http.StatusOK, policy)
}

// Update cập nhật cấu hình retention policy
func (h *RetentionPolicyHandler) Update(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var input struct {
		AuditLogsDays       *int `json:"audit_logs_days,omitempty"`
		SystemEventLogsDays *int `json:"system_event_logs_days,omitempty"`
		AlertLogsDays       *int `json:"alert_logs_days,omitempty"`
		TelemetryInfluxDays *int `json:"telemetry_influx_days,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu đầu vào không hợp lệ")
		return
	}

	// Kiểm tra giá trị hợp lệ
	if input.AuditLogsDays != nil && *input.AuditLogsDays < 1 {
		response.Error(w, http.StatusBadRequest, "INVALID_INPUT", "Giữa lại Logs kiểm toán phải lớn hơn hoặc bằng 1 ngày")
		return
	}
	if input.SystemEventLogsDays != nil && *input.SystemEventLogsDays < 1 {
		response.Error(w, http.StatusBadRequest, "INVALID_INPUT", "Giữa lại System Event Logs phải lớn hơn hoặc bằng 1 ngày")
		return
	}
	if input.AlertLogsDays != nil && *input.AlertLogsDays < 1 {
		response.Error(w, http.StatusBadRequest, "INVALID_INPUT", "Giữa lại Alert Logs phải lớn hơn hoặc bằng 1 ngày")
		return
	}
	if input.TelemetryInfluxDays != nil && *input.TelemetryInfluxDays < 1 {
		response.Error(w, http.StatusBadRequest, "INVALID_INPUT", "Giữa lại Đo xa phải lớn hơn hoặc bằng 1 ngày")
		return
	}

	userID := GetClaims(r).UserID

	// Upsert: INSERT ... ON CONFLICT UPDATE
	var policy models.SiteRetentionPolicy
	err := h.store.Pool.QueryRow(r.Context(),
		`INSERT INTO site_retention_policies (site_id, audit_logs_days, system_event_logs_days, alert_logs_days, telemetry_influx_days, updated_by)
		 VALUES ($1, 
		         COALESCE($2, 90), 
		         COALESCE($3, 90), 
		         COALESCE($4, 180), 
		         COALESCE($5, 30), 
		         $6)
		 ON CONFLICT (site_id) DO UPDATE SET
		     audit_logs_days = COALESCE($2, site_retention_policies.audit_logs_days),
		     system_event_logs_days = COALESCE($3, site_retention_policies.system_event_logs_days),
		     alert_logs_days = COALESCE($4, site_retention_policies.alert_logs_days),
		     telemetry_influx_days = COALESCE($5, site_retention_policies.telemetry_influx_days),
		     updated_at = NOW(),
		     updated_by = $6
		 RETURNING id, site_id, audit_logs_days, system_event_logs_days, alert_logs_days, telemetry_influx_days, updated_at, updated_by`,
		membership.SiteID, input.AuditLogsDays, input.SystemEventLogsDays, input.AlertLogsDays, input.TelemetryInfluxDays, userID,
	).Scan(&policy.ID, &policy.SiteID, &policy.AuditLogsDays, &policy.SystemEventLogsDays,
		&policy.AlertLogsDays, &policy.TelemetryInfluxDays, &policy.UpdatedAt, &policy.UpdatedBy)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "FAILED_TO_UPDATE_RETENTION_POLICY", "Không thể cập nhật chính sách giữ dữ liệu")
		return
	}

	response.JSON(w, http.StatusOK, policy)
}

// TriggerManual kích hoạt dọn dẹp dữ liệu cũ ngay lập tức
func (h *RetentionPolicyHandler) TriggerManual(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	// TODO: Chạy tác vụ dọn dẹp bất đồng bộ
	// go func() {
	//     cleanOldData(h.store, membership.SiteID)
	// }()

	response.JSON(w, http.StatusAccepted, nil)
}
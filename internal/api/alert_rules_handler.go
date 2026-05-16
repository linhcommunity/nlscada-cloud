package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/mqtt"
	"nlscada-cloud/internal/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AlertRuleHandler struct {
	store      *postgres.Store
	mqttClient *mqtt.Client
}

func NewAlertRuleHandler(store *postgres.Store, mqttClient *mqtt.Client) *AlertRuleHandler {
	return &AlertRuleHandler{store: store, mqttClient: mqttClient}
}

// @Summary Danh sách alert rules
// @Tags Alert Rules
// @Produce json
// @Param siteID path string true "Site UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=[]models.AlertRule} "Danh sách rule"
// @Router /sites/{siteID}/alert-rules [get]
func (h *AlertRuleHandler) List(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT id, site_id, tag_id, name, description, min_value, max_value,
		        severity, message_template, is_enabled, created_at, updated_at
		 FROM alert_rules WHERE site_id = $1 ORDER BY created_at DESC`, membership.SiteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Truy vấn thất bại")
		return
	}
	defer rows.Close()

	var rules []models.AlertRule
	for rows.Next() {
		var r models.AlertRule
		if err := rows.Scan(&r.ID, &r.SiteID, &r.TagID, &r.Name, &r.Description,
			&r.MinValue, &r.MaxValue, &r.Severity, &r.MessageTemplate,
			&r.IsEnabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Quyét dữ liệu thất bại")
			return
		}
		rules = append(rules, r)
	}

	response.JSON(w, http.StatusOK, rules)
}

type CreateAlertRuleRequest struct {
	TagID           string   `json:"tag_id"`
	Name            string   `json:"name"`
	Description     *string  `json:"description,omitempty"`
	MinValue        *float64 `json:"min_value,omitempty"`
	MaxValue        *float64 `json:"max_value,omitempty"`
	Severity        string   `json:"severity"`
	MessageTemplate *string  `json:"message_template,omitempty"`
	IsEnabled       *bool    `json:"is_enabled,omitempty"`
}

// @Summary Tạo alert rule
// @Tags Alert Rules
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param request body CreateAlertRuleRequest true "Thông tin rule"
// @Security BearerAuth
// @Success 201 {object} response.SuccessResponse{data=models.AlertRule} "Rule đã tạo"
// @Failure 400 {object} response.ErrorResponse "Dữ liệu không hợp lệ"
// @Router /sites/{siteID}/alert-rules [post]
func (h *AlertRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var input CreateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Data vào yêu cầu không hợp lệ")
		return
	}
	if input.Severity != "INFO" && input.Severity != "WARNING" && input.Severity != "CRITICAL" {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Thiết lập mức độ nghiêm trọng không hợp lệ")
		return
	}
	tagID, err := uuid.Parse(input.TagID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "ID thẻ không hợp lệ")
		return
	}

	isEnabled := true
	if input.IsEnabled != nil {
		isEnabled = *input.IsEnabled
	}

	var rule models.AlertRule
	err = h.store.Pool.QueryRow(r.Context(),
		`INSERT INTO alert_rules (site_id, tag_id, name, description, min_value, max_value, severity, message_template, is_enabled)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, site_id, tag_id, name, description, min_value, max_value, severity, message_template, is_enabled, created_at, updated_at`,
		membership.SiteID, tagID, input.Name, input.Description, input.MinValue, input.MaxValue,
		input.Severity, input.MessageTemplate, isEnabled,
	).Scan(&rule.ID, &rule.SiteID, &rule.TagID, &rule.Name, &rule.Description,
		&rule.MinValue, &rule.MaxValue, &rule.Severity, &rule.MessageTemplate,
		&rule.IsEnabled, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Tạo vai trò cảnh báo thất bại")
		return
	}

	h.publishConfig(membership.SiteID)
	response.JSON(w, http.StatusCreated, rule)
}

// @Summary Chi tiết alert rule
// @Tags Alert Rules
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param ruleID path string true "Rule UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.AlertRule}
// @Failure 404 {object} response.ErrorResponse "Rule không tồn tại"
// @Router /sites/{siteID}/alert-rules/{ruleID} [get]
func (h *AlertRuleHandler) Get(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	ruleID, err := uuid.Parse(chi.URLParam(r, "ruleID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "ID quy tắc cảnh báo không hợp lệ")
		return
	}

	var rule models.AlertRule
	err = h.store.Pool.QueryRow(r.Context(),
		`SELECT id, site_id, tag_id, name, description, min_value, max_value,
		        severity, message_template, is_enabled, created_at, updated_at
		 FROM alert_rules WHERE id = $1 AND site_id = $2`, ruleID, membership.SiteID,
	).Scan(&rule.ID, &rule.SiteID, &rule.TagID, &rule.Name, &rule.Description,
		&rule.MinValue, &rule.MaxValue, &rule.Severity, &rule.MessageTemplate,
		&rule.IsEnabled, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		response.Error(w, http.StatusNotFound, "UNAUTHORIZED", "Quy tắc cảnh báo không tìm thấy")
		return
	}

	response.JSON(w, http.StatusOK, rule)
}

type UpdateAlertRuleRequest struct {
	Name            *string  `json:"name,omitempty"`
	Description     *string  `json:"description,omitempty"`
	MinValue        *float64 `json:"min_value,omitempty"`
	MaxValue        *float64 `json:"max_value,omitempty"`
	Severity        *string  `json:"severity,omitempty"`
	MessageTemplate *string  `json:"message_template,omitempty"`
	IsEnabled       *bool    `json:"is_enabled,omitempty"`
}

// @Summary Cập nhật alert rule
// @Tags Alert Rules
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param ruleID path string true "Rule UUID"
// @Param request body UpdateAlertRuleRequest true "Các trường cần cập nhật"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.AlertRule} "Rule đã cập nhật"
// @Failure 400 {object} response.ErrorResponse "Không có gì để cập nhật"
// @Router /sites/{siteID}/alert-rules/{ruleID} [put]
func (h *AlertRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	ruleID, err := uuid.Parse(chi.URLParam(r, "ruleID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "ID quy tắc cảnh báo không hợp lệ")
		return
	}

	var input UpdateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Data vào yêu cầu không hợp lệ")
		return
	}

	// Build dynamic update
	query := "UPDATE alert_rules SET "
	args := []interface{}{}
	argIdx := 1
	if input.Name != nil {
		query += `name = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.Name)
		argIdx++
	}
	if input.Description != nil {
		query += `description = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.Description)
		argIdx++
	}
	if input.MinValue != nil {
		query += `min_value = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.MinValue)
		argIdx++
	}
	if input.MaxValue != nil {
		query += `max_value = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.MaxValue)
		argIdx++
	}
	if input.Severity != nil {
		query += `severity = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.Severity)
		argIdx++
	}
	if input.MessageTemplate != nil {
		query += `message_template = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.MessageTemplate)
		argIdx++
	}
	if input.IsEnabled != nil {
		query += `is_enabled = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.IsEnabled)
		argIdx++
	}
	if len(args) == 0 {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Không có gì để cập nhật")
		return
	}
	query = query[:len(query)-2] + `, updated_at = NOW() WHERE id = $` + strconv.Itoa(argIdx) + ` AND site_id = $` + strconv.Itoa(argIdx+1) + ` RETURNING id, site_id, tag_id, name, description, min_value, max_value, severity, message_template, is_enabled, created_at, updated_at`
	args = append(args, ruleID, membership.SiteID)

	var rule models.AlertRule
	err = h.store.Pool.QueryRow(r.Context(), query, args...).Scan(
		&rule.ID, &rule.SiteID, &rule.TagID, &rule.Name, &rule.Description,
		&rule.MinValue, &rule.MaxValue, &rule.Severity, &rule.MessageTemplate,
		&rule.IsEnabled, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Cập nhật quy tắc cảnh báo thất bại")
		return
	}

	h.publishConfig(membership.SiteID)
	response.JSON(w, http.StatusOK, rule)
}

// @Summary Xóa alert rule
// @Tags Alert Rules
// @Param siteID path string true "Site UUID"
// @Param ruleID path string true "Rule UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse "Rule đã xóa"
// @Router /sites/{siteID}/alert-rules/{ruleID} [delete]
func (h *AlertRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	ruleID, err := uuid.Parse(chi.URLParam(r, "ruleID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "ID quy tắc cảnh báo không hợp lệ")
		return
	}

	_, err = h.store.Pool.Exec(r.Context(), "DELETE FROM alert_rules WHERE id = $1 AND site_id = $2", ruleID, membership.SiteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Xóa quy tắc cảnh báo thất bại")
		return
	}

	h.publishConfig(membership.SiteID)
	response.JSON(w, http.StatusOK, nil)
}

// publishConfig gửi cấu hình mới xuống Gateway qua MQTT
func (h *AlertRuleHandler) publishConfig(siteID uuid.UUID) {
	if h.mqttClient == nil {
		return
	}
	topic := "site/" + siteID.String() + "/config"
	h.mqttClient.Publish(topic, 1, `{"type":"alert_rules_updated"}`)
}

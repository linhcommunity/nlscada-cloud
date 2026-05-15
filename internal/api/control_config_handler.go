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

type ControlConfigHandler struct {
	store      *postgres.Store
	mqttClient *mqtt.Client
}

func NewControlConfigHandler(store *postgres.Store, mqttClient *mqtt.Client) *ControlConfigHandler {
	return &ControlConfigHandler{store: store, mqttClient: mqttClient}
}

// List trả về danh sách control config của site
func (h *ControlConfigHandler) List(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT id, site_id, tag_id, control_type, allowed_values, is_enabled, created_at, updated_at
		 FROM control_config WHERE site_id = $1 ORDER BY created_at DESC`, membership.SiteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	defer rows.Close()

	var configs []models.ControlConfig
	for rows.Next() {
		var c models.ControlConfig
		if err := rows.Scan(&c.ID, &c.SiteID, &c.TagID, &c.ControlType,
			&c.AllowedValues, &c.IsEnabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "LỖI NỘI BỘ", "Lỗi quyét dữ liệu")
			return
		}
		configs = append(configs, c)
	}

	response.JSON(w, http.StatusOK, configs)
}

// Create tạo control config mới
func (h *ControlConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var input struct {
		TagID         string           `json:"tag_id"`
		ControlType   string           `json:"control_type"`
		AllowedValues *json.RawMessage `json:"allowed_values,omitempty"`
		IsEnabled     *bool            `json:"is_enabled,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Data vào yêu cầu không hợp lệ")
		return
	}
	tagID, err := uuid.Parse(input.TagID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Tag ID không hợp lệ")
		return
	}

	isEnabled := true
	if input.IsEnabled != nil {
		isEnabled = *input.IsEnabled
	}

	var cfg models.ControlConfig
	err = h.store.Pool.QueryRow(r.Context(),
		`INSERT INTO control_config (site_id, tag_id, control_type, allowed_values, is_enabled)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, site_id, tag_id, control_type, allowed_values, is_enabled, created_at, updated_at`,
		membership.SiteID, tagID, input.ControlType, input.AllowedValues, isEnabled,
	).Scan(&cfg.ID, &cfg.SiteID, &cfg.TagID, &cfg.ControlType,
		&cfg.AllowedValues, &cfg.IsEnabled, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LỖI NỘI BỘ", "Failed to create control config")
		return
	}

	h.publishConfig(membership.SiteID)
	response.JSON(w, http.StatusCreated, cfg)
}

// Get trả về chi tiết một control config
func (h *ControlConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	configID, err := uuid.Parse(chi.URLParam(r, "configID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BADREQUEST", "Config ID không hợp lệ")
		return
	}

	var cfg models.ControlConfig
	err = h.store.Pool.QueryRow(r.Context(),
		`SELECT id, site_id, tag_id, control_type, allowed_values, is_enabled, created_at, updated_at
		 FROM control_config WHERE id = $1 AND site_id = $2`, configID, membership.SiteID,
	).Scan(&cfg.ID, &cfg.SiteID, &cfg.TagID, &cfg.ControlType,
		&cfg.AllowedValues, &cfg.IsEnabled, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOTFOUND", "Control config not found")
		return
	}

	response.JSON(w, http.StatusOK, cfg)
}

// Update cập nhật control config
func (h *ControlConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	configID, err := uuid.Parse(chi.URLParam(r, "configID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BADREQUEST", "Config ID không hợp lệ")
		return
	}

	var input struct {
		ControlType   *string          `json:"control_type,omitempty"`
		AllowedValues *json.RawMessage `json:"allowed_values,omitempty"`
		IsEnabled     *bool            `json:"is_enabled,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "BADREQUEST", "Data vào yêu cầu không hợp lệ")
		return
	}

	query := "UPDATE control_config SET "
	args := []interface{}{}
	argIdx := 1
	if input.ControlType != nil {
		query += `control_type = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.ControlType)
		argIdx++
	}
	if input.AllowedValues != nil {
		query += `allowed_values = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.AllowedValues)
		argIdx++
	}
	if input.IsEnabled != nil {
		query += `is_enabled = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.IsEnabled)
		argIdx++
	}
	if len(args) == 0 {
		response.Error(w, http.StatusBadRequest, "BADREQUEST", "Không có dữ liệu để cập nhật")
		return
	}
	query = query[:len(query)-2] + `, updated_at = NOW() WHERE id = $` + strconv.Itoa(argIdx) + ` AND site_id = $` + strconv.Itoa(argIdx+1) + ` RETURNING id, site_id, tag_id, control_type, allowed_values, is_enabled, created_at, updated_at`
	args = append(args, configID, membership.SiteID)

	var cfg models.ControlConfig
	err = h.store.Pool.QueryRow(r.Context(), query, args...).Scan(
		&cfg.ID, &cfg.SiteID, &cfg.TagID, &cfg.ControlType,
		&cfg.AllowedValues, &cfg.IsEnabled, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LỖI NỘI BỘ", "Failed to update control config")
		return
	}

	h.publishConfig(membership.SiteID)
	response.JSON(w, http.StatusOK, cfg)
}

// Delete xóa control config
func (h *ControlConfigHandler) Delete(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	configID, err := uuid.Parse(chi.URLParam(r, "configID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BADREQUEST", "Config ID không hợp lệ")
		return
	}

	_, err = h.store.Pool.Exec(r.Context(), "DELETE FROM control_config WHERE id = $1 AND site_id = $2", configID, membership.SiteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LỖI NỘI BỘ", "Failed to delete control config")
		return
	}

	h.publishConfig(membership.SiteID)
	response.JSON(w, http.StatusOK, nil)
}

// publishConfig gửi cấu hình mới xuống Gateway qua MQTT
func (h *ControlConfigHandler) publishConfig(siteID uuid.UUID) {
	if h.mqttClient == nil {
		return
	}
	topic := "site/" + siteID.String() + "/config"
	h.mqttClient.Publish(topic, 1, `{"type":"control_config_updated"}`)
}

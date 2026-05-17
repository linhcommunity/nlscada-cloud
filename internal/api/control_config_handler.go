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

type CreateControlConfigRequest struct {
	TagID         string           `json:"tag_id"`
	ControlType   string           `json:"control_type"`
	AllowedValues *json.RawMessage `json:"allowed_values,omitempty" swaggertype:"object"`
	IsEnabled     *bool            `json:"is_enabled,omitempty"`
}

type UpdateControlConfigRequest struct {
	ControlType   *string          `json:"control_type,omitempty"`
	AllowedValues *json.RawMessage `json:"allowed_values,omitempty" swaggertype:"object"`
	IsEnabled     *bool            `json:"is_enabled,omitempty"`
}

// @Summary Danh sách control config
// @Tags Control Config
// @Produce json
// @Param siteID path string true "Site UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=[]models.ControlConfig} "Danh sách config"
// @Router /sites/{siteID}/control-configs [get]
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
		response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", "Truy vấn thất bại")
		return
	}
	defer rows.Close()

	var configs []models.ControlConfig
	for rows.Next() {
		var c models.ControlConfig
		if err := rows.Scan(&c.ID, &c.SiteID, &c.TagID, &c.ControlType,
			&c.AllowedValues, &c.IsEnabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "SCAN_FAILED", "Lỗi quét dữ liệu")
			return
		}
		configs = append(configs, c)
	}

	response.ListJSON(w, http.StatusOK, configs, 1, len(configs), int64(len(configs)))
}

// @Summary Tạo control config
// @Tags Control Config
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param request body CreateControlConfigRequest true "Thông tin config"
// @Security BearerAuth
// @Success 201 {object} response.SuccessResponse{data=models.ControlConfig}
// @Router /sites/{siteID}/control-configs [post]
func (h *ControlConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var input CreateControlConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu yêu cầu không hợp lệ")
		return
	}
	tagID, err := uuid.Parse(input.TagID)
	if err != nil {
		response.ValidationError(w, []response.ValidationErrorDetail{
			{Field: "tag_id", Issue: "ID thẻ không hợp lệ"},
		})
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
		response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Tạo cấu hình điều khiển thất bại")
		return
	}

	h.publishConfig(membership.SiteID)
	response.JSON(w, http.StatusCreated, cfg)
}

// @Summary Chi tiết control config
// @Tags Control Config
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param configID path string true "Config UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.ControlConfig}
// @Router /sites/{siteID}/control-configs/{configID} [get]
func (h *ControlConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	configID, err := uuid.Parse(chi.URLParam(r, "configID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_CONFIG_ID", "ID cấu hình không hợp lệ")
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

// @Summary Cập nhật control config
// @Tags Control Config
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param configID path string true "Config UUID"
// @Param request body UpdateControlConfigRequest true "Các trường cần cập nhật"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.ControlConfig}
// @Router /sites/{siteID}/control-configs/{configID} [put]
func (h *ControlConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	configID, err := uuid.Parse(chi.URLParam(r, "configID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_CONFIG_ID", "ID cấu hình không hợp lệ")
		return
	}

	var input UpdateControlConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu yêu cầu không hợp lệ")
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
		response.Error(w, http.StatusBadRequest, "NO_FIELDS_TO_UPDATE", "Không có trường nào để cập nhật")
		return
	}
	query = query[:len(query)-2] + `, updated_at = NOW() WHERE id = $` + strconv.Itoa(argIdx) + ` AND site_id = $` + strconv.Itoa(argIdx+1) + ` RETURNING id, site_id, tag_id, control_type, allowed_values, is_enabled, created_at, updated_at`
	args = append(args, configID, membership.SiteID)

	var cfg models.ControlConfig
	err = h.store.Pool.QueryRow(r.Context(), query, args...).Scan(
		&cfg.ID, &cfg.SiteID, &cfg.TagID, &cfg.ControlType,
		&cfg.AllowedValues, &cfg.IsEnabled, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Cập nhật cấu hình điều khiển thất bại")
		return
	}

	h.publishConfig(membership.SiteID)
	response.JSON(w, http.StatusOK, cfg)
}

// @Summary Xóa control config
// @Tags Control Config
// @Param siteID path string true "Site UUID"
// @Param configID path string true "Config UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Router /sites/{siteID}/control-configs/{configID} [delete]
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
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", "Xóa cấu hình điều khiển thất bại")
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

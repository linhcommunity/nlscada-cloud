package api

import (
	"net/http"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/response"

	"github.com/google/uuid"
)

type SiteConfigHandler struct {
	store *postgres.Store
}

func NewSiteConfigHandler(store *postgres.Store) *SiteConfigHandler {
	return &SiteConfigHandler{store: store}
}

// @Summary Cấu hình tổng hợp cho Gateway
// @Tags Site Config
// @Produce json
// @Param siteID path string true "Site UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse "Cấu hình đầy đủ"
// @Router /sites/{siteID}/config [get]
func (h *SiteConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	siteID := membership.SiteID

	// Lấy devices
	devices, _ := h.getDevices(r, siteID)
	// Lấy tags
	tags, _ := h.getTags(r, siteID)
	// Lấy alert rules
	alertRules, _ := h.getAlertRules(r, siteID)
	// Lấy control configs
	controlConfigs, _ := h.getControlConfigs(r, siteID)

	config := map[string]interface{}{
		"devices":         devices,
		"tags":            tags,
		"alert_rules":     alertRules,
		"control_configs": controlConfigs,
	}

	response.JSON(w, http.StatusOK, config)
}

func (h *SiteConfigHandler) getDevices(r *http.Request, siteID uuid.UUID) ([]models.Device, error) {
	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT id, site_id, name, device_type, mqtt_client_id, status, last_heartbeat, created_at
		 FROM devices WHERE site_id = $1`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var devices []models.Device
	for rows.Next() {
		var d models.Device
		if err := rows.Scan(&d.ID, &d.SiteID, &d.Name, &d.DeviceType,
			&d.MqttClientID, &d.Status, &d.LastHeartbeat, &d.CreatedAt); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func (h *SiteConfigHandler) getTags(r *http.Request, siteID uuid.UUID) ([]models.Tag, error) {
	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT t.id, t.device_id, t.name, t.data_type, t.unit, t.description, t.created_at
		 FROM tags t
		 JOIN devices d ON t.device_id = d.id
		 WHERE d.site_id = $1`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.DeviceID, &t.Name, &t.DataType,
			&t.Unit, &t.Description, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

func (h *SiteConfigHandler) getAlertRules(r *http.Request, siteID uuid.UUID) ([]models.AlertRule, error) {
	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT id, site_id, tag_id, name, description, min_value, max_value, severity, message_template, is_enabled, created_at, updated_at
		 FROM alert_rules WHERE site_id = $1 AND is_enabled = true`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []models.AlertRule
	for rows.Next() {
		var rule models.AlertRule
		if err := rows.Scan(&rule.ID, &rule.SiteID, &rule.TagID, &rule.Name,
			&rule.Description, &rule.MinValue, &rule.MaxValue, &rule.Severity,
			&rule.MessageTemplate, &rule.IsEnabled, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (h *SiteConfigHandler) getControlConfigs(r *http.Request, siteID uuid.UUID) ([]models.ControlConfig, error) {
	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT id, site_id, tag_id, control_type, allowed_values, is_enabled, created_at, updated_at
		 FROM control_config WHERE site_id = $1 AND is_enabled = true`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var configs []models.ControlConfig
	for rows.Next() {
		var c models.ControlConfig
		if err := rows.Scan(&c.ID, &c.SiteID, &c.TagID, &c.ControlType,
			&c.AllowedValues, &c.IsEnabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, nil
}

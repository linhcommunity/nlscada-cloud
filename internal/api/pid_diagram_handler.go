package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// PidDiagramHandler xử lý các endpoint quản lý sơ đồ P&ID
type PidDiagramHandler struct {
	store *postgres.Store
}

// NewPidDiagramHandler tạo instance mới
func NewPidDiagramHandler(store *postgres.Store) *PidDiagramHandler {
	return &PidDiagramHandler{store: store}
}

type CreateDiagramRequest struct {
	Name          string `json:"name"`
	BackgroundURL string `json:"background_url"`
}

type AddWidgetRequest struct {
	DeviceID   string  `json:"device_id"`
	TagName    string  `json:"tag_name"`
	PositionX  float64 `json:"position_x"`
	PositionY  float64 `json:"position_y"`
	WidgetType string  `json:"widget_type"`
}

type UpdateWidgetRequest struct {
	DeviceID   *string  `json:"device_id,omitempty"`
	TagName    *string  `json:"tag_name,omitempty"`
	PositionX  *float64 `json:"position_x,omitempty"`
	PositionY  *float64 `json:"position_y,omitempty"`
	WidgetType *string  `json:"widget_type,omitempty"`
}

// @Summary Tạo sơ đồ P&ID
// @Tags P&ID
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param request body CreateDiagramRequest true "Thông tin sơ đồ"
// @Security BearerAuth
// @Success 201 {object} response.SuccessResponse{data=models.PidDiagram}
// @Router /sites/{siteID}/pid-diagrams [post]
func (h *PidDiagramHandler) Create(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var input CreateDiagramRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	if input.Name == "" || input.BackgroundURL == "" {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var diagram models.PidDiagram
	err := h.store.Pool.QueryRow(r.Context(),
		`INSERT INTO pid_diagrams (site_id, name, background_url) 
		 VALUES ($1, $2, $3) 
		 RETURNING id, site_id, name, background_url, created_at, updated_at`,
		membership.SiteID, input.Name, input.BackgroundURL,
	).Scan(&diagram.ID, &diagram.SiteID, &diagram.Name, &diagram.BackgroundURL, &diagram.CreatedAt, &diagram.UpdatedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	response.JSON(w, http.StatusCreated, diagram)
}

// @Summary Danh sách sơ đồ P&ID
// @Tags P&ID
// @Produce json
// @Param siteID path string true "Site UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=[]models.PidDiagram}
// @Router /sites/{siteID}/pid-diagrams [get]
func (h *PidDiagramHandler) List(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT id, site_id, name, background_url, created_at, updated_at 
		 FROM pid_diagrams WHERE site_id = $1 ORDER BY created_at DESC`, membership.SiteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	defer rows.Close()

	var diagrams []models.PidDiagram
	for rows.Next() {
		var d models.PidDiagram
		if err := rows.Scan(&d.ID, &d.SiteID, &d.Name, &d.BackgroundURL, &d.CreatedAt, &d.UpdatedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
			return
		}
		diagrams = append(diagrams, d)
	}

	response.JSON(w, http.StatusOK, diagrams)
}

// @Summary Chi tiết sơ đồ P&ID
// @Tags P&ID
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param diagramID path string true "Diagram UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.PidDiagram}
// @Router /sites/{siteID}/pid-diagrams/{diagramID} [get]
func (h *PidDiagramHandler) Get(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	diagramID, err := uuid.Parse(chi.URLParam(r, "diagramID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var d models.PidDiagram
	err = h.store.Pool.QueryRow(r.Context(),
		`SELECT id, site_id, name, background_url, created_at, updated_at 
		 FROM pid_diagrams WHERE id = $1 AND site_id = $2`, diagramID, membership.SiteID,
	).Scan(&d.ID, &d.SiteID, &d.Name, &d.BackgroundURL, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		response.Error(w, http.StatusNotFound, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	response.JSON(w, http.StatusOK, d)
}

// @Summary Xóa sơ đồ P&ID
// @Tags P&ID
// @Param siteID path string true "Site UUID"
// @Param diagramID path string true "Diagram UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Router /sites/{siteID}/pid-diagrams/{diagramID} [delete]
func (h *PidDiagramHandler) Delete(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	diagramID, err := uuid.Parse(chi.URLParam(r, "diagramID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	_, err = h.store.Pool.Exec(r.Context(),
		"DELETE FROM pid_diagrams WHERE id = $1 AND site_id = $2", diagramID, membership.SiteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	response.JSON(w, http.StatusOK, nil)
}

// @Summary Thêm widget vào sơ đồ
// @Tags P&ID
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param diagramID path string true "Diagram UUID"
// @Param request body AddWidgetRequest true "Thông tin widget"
// @Security BearerAuth
// @Success 201 {object} response.SuccessResponse{data=models.PidWidget}
// @Router /sites/{siteID}/pid-diagrams/{diagramID}/widgets [post]
func (h *PidDiagramHandler) AddWidget(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	diagramID, err := uuid.Parse(chi.URLParam(r, "diagramID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var input AddWidgetRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	if input.WidgetType != "TEXT" && input.WidgetType != "PUMP" && input.WidgetType != "VALVE" {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	deviceID, err := uuid.Parse(input.DeviceID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var widget models.PidWidget
	err = h.store.Pool.QueryRow(r.Context(),
		`INSERT INTO pid_widgets (diagram_id, device_id, tag_name, position_x, position_y, widget_type)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, diagram_id, device_id, tag_name, position_x, position_y, widget_type`,
		diagramID, deviceID, input.TagName, input.PositionX, input.PositionY, input.WidgetType,
	).Scan(&widget.ID, &widget.DiagramID, &widget.DeviceID, &widget.TagName, &widget.PositionX, &widget.PositionY, &widget.WidgetType)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	response.JSON(w, http.StatusCreated, widget)
}

// @Summary Cập nhật widget
// @Tags P&ID
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param diagramID path string true "Diagram UUID"
// @Param widgetID path string true "Widget UUID"
// @Param request body UpdateWidgetRequest true "Các trường cần cập nhật"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.PidWidget}
// @Router /sites/{siteID}/pid-diagrams/{diagramID}/widgets/{widgetID} [put]
func (h *PidDiagramHandler) UpdateWidget(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	widgetID, err := uuid.Parse(chi.URLParam(r, "widgetID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var input UpdateWidgetRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	// Build update query động
	query := "UPDATE pid_widgets SET "
	args := []interface{}{}
	argIdx := 1

	if input.DeviceID != nil {
		query += `device_id = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.DeviceID)
		argIdx++
	}
	if input.TagName != nil {
		query += `tag_name = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.TagName)
		argIdx++
	}
	if input.PositionX != nil {
		query += `position_x = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.PositionX)
		argIdx++
	}
	if input.PositionY != nil {
		query += `position_y = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.PositionY)
		argIdx++
	}
	if input.WidgetType != nil {
		query += `widget_type = $` + strconv.Itoa(argIdx) + `, `
		args = append(args, *input.WidgetType)
		argIdx++
	}
	if len(args) == 0 {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	query = query[:len(query)-2]
	query += ` WHERE id = $` + strconv.Itoa(argIdx) + ` RETURNING id, diagram_id, device_id, tag_name, position_x, position_y, widget_type`
	args = append(args, widgetID)

	var widget models.PidWidget
	err = h.store.Pool.QueryRow(r.Context(), query, args...).Scan(
		&widget.ID, &widget.DiagramID, &widget.DeviceID, &widget.TagName, &widget.PositionX, &widget.PositionY, &widget.WidgetType)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	response.JSON(w, http.StatusOK, widget)
}

// @Summary Xóa widget
// @Tags P&ID
// @Param siteID path string true "Site UUID"
// @Param diagramID path string true "Diagram UUID"
// @Param widgetID path string true "Widget UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Router /sites/{siteID}/pid-diagrams/{diagramID}/widgets/{widgetID} [delete]
func (h *PidDiagramHandler) DeleteWidget(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	widgetID, err := uuid.Parse(chi.URLParam(r, "widgetID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	_, err = h.store.Pool.Exec(r.Context(), "DELETE FROM pid_widgets WHERE id = $1", widgetID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	response.JSON(w, http.StatusOK, nil)
}

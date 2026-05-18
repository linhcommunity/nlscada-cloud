package api

import (
	"encoding/json"
	"net/http"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TagHandler struct {
	store *postgres.Store
}

func NewTagHandler(store *postgres.Store) *TagHandler {
	return &TagHandler{store: store}
}

type CreateTagRequest struct {
	Name        string `json:"name"`
	DataType    string `json:"data_type"`
	Unit        string `json:"unit,omitempty"`
	Description string `json:"description,omitempty"`
}

type UpdateTagRequest struct {
	Name        string `json:"name,omitempty"`
	DataType    string `json:"data_type,omitempty"`
	Unit        string `json:"unit,omitempty"`
	Description string `json:"description,omitempty"`
}

// @Summary Danh sách tag của thiết bị
// @Tags Tags
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param deviceID path string true "Device UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=[]models.Tag} "Danh sách tag"
// @Router /sites/{siteID}/devices/{deviceID}/tags [get]
func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
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

	// Kiểm tra device thuộc site của user
	var siteID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(), "SELECT site_id FROM devices WHERE id = $1", deviceID).Scan(&siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Thiết bị không tồn tại")
		return
	}

	if siteID != membership.SiteID {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Thiết bị không thuộc site của bạn")
		return
	}

	rows, err := h.store.Pool.Query(r.Context(),
		"SELECT id, device_id, name, data_type, unit, description, created_at FROM tags WHERE device_id=$1", deviceID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", "Truy vấn thất bại")
		return
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.DeviceID, &t.Name, &t.DataType, &t.Unit, &t.Description, &t.CreatedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", "Xóa tag thất bại")
			return
		}
		tags = append(tags, t)
	}

	response.ListJSON(w, http.StatusOK, tags, 1, len(tags), int64(len(tags)))
}

// @Summary Tạo tag mới
// @Tags Tags
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param deviceID path string true "Device UUID"
// @Param request body CreateTagRequest true "Thông tin tag"
// @Security BearerAuth
// @Success 201 {object} response.SuccessResponse{data=models.Tag} "Tag đã tạo"
// @Router /sites/{siteID}/devices/{deviceID}/tags [post]
func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	// Kiểm tra device thuộc site của user
	var siteID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(), "SELECT site_id FROM devices WHERE id = $1", deviceID).Scan(&siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Thiết bị không tồn tại")
		return
	}

	if siteID != membership.SiteID {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Thiết bị không thuộc site của bạn")
		return
	}

	var input CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu yêu cầu không hợp lệ")
		return
	}

	var t models.Tag
	err = h.store.Pool.QueryRow(r.Context(),
		`INSERT INTO tags (device_id, name, data_type, unit, description) 
		 VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, '')) 
		 RETURNING id, device_id, name, data_type, unit, description, created_at`,
		deviceID, input.Name, input.DataType, input.Unit, input.Description).
		Scan(&t.ID, &t.DeviceID, &t.Name, &t.DataType, &t.Unit, &t.Description, &t.CreatedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "SCAN_FAILED", "Quét dữ liệu thất bại")
		return
	}

	response.JSON(w, http.StatusCreated, t)
}

// @Summary Cập nhật tag
// @Tags Tags
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param tagID path string true "Tag UUID"
// @Param request body UpdateTagRequest true "Các trường cần cập nhật"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.Tag} "Tag đã cập nhật"
// @Router /sites/{siteID}/tags/{tagID} [put]
func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "tagID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_TAG_ID", "ID tag không hợp lệ")
		return
	}

	// Lấy device_id của tag và kiểm tra quyền
	var deviceID, siteID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(),
		"SELECT t.device_id, d.site_id FROM tags t JOIN devices d ON t.device_id = d.id WHERE t.id = $1", id).
		Scan(&deviceID, &siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "TAG_NOT_FOUND", "Tag không tồn tại")
		return
	}

	if siteID != membership.SiteID {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Tag không thuộc site của bạn")
		return
	}

	var input UpdateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu yêu cầu không hợp lệ")
		return
	}

	var t models.Tag
	err = h.store.Pool.QueryRow(r.Context(),
		`UPDATE tags SET 
		    name = COALESCE(NULLIF($1, ''), name),
		    data_type = COALESCE(NULLIF($2, ''), data_type),
		    unit = CASE WHEN $3::text IS NOT NULL THEN $3 ELSE unit END,
		    description = CASE WHEN $4::text IS NOT NULL THEN $4 ELSE description END
		 WHERE id=$5 
		 RETURNING id, device_id, name, data_type, unit, description, created_at`,
		input.Name, input.DataType, input.Unit, input.Description, id).
		Scan(&t.ID, &t.DeviceID, &t.Name, &t.DataType, &t.Unit, &t.Description, &t.CreatedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Tạo tag thất bại")
		return
	}

	response.JSON(w, http.StatusOK, t)
}

// @Summary Xóa tag
// @Tags Tags
// @Param siteID path string true "Site UUID"
// @Param tagID path string true "Tag UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Router /sites/{siteID}/tags/{tagID} [delete]
func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "tagID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_TAG_ID", "ID tag không hợp lệ")
		return
	}

	// Kiểm tra quyền sở hữu tag
	var siteID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(),
		"SELECT d.site_id FROM tags t JOIN devices d ON t.device_id = d.id WHERE t.id = $1", id).
		Scan(&siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "TAG_NOT_FOUND", "Tag không tồn tại")
		return
	}
	if siteID != membership.SiteID {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Tag không thuộc site của bạn")
		return
	}

	_, err = h.store.Pool.Exec(r.Context(), "DELETE FROM tags WHERE id=$1", id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Cập nhật tag thất bại")
		return
	}

	response.JSON(w, http.StatusOK, nil)
}

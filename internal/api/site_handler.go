package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/response"

	"github.com/google/uuid"
)

type SiteHandler struct {
	store *postgres.Store
}

func NewSiteHandler(store *postgres.Store) *SiteHandler {
	return &SiteHandler{store: store}
}

type CreateSiteRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type UpdateSiteRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// @Summary Tạo site mới
// @Description Tạo site mới và user hiện tại trở thành admin của site đó
// @Tags Sites
// @Accept json
// @Produce json
// @Param request body CreateSiteRequest true "Tên site"
// @Success 201 {object} response.SuccessResponse{data=models.Site} "Site đã tạo"
// @Failure 400 {object} response.ErrorResponse "Dữ liệu không hợp lệ"
// @Security BearerAuth
// @Router /sites [post]
func (h *SiteHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r) // chứa userID
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var input CreateSiteRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu yêu cầu không hợp lệ")
		return
	}
	if input.Name == "" { // loại trừ tên rỗng
		response.Error(w, http.StatusBadRequest, "NAME_REQUIRED", "Tên site là bắt buộc")
		return
	}

	ctx := r.Context()
	tx, err := h.store.Pool.Begin(ctx) // bắt đầu transaction
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Lỗi nội bộ")
		return
	}
	defer tx.Rollback(ctx)

	// 1. Tạo site
	var siteID uuid.UUID
	err = tx.QueryRow(ctx, "INSERT INTO sites (name, description) VALUES ($1, $2) RETURNING id", input.Name, input.Description).Scan(&siteID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			response.Error(w, http.StatusConflict, "SITE_NAME_EXISTS", "Tên site đã tồn tại")
		} else {
			response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Tạo site thất bại")
		}
		return
	}
	// siteID lấy từ database, đảm bảo là UUID hợp lệ nên không cần parse lại
	fmt.Printf("Created site with ID: %s\n, name: %s", siteID, input.Name)
	// 2. Tạo membership (admin) cho user
	_, err = tx.Exec(ctx, "INSERT INTO memberships (user_id, site_id, role) VALUES ($1, $2, 'admin')", claims.UserID, siteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Tạo thành viên thất bại")
		return
	}
	fmt.Printf("Created membership for user %s as admin of site %s\n", claims.UserID, siteID)

	if err := tx.Commit(ctx); err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Lỗi nội bộ")
		return
	}

	// Lấy thông tin site vừa tạo để trả về
	var s models.Site
	err = h.store.Pool.QueryRow(ctx, "SELECT id, name, description created_at FROM sites WHERE id = $1", siteID).Scan(&s.ID, &s.Name, &s.Description, &s.CreatedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Lỗi nội bộ")
		return
	}

	resp := struct {
		models.Site
		Role string `json:"role"`
	}{
		Site: s,
		Role: "admin",
	}

	fmt.Printf("Site created successfully: %+v\n", resp)

	response.JSON(w, http.StatusCreated, resp)
}

// @Summary Danh sách site của user
// @Description Trả về tất cả site mà user hiện tại là thành viên, kèm role
// @Tags Sites
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=[]models.Site} "Danh sách site"
// @Failure 401 {object} response.ErrorResponse "Chưa đăng nhập"
// @Security BearerAuth
// @Router /sites [get]
func (h *SiteHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT s.id, s.name, s.description, s.created_at, m.role 
		 FROM sites s 
		 INNER JOIN memberships m ON s.id = m.site_id 
		 WHERE m.user_id = $1`, claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", "Truy vấn thất bại")
		return
	}
	defer rows.Close()

	type SiteInfo struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		CreatedAt   time.Time `json:"created_at"`
		Role        string    `json:"role"`
	}

	var sites []SiteInfo
	for rows.Next() {
		var s SiteInfo
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.CreatedAt, &s.Role); err != nil {
			response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", "Xóa site thất bại")
			return
		}
		sites = append(sites, s)
	}

	response.ListJSON(w, http.StatusOK, sites, 1, len(sites), int64(len(sites)))
}

// @Summary Cập nhật site
// @Tags Sites
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param request body UpdateSiteRequest true "Tên mới"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.Site} "Site đã cập nhật"
// @Failure 403 {object} response.ErrorResponse "Không có quyền admin"
// @Router /sites/{siteID}/ [put]
func (h *SiteHandler) Update(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	if membership.Role != "admin" {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Bạn không có quyền admin")
		return
	}

	var input UpdateSiteRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu yêu cầu không hợp lệ")
		return
	}
	if input.Name == "" {
		response.Error(w, http.StatusBadRequest, "NAME_REQUIRED", "Tên site là bắt buộc")
		return
	}

	var s models.Site
	err := h.store.Pool.QueryRow(r.Context(),
		"UPDATE sites SET name = $1, description = $2 WHERE id = $3 RETURNING id, name, description, created_at",
		input.Name, input.Description, membership.SiteID,
	).Scan(&s.ID, &s.Name, &s.Description, &s.CreatedAt)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", "Cập nhật site thất bại")
		return
	}
	response.JSON(w, http.StatusOK, s)
}

// @Summary Xóa site
// @Tags Sites
// @Param siteID path string true "Site UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse "Site đã xóa"
// @Failure 403 {object} response.ErrorResponse "Không có quyền admin"
// @Router /sites/{siteID}/ [delete]
func (h *SiteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	if membership.Role != "admin" {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Bạn không có quyền admin")
		return
	}

	_, err := h.store.Pool.Exec(r.Context(),
		"DELETE FROM sites WHERE id=$1", membership.SiteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "SCAN_FAILED", "Quét dữ liệu thất bại")
		return
	}

	response.JSON(w, http.StatusOK, nil)
}

// @Summary Chi tiết site
// @Tags Sites
// @Produce json
// @Param siteID path string true "Site UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.Site} "Thông tin site"
// @Failure 404 {object} response.ErrorResponse "Không tìm thấy"
// @Router /sites/{siteID}/ [get]
func (h *SiteHandler) Get(w http.ResponseWriter, r *http.Request) {
	log.Print("GetSite called")
	claims := GetClaims(r)
	membership := GetMembership(r)
	if claims == nil || membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	fmt.Printf("User %s requested site details for site %s\n", claims.UserID, membership.SiteID)

	// Kiểm tra membership
	var role string
	err := h.store.Pool.QueryRow(r.Context(),
		`SELECT role FROM memberships WHERE user_id=$1 AND site_id=$2`,
		claims.UserID, membership.SiteID).Scan(&role)
	if err != nil {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Bạn không có quyền truy cập site này")
		return
	}

	var site models.Site
	err = h.store.Pool.QueryRow(r.Context(),
		`SELECT id, name, description, created_at FROM sites WHERE id=$1`, membership.SiteID).Scan(&site.ID, &site.Name, &site.Description, &site.CreatedAt)
	if err != nil {
		response.Error(w, http.StatusNotFound, "SITE_NOT_FOUND", "Site không tồn tại")
		return
	}
	response.JSON(w, http.StatusOK, site)
}

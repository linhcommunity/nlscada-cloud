package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type MembershipHandler struct {
	store *postgres.Store
}

func NewMembershipHandler(store *postgres.Store) *MembershipHandler {
	return &MembershipHandler{store: store}
}

type InviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UpdateRoleRequest struct {
	Role string `json:"role"`
}

// @Summary Danh sách thành viên
// @Tags Memberships
// @Produce json
// @Param siteID path string true "Site UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=[]models.Membership} "Danh sách thành viên"
// @Router /sites/{siteID}/members [get]
func (h *MembershipHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SITE_ID", "ID site không hợp lệ")
		return
	}

	claims := GetClaims(r)
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	// Chỉ admin mới có quyền, middleware đã kiểm tra, nhưng kiểm tra thêm siteID
	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT m.id, m.user_id, m.site_id, m.role, m.created_at, u.email 
		 FROM memberships m JOIN users u ON m.user_id = u.id 
		 WHERE m.site_id = $1`, siteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", "Xóa thiết bị thất bại")
		return
	}
	defer rows.Close()

	type MemberWithEmail struct {
		models.Membership
		Email string `json:"email"`
	}

	var members []MemberWithEmail
	for rows.Next() {
		var m MemberWithEmail
		if err := rows.Scan(&m.ID, &m.UserID, &m.SiteID, &m.Role, &m.CreatedAt, &m.Email); err != nil {
			response.Error(w, http.StatusInternalServerError, "SCAN_FAILED", "Quét dữ liệu thất bại")
			return
		}
		members = append(members, m)
	}

	response.ListJSON(w, http.StatusOK, members, 1, len(members), int64(len(members)))
}

// @Summary Mời thành viên
// @Tags Memberships
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param request body InviteRequest true "Email và role"
// @Security BearerAuth
// @Success 201 {object} response.SuccessResponse{data=models.Membership} "Đã mời thành công"
// @Failure 404 {object} response.ErrorResponse "User chưa đăng ký"
// @Router /sites/{siteID}/members [post]
func (h *MembershipHandler) Invite(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SITE_ID", "ID site không hợp lệ")
		return
	}

	var input InviteRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu yêu cầu không hợp lệ")
		return
	}

	if input.Role == "" {
		input.Role = "viewer"
	}

	// Tìm user theo email
	var userID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(), "SELECT id FROM users WHERE email = $1", input.Email).Scan(&userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "Người dùng không tồn tại")
		return
	}

	// Tạo membership
	var m models.Membership
	err = h.store.Pool.QueryRow(r.Context(),
		"INSERT INTO memberships (user_id, site_id, role) VALUES ($1, $2, $3) RETURNING id, user_id, site_id, role, created_at",
		userID, siteID, input.Role).Scan(&m.ID, &m.UserID, &m.SiteID, &m.Role, &m.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			response.Error(w, http.StatusConflict, "USER_ALREADY_IN_SITE", "Người dùng đã là thành viên của site này")
		} else {
			response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", "Tạo thành viên thất bại")
		}
		return
	}

	response.JSON(w, http.StatusCreated, m)
}

// @Summary Đổi role thành viên
// @Tags Memberships
// @Accept json
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param userID path string true "User UUID"
// @Param request body UpdateRoleRequest true "Role mới"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=models.Membership} "Role đã cập nhật"
// @Router /sites/{siteID}/members/{userID} [put]
func (h *MembershipHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SITE_ID", "ID site không hợp lệ")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "ID người dùng không hợp lệ")
		return
	}

	var input UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu yêu cầu không hợp lệ")
		return
	}

	var m models.Membership
	err = h.store.Pool.QueryRow(r.Context(),
		"UPDATE memberships SET role = $1 WHERE user_id = $2 AND site_id = $3 RETURNING id, user_id, site_id, role, created_at",
		input.Role, userID, siteID).Scan(&m.ID, &m.UserID, &m.SiteID, &m.Role, &m.CreatedAt)
	if err != nil {
		response.Error(w, http.StatusNotFound, "MEMBERSHIP_NOT_FOUND", "Thành viên không tồn tại")
		return
	}

	response.JSON(w, http.StatusOK, m)
}

// @Summary Xóa thành viên
// @Tags Memberships
// @Param siteID path string true "Site UUID"
// @Param userID path string true "User UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse
// @Router /sites/{siteID}/members/{userID} [delete]
func (h *MembershipHandler) Remove(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SITE_ID", "ID site không hợp lệ")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_USER_ID", "ID người dùng không hợp lệ")
		return
	}

	_, err = h.store.Pool.Exec(r.Context(), "DELETE FROM memberships WHERE user_id = $1 AND site_id = $2", userID, siteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", "Truy vấn thất bại")
		return
	}

	response.JSON(w, http.StatusOK, nil)
}

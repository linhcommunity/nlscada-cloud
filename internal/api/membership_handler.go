package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"
	"nlscada-cloud/internal/response"
	"nlscada-cloud/internal/ws"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type MembershipHandler struct {
	store *postgres.Store
	hub   *ws.Hub
}

func NewMembershipHandler(store *postgres.Store, hub *ws.Hub) *MembershipHandler {
	return &MembershipHandler{store: store, hub: hub}
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
	var siteName string
	h.store.Pool.QueryRow(r.Context(), "SELECT name FROM sites WHERE id = $1", siteID).Scan(&siteName)
	h.hub.NotifyNewMembership(userID, siteID, siteName, input.Role)

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

	// Nếu role bị hạ xuống viewer hoặc auditor, có thể force disconnect để áp dụng quyền mới
	// if input.Role == "viewer" || input.Role == "auditor" {
	// 	h.hub.ForceDisconnect(userID, siteID)
	// }
	h.hub.ReloadPermissions(userID, siteID) // Thay vì ForceDisconnect

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

	// Force disconnect user khỏi site này
	h.hub.ForceDisconnect(userID, siteID)

	response.JSON(w, http.StatusOK, nil)
}

// Leave cho phép user tự rời khỏi site
// @Summary Tự rời khỏi site
// @Description Người dùng tự xóa membership của chính mình khỏi site. Không thể rời nếu là admin cuối cùng.
// @Tags Memberships
// @Param siteID path string true "Site UUID"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse "Đã rời khỏi site"
// @Failure 403 {object} response.ErrorResponse "Không có quyền"
// @Failure 409 {object} response.ErrorResponse "Không thể rời vì là admin cuối cùng"
// @Router /sites/{siteID}/leave [post]
func (h *MembershipHandler) Leave(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SITE_ID", "ID site không hợp lệ")
		return
	}

	// Kiểm tra user có membership trong site không
	var currentRole string
	err = h.store.Pool.QueryRow(r.Context(),
		"SELECT role FROM memberships WHERE user_id = $1 AND site_id = $2",
		claims.UserID, siteID).Scan(&currentRole)
	if err != nil {
		response.Error(w, http.StatusNotFound, "MEMBERSHIP_NOT_FOUND", "Bạn không phải là thành viên của site này")
		return
	}

	// Nếu user là admin, kiểm tra xem có phải admin cuối cùng không
	if currentRole == "admin" {
		var adminCount int
		err = h.store.Pool.QueryRow(r.Context(),
			"SELECT COUNT(*) FROM memberships WHERE site_id = $1 AND role = 'admin'",
			siteID).Scan(&adminCount)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "FAILED_TO_CHECK_ADMIN_COUNT", "Failed to check admin count")
			return
		}
		if adminCount <= 1 {
			response.Error(w, http.StatusConflict, "LAST_ADMIN_CANNOT_LEAVE", "Bạn là quản trị viên cuối cùng. Vui lòng chuyển quyền quản trị viên sang thành viên khác trước khi rời khỏi site.")
			return
		}
	}

	// Xóa membership
	_, err = h.store.Pool.Exec(r.Context(),
		"DELETE FROM memberships WHERE user_id = $1 AND site_id = $2",
		claims.UserID, siteID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "FAILED_TO_LEAVE_SITE", "Failed to leave site")
		return
	}

	// Force disconnect WebSocket của user khỏi site này
	h.hub.ForceDisconnect(claims.UserID, siteID)

	response.JSON(w, http.StatusOK, nil)
}

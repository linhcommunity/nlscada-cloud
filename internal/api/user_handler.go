package api

import (
	"encoding/json"
	"net/http"
	"time"

	"nlscada-cloud/internal/db/postgres"

	"github.com/google/uuid"
)

type UserHandler struct {
	store *postgres.Store
}

func NewUserHandler(store *postgres.Store) *UserHandler {
	return &UserHandler{store: store}
}

type UserInfo struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// @Summary Thông tin cá nhân
// @Description Trả về thông tin tài khoản của user hiện đang đăng nhập
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=UserInfo} "Thông tin cá nhân"
// @Failure 401 {object} response.ErrorResponse "Chưa xác thực"
// @Router /users/me [get]
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}

	var user struct {
		ID    uuid.UUID `json:"id"`
		Email string    `json:"email"`
		Name  string    `json:"name,omitempty"`
		// TODO: email_verified sẽ được thêm ở phiên bản sau
		// EmailVerified bool   `json:"email_verified"`
		CreatedAt time.Time `json:"created_at"`
	}
	err := h.store.Pool.QueryRow(r.Context(),
		"SELECT id, email, name, created_at FROM users WHERE id = $1", claims.UserID).
		Scan(&user.ID,
			&user.Email,
			&user.Name,
			// &user.EmailVerified,
			&user.CreatedAt)

	if err != nil {
		response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "Người dùng không tồn tại")
		return
	}

	response.JSON(w, http.StatusOK, user)
}

// ListVerified trả về danh sách user đã xác thực email (dùng để mời vào site)
// @Summary Danh sách user đã xác thực
// @Description Trả về danh sách tài khoản đã xác thực email để có thể mời vào site
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=[]UserInfo} "Danh sách user"
// @Router /users [get]
func (h *UserHandler) ListVerified(w http.ResponseWriter, r *http.Request) {
	// Chỉ cần trả về danh sách user đã verified (không phân biệt tenant)
	rows, err := h.store.Pool.Query(r.Context(),
		"SELECT id, email, name, created_at FROM users WHERE email_verified = true")
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", "Xóa thành viên thất bại")
		return
	}
	defer rows.Close()

	var users []UserInfo
	for rows.Next() {
		var u UserInfo
		if err := rows.Scan(&u.ID,
			&u.Email,
			&u.Name,
			// &u.EmailVerified,
			&u.CreatedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", "Truy vấn thất bại")
			return
		}
		users = append(users, u)
	}
	response.ListJSON(w, http.StatusOK, users, 1, len(users), int64(len(users)))
}

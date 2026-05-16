package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"nlscada-cloud/internal/auth"
	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/response"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	store     *postgres.Store
	jwtSecret string
}

func NewAuthHandler(store *postgres.Store, jwtSecret string) *AuthHandler {
	return &AuthHandler{store: store, jwtSecret: jwtSecret}
}

// --- Register ---

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	// SiteName string `json:"site_name,omitempty"` // nếu bỏ trống sẽ tự sinh từ email
}

type registerResponse struct {
	Email string `json:"email"`
	// Token    string    `json:"token"`
	Message string `json:"message"`
}

// Register tạo tài khoản người dùng mới
// @Summary      Đăng ký
// @Description  Tạo người dùng mới. Không tạo site hay membership.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body registerRequest true "Thông tin đăng ký"
// @Success      201  {object}  response.SuccessResponse{data=registerResponse} "Đăng ký thành công"
// @Failure      400  {object}  response.ErrorResponse "Lỗi dữ liệu"
// @Failure      409  {object}  response.ErrorResponse "Email đã tồn tại"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Dữ liệu không hợp lệ")
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_FIELDS", "Email, tên và mật khẩu là bắt buộc")
		return
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Đã xảy ra lỗi nội bộ!")
		return
	}

	// Tạo user
	var userID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(),
		"INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING id",
		req.Email, req.Name, string(hashed),
	).Scan(&userID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			response.Error(w, http.StatusConflict, "EMAIL_EXISTS", "Email đã được sử dụng!")
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Đã xảy ra lỗi nội bộ!")
		return
	}

	response.JSON(w, http.StatusCreated, registerResponse{Email: req.Email, Message: "Đăng ký thành công. Vui lòng đăng nhập để nhận token truy cập."})
}

// --- Login ---

type loginRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name,omitempty"` // không bắt buộc, chỉ để phản hồi lại cho client
	Password string `json:"password"`
}

type loginResponse struct {
	Email   string `json:"email,omitempty"`
	Name    string `json:"name,omitempty"`
	Message string `json:"message,omitempty"`
}

// Login xác thực người dùng và trả về JWT token
// @Summary      Đăng nhập
// @Description  Xác thực người dùng và trả về JWT token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body loginRequest true "Thông tin đăng nhập"
// @Success      200  {object}  response.SuccessResponse{data=loginResponse} "Đăng nhập thành công"
// @Failure      400  {object}  response.ErrorResponse "Lỗi dữ liệu"
// @Failure      401  {object}  response.ErrorResponse "Tài khoản hoặc mật khẩu không chính xác"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Phương thức không được hỗ trợ")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Dữ liệu không hợp lệ")
		return
	}

	ctx := r.Context()

	// Tìm user
	var userID uuid.UUID
	var passwordHash string
	err := h.store.Pool.QueryRow(ctx, "SELECT id, password_hash FROM users WHERE email = $1", req.Email).Scan(&userID, &passwordHash)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Tài khoản hoặc mật khẩu không chính xác!")
		return
	}
	// fmt.Printf("UserID: %s, PasswordHash: %s\n", userID, passwordHash)
	// Kiểm tra password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		response.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Tài khoản hoặc mật khẩu không chính xác!")
		return
	}

	// Tạo token
	token, err := auth.GenerateToken(h.jwtSecret, userID, req.Email, req.Name)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Đã xảy ra lỗi nội bộ!")
		return
	}
	response.JSONWithCookie(w, http.StatusOK, loginResponse{Email: req.Email, Name: req.Name, Message: "Đăng nhập thành công."}, token, "http://localhost:8080") // Cần điều chỉnh domain khi deploy thực tế
}

// Refesh tạo token mới dựa trên refresh token (ở đây dùng cookie session_token)
// @Summary      Làm mới token
// @Description  Tạo token mới dựa trên refresh token (ở đây dùng cookie session_token)
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.SuccessResponse{data=loginResponse} "Làm mới token thành công"
// @Failure      401  {object}  response.ErrorResponse "Phiên đăng nhập không hợp lệ hoặc đã hết hạn"
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Phương thức không hỗ trợ")
		return
	}
	// 1. Đọc refresh token từ cookie đặc thù
	cookie, err := r.Cookie("session_token")
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "SESSION_NOT_FOUND", "Không tìm thấy phiên đăng nhập. Vui lòng đăng nhập lại!")
		return
	}

	claims, err := auth.VerifyToken(h.jwtSecret, cookie.Value)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Đăng nhập hết hạn. Vui lòng đăng nhập lại!")
		return
	}

	newToken, err := auth.GenerateToken(h.jwtSecret, claims.UserID, claims.Email, claims.Name)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Đã xảy ra lỗi nội bộ!")
		return
	}

	response.JSONWithCookie(w, http.StatusOK, loginResponse{Email: claims.Email, Name: claims.Name, Message: "Làm mới token thành công."}, newToken, "http://localhost:8080") // Cần điều chỉnh domain khi deploy thực tế
}

// Logout xóa cookie session_token để đăng xuất người dùng
// @Summary      Đăng xuất
// @Description  Xóa cookie session_token để đăng xuất người dùng
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.SuccessResponse "Đăng xuất thành công"
// @Failure      405  {object}  response.ErrorResponse "Phương thức không được hỗ trợ"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Kiểm tra nếu không phải phương thức POST (Khuyến nghị bảo mật, tránh thu thập link)
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Phương thức không được hỗ trợ")
		return
	}

	response.JSONClearCookie(w, "http://localhost:8080") // Cần điều chỉnh domain khi deploy thực tế
}

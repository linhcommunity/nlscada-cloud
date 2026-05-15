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
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	// Token    string    `json:"token"`
	Message string `json:"message"`
}

// Register chỉ tạo user mới. Không tạo site, không tạo membership, không trả về token.
// Người dùng phải login để lấy token, sau đó tự tạo site hoặc được mời vào site.
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

	// Phản hồi
	resp := map[string]interface{}{
		"user_id": userID,
		"email":   req.Email,
		// "name":    req.Name,
		"message": "registration successful. Please login to get your access token.",
	}

	response.JSON(w, http.StatusCreated, resp)
}

// --- Login ---

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	// Token   string `json:"token"`
	Message string `json:"message,omitempty"`
}

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
	token, err := auth.GenerateToken(h.jwtSecret, userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Đã xảy ra lỗi nội bộ!")
		return
	}
	response.JSONWithCookie(w, http.StatusOK, loginResponse{Message: "Đăng nhập thành công."}, token, "http://localhost:8080") // Cần điều chỉnh domain khi deploy thực tế
}

// --- Refresh ---

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Phương thức không hỗ trợ")
		return
	}
	// 1. Đọc refresh token từ cookie đặc thù
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "REFRESH_TOKEN_MISSING", "Phiên làm việc đã kết thúc, vui lòng đăng nhập lại.")
		return
	}

	claims, err := auth.VerifyToken(h.jwtSecret, cookie.Value)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Đăng nhập hết hạn. Vui lòng đăng nhập lại!")
		return
	}

	newToken, err := auth.GenerateToken(h.jwtSecret, claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Đã xảy ra lỗi nội bộ!")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": newToken})
	response.JSONWithCookie(w, http.StatusOK, loginResponse{Message: "Làm mới token thành công."}, newToken, "http://localhost:8080") // Cần điều chỉnh domain khi deploy thực tế
}

// --- Logout ---

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Kiểm tra nếu không phải phương thức POST (Khuyến nghị bảo mật, tránh thu thập link)
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Phương thức không được hỗ trợ")
		return
	}

	response.JSONClearCookie(w, "http://localhost:8080") // Cần điều chỉnh domain khi deploy thực tế
}

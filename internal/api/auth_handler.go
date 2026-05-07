package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"nlscada-cloud/internal/auth"
	"nlscada-cloud/internal/db/postgres"

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

// ---- Register (đã sửa trước đó) ----
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID      uuid.UUID `json:"id"`
	Email   string    `json:"email"`
	Message string    `json:"message"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password required"}`, http.StatusBadRequest)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	var userID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(),
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		req.Email, string(hashed),
	).Scan(&userID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			http.Error(w, `{"error":"email already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"failed to create user"}`, http.StatusInternalServerError)
		return
	}

	resp := registerResponse{
		ID:      userID,
		Email:   req.Email,
		Message: "registration successful. Please verify your email.",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// ---- Login (cập nhật) ----
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	// Tìm user
	var user struct {
		ID           string `db:"id"`
		Email        string `db:"email"`
		PasswordHash string `db:"password_hash"`
	}
	err := h.store.Pool.QueryRow(r.Context(),
		"SELECT id, email, password_hash FROM users WHERE email = $1", req.Email).
		Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Tạo token với user_id và email (không role, không tenant)
	accessToken, err := auth.GenerateToken(h.jwtSecret, uuid.MustParse(user.ID), user.Email)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	refreshToken := accessToken // tạm thời

	resp := loginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ---- Refresh (giữ nguyên, nhưng cần đảm bảo GenerateToken mới không thêm role) ----
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	claims, err := auth.VerifyToken(h.jwtSecret, req.RefreshToken)
	if err != nil {
		http.Error(w, `{"error":"invalid refresh token"}`, http.StatusUnauthorized)
		return
	}

	newToken, err := auth.GenerateToken(h.jwtSecret, claims.UserID, claims.Email)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"access_token": newToken})
}

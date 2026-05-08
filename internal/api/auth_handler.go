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

// --- Register ---

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	SiteName string `json:"site_name,omitempty"` // nếu bỏ trống sẽ tự sinh từ email
}

type registerResponse struct {
	SiteID   uuid.UUID `json:"site_id"`
	SiteName string    `json:"site_name"`
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	Token    string    `json:"token"`
	Message  string    `json:"message"`
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
	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	tx, err := h.store.Pool.Begin(ctx)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// 1. Tạo site - không dùng nữa

	// 2. Tạo user
	var userID uuid.UUID
	err = tx.QueryRow(ctx, "INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id", req.Email, string(hashed)).Scan(&userID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			http.Error(w, `{"error":"email already exists"}`, http.StatusConflict)
		} else {
			http.Error(w, `{"error":"failed to create user"}`, http.StatusInternalServerError)
		}
		return
	}
	// token, err := auth.GenerateToken(h.jwtSecret, userID)
	// if err != nil {
	// 	http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
	// 	return
	// }
	resp := registerResponse{
		UserID:  userID,
		Email:   req.Email,
		Message: "registration successful",
		// Token:   token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// --- Login ---

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Tìm user
	var userID uuid.UUID
	var passwordHash string
	err := h.store.Pool.QueryRow(ctx, "SELECT id, password_hash FROM users WHERE email = $1", req.Email).Scan(&userID, &passwordHash)
	if err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Kiểm tra password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Tạo token
	token, err := auth.GenerateToken(h.jwtSecret, userID)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{Token: token})
}

// --- Refresh ---

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

	newToken, err := auth.GenerateToken(h.jwtSecret, claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": newToken})
}

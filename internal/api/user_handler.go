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

// Me - GET /v1/users/me
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var user struct {
		ID    uuid.UUID `json:"id"`
		Email string    `json:"email"`
		// TODO: email_verified sẽ được thêm ở phiên bản sau
		// EmailVerified bool   `json:"email_verified"`
		CreatedAt time.Time `json:"created_at"`
	}
	err := h.store.Pool.QueryRow(r.Context(),
		"SELECT id, email, email_verified, created_at FROM users WHERE id = $1", claims.UserID).
		Scan(&user.ID,
			&user.Email,
			// &user.EmailVerified,
			&user.CreatedAt)

	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ListVerified - GET /v1/users (chỉ trả về user đã verified để mời)
func (h *UserHandler) ListVerified(w http.ResponseWriter, r *http.Request) {
	// Chỉ cần trả về danh sách user đã verified (không phân biệt tenant)
	rows, err := h.store.Pool.Query(r.Context(),
		"SELECT id, email, email_verified, created_at FROM users WHERE email_verified = true")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type UserInfo struct {
		ID    uuid.UUID `json:"id"`
		Email string    `json:"email"`
		// EmailVerified bool      `json:"email_verified"`
		CreatedAt time.Time `json:"created_at"`
	}
	var users []UserInfo
	for rows.Next() {
		var u UserInfo
		if err := rows.Scan(&u.ID,
			&u.Email,
			// &u.EmailVerified,
			&u.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

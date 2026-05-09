package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type MembershipHandler struct {
	store *postgres.Store
}

func NewMembershipHandler(store *postgres.Store) *MembershipHandler {
	return &MembershipHandler{store: store}
}

// ListMembers trả về danh sách thành viên của site
func (h *MembershipHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		http.Error(w, `{"error":"invalid site id"}`, http.StatusBadRequest)
		return
	}

	claims := GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Chỉ admin mới có quyền, middleware đã kiểm tra, nhưng kiểm tra thêm siteID
	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT m.id, m.user_id, m.site_id, m.role, m.created_at, u.email 
		 FROM memberships m JOIN users u ON m.user_id = u.id 
		 WHERE m.site_id = $1`, siteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		members = append(members, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

// Invite mời user vào site (chỉ admin)
func (h *MembershipHandler) Invite(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		http.Error(w, `{"error":"invalid site id"}`, http.StatusBadRequest)
		return
	}

	var input struct {
		Email string `json:"email"`
		Role  string `json:"role"` // "operator" hoặc "viewer"
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	if input.Role == "" {
		input.Role = "viewer"
	}

	// Tìm user theo email
	var userID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(), "SELECT id FROM users WHERE email = $1", input.Email).Scan(&userID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	// Tạo membership
	var m models.Membership
	err = h.store.Pool.QueryRow(r.Context(),
		"INSERT INTO memberships (user_id, site_id, role) VALUES ($1, $2, $3) RETURNING id, user_id, site_id, role, created_at",
		userID, siteID, input.Role).Scan(&m.ID, &m.UserID, &m.SiteID, &m.Role, &m.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			http.Error(w, `{"error":"user already in site"}`, http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(m)
}

// UpdateRole sửa role của thành viên
func (h *MembershipHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		http.Error(w, `{"error":"invalid site id"}`, http.StatusBadRequest)
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	var input struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	var m models.Membership
	err = h.store.Pool.QueryRow(r.Context(),
		"UPDATE memberships SET role = $1 WHERE user_id = $2 AND site_id = $3 RETURNING id, user_id, site_id, role, created_at",
		input.Role, userID, siteID).Scan(&m.ID, &m.UserID, &m.SiteID, &m.Role, &m.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"membership not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

// Remove xóa thành viên khỏi site
func (h *MembershipHandler) Remove(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
	if err != nil {
		http.Error(w, `{"error":"invalid site id"}`, http.StatusBadRequest)
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	_, err = h.store.Pool.Exec(r.Context(), "DELETE FROM memberships WHERE user_id = $1 AND site_id = $2", userID, siteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

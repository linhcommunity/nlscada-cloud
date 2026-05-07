package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SiteHandler struct {
	store *postgres.Store
}

func NewSiteHandler(store *postgres.Store) *SiteHandler {
	return &SiteHandler{store: store}
}

// CreateSite - POST /v1/sites
func (h *SiteHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r) // user đã xác thực
	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if input.Name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	tx, err := h.store.Pool.Begin(ctx)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// Tạo site
	var siteID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO sites (name) VALUES ($1) RETURNING id`, input.Name,
	).Scan(&siteID)
	if err != nil {
		http.Error(w, `{"error":"site name may already exist"}`, http.StatusInternalServerError)
		return
	}

	// Thêm user hiện tại làm admin
	_, err = tx.Exec(ctx,
		`INSERT INTO memberships (user_id, site_id, role) VALUES ($1, $2, 'admin')`,
		claims.UserID, siteID,
	)
	if err != nil {
		http.Error(w, `{"error":"failed to set admin"}`, http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.Site{ID: siteID, Name: input.Name})
}

// ListSites - GET /v1/sites (các site user tham gia)
func (h *SiteHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT s.id, s.name, s.created_at 
		 FROM sites s
		 JOIN memberships m ON s.id = m.site_id
		 WHERE m.user_id = $1`, claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var sites []models.Site
	for rows.Next() {
		var s models.Site
		if err := rows.Scan(&s.ID, &s.Name, &s.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sites = append(sites, s)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sites)
}

// GetSite - GET /v1/sites/{id}
func (h *SiteHandler) Get(w http.ResponseWriter, r *http.Request) {
	log.Print("GetSite called")
	claims := GetClaims(r)
	fmt.Printf("User %s requested site details for site %s\n", claims.UserID, chi.URLParam(r, "id"))
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid site id"}`, http.StatusBadRequest)
		return
	}

	// Kiểm tra membership
	var role string
	err = h.store.Pool.QueryRow(r.Context(),
		`SELECT role FROM memberships WHERE user_id=$1 AND site_id=$2`,
		claims.UserID, siteID).Scan(&role)
	if err != nil {
		http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
		return
	}

	var site models.Site
	err = h.store.Pool.QueryRow(r.Context(),
		`SELECT id, name, created_at FROM sites WHERE id=$1`, siteID).Scan(&site.ID, &site.Name, &site.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(site)
}

// AddMember - POST /v1/sites/{id}/members
func (h *SiteHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	siteID, _ := uuid.Parse(chi.URLParam(r, "id"))
	var input struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	// Kiểm tra user hiện tại có phải admin của site không
	var adminRole string
	err := h.store.Pool.QueryRow(r.Context(),
		`SELECT role FROM memberships WHERE user_id=$1 AND site_id=$2`,
		claims.UserID, siteID).Scan(&adminRole)
	if err != nil || adminRole != "admin" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	// Tìm user được mời (phải đã verified email)
	var userID uuid.UUID
	err = h.store.Pool.QueryRow(r.Context(),
		`SELECT id FROM users WHERE email=$1 AND email_verified=true`, input.Email).Scan(&userID)
	if err != nil {
		http.Error(w, `{"error":"user not found or not verified"}`, http.StatusNotFound)
		return
	}

	// Thêm membership
	_, err = h.store.Pool.Exec(r.Context(),
		`INSERT INTO memberships (user_id, site_id, role) VALUES ($1, $2, $3)`,
		userID, siteID, input.Role)
	if err != nil {
		http.Error(w, `{"error":"could not add member (duplicate?)"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "member added"})
}

// RemoveMember - DELETE /v1/sites/{id}/members/{userID}
func (h *SiteHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	siteID, _ := uuid.Parse(chi.URLParam(r, "id"))
	targetUserID, _ := uuid.Parse(chi.URLParam(r, "userID"))

	// Chỉ admin mới xóa được (và không tự xóa chính mình)
	var adminRole string
	err := h.store.Pool.QueryRow(r.Context(),
		`SELECT role FROM memberships WHERE user_id=$1 AND site_id=$2`,
		claims.UserID, siteID).Scan(&adminRole)
	if err != nil || adminRole != "admin" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if targetUserID == claims.UserID {
		http.Error(w, `{"error":"cannot remove yourself"}`, http.StatusBadRequest)
		return
	}

	_, err = h.store.Pool.Exec(r.Context(),
		`DELETE FROM memberships WHERE user_id=$1 AND site_id=$2`, targetUserID, siteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListMembers - GET /v1/sites/{id}/members
func (h *SiteHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	siteID, _ := uuid.Parse(chi.URLParam(r, "id"))
	// Kiểm tra membership bất kỳ
	var role string
	err := h.store.Pool.QueryRow(r.Context(),
		`SELECT role FROM memberships WHERE user_id=$1 AND site_id=$2`,
		claims.UserID, siteID).Scan(&role)
	if err != nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	rows, err := h.store.Pool.Query(r.Context(),
		`SELECT u.id, u.email, m.role, m.created_at
		 FROM memberships m
		 JOIN users u ON m.user_id = u.id
		 WHERE m.site_id = $1`, siteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Member struct {
		ID    uuid.UUID `json:"id"`
		Email string    `json:"email"`
		Role  string    `json:"role"`
		Since time.Time `json:"since"`
	}
	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Email, &m.Role, &m.Since); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		members = append(members, m)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

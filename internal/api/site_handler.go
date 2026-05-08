package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"

	"github.com/google/uuid"
)

type SiteHandler struct {
	store *postgres.Store
}

func NewSiteHandler(store *postgres.Store) *SiteHandler {
	return &SiteHandler{store: store}
}

// CreateSite - POST /v1/sites
// Create tạo site mới và gán user hiện tại làm admin
func (h *SiteHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input struct {
		Name string `json:"name"`
		// Sau này có thể thêm các trường khác như description, config, v.v.
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

	// 1. Tạo site
	var siteID uuid.UUID
	err = tx.QueryRow(ctx, "INSERT INTO sites (name) VALUES ($1) RETURNING id", input.Name).Scan(&siteID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			http.Error(w, `{"error":"site name already exists"}`, http.StatusConflict)
		} else {
			http.Error(w, `{"error":"failed to create site"}`, http.StatusInternalServerError)
		}
		return
	}
	// siteID lấy từ database, đảm bảo là UUID hợp lệ nên không cần parse lại

	// 2. Tạo membership (admin) cho user
	_, err = tx.Exec(ctx, "INSERT INTO memberships (user_id, site_id, role) VALUES ($1, $2, 'admin')", claims.UserID, siteID)
	if err != nil {
		http.Error(w, `{"error":"failed to create membership"}`, http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Lấy thông tin site vừa tạo để trả về
	var s models.Site
	err = h.store.Pool.QueryRow(ctx, "SELECT id, name, created_at FROM sites WHERE id = $1", siteID).Scan(&s.ID, &s.Name, &s.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	resp := struct {
		models.Site
		Role string `json:"role"`
	}{
		Site: s,
		Role: "admin",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
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

// UpdateSite - PUT /v1/sites/{id} (chỉ admin mới được phép)
func (h *SiteHandler) Update(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if membership.Role != "admin" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

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

	var s models.Site
	err := h.store.Pool.QueryRow(r.Context(),
		"UPDATE sites SET name = $1 WHERE id = $2 RETURNING id, name, created_at",
		input.Name, membership.SiteID,
	).Scan(&s.ID, &s.Name, &s.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

// / DeleteSite - DELETE /v1/sites/{id} (chỉ admin mới được phép)
func (h *SiteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	membership := GetMembership(r)
	if membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if membership.Role != "admin" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	_, err := h.store.Pool.Exec(r.Context(),
		"DELETE FROM sites WHERE id=$1", membership.SiteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSite - GET /v1/sites/{id}
func (h *SiteHandler) Get(w http.ResponseWriter, r *http.Request) {
	log.Print("GetSite called")
	claims := GetClaims(r)
	membership := GetMembership(r)
	if claims == nil || membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	fmt.Printf("User %s requested site details for site %s\n", claims.UserID, membership.SiteID)

	// Kiểm tra membership
	var role string
	err := h.store.Pool.QueryRow(r.Context(),
		`SELECT role FROM memberships WHERE user_id=$1 AND site_id=$2`,
		claims.UserID, membership.SiteID).Scan(&role)
	if err != nil {
		http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
		return
	}

	var site models.Site
	err = h.store.Pool.QueryRow(r.Context(),
		`SELECT id, name, created_at FROM sites WHERE id=$1`, membership.SiteID).Scan(&site.ID, &site.Name, &site.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(site)
}

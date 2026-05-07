package api

import (
	"encoding/json"
	"net/http"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TagHandler struct {
	store *postgres.Store
}

func NewTagHandler(store *postgres.Store) *TagHandler {
	return &TagHandler{store: store}
}

// ListTags trả về tags của một device
func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	deviceID, err := uuid.Parse(chi.URLParam(r, "deviceID"))
	if err != nil {
		http.Error(w, "invalid device id", http.StatusBadRequest)
		return
	}

	rows, err := h.store.Pool.Query(r.Context(),
		"SELECT id, device_id, name, data_type, unit, description, created_at FROM tags WHERE device_id=$1", deviceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.DeviceID, &t.Name, &t.DataType, &t.Unit, &t.Description, &t.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tags = append(tags, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

// CreateTag tạo tag mới
func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	deviceID, err := uuid.Parse(chi.URLParam(r, "deviceID"))
	if err != nil {
		http.Error(w, "invalid device id", http.StatusBadRequest)
		return
	}

	var input struct {
		Name        string `json:"name"`
		DataType    string `json:"data_type"`
		Unit        string `json:"unit,omitempty"`
		Description string `json:"description,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	var t models.Tag
	err = h.store.Pool.QueryRow(r.Context(),
		`INSERT INTO tags (device_id, name, data_type, unit, description) 
		 VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, '')) 
		 RETURNING id, device_id, name, data_type, unit, description, created_at`,
		deviceID, input.Name, input.DataType, input.Unit, input.Description).
		Scan(&t.ID, &t.DeviceID, &t.Name, &t.DataType, &t.Unit, &t.Description, &t.CreatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

// UpdateTag cập nhật tag
func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "tagID"))
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}

	var input struct {
		Name        string `json:"name,omitempty"`
		DataType    string `json:"data_type,omitempty"`
		Unit        string `json:"unit,omitempty"`
		Description string `json:"description,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	// Cập nhật động các trường không rỗng (có thể dùng COALESCE)
	var t models.Tag
	err = h.store.Pool.QueryRow(r.Context(),
		`UPDATE tags SET 
		    name = COALESCE(NULLIF($1, ''), name),
		    data_type = COALESCE(NULLIF($2, ''), data_type),
		    unit = CASE WHEN $3::text IS NOT NULL THEN $3 ELSE unit END,
		    description = CASE WHEN $4::text IS NOT NULL THEN $4 ELSE description END
		 WHERE id=$5 
		 RETURNING id, device_id, name, data_type, unit, description, created_at`,
		input.Name, input.DataType, input.Unit, input.Description, id).
		Scan(&t.ID, &t.DeviceID, &t.Name, &t.DataType, &t.Unit, &t.Description, &t.CreatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// DeleteTag xóa tag
func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "tagID"))
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}

	_, err = h.store.Pool.Exec(r.Context(), "DELETE FROM tags WHERE id=$1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

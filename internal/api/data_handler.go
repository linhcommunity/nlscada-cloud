package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"nlscada-cloud/internal/db/influxdb"
	"nlscada-cloud/internal/db/postgres"

	"github.com/go-chi/chi/v5"
)

type DataHandler struct {
	influxReader *influxdb.Reader
	store        *postgres.Store
}

func NewDataHandler(reader *influxdb.Reader) *DataHandler {
	return &DataHandler{influxReader: reader}
}

func (h *DataHandler) Query(w http.ResponseWriter, r *http.Request) {
	// 1. Kiểm tra membership
	membership := GetMembership(r)
	if membership == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	// 2. Lấy deviceID từ URL
	deviceID := chi.URLParam(r, "deviceID")
	if deviceID == "" {
		http.Error(w, `{"error":"deviceID required"}`, http.StatusBadRequest)
		return
	}

	// 3. Kiểm tra device có thuộc site của membership không
	var exists bool
	err := h.store.Pool.QueryRow(r.Context(),
		"SELECT EXISTS(SELECT 1 FROM devices WHERE id = $1 AND site_id = $2)",
		deviceID, membership.SiteID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, `{"error":"device not found or access denied"}`, http.StatusNotFound)
		return
	}
	tagsParam := r.URL.Query().Get("tags")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "100"
	}

	// Xây dựng Flux query
	// Giả sử measurement là "metrics", filter theo device_id và tag_name
	query := fmt.Sprintf(`
		from(bucket: "nlscada_metrics")
		  |> range(start: %s, stop: %s)
		  |> filter(fn: (r) => r._measurement == "metrics" and r.device_id == "%s")
		  |> filter(fn: (r) => r._field == "value")
		  |> limit(n: %s)
	`, fromStr, toStr, deviceID, limit)

	// Nếu có lọc theo tag, thêm điều kiện
	if tagsParam != "" {
		tagList := strings.Split(tagsParam, ",")
		// Tạo filter chứa các tag (dùng OR)
		var tagFilters []string
		for _, t := range tagList {
			tagFilters = append(tagFilters, fmt.Sprintf(`r.tag_name == "%s"`, strings.TrimSpace(t)))
		}
		query += fmt.Sprintf("  |> filter(fn: (r) => %s)\n", strings.Join(tagFilters, " or "))
	}

	query += fmt.Sprintf("  |> limit(n: %s)", limit)
	fmt.Println("InfluxDB Query:\n", query)
	result, err := h.influxReader.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var data []map[string]interface{}
	for result.Next() {
		record := result.Record()
		data = append(data, map[string]interface{}{
			"time":  record.Time().Format(time.RFC3339),
			"tag":   record.ValueByKey("tag_name"),
			"value": record.Value(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

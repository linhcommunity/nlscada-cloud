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

func NewDataHandler(store *postgres.Store, reader *influxdb.Reader) *DataHandler {
	return &DataHandler{influxReader: reader, store: store}
}

// DataPoint là một điểm dữ liệu time-series trả về từ InfluxDB
type DataPoint struct {
	Time  string      `json:"time"`
	Tag   interface{} `json:"tag"`
	Value interface{} `json:"value"`
}

// @Summary Truy vấn dữ liệu lịch sử
// @Tags Data
// @Produce json
// @Param siteID path string true "Site UUID"
// @Param deviceID path string true "Device UUID"
// @Param tags query string false "Tên tag (phân cách bằng dấu phẩy)"
// @Param from query string false "Thời gian bắt đầu (ISO 8601)"
// @Param to query string false "Thời gian kết thúc (ISO 8601)"
// @Param limit query int false "Số bản ghi tối đa"
// @Security BearerAuth
// @Success 200 {object} response.SuccessResponse{data=[]DataPoint}
// @Router /sites/{siteID}/devices/{deviceID}/data [get]
func (h *DataHandler) Query(w http.ResponseWriter, r *http.Request) {
	// 1. Kiểm tra membership
	membership := GetMembership(r)
	if membership == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bạn không có quyền truy cập tài nguyên này")
		return
	}
	// 2. Lấy deviceID từ URL
	deviceID := chi.URLParam(r, "deviceID")
	if deviceID == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_DEVICE_ID", "ID thiết bị không hợp lệ")
		return
	}

	// 3. Kiểm tra device có thuộc site của membership không
	var exists bool
	err := h.store.Pool.QueryRow(r.Context(),
		"SELECT EXISTS(SELECT 1 FROM devices WHERE id = $1 AND site_id = $2)",
		deviceID, membership.SiteID).Scan(&exists)
	if err != nil || !exists {
		response.Error(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Thiết bị không tồn tại hoặc không có quyền truy cập")
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
		response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", "Truy vấn thất bại")
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

	response.JSON(w, http.StatusOK, data)
}

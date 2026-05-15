package response

import (
	"encoding/json"
	"net/http"
	"time"
)

// --- ĐỊNH NGHĨA CẤU TRÚC JSON SCHEMA CHUẨN ---

// Pagination thông tin phân trang bắt buộc cho các API dạng danh sách (List)
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
	TotalItems int64 `json:"total_items"`
}

type Meta struct {
	Timestamp  string      `json:"timestamp"`
	Pagination *Pagination `json:"pagination,omitempty"` // Chỉ xuất hiện khi có phân trang, nếu không có sẽ tự động ẩn đi
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Meta    Meta        `json:"meta"`
}

type ValidationErrorDetail struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

type ErrorPayload struct {
	Code      string                  `json:"code"`
	Message   string                  `json:"message"`
	Timestamp string                  `json:"timestamp"`
	Details   []ValidationErrorDetail `json:"details,omitempty"`
}

type ErrorResponse struct {
	Success bool         `json:"success"`
	Error   ErrorPayload `json:"error"`
}

// --- CÁC HÀM TIỆN ÍCH (HELPER FUNCTIONS) ĐÃ NÂNG CẤP ---

// 1. JSONWithPagination: Phản hồi danh sách có phân trang + Thiết lập HttpOnly Cookie (Tùy chọn)
// Nếu không muốn ghi đè Cookie, bạn chỉ cần truyền chuỗi tokenString rỗng ("")
func respondWithPagination(w http.ResponseWriter, statusCode int, data interface{}, page int, limit int, totalItems int64, tokenString string, domain string) {
	// Phòng chống lỗi mảng rỗng (null) trên giao diện Web Client
	if data == nil {
		data = []interface{}{}
	}

	// Tự động tính toán tổng số trang dựa trên tổng số phần tử và giới hạn bản ghi
	totalPages := int(totalItems) / limit
	if int(totalItems)%limit != 0 {
		totalPages++
	}

	// Nếu có truyền Token, tiến hành thiết lập mã HttpOnly Cookie bảo mật
	if tokenString != "" {
		cookie := &http.Cookie{
			Name:     "session_token",
			Value:    tokenString,
			Path:     "/",
			Domain:   domain, // Ví dụ: ".myiiot.com"
			HttpOnly: true,   // Ngăn chặn mã độc JavaScript đọc mã phiên (Chống XSS)
			Secure:   true,   // Ép buộc trình duyệt chỉ truyền qua mạng mã hóa HTTPS
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400, // Phiên làm việc có hiệu lực trong 24 giờ
		}
		http.SetCookie(w, cookie)
	}

	// Đóng gói cấu trúc phản hồi JSON chuẩn hóa 100%
	res := SuccessResponse{
		Success: true,
		Data:    data,
		Meta: Meta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Pagination: &Pagination{
				Page:       page,
				Limit:      limit,
				TotalPages: totalPages,
				TotalItems: totalItems,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(res)
}

// Hàm 1: Chỉ xuất dữ liệu danh sách phân trang thuần túy (Dùng cho 90% API)
func JSONWithPagination(w http.ResponseWriter, statusCode int, data interface{}, page int, limit int, totalItems int64) {
	// Không xử lý Cookie, chỉ gọi luồng đóng gói JSON như bình thường
	respondWithPagination(w, statusCode, data, page, limit, totalItems, "", "")
}

// Hàm 2: Xuất danh sách phân trang + Ép ghi đè Cookie (Chỉ dùng khi cần gia hạn phiên hoặc Đăng nhập/Đăng ký)
func JSONWithPaginationAndCookie(w http.ResponseWriter, statusCode int, data interface{}, page int, limit int, totalItems int64, tokenString string, domain string) {
	respondWithPagination(w, statusCode, data, page, limit, totalItems, tokenString, domain)
}

// 2. Gửi phản hồi thành công và thiết lập HttpOnly Cookie (Dùng khi Đăng nhập / Đăng ký)
func JSONWithCookie(w http.ResponseWriter, statusCode int, data interface{}, tokenString string, domain string) {
	if data == nil {
		data = struct{}{}
	}

	// 1. Cấu hình HttpOnly Cookie chuẩn bảo mật
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    tokenString,
		Path:     "/",
		Domain:   domain,               // Ví dụ: ".myiiot.com"
		HttpOnly: true,                 // Chặn JavaScript truy cập (Chống XSS)
		Secure:   true,                 // Bắt buộc chạy qua HTTPS
		SameSite: http.SameSiteLaxMode, // Phòng chống tấn công CSRF
		MaxAge:   86400,                // Hết hạn sau 24 giờ (tính bằng giây)
	}
	http.SetCookie(w, cookie)

	// 2. Xuất dữ liệu JSON ra ngoài đường truyền
	res := SuccessResponse{
		Success: true,
		Data:    data,
		Meta: Meta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(res)
}

// 3. Xóa HttpOnly Cookie khỏi trình duyệt (Dùng khi Đăng xuất)
func JSONClearCookie(w http.ResponseWriter, domain string) {
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Domain:   domain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // Giá trị âm ép trình duyệt xóa ngay lập tức
	}
	http.SetCookie(w, cookie)

	res := SuccessResponse{
		Success: true,
		Data:    struct{}{},
		Meta: Meta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

// 4. JSON: Phản hồi thành công cho tài nguyên đơn lẻ (Ví dụ: Chi tiết 1 thiết bị, Đăng ký xong 1 User)
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	// Phòng chống lỗi dữ liệu rỗng (null) phá vỡ UI của Frontend
	if data == nil {
		data = struct{}{}
	}

	res := SuccessResponse{
		Success: true,
		Data:    data,
		Meta: Meta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(res)
}

// 5. ListJSON: Phản hồi thành công cho API danh sách BẮT BUỘC có phân trang (Ví dụ: Lấy danh sách thiết bị)
func ListJSON(w http.ResponseWriter, statusCode int, data interface{}, page int, limit int, totalItems int64) {
	// Nếu mảng dữ liệu rỗng, ép hệ thống trả về mảng rỗng [] thay vì null
	if data == nil {
		data = []interface{}{}
	}

	// Tự động tính toán tổng số trang từ số lượng phần tử
	totalPages := int(totalItems) / limit
	if int(totalItems)%limit != 0 {
		totalPages++
	}

	res := SuccessResponse{
		Success: true,
		Data:    data,
		Meta: Meta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Pagination: &Pagination{
				Page:       page,
				Limit:      limit,
				TotalPages: totalPages,
				TotalItems: totalItems,
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(res)
}

// 6. Error: Giữ nguyên hàm báo lỗi hệ thống của bạn (Chuẩn)
func Error(w http.ResponseWriter, statusCode int, errorCode string, message string) {
	res := ErrorResponse{
		Success: false,
		Error: ErrorPayload{
			Code:      errorCode,
			Message:   message,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(res)
}

// 7. ValidationError: Giữ nguyên hàm báo lỗi nhập liệu của bạn (Chuẩn)
func ValidationError(w http.ResponseWriter, details []ValidationErrorDetail) {
	res := ErrorResponse{
		Success: false,
		Error: ErrorPayload{
			Code:      "INVALID_INPUT",
			Message:   "Dữ liệu gửi lên không đúng định dạng hoặc thiếu các trường bắt buộc.",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Details:   details,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(res)
}

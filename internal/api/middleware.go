package api

import (
	"context"
	"net/http"

	"nlscada-cloud/internal/auth"
	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	ClaimsKey     contextKey = "claims"
	MembershipKey contextKey = "membership" // lưu struct { Role, SiteID }
)

// AuthMiddleware xác thực JWT và đưa claims vào context
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Trích xuất cookie mang tên "session_token" từ request gửi lên backend
			cookie, err := r.Cookie("session_token")
			if err != nil {
				// Trả về lỗi nếu không tìm thấy cookie lưu phiên đăng nhập
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Vui lòng đăng nhập để thực hiện thao tác này.")
				return
			}

			// 2. Lấy chuỗi token thô (raw string) từ trong đối tượng cookie
			tokenString := cookie.Value

			claims, err := auth.VerifyToken(secret, tokenString)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token không hợp lệ.")
				return
			}
			// 3. Lưu thông tin claims vào context để các middleware và handler phía sau có thể truy cập
			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims lấy claims từ context
func GetClaims(r *http.Request) *auth.Claims {
	claims, ok := r.Context().Value(ClaimsKey).(*auth.Claims)
	if !ok {
		return nil
	}
	return claims
}

// SiteMiddleware xác định site và role cho các route có {siteID}
func SiteMiddleware(store *postgres.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Vui lòng đăng nhập để thực hiện thao tác này.")
				return
			}

			siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
			if err != nil {
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "siteID không hợp lệ.")
				return
			}

			// Truy vấn membership
			var role string
			err = store.Pool.QueryRow(r.Context(),
				"SELECT role FROM memberships WHERE user_id = $1 AND site_id = $2",
				claims.UserID, siteID).Scan(&role)
			if err != nil {
				response.Error(w, http.StatusForbidden, "FORBIDDEN", "Bạn không có quyền truy cập vào site này.")
				return
			}

			// Lưu thông tin membership vào context
			membership := struct {
				SiteID uuid.UUID
				Role   string
			}{siteID, role}
			ctx := context.WithValue(r.Context(), MembershipKey, membership)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole kiểm tra vai trò của user
func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// truy vấn từ bảng memberships để lấy role của user hiện tại trong site hiện tại
			membership := GetMembership(r)
			if membership == nil {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Vui lòng đăng nhập để thực hiện thao tác này.")
				return
			}
			for _, role := range allowed {
				if membership.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Bạn không có quyền truy cập vào site này.")
		})
	}
}

// GetMembership lấy membership từ context
func GetMembership(r *http.Request) *struct {
	SiteID uuid.UUID
	Role   string
} {
	m, ok := r.Context().Value(MembershipKey).(struct {
		SiteID uuid.UUID
		Role   string
	})
	if !ok {
		return nil
	}
	return &m
}

package api

import (
	"context"
	"net/http"
	"strings"

	"nlscada-cloud/internal/auth"
	"nlscada-cloud/internal/db/postgres"

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
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := auth.VerifyToken(secret, tokenStr)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

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
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			siteID, err := uuid.Parse(chi.URLParam(r, "siteID"))
			if err != nil {
				http.Error(w, `{"error":"invalid siteID"}`, http.StatusBadRequest)
				return
			}

			// Truy vấn membership
			var role string
			err = store.Pool.QueryRow(r.Context(),
				"SELECT role FROM memberships WHERE user_id = $1 AND site_id = $2",
				claims.UserID, siteID).Scan(&role)
			if err != nil {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
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
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			for _, role := range allowed {
				if membership.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
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

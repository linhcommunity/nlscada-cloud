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

const ClaimsKey contextKey = "claims"

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

// GetClaims giúp handler lấy claims từ context
func GetClaims(r *http.Request) *auth.Claims {
	claims, ok := r.Context().Value(ClaimsKey).(*auth.Claims)
	if !ok {
		return nil
	}
	return claims
}

// // RequireRole middleware kiểm tra role của user có nằm trong allowed không
// func RequireRole(allowed ...string) func(http.Handler) http.Handler {
// 	return func(next http.Handler) http.Handler {
// 		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			claims := GetClaims(r)
// 			if claims == nil {
// 				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
// 				return
// 			}
// 			for _, role := range allowed {
// 				if claims.Role == role {
// 					next.ServeHTTP(w, r)
// 					return
// 				}
// 			}
// 			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
// 		})
// 	}
// }

func RequireSiteRole(store *postgres.Store, allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			siteID, err := uuid.Parse(chi.URLParam(r, "siteID")) // mong đợi URL param là siteID
			if err != nil {
				http.Error(w, `{"error":"invalid site id"}`, http.StatusBadRequest)
				return
			}
			var role string
			err = store.Pool.QueryRow(r.Context(),
				`SELECT role FROM memberships WHERE user_id=$1 AND site_id=$2`,
				claims.UserID, siteID).Scan(&role)
			if err != nil {
				http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
				return
			}
			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		})
	}
}

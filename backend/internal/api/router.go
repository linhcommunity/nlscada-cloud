package api

import (
	"net/http"

	"nlscada-cloud/internal/auth"
	"nlscada-cloud/internal/db/influxdb"
	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/ws"

	"github.com/go-chi/chi/v5"
)

func NewRouter(store *postgres.Store, influxReader *influxdb.Reader, influxWriter *influxdb.Writer, jwtSecret string, hub *ws.Hub) *chi.Mux {
	r := chi.NewRouter()

	// Public
	authHandler := NewAuthHandler(store, jwtSecret)
	r.Post("/v1/auth/register", authHandler.Register)
	r.Post("/v1/auth/login", authHandler.Login)
	r.Post("/v1/auth/refresh", authHandler.Refresh)

	// Authenticated
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware(jwtSecret))

		// User
		userHandler := NewUserHandler(store)
		r.Get("/v1/users/me", userHandler.Me)

		// Sites (list, create - không cần siteID trong URL)
		siteHandler := NewSiteHandler(store)
		r.Get("/v1/sites", siteHandler.List)
		r.Post("/v1/sites", siteHandler.Create)

		// Các route liên quan đến 1 site cụ thể
		r.Route("/v1/sites/{siteID}", func(r chi.Router) {
			r.Use(SiteMiddleware(store)) // Áp dụng SiteMiddleware cho tất cả route con

			// Site detail/update/delete
			r.Get("/", siteHandler.Get)
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("admin"))
				r.Put("/", siteHandler.Update)
				r.Delete("/", siteHandler.Delete)
			})

			// Memberships
			membershipHandler := NewMembershipHandler(store)
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("admin"))
				r.Get("/members", membershipHandler.List)
				r.Post("/members", membershipHandler.Invite)             // Admin only
				r.Put("/members/{userID}", membershipHandler.UpdateRole) // Admin
				r.Delete("/members/{userID}", membershipHandler.Remove)  // Admin
			})

			// Devices
			deviceHandler := NewDeviceHandler(store)
			r.Get("/devices", deviceHandler.List)
			r.Get("/devices/{deviceID}", deviceHandler.Get)
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("admin", "operator"))
				r.Post("/devices", deviceHandler.Create)
				r.Put("/devices/{deviceID}", deviceHandler.Update)
			})
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("admin"))
				r.Delete("/devices/{deviceID}", deviceHandler.Delete)
			})

			// Tags
			tagHandler := NewTagHandler(store)
			r.Get("/devices/{deviceID}/tags", tagHandler.List)
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("admin", "operator"))
				r.Post("/devices/{deviceID}/tags", tagHandler.Create)
				r.Put("/tags/{tagID}", tagHandler.Update)
			})
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("admin"))
				r.Delete("/tags/{tagID}", tagHandler.Delete)
			})

			// Data Query
			dataHandler := NewDataHandler(influxReader)
			r.Get("/devices/{deviceID}/data", dataHandler.Query)
		})
	})

	// WebSocket (authenticated qua query string)
	r.Get("/v1/ws", func(w http.ResponseWriter, r *http.Request) {
		// Lấy token từ query string
		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		claims, err := auth.VerifyToken(jwtSecret, tokenStr)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		ws.ServeWebSocket(hub, store, claims.UserID, w, r)
	})

	return r
}

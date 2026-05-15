package api

import (
	"net/http"

	"nlscada-cloud/internal/auth"
	"nlscada-cloud/internal/db/influxdb"
	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/ws"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func NewRouter(store *postgres.Store, influxReader *influxdb.Reader, influxWriter *influxdb.Writer, jwtSecret string, hub *ws.Hub) *chi.Mux {
	r := chi.NewRouter()
	// Middleware cơ bản của chi (logger, recover)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// Áp dụng CORSMiddleware tự viết của bạn lên toàn cục hệ thống
	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "https://your-production-domain.com"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		MaxAge:           300, // Thời gian cache preflight (giây)
	}))
	// 1. Public - Authentication & User
	authHandler := NewAuthHandler(store, jwtSecret)
	r.Post("/v1/auth/register", authHandler.Register)
	r.Post("/v1/auth/login", authHandler.Login)
	r.Post("/v1/auth/refresh", authHandler.Refresh)
	r.Post("/v1/auth/logout", authHandler.Logout)

	// Authenticated
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware(jwtSecret))

		// User
		userHandler := NewUserHandler(store)
		r.Get("/v1/users/me", userHandler.Me)

		// 2. Quản lý Nhà máy (Sites)
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

			// 3. Quản lý Thành viên (Memberships)
			membershipHandler := NewMembershipHandler(store)
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("admin"))
				r.Get("/members", membershipHandler.List)
				r.Post("/members", membershipHandler.Invite)             // Admin only
				r.Put("/members/{userID}", membershipHandler.UpdateRole) // Admin
				r.Delete("/members/{userID}", membershipHandler.Remove)  // Admin
			})

			// 4. Quản lý Thiết bị (Devices)
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

			// 5. Cấu hình Thẻ & Hạn mức (Tags & Thresholds)
			tagHandler := NewTagHandler(store)
			r.Route("/devices/{deviceID}", func(r chi.Router) {
				r.Get("/tags", tagHandler.List)
				r.Group(func(r chi.Router) {
					r.Use(RequireRole("admin", "operator"))
					r.Post("/tags", tagHandler.Create)
					r.Put("/tags/{tagID}", tagHandler.Update)
				})
				r.Group(func(r chi.Router) {
					r.Use(RequireRole("admin"))
					r.Delete("/tags/{tagID}", tagHandler.Delete)
				})

				// 6. Dữ liệu Chuỗi thời gian & Vận hành (Telemetry & Control)
				// GET /api/v1/sites/{site_id}/devices/{device_id}/telemetry
				// POST /api/v1/sites/{site_id}/devices/{device_id}/control
				// GET /api/v1/sites/{site_id}/devices/{device_id}/control/logs
			})
			// 7. Triggers & Notifications (Cảnh báo & Thông báo)
			// GET /api/v1/sites/{site_id}/alerts
			// POST /api/v1/sites/{site_id}/alerts/{alert_id}/acknowledge
			// POST /api/v1/sites/{site_id}/alerts/{alert_id}/resolve

			// 8. Quản lý Sơ đồ Công nghệ (P&ID Diagrams)
			// POST /api/v1/sites/{site_id}/pid-diagrams
			// GET /api/v1/sites/{site_id}/pid-diagrams
			// GET /api/v1/sites/{site_id}/pid-diagrams/{diagram_id}
			// DELETE /api/v1/sites/{site_id}/pid-diagrams/{diagram_id}
			// POST /api/v1/sites/{site_id}/pid-diagrams/{diagram_id}/widgets
			// PUT /api/v1/sites/{site_id}/pid-diagrams/{diagram_id}/widgets/{widget_id}
			// DELETE /api/v1/sites/{site_id}/pid-diagrams/{diagram_id}/widgets/{widget_id}

			// 9. Báo cáo & Phân tích (Reports & Analytics)
			// GET /api/v1/sites/{site_id}/reports/production
			// GET /api/v1/sites/{site_id}/reports/energy
			// GET /api/v1/sites/{site_id}/reports/custom?from=...&to=...

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

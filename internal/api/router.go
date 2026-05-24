package api

import (
	"net/http"

	_ "nlscada-cloud/docs" // import để swagger có thể tìm thấy doc.go
	"nlscada-cloud/internal/auth"
	"nlscada-cloud/internal/control"
	"nlscada-cloud/internal/db/influxdb"
	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/mqtt"
	"nlscada-cloud/internal/ws"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

func NewRouter(store *postgres.Store, influxReader *influxdb.Reader, influxWriter *influxdb.Writer, jwtSecret string, hub *ws.Hub, mqttClient *mqtt.Client, controlService *control.Service) *chi.Mux {
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

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.WrapHandler)
	// ============================================
	// PUBLIC
	// ============================================
	authHandler := NewAuthHandler(store, jwtSecret)
	r.Post("/v1/auth/register", authHandler.Register)
	r.Post("/v1/auth/login", authHandler.Login)
	r.Post("/v1/auth/refresh", authHandler.Refresh)
	r.Post("/v1/auth/logout", authHandler.Logout)

	// ============================================
	// AUTHENTICATED (không cần site)
	// ============================================
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware(jwtSecret))

		userHandler := NewUserHandler(store)
		r.Get("/v1/users/me", userHandler.Me)

		siteHandler := NewSiteHandler(store)
		r.Get("/v1/sites", siteHandler.List)
		r.Post("/v1/sites", siteHandler.Create)
	})

	// ============================================
	// SITE-SCOPED (yêu cầu membership)
	// ============================================
	r.Route("/v1/sites/{siteID}", func(r chi.Router) {
		r.Use(AuthMiddleware(jwtSecret))
		r.Use(SiteMiddleware(store))

		// --- Site detail ---
		siteHandler := NewSiteHandler(store)
		r.Get("/", siteHandler.Get)       // ai cũng xem
		r.Put("/", siteHandler.Update)    // tự kiểm tra admin trong handler
		r.Delete("/", siteHandler.Delete) // tự kiểm tra admin trong handler
		// --- Memberships (chỉ admin) ---
		membershipHandler := NewMembershipHandler(store, hub)
		r.Post("/leave", membershipHandler.Leave)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin"))
			r.Get("/members", membershipHandler.List)
			r.Post("/members", membershipHandler.Invite)
			r.Put("/members/{userID}", membershipHandler.UpdateRole)
			r.Delete("/members/{userID}", membershipHandler.Remove)
		})

		// --- Devices (xem: tất cả; tạo/sửa/xóa: admin) ---
		deviceHandler := NewDeviceHandler(store)
		tagHandler := NewTagHandler(store)
		dataHandler := NewDataHandler(store, influxReader)
		controlHandler := NewControlHandler(store, mqttClient, controlService)

		r.Get("/devices", deviceHandler.List)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin"))
			r.Post("/devices", deviceHandler.Create)
		})

		r.Route("/devices/{deviceID}", func(r chi.Router) {
			// Device CRUD
			r.Get("/", deviceHandler.Get)
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("admin"))
				r.Put("/", deviceHandler.Update)
				r.Delete("/", deviceHandler.Delete)
			})

			// Tags
			r.Get("/tags", tagHandler.List)
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("admin"))
				r.Post("/tags", tagHandler.Create)
			})

			// Data Query
			r.Get("/data", dataHandler.Query)

			// Control
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("admin", "operator"))
				r.Post("/control", controlHandler.Send)
			})
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("admin", "operator", "auditor"))
				r.Get("/control/logs", controlHandler.Logs)
			})
		})

		// Tags không theo device
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin"))
			r.Put("/tags/{tagID}", tagHandler.Update)
			r.Delete("/tags/{tagID}", tagHandler.Delete)
		})

		// --- Alerts (xem: admin, auditor; tạo/sửa/xóa/thao tác: admin) ---
		alertHandler := NewAlertHandler(store)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin", "auditor"))
			r.Get("/alerts", alertHandler.List)
		})
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin"))
			r.Post("/alerts", alertHandler.Create)
			r.Put("/alerts/{alertID}", alertHandler.Update)
			r.Delete("/alerts/{alertID}", alertHandler.Delete)
			r.Post("/alerts/{alertID}/acknowledge", alertHandler.Acknowledge)
			r.Post("/alerts/{alertID}/resolve", alertHandler.Resolve)
		})

		// --- Audit & Event Logs (xem: admin, auditor) ---
		auditLogHandler := NewAuditLogHandler(store)
		systemEventHandler := NewSystemEventHandler(store)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin", "auditor"))
			r.Get("/logs/audit", auditLogHandler.List)
			r.Get("/logs/audit/{logID}", auditLogHandler.Get)
			r.Get("/logs/events", systemEventHandler.List)
			r.Get("/logs/events/{eventID}", systemEventHandler.Get)
		})

		// --- Retention Policies (xem: admin, auditor; sửa/kích hoạt: admin) ---
		retentionPolicyHandler := NewRetentionPolicyHandler(store)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin", "auditor"))
			r.Get("/retention-policies", retentionPolicyHandler.Get)
		})
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin"))
			r.Put("/retention-policies", retentionPolicyHandler.Update)
			r.Post("/retention-policies/trigger-manual", retentionPolicyHandler.TriggerManual)
		})

		// --- P&ID Diagrams (xem: tất cả; quản lý: admin) ---
		pidDiagramHandler := NewPidDiagramHandler(store)
		r.Get("/pid-diagrams", pidDiagramHandler.List)
		r.Get("/pid-diagrams/{diagramID}", pidDiagramHandler.Get)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin"))
			r.Post("/pid-diagrams", pidDiagramHandler.Create)
			r.Delete("/pid-diagrams/{diagramID}", pidDiagramHandler.Delete)
			r.Post("/pid-diagrams/{diagramID}/widgets", pidDiagramHandler.AddWidget)
			r.Put("/pid-diagrams/{diagramID}/widgets/{widgetID}", pidDiagramHandler.UpdateWidget)
			r.Delete("/pid-diagrams/{diagramID}/widgets/{widgetID}", pidDiagramHandler.DeleteWidget)
		})

		// --- Alert Rules ---
		alertRuleHandler := NewAlertRuleHandler(store, mqttClient)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin", "auditor"))
			r.Get("/alert-rules", alertRuleHandler.List)
			r.Get("/alert-rules/{ruleID}", alertRuleHandler.Get)
		})
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin"))
			r.Post("/alert-rules", alertRuleHandler.Create)
			r.Put("/alert-rules/{ruleID}", alertRuleHandler.Update)
			r.Delete("/alert-rules/{ruleID}", alertRuleHandler.Delete)
		})

		// --- Control Config ---
		controlConfigHandler := NewControlConfigHandler(store, mqttClient)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin", "operator"))
			r.Get("/control-configs", controlConfigHandler.List)
			r.Get("/control-configs/{configID}", controlConfigHandler.Get)
		})
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin"))
			r.Post("/control-configs", controlConfigHandler.Create)
			r.Put("/control-configs/{configID}", controlConfigHandler.Update)
			r.Delete("/control-configs/{configID}", controlConfigHandler.Delete)
		})

		// --- Site Config (tổng hợp cho Gateway) ---
		siteConfigHandler := NewSiteConfigHandler(store)
		r.Get("/config", siteConfigHandler.GetConfig)
	})

	// ============================================
	// WEBSOCKET
	// ============================================
	r.Get("/v1/ws", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "missing session", http.StatusUnauthorized)
			return
		}
		claims, err := auth.VerifyToken(jwtSecret, cookie.Value)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		ws.ServeWebSocket(hub, store, claims.UserID, w, r)
	})

	return r
}

package api

import (
	"net/http"
	"nlscada-cloud/internal/db/influxdb"
	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/ws"

	"github.com/go-chi/chi/v5"
)

func NewRouter(store *postgres.Store, influxReader *influxdb.Reader, influxWriter *influxdb.Writer, jwtSecret string, hub *ws.Hub) *chi.Mux {
	r := chi.NewRouter()

	// Auth handlers (public)
	authHandler := NewAuthHandler(store, jwtSecret)
	r.Post("/v1/auth/register", authHandler.Register)
	r.Post("/v1/auth/login", authHandler.Login)
	r.Post("/v1/auth/refresh", authHandler.Refresh)

	// Authenticated (mọi user đã login)
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware(jwtSecret))

		// User
		userHandler := NewUserHandler(store)
		r.Get("/v1/users/me", userHandler.Me)        // good
		r.Get("/v1/users", userHandler.ListVerified) // good

		// Site management (user đã đăng nhập)
		siteHandler := NewSiteHandler(store)
		r.Post("/v1/sites", siteHandler.Create)  // good
		r.Get("/v1/sites", siteHandler.List)     // good
		r.Get("/v1/sites/{id}", siteHandler.Get) // lỗi 404, endpoint k được khai báo

		// Site member management (chỉ admin của site mới dùng được)
		r.Post("/v1/sites/{id}/members", siteHandler.AddMember)               // thêm member, loại trừ được mail không tồn tại
		r.Get("/v1/sites/{id}/members", siteHandler.ListMembers)              // good
		r.Delete("/v1/sites/{id}/members/{userID}", siteHandler.RemoveMember) // good

		// Các routes dành riêng cho từng site: devices, tags, data
		r.Route("/v1/sites/{siteID}", func(r chi.Router) {
			// Middleware kiểm tra membership: mọi role đều cần có membership để truy cập site
			r.Use(RequireSiteRole(store, "admin", "operator", "viewer"))

			// Devices
			deviceHandler := NewDeviceHandler(store)
			r.Get("/devices", deviceHandler.List)     // good
			r.Get("/devices/{id}", deviceHandler.Get) // {id} = deviceID, good

			// Tạo / sửa device: yêu cầu operator hoặc admin
			r.Group(func(r chi.Router) {
				r.Use(RequireSiteRole(store, "admin", "operator"))
				r.Post("/devices", deviceHandler.Create)     // good
				r.Put("/devices/{id}", deviceHandler.Update) // good
			})

			// Xóa device: chỉ admin
			r.Group(func(r chi.Router) {
				r.Use(RequireSiteRole(store, "admin"))
				r.Delete("/devices/{id}", deviceHandler.Delete) // good
			})

			// Tags
			tagHandler := NewTagHandler(store)
			r.Get("/devices/{deviceID}/tags", tagHandler.List) // good
			r.Group(func(r chi.Router) {
				r.Use(RequireSiteRole(store, "admin", "operator"))
				r.Post("/devices/{deviceID}/tags", tagHandler.Create) // good
				r.Put("/tags/{tagID}", tagHandler.Update)             // good
			})
			r.Group(func(r chi.Router) {
				r.Use(RequireSiteRole(store, "admin"))
				r.Delete("/tags/{tagID}", tagHandler.Delete) // good
			})

			// Data query (tất cả role đều xem được)
			dataHandler := NewDataHandler(influxReader)
			r.Get("/data/{deviceID}", dataHandler.Query) // nice
		})
	})

	// WebSocket (public endpoint, xác thực qua message)
	r.Get("/v1/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWebSocket(hub, w, r)
	})

	return r
}

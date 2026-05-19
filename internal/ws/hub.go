package ws

import (
	"log"
	"net/http"
	"sync"
	"time"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/pkg/channel"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type subscription struct {
	client   *Client
	siteID   string
	deviceID string
}

type controlRequest struct {
	client   *Client
	siteID   string
	deviceID string
	tagName  string
	value    string
}

type forceDisconnectRequest struct {
	UserID uuid.UUID
	SiteID uuid.UUID
}

type Hub struct {
	store           *postgres.Store
	clients         map[*Client]bool
	register        chan *Client
	unregister      chan *Client
	subscribe       chan subscription
	unsubscribe     chan subscription
	controlRequests chan controlRequest
	forceDisconnect chan forceDisconnectRequest
	jwtSecret       string

	// Map: siteID -> deviceID -> set of clients
	devices map[string]map[string]map[*Client]bool
	mu      sync.RWMutex
}

func NewHub(store *postgres.Store, jwtSecret string) *Hub {
	return &Hub{
		store:           store,
		clients:         make(map[*Client]bool),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		subscribe:       make(chan subscription),
		unsubscribe:     make(chan subscription),
		controlRequests: make(chan controlRequest, 100),
		forceDisconnect: make(chan forceDisconnectRequest),
		devices:         make(map[string]map[string]map[*Client]bool),
		jwtSecret:       jwtSecret,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Println("WS client registered")

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.mu.Lock()
				for siteID, devs := range h.devices {
					for devID, cls := range devs {
						delete(cls, client)
						if len(cls) == 0 {
							delete(devs, devID)
						}
					}
					if len(devs) == 0 {
						delete(h.devices, siteID)
					}
				}
				h.mu.Unlock()
				log.Println("WS client unregistered")
			}

		case sub := <-h.subscribe:
			h.mu.Lock()
			if h.devices[sub.siteID] == nil {
				h.devices[sub.siteID] = make(map[string]map[*Client]bool)
			}
			if h.devices[sub.siteID][sub.deviceID] == nil {
				h.devices[sub.siteID][sub.deviceID] = make(map[*Client]bool)
			}
			h.devices[sub.siteID][sub.deviceID][sub.client] = true
			h.mu.Unlock()

		case unsub := <-h.unsubscribe:
			h.mu.Lock()
			if devs, ok := h.devices[unsub.siteID]; ok {
				if cls, ok := devs[unsub.deviceID]; ok {
					delete(cls, unsub.client)
					if len(cls) == 0 {
						delete(devs, unsub.deviceID)
					}
				}
			}
			h.mu.Unlock()

		case update := <-channel.RealTimeDataChan:
			siteID := update.SiteID.String()
			deviceID := update.DeviceID.String()
			h.mu.RLock()
			if devs, ok := h.devices[siteID]; ok {
				if cls, ok := devs[deviceID]; ok {
					for client := range cls {
						msg := client.makeMessage("tag_update", TagUpdatePayload{
							DeviceID:  deviceID,
							Tags:      update.Tags,
							Timestamp: update.Timestamp,
						})
						select {
						case client.send <- msg:
						default:
						}
					}
				}
			}
			h.mu.RUnlock()

		case alert := <-channel.AlertNotificationChan:
			siteID := alert.SiteID.String()
			h.mu.RLock()
			// Gửi alert cho tất cả client trong site có quyền (admin, auditor)
			// Duyệt qua tất cả client, kiểm tra permissions
			for client := range h.clients {
				if role, ok := client.permissions[siteID]; ok && (role == "admin" || role == "auditor") {
					msg := client.makeMessage("alert_new", AlertNewPayload{
						AlertID:  alert.AlertID,
						Severity: alert.Severity,
						Message:  alert.Message,
						DeviceID: alert.DeviceID,
						TagName:  alert.TagName,
					})
					select {
					case client.send <- msg:
					default:
					}
				}
			}
			h.mu.RUnlock()

		case req := <-h.controlRequests:
			// Xử lý điều khiển: gọi ControlHandler hoặc xử lý trực tiếp
			// Tạm thời chỉ gửi phản hồi giả lập
			ack := req.client.makeMessage("control_ack", ControlAckPayload{
				LogID:  "placeholder-log-id",
				Status: "SENT",
			})
			select {
			case req.client.send <- ack:
			default:
			}

		case req := <-h.forceDisconnect:
			h.mu.RLock()
			// Tìm tất cả client có userID trùng và đang hoạt động trong siteID đó
			for client := range h.clients {
				if client.userID == req.UserID {
					// Nếu siteID không nil, kiểm tra client có đang trong site đó không
					if req.SiteID != uuid.Nil {
						if _, ok := client.permissions[req.SiteID.String()]; !ok {
							continue
						}
					}
					// Gửi close message với code 4001
					msg := websocket.FormatCloseMessage(4001, "membership changed")
					client.conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(writeWait))
					client.conn.Close()
					h.unregister <- client
					log.Printf("Force disconnected user %s from site %s", req.UserID, req.SiteID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func ServeWebSocket(hub *Hub, store *postgres.Store, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}
	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}
	client.loadPermissions()
	hub.register <- client
	go client.writePump()
	go client.readPump()
}

// ForceDisconnect được gọi từ bên ngoài (handler) để yêu cầu Hub ngắt kết nối user
func (h *Hub) ForceDisconnect(userID, siteID uuid.UUID) {
	h.forceDisconnect <- forceDisconnectRequest{
		UserID: userID,
		SiteID: siteID,
	}
}

package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"nlscada-cloud/internal/control"
	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/pkg/channel"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
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

type closingRequest struct {
	userID uuid.UUID
	siteID uuid.UUID // có thể nil nếu muốn đóng tất cả
}

type siteInviteRequest struct {
	userID   uuid.UUID
	siteID   uuid.UUID
	siteName string
	role     string
}

type Hub struct {
	store           *postgres.Store
	clients         map[*Client]bool
	register        chan *Client
	unregister      chan *Client // Chỉ nhận từ readPump
	subscribe       chan subscription
	unsubscribe     chan subscription
	controlRequests chan controlRequest
	closing         chan closingRequest // Yêu cầu đóng client từ bên ngoài
	siteInvite      chan siteInviteRequest
	jwtSecret       string

	devices map[string]map[string]map[*Client]bool
	mu      sync.RWMutex

	controlService *control.Service
}

func NewHub(store *postgres.Store, jwtSecret string, controlService *control.Service) *Hub {
	return &Hub{
		store:           store,
		clients:         make(map[*Client]bool),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		subscribe:       make(chan subscription, 100),
		unsubscribe:     make(chan subscription, 100),
		controlRequests: make(chan controlRequest, 100),
		closing:         make(chan closingRequest, 100),
		devices:         make(map[string]map[string]map[*Client]bool),
		siteInvite:      make(chan siteInviteRequest, 100),
		jwtSecret:       jwtSecret,
		controlService:  controlService,
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
						// Gửi tag_update, nếu kênh đầy thì log cảnh báo và bỏ qua
						msg := client.makeMessage("tag_update", TagUpdatePayload{
							DeviceID:  deviceID,
							Tags:      update.Tags,
							Timestamp: update.Timestamp,
						})
						select {
						case client.send <- msg:
						default:
							log.Printf("WS: client %s send buffer full, dropping tag_update", client.userID)
						}
					}
				}
			}
			h.mu.RUnlock()

		case alert := <-channel.AlertNotificationChan:
			siteID := alert.SiteID.String()
			h.mu.RLock()
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
						log.Printf("WS: client %s send buffer full, dropping alert_new", client.userID)
					}
				}
			}
			h.mu.RUnlock()

		case sysEvent := <-channel.SystemEventNotificationChan:
			siteID := sysEvent.SiteID.String()
			h.mu.RLock()
			for client := range h.clients {
				if _, ok := client.permissions[siteID]; ok {
					msg := client.makeMessage("system_event", SystemEventPayload{
						EventID:   sysEvent.EventID,
						EventType: sysEvent.EventType,
						Severity:  sysEvent.Severity,
						Message:   sysEvent.Message,
						DeviceID:  sysEvent.DeviceID,
						Timestamp: sysEvent.Timestamp,
					})
					select {
					case client.send <- msg:
					default:
						log.Printf("WS: client %s send buffer full, dropping system_event", client.userID)
					}
				}
			}
			h.mu.RUnlock()

		case req := <-h.controlRequests:
			siteID, _ := uuid.Parse(req.siteID)
			deviceID, _ := uuid.Parse(req.deviceID)
			logEntry, err := h.controlService.SendControl(context.Background(), siteID, deviceID, req.client.userID, req.tagName, req.value)
			if err != nil {
				ack := req.client.makeMessage("control_ack", ControlAckPayload{LogID: "", Status: "FAILED"})
				select {
				case req.client.send <- ack:
				default:
				}
				continue
			}
			ack := req.client.makeMessage("control_ack", ControlAckPayload{LogID: logEntry.ID.String(), Status: logEntry.Status})
			select {
			case req.client.send <- ack:
			default:
				log.Printf("WS: client %s send buffer full, dropping control_ack", req.client.userID)
			}

		case req := <-h.closing:
			h.mu.RLock()
			for client := range h.clients {
				if client.userID == req.userID {
					if req.siteID != uuid.Nil {
						if _, ok := client.permissions[req.siteID.String()]; !ok {
							continue
						}
					}
					// Gửi tín hiệu đóng, không chặn
					select {
					case client.closeSignal <- struct{}{}:
					default:
						// Tín hiệu đã được gửi trước đó, bỏ qua
					}
				}
			}
			h.mu.RUnlock()

		case invite := <-h.siteInvite:
			h.mu.RLock()
			for client := range h.clients {
				if client.userID == invite.userID {
					// Gửi event site_invited
					msg := client.makeMessage("site_invited", SiteInvitedPayload{
						SiteID:   invite.siteID.String(),
						SiteName: invite.siteName,
						Role:     invite.role,
						Message:  "You have been invited to a new site.",
					})
					select {
					case client.send <- msg:
					default:
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// ReloadPermissions gửi event yêu cầu client reload permissions (dùng khi đổi role)
func (h *Hub) ReloadPermissions(userID, siteID uuid.UUID) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.userID == userID {
			if siteID != uuid.Nil {
				if _, ok := client.permissions[siteID.String()]; !ok {
					continue
				}
			}
			msg := client.makeMessage("permissions_changed", map[string]string{
				"message": "Your permissions have been updated.",
			})
			select {
			case client.send <- msg:
			default:
			}
		}
	}
}

// ForceDisconnect yêu cầu đóng kết nối của một user khỏi site cụ thể
func (h *Hub) ForceDisconnect(userID, siteID uuid.UUID) {
	h.closing <- closingRequest{userID: userID, siteID: siteID}
}

// SendToUser gửi message đến tất cả client có userID trùng khớp
func (h *Hub) SendToUser(userID uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- message:
			default:
				log.Printf("WS: SendToUser buffer full for user %s", userID)
			}
		}
	}
}

// ServeWebSocket nâng cấp HTTP lên WS và đăng ký client mới
func ServeWebSocket(hub *Hub, store *postgres.Store, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}
	client := &Client{
		hub:         hub,
		conn:        conn,
		send:        make(chan []byte, 256),
		closeSignal: make(chan struct{}, 1),
		userID:      userID,
		limiter:     rate.NewLimiter(rate.Limit(maxMsgPerSec), burstSize),
	}
	client.loadPermissions()
	hub.register <- client
	go client.writePump()
	go client.readPump()
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

// NotifyNewMembership gửi thông báo đến user khi được mời vào site mới
func (h *Hub) NotifyNewMembership(userID, siteID uuid.UUID, siteName, role string) {
	h.siteInvite <- siteInviteRequest{
		userID:   userID,
		siteID:   siteID,
		siteName: siteName,
		role:     role,
	}
}

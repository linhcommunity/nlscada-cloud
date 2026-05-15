package ws

import (
	"log"
	"net/http"
	"sync"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/pkg/channel"

	"github.com/google/uuid"
)

type subscription struct {
	client   *Client
	siteID   string
	deviceID string
}

type Hub struct {
	store       *postgres.Store
	clients     map[*Client]bool
	register    chan *Client
	unregister  chan *Client
	subscribe   chan subscription
	unsubscribe chan subscription
	jwtSecret   string

	// Map: siteID -> deviceID -> set of clients
	devices map[string]map[string]map[*Client]bool
	mu      sync.RWMutex
}

func NewHub(store *postgres.Store, jwtSecret string) *Hub {
	return &Hub{
		store:       store,
		clients:     make(map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		subscribe:   make(chan subscription),
		unsubscribe: make(chan subscription),
		devices:     make(map[string]map[string]map[*Client]bool),
		jwtSecret:   jwtSecret,
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
						if _, ok := cls[client]; ok {
							delete(cls, client)
							if len(cls) == 0 {
								delete(devs, devID)
							}
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
			log.Printf("WS client subscribed to device %s in site %s", sub.deviceID, sub.siteID)

		case unsub := <-h.unsubscribe:
			h.mu.Lock()
			if devs, ok := h.devices[unsub.siteID]; ok {
				if cls, ok := devs[unsub.deviceID]; ok {
					delete(cls, unsub.client)
					if len(cls) == 0 {
						delete(devs, unsub.deviceID)
					}
				}
				if len(devs) == 0 {
					delete(h.devices, unsub.siteID)
				}
			}
			h.mu.Unlock()

		case update := <-channel.RealTimeDataChan:
			siteID := update.SiteID.String()
			deviceID := update.DeviceID.String()
			h.mu.RLock()
			if devs, ok := h.devices[siteID]; ok {
				if cls, ok := devs[deviceID]; ok {
					msg := mustMarshal(ServerMessage{
						Type:      "tag_update",
						DeviceID:  deviceID,
						Timestamp: update.Timestamp,
						Tags:      update.Tags,
					})
					for client := range cls {
						select {
						case client.send <- msg:
						default:
						}
					}
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

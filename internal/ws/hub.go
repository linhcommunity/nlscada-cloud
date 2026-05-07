package ws

import (
	"log"
	"net/http"
	"sync"

	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/pkg/channel"
)

type subscription struct {
	client   *Client
	siteID   string
	deviceID string
}

type Hub struct {
	clients     map[*Client]bool
	register    chan *Client
	unregister  chan *Client
	subscribe   chan subscription
	unsubscribe chan subscription
	jwtSecret   string
	store       *postgres.Store

	// Map: siteID -> deviceID -> set of clients
	devices map[string]map[string]map[*Client]bool
	mu      sync.RWMutex
}

func NewHub(jwtSecret string, store *postgres.Store) *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		subscribe:   make(chan subscription),
		unsubscribe: make(chan subscription),
		devices:     make(map[string]map[string]map[*Client]bool),
		jwtSecret:   jwtSecret,
		store:       store,
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
				// Xóa client khỏi tất cả subscription
				h.mu.Lock()
				for _, devs := range h.devices {
					for _, cls := range devs {
						delete(cls, client)
					}
				}
				h.mu.Unlock()
				log.Println("WS client unregistered")
			}

		case sub := <-h.subscribe:
			siteID := sub.siteID
			deviceID := sub.deviceID
			h.mu.Lock()
			if h.devices[siteID] == nil {
				h.devices[siteID] = make(map[string]map[*Client]bool)
			}
			if h.devices[siteID][deviceID] == nil {
				h.devices[siteID][deviceID] = make(map[*Client]bool)
			}
			h.devices[siteID][deviceID][sub.client] = true
			h.mu.Unlock()
			log.Printf("WS client subscribed to device %s in site %s", deviceID, siteID)

		case unsub := <-h.unsubscribe:
			siteID := unsub.siteID
			deviceID := unsub.deviceID
			h.mu.Lock()
			if devs, ok := h.devices[siteID]; ok {
				if cls, ok := devs[deviceID]; ok {
					delete(cls, unsub.client)
					if len(cls) == 0 {
						delete(devs, deviceID)
					}
				}
				if len(devs) == 0 {
					delete(h.devices, siteID)
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

func ServeWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}
	client := &Client{
		hub:     hub,
		conn:    conn,
		send:    make(chan []byte, 256),
		devices: make(map[string]string), // lưu deviceID -> siteID để dễ unsubscribe
	}
	hub.register <- client

	// go client.writePump()
	go client.readPump()
}

package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"nlscada-cloud/internal/auth"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Trong production nên giới hạn origin
	},
}

type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	userID  uuid.UUID
	email   string
	devices map[string]string // deviceID -> siteID
	mu      sync.Mutex
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WS read error: %v", err)
			}
			break
		}

		var msg ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.send <- mustMarshal(ServerMessage{Type: "error", Message: "invalid message format"})
			continue
		}

		switch msg.Action {
		case "auth":
			claims, err := auth.VerifyToken(c.hub.jwtSecret, msg.Token)
			if err != nil {
				c.send <- mustMarshal(ServerMessage{Type: "auth_error", Message: "invalid token"})
				log.Printf("WS auth failed: %v", err)
				return
			}
			c.userID = claims.UserID
			c.email = claims.Email
			c.send <- mustMarshal(ServerMessage{Type: "auth_ok"})
			log.Printf("WS client authenticated: user=%s", claims.UserID)

		case "subscribe":
			if c.userID == uuid.Nil {
				c.send <- mustMarshal(ServerMessage{Type: "error", Message: "authenticate first"})
				continue
			}
			if msg.DeviceID == "" {
				c.send <- mustMarshal(ServerMessage{Type: "error", Message: "device_id required"})
				continue
			}
			// Kiểm tra xem device thuộc site nào và user có quyền không
			var siteID, role string
			err := c.hub.store.Pool.QueryRow(context.Background(),
				`SELECT d.site_id, m.role 
                 FROM devices d
                 JOIN memberships m ON m.site_id = d.site_id AND m.user_id = $1
                 WHERE d.id = $2`,
				c.userID, msg.DeviceID,
			).Scan(&siteID, &role)
			if err != nil {
				c.send <- mustMarshal(ServerMessage{Type: "error", Message: "access denied or device not found"})
				continue
			}

			c.mu.Lock()
			c.devices[msg.DeviceID] = siteID
			c.mu.Unlock()
			c.hub.subscribe <- subscription{client: c, siteID: siteID, deviceID: msg.DeviceID}
			c.send <- mustMarshal(ServerMessage{Type: "subscribed", DeviceID: msg.DeviceID})

		case "unsubscribe":
			if msg.DeviceID == "" {
				c.send <- mustMarshal(ServerMessage{Type: "error", Message: "device_id required"})
				continue
			}
			c.mu.Lock()
			siteID, ok := c.devices[msg.DeviceID]
			if ok {
				delete(c.devices, msg.DeviceID)
			}
			c.mu.Unlock()
			if ok {
				c.hub.unsubscribe <- subscription{client: c, siteID: siteID, deviceID: msg.DeviceID}
				c.send <- mustMarshal(ServerMessage{Type: "unsubscribed", DeviceID: msg.DeviceID})
			}
		}
	}
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

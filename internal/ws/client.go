package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

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
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	userID      uuid.UUID
	permissions map[string]string // key: siteID, value: role
	mu          sync.Mutex
}

func (c *Client) loadPermissions() {
	rows, err := c.hub.store.Pool.Query(context.Background(), // hoặc dùng context từ request
		"SELECT site_id, role FROM memberships WHERE user_id = $1", c.userID)
	if err != nil {
		return
	}
	defer rows.Close()
	c.permissions = make(map[string]string)
	for rows.Next() {
		var siteID uuid.UUID
		var role string
		if err := rows.Scan(&siteID, &role); err == nil {
			c.permissions[siteID.String()] = role
		}
	}
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
		case "sub":
			if msg.SiteID == "" || msg.DeviceID == "" {
				c.send <- mustMarshal(ServerMessage{Type: "error", Message: "site_id and device_id required"})
				continue
			}
			// Kiểm tra quyền
			if _, ok := c.permissions[msg.SiteID]; !ok {
				c.send <- mustMarshal(ServerMessage{Type: "error", Message: "forbidden"})
				continue
			}
			c.hub.subscribe <- subscription{client: c, siteID: msg.SiteID, deviceID: msg.DeviceID}
			c.send <- mustMarshal(ServerMessage{Type: "sub_ok", SiteID: msg.SiteID, DeviceID: msg.DeviceID})
			log.Printf("WS client subscribed to device %s in site %s", msg.DeviceID, msg.SiteID)

		case "unsub":
			if msg.SiteID == "" || msg.DeviceID == "" {
				c.send <- mustMarshal(ServerMessage{Type: "error", Message: "site_id and device_id required"})
				continue
			}
			c.hub.unsubscribe <- subscription{client: c, siteID: msg.SiteID, deviceID: msg.DeviceID}
		}
	}
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			log.Printf("WS sending: %s", string(message)) // <-- thêm dòng này
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

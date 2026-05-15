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
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	userID      uuid.UUID
	permissions map[string]string // siteID -> role
	mu          sync.Mutex
}

func (c *Client) loadPermissions() {
	rows, err := c.hub.store.Pool.Query(context.Background(),
		"SELECT site_id, role FROM memberships WHERE user_id = $1", c.userID)
	if err != nil {
		log.Printf("WS loadPermissions error: %v", err)
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
	log.Printf("WS client %s loaded %d sites", c.userID, len(c.permissions))
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

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.send <- c.errorMessage("invalid message format", "PARSE_ERROR")
			continue
		}

		switch msg.Event {
		case "subscribe":
			var p SubscribePayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				c.send <- c.errorMessage("invalid subscribe payload", "PARSE_ERROR")
				continue
			}
			if p.SiteID == "" || p.DeviceID == "" {
				c.send <- c.errorMessage("site_id and device_id required", "INVALID_PARAMS")
				continue
			}

			// Kiểm tra quyền
			if _, ok := c.permissions[p.SiteID]; !ok {
				c.send <- c.makeMessage("sub_error", ErrorPayload{Message: "forbidden", Code: "FORBIDDEN"})
				continue
			}

			// Kiểm tra device tồn tại trong site
			var exists bool
			err := c.hub.store.Pool.QueryRow(context.Background(),
				"SELECT EXISTS(SELECT 1 FROM devices WHERE id = $1 AND site_id = $2)",
				p.DeviceID, p.SiteID).Scan(&exists)
			if err != nil || !exists {
				c.send <- c.makeMessage("sub_error", ErrorPayload{Message: "device not found in site", Code: "NOT_FOUND"})
				continue
			}

			c.hub.subscribe <- subscription{client: c, siteID: p.SiteID, deviceID: p.DeviceID}
			c.send <- c.makeMessage("sub_ok", SubOkPayload{SiteID: p.SiteID, DeviceID: p.DeviceID})
			log.Printf("WS client subscribed to device %s in site %s", p.DeviceID, p.SiteID)

		case "unsubscribe":
			var p SubscribePayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				c.send <- c.errorMessage("invalid unsubscribe payload", "PARSE_ERROR")
				continue
			}
			if p.SiteID == "" || p.DeviceID == "" {
				c.send <- c.errorMessage("site_id and device_id required", "INVALID_PARAMS")
				continue
			}
			c.hub.unsubscribe <- subscription{client: c, siteID: p.SiteID, deviceID: p.DeviceID}

		case "control":
			var p ControlPayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				c.send <- c.errorMessage("invalid control payload", "PARSE_ERROR")
				continue
			}
			if p.DeviceID == "" || p.TagName == "" || p.Value == "" {
				c.send <- c.errorMessage("device_id, tag_name, value required", "INVALID_PARAMS")
				continue
			}

			// Xác định site của device và kiểm tra quyền
			var siteID uuid.UUID
			err := c.hub.store.Pool.QueryRow(context.Background(),
				"SELECT site_id FROM devices WHERE id = $1", p.DeviceID).Scan(&siteID)
			if err != nil {
				c.send <- c.makeMessage("control_ack", ControlAckPayload{LogID: "", Status: "FAILED"})
				continue
			}
			role, ok := c.permissions[siteID.String()]
			if !ok || (role != "admin" && role != "operator") {
				c.send <- c.makeMessage("control_ack", ControlAckPayload{LogID: "", Status: "FORBIDDEN"})
				continue
			}

			// Gửi vào channel để ControlHandler xử lý (hoặc xử lý trực tiếp ở đây)
			c.hub.controlRequests <- controlRequest{
				client:   c,
				siteID:   siteID.String(),
				deviceID: p.DeviceID,
				tagName:  p.TagName,
				value:    p.Value,
			}
		}
	}
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

func (c *Client) makeMessage(event string, payload interface{}) []byte {
	data, _ := json.Marshal(payload)
	msg := WSMessage{
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   data,
	}
	raw, _ := json.Marshal(msg)
	return raw
}

func (c *Client) errorMessage(message, code string) []byte {
	return c.makeMessage("error", ErrorPayload{Message: message, Code: code})
}

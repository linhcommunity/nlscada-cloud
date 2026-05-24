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
	"golang.org/x/time/rate"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	// Rate limiting: tối đa 10 message/giây, burst 5
	maxMsgPerSec = 10
	burstSize    = 5
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Kiểm tra Origin – chỉ cho phép domain của WebBase
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// Cho phép localhost cho dev, thêm domain production sau
		allowedOrigins := []string{"http://localhost:3000", "http://localhost:5173", "https://yourdomain.com"}
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}
		log.Printf("WS: Origin not allowed: %s", origin)
		return false
	},
}

type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte   // buffered (256)
	closeSignal chan struct{} // buffered 1, tín hiệu yêu cầu đóng từ Hub
	userID      uuid.UUID
	permissions map[string]string // siteID -> role
	limiter     *rate.Limiter     // giới hạn tốc độ gửi message
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
		// Chỉ unregister một lần duy nhất
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
		select {
		case <-c.closeSignal:
			// Hub yêu cầu đóng kết nối (ví dụ membership thay đổi)
			msg := websocket.FormatCloseMessage(4001, "membership changed")
			c.conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(writeWait))
			return
		default:
		}

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WS read error: %v", err)
			}
			break
		}

		// Rate limiting
		if !c.limiter.Allow() {
			c.send <- c.errorMessage("rate limit exceeded", "RATE_LIMITED")
			continue
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
			// Kiểm tra quyền
			if _, ok := c.permissions[p.SiteID]; !ok {
				c.send <- c.makeMessage("sub_error", ErrorPayload{Message: "forbidden", Code: "FORBIDDEN"})
				continue
			}
			// Kiểm tra device
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

		case "unsubscribe":
			var p SubscribePayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				continue
			}
			c.hub.unsubscribe <- subscription{client: c, siteID: p.SiteID, deviceID: p.DeviceID}
			c.send <- c.makeMessage("unsub_ok", SubOkPayload{SiteID: p.SiteID, DeviceID: p.DeviceID})

		case "control":
			var p ControlPayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				c.send <- c.errorMessage("invalid control payload", "PARSE_ERROR")
				continue
			}
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

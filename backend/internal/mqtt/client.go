package mqtt

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client wraps paho MQTT client with auto-reconnect
type Client struct {
	client mqtt.Client
}

// NewClient tạo MQTT client mới và kết nối đến broker
func NewClient(broker, clientID, username, password string, handlers map[string]struct {
	Qos     byte
	Handler mqtt.MessageHandler
}) *Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(5 * time.Second)
	opts.SetCleanSession(false) // giữ session để không mất message khi mất kết nối tạm thời

	opts.OnConnect = func(c mqtt.Client) {
		log.Println("MQTT connected, subscribing topics...")
		for topic, cfg := range handlers {
			if token := c.Subscribe(topic, cfg.Qos, cfg.Handler); token.Wait() && token.Error() != nil {
				log.Printf("Subscribe error [%s]: %v", topic, token.Error())
			} else {
				log.Printf("Subscribed to %s (qos=%d)", topic, cfg.Qos)
			}
		}
	}

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	}

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connect failed: %v", token.Error())
	}

	return &Client{client: c}
}

// Subscribe đăng ký một subscription với handler
func (c *Client) Subscribe(topic string, qos byte, handler mqtt.MessageHandler) {
	token := c.client.Subscribe(topic, qos, handler)
	if token.Wait() && token.Error() != nil {
		log.Printf("Subscribe error [%s]: %v", topic, token.Error())
	} else {
		log.Printf("Subscribed to %s", topic)
	}
}

// Publish gửi message lên topic
func (c *Client) Publish(topic string, qos byte, payload interface{}) {
	token := c.client.Publish(topic, qos, false, payload)
	if token.Wait() && token.Error() != nil {
		log.Printf("Publish error [%s]: %v", topic, token.Error())
	}
}

// Disconnect ngắt kết nối MQTT
func (c *Client) Disconnect() {
	c.client.Disconnect(250)
	fmt.Println("MQTT disconnected")
}

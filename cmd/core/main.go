package main

import (
	"log"
	"net/http"
	"nlscada-cloud/internal/api"
	"nlscada-cloud/internal/config"
	"nlscada-cloud/internal/control"
	"nlscada-cloud/internal/db/influxdb"
	"nlscada-cloud/internal/db/postgres"
	"nlscada-cloud/internal/ingest"
	"nlscada-cloud/internal/mqtt"
	"nlscada-cloud/internal/ws"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-chi/chi/v5"
)

// @title           NL SCADA Center API
// @version         1.0
// @description     Backend API for NL SCADA Cloud
// @contact.name    Linh Community
// @contact.url     https://github.com/linhcommunity/nlscada-cloud

// @host            localhost:8080
// @BasePath        /v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Sử dụng "Bearer <token>", nhưng cookie session_token cũng được chấp nhận khi gọi từ browser
func main() {
	cfg := config.Load()

	// PostgreSQL
	pgStore, err := postgres.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("PostgreSQL: %v", err)
	}
	defer pgStore.Close()

	// InfluxDB writer
	influxWriter, err := influxdb.NewWriter(cfg.InfluxURL, cfg.InfluxToken, "nlscada", "nlscada_metrics")
	if err != nil {
		log.Fatalf("InfluxDB writer: %v", err)
	}
	defer influxWriter.Close()

	// InfluxDB reader
	influxReader, err := influxdb.NewReader(cfg.InfluxURL, cfg.InfluxToken, "nlscada")
	if err != nil {
		log.Fatalf("InfluxDB reader: %v", err)
	}
	defer influxReader.Close()

	// Khai báo ingestService trước để dùng trong closure
	var ingestService *ingest.Service
	// Chuẩn bị handlers cho MQTT
	handlers := map[string]struct {
		Qos     byte
		Handler paho.MessageHandler
	}{
		"site/+/device/+/data": {
			Qos: 1,
			Handler: func(client paho.Client, msg paho.Message) {
				log.Printf("MQTT received: topic=%s payload=%s", msg.Topic(), string(msg.Payload()))
				ingestService.HandleData(msg.Topic(), msg.Payload())
			},
		},
		"site/+/device/+/heartbeat": {
			Qos: 1,
			Handler: func(client paho.Client, msg paho.Message) {
				log.Printf("MQTT heartbeat: topic=%s", msg.Topic())
				ingestService.HandleData(msg.Topic(), msg.Payload())
			},
		},
		"site/+/device/+/event": {
			Qos: 1,
			Handler: func(client paho.Client, msg paho.Message) {
				log.Printf("MQTT event: topic=%s payload=%s", msg.Topic(), string(msg.Payload()))
				ingestService.HandleEvent(msg.Topic(), msg.Payload())
			},
		},
	}

	// Tạo MQTT client (sẽ tự connect và subscribe trong OnConnect)
	mqttClient := mqtt.NewClient(cfg.MQTTBroker, "core-service", cfg.MQTTUser, cfg.MQTTPass, handlers)
	defer mqttClient.Disconnect()

	controlService := control.NewService(pgStore, mqttClient)
	// WebSocket Hub
	wsHub := ws.NewHub(pgStore, cfg.JWTSecret, controlService)
	go wsHub.Run()

	// Gán ingestService sau khi có wsHub
	ingestService = ingest.NewService(pgStore, influxWriter, wsHub)

	// Bắt đầu kiểm tra offline định kỳ
	ingestService.StartOfflineChecker(30 * time.Second)

	// Router
	router := api.NewRouter(pgStore, influxReader, influxWriter, cfg.JWTSecret, wsHub, mqttClient, controlService)
	err = chi.Walk(router, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		log.Printf("Route: %s %s", method, route)
		return nil
	})
	if err != nil {
		log.Fatalf("Chi Walk: %v", err)
	}

	log.Println("NL SCADA Core listening on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("HTTP server: %v", err)
	}
}

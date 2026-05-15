package config

import (
	"os"
)

type Config struct {
	DatabaseURL string
	InfluxURL   string
	InfluxToken string
	MQTTBroker  string
	MQTTUser    string
	MQTTPass    string
	JWTSecret   string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://nlscada:Nlscada@2025@localhost:5432/nlscada?sslmode=disable"),
		InfluxURL:   getEnv("INFLUXDB_URL", "http://localhost:8086"),
		InfluxToken: getEnv("INFLUXDB_TOKEN", "my-super-secret-token"),
		MQTTBroker:  getEnv("MQTT_BROKER", "tcp://localhost:1883"),
		MQTTUser:    getEnv("MQTT_USER", "core-service"),
		MQTTPass:    getEnv("MQTT_PASS", "CoreService@2025"),
		JWTSecret:   getEnv("JWT_SECRET", "dev-secret-change-me"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

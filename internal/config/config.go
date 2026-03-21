package config

import (
	"os"
)

type Config struct {
	APIPort       string
	SignalingPort string
	SFUPort       string
	PostgresDSN   string
	RedisAddr     string
	JWTSecret     string
	JWTRefresh    string
	STUNServers   []string
	PublicIP      string
}

func Load() *Config {
	return &Config{
		APIPort:       getEnv("API_PORT", "8080"),
		SignalingPort: getEnv("SIGNALING_PORT", "8081"),
		SFUPort:       getEnv("SFU_PORT", "8082"),
		PostgresDSN:   getEnv("POSTGRES_DSN", "host=localhost user=postgres password=postgres dbname=meeting port=5432 sslmode=disable"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:     getEnv("JWT_SECRET", "meeting-secret-key-2024"),
		JWTRefresh:    getEnv("JWT_REFRESH_SECRET", "meeting-refresh-secret-2024"),
		STUNServers:   []string{"stun:stun.l.google.com:19302"},
		PublicIP:      getEnv("PUBLIC_IP", "127.0.0.1"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

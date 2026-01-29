package realtime

import "os"

type Config struct {
	NatsURL      string
	TenantID     string
	JWTSecret    string
	RealtimePort string
}

func LoadConfig() Config {
	return Config{
		NatsURL:      getEnv("NATS_URL", "nats://localhost:4222"),
		TenantID:     getEnv("TENANT_ID", "default"),
		JWTSecret:    getEnv("JWT_SECRET", ""),
		RealtimePort: getEnv("REALTIME_PORT", ":8081"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

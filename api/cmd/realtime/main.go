package main

import (
	"api/internal/realtime"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := realtime.LoadConfig()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	hub := realtime.NewHub()
	go hub.Run()

	bridge, err := realtime.NewNATSBridge(cfg.NatsURL, cfg.TenantID, hub)
	if err != nil {
		log.Fatalf("NATS bridge: %v", err)
	}
	defer bridge.Close()

	if err := bridge.Subscribe(); err != nil {
		log.Fatalf("NATS subscribe: %v", err)
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		realtime.ServeWS(hub, cfg.JWTSecret, w, r)
	})

	log.Printf("Realtime service listening on %s", cfg.RealtimePort)
	if err := http.ListenAndServe(cfg.RealtimePort, nil); err != nil {
		log.Fatalf("server: %v", err)
	}
}

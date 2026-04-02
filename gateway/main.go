package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newProxy(target string) *httputil.ReverseProxy {
	log.Printf("Proxying to %s", target)
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid upstream URL %s: %v", target, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 60 * time.Second,
	}
	return proxy
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	pipelineOrigin := getEnv("PIPELINE_API_URL", "http://localhost:8080")
	vizOrigin := getEnv("VIZ_API_URL", "http://localhost:3030")
	wsOrigin := getEnv("WS_API_URL", "http://localhost:8081")
	port := getEnv("GATEWAY_PORT", ":8000")

	pipelineProxy := newProxy(pipelineOrigin)
	vizProxy := newProxy(vizOrigin)
	wsProxy := newProxy(wsOrigin)

	mux := http.NewServeMux()

	// /api/* → pipeline API (Go/Gin, :8080)
	mux.Handle("/api/", pipelineProxy)

	// /viz/* → viz API (Rust/Axum, :3030), strip /viz prefix
	mux.HandleFunc("/viz/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/viz")
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		if r.URL.RawPath != "" {
			r.URL.RawPath = strings.TrimPrefix(r.URL.RawPath, "/viz")
		}
		vizProxy.ServeHTTP(w, r)
	})

	// /ws → realtime WebSocket server (:8081)
	mux.Handle("/ws", wsProxy)
	mux.Handle("/ws/", wsProxy)

	log.Printf("Gateway listening on %s", port)
	log.Printf("  /api/*  → %s", pipelineOrigin)
	log.Printf("  /viz/*  → %s (strips /viz prefix)", vizOrigin)
	log.Printf("  /ws     → %s", wsOrigin)

	log.Fatal(http.ListenAndServe(port, corsMiddleware(mux)))
}

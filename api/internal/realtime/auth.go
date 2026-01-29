package realtime

import (
	"api/pkg"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS handles the WebSocket upgrade with JWT authentication via query param.
func ServeWS(hub *Hub, jwtSecret string, w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	if _, err := pkg.ValidateToken(token, jwtSecret); err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(hub, conn)
	hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

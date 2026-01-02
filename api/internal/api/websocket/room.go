package websocket

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Room represents a collaboration room for a specific job
type Room struct {
	ID      uint
	JobID   uint
	Clients map[string]*Client
	mu      sync.RWMutex
	Logger  zerolog.Logger
}

func NewRoom(jobID uint, logger zerolog.Logger) *Room {
	return &Room{
		JobID:   jobID,
		Clients: make(map[string]*Client),
		Logger:  logger,
	}
}

// AddClient adds a client to the room
func (r *Room) AddClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Clients[client.ID] = client
	r.Logger.Info().
		Uint("jobId", r.JobID).
		Str("clientId", client.ID).
		Uint("userId", client.UserID).
		Int("totalClients", len(r.Clients)).
		Msg("Client joined room")

	// Notify other users in the room
	r.broadcastUserJoin(client)
}

// RemoveClient removes a client from the room
func (r *Room) RemoveClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Clients[client.ID]; exists {
		delete(r.Clients, client.ID)
		r.Logger.Info().
			Uint("jobId", r.JobID).
			Str("clientId", client.ID).
			Uint("userId", client.UserID).
			Int("remainingClients", len(r.Clients)).
			Msg("Client left room")

		// Notify other users in the room
		r.broadcastUserLeave(client)
	}
}

// Broadcast sends a message to all clients in the room
func (r *Room) Broadcast(message Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, client := range r.Clients {
		select {
		case client.Send <- message:
		default:
			// Client's send channel is full, skip
			r.Logger.Warn().
				Str("clientId", client.ID).
				Msg("Client send buffer full, message dropped")
		}
	}
}

// BroadcastExcept sends a message to all clients in the room except the sender
func (r *Room) BroadcastExcept(message Message, senderID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, client := range r.Clients {
		if client.ID == senderID {
			continue
		}

		select {
		case client.Send <- message:
		default:
			r.Logger.Warn().
				Str("clientId", client.ID).
				Msg("Client send buffer full, message dropped")
		}
	}
}

// GetActiveUsers returns a list of active users in the room
func (r *Room) GetActiveUsers() []UserInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]UserInfo, 0, len(r.Clients))
	seen := make(map[uint]bool) // Track unique user IDs

	for _, client := range r.Clients {
		if !seen[client.UserID] {
			users = append(users, UserInfo{
				UserID:   client.UserID,
				Username: client.Username,
				Color:    client.Color,
			})
			seen[client.UserID] = true
		}
	}

	return users
}

// IsEmpty returns true if the room has no clients
func (r *Room) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Clients) == 0
}

// ClientCount returns the number of clients in the room
func (r *Room) ClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Clients)
}

// broadcastUserJoin notifies all clients that a user joined
func (r *Room) broadcastUserJoin(client *Client) {
	message := NewUserJoinMessage(
		r.JobID,
		client.UserID,
		client.Username,
		UserInfo{
			UserID:   client.UserID,
			Username: client.Username,
			Color:    client.Color,
		},
	)

	// Send to all clients including the new one
	for _, c := range r.Clients {
		select {
		case c.Send <- message:
		default:
		}
	}

	// Send active users list to the new client
	activeUsers := r.GetActiveUsers()
	if len(activeUsers) > 0 {
		usersMessage := Message{
			Type:      MessageTypeUserJoin,
			JobID:     r.JobID,
			UserID:    0, // System message
			Username:  "system",
			Timestamp: time.Now(),
			Data: map[string]any{
				"activeUsers": activeUsers,
			},
		}
		select {
		case client.Send <- usersMessage:
		default:
		}
	}
}

// broadcastUserLeave notifies all clients that a user left
func (r *Room) broadcastUserLeave(client *Client) {
	message := NewUserLeaveMessage(
		r.JobID,
		client.UserID,
		client.Username,
		UserInfo{
			UserID:   client.UserID,
			Username: client.Username,
			Color:    client.Color,
		},
	)

	for _, c := range r.Clients {
		select {
		case c.Send <- message:
		default:
		}
	}
}

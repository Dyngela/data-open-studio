package websocket

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Hub maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	// Rooms indexed by job ID
	Rooms map[uint]*Room

	// Register requests from clients
	Register chan *Client

	// Unregister requests from clients
	Unregister chan *Client

	// Broadcast messages to clients in a specific room
	Broadcast chan Message

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Logger
	Logger zerolog.Logger
}

func NewHub(logger zerolog.Logger) *Hub {
	return &Hub{
		Rooms:      make(map[uint]*Room),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan Message, 256),
		Logger:     logger,
	}
}

// Run starts the hub's main event loop
func (h *Hub) Run() {
	// Cleanup ticker for removing empty rooms
	cleanupTicker := time.NewTicker(5 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case message := <-h.Broadcast:
			h.broadcastMessage(message)

		case <-cleanupTicker.C:
			h.cleanupEmptyRooms()
		}
	}
}

// registerClient registers a new client to a room
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Get or create room for the job
	room, exists := h.Rooms[client.JobID]
	if !exists {
		room = NewRoom(client.JobID, h.Logger)
		h.Rooms[client.JobID] = room
		h.Logger.Info().Uint("jobId", client.JobID).Msg("Created new room")
	}

	// Add client to room
	room.AddClient(client)
}

// unregisterClient unregisters a client from a room
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.Rooms[client.JobID]
	if !exists {
		return
	}

	// Remove client from room
	room.RemoveClient(client)
	close(client.Send)

	// Remove room if empty
	if room.IsEmpty() {
		delete(h.Rooms, client.JobID)
		h.Logger.Info().Uint("jobId", client.JobID).Msg("Removed empty room")
	}
}

// broadcastMessage broadcasts a message to the appropriate room
// Note: Messages are already processed by clients before reaching here
func (h *Hub) broadcastMessage(message Message) {
	h.mu.RLock()
	room, exists := h.Rooms[message.JobID]
	h.mu.RUnlock()

	if !exists {
		h.Logger.Warn().
			Uint("jobId", message.JobID).
			Str("type", string(message.Type)).
			Msg("Room not found for broadcast")
		return
	}

	// Just broadcast - no processing here (already done in client)
	room.Broadcast(message)

	h.Logger.Debug().
		Str("type", string(message.Type)).
		Uint("jobId", message.JobID).
		Uint("userId", message.UserID).
		Msg("Broadcasted message")
}

// cleanupEmptyRooms removes empty rooms
func (h *Hub) cleanupEmptyRooms() {
	h.mu.Lock()
	defer h.mu.Unlock()

	emptyRooms := make([]uint, 0)
	for jobID, room := range h.Rooms {
		if room.IsEmpty() {
			emptyRooms = append(emptyRooms, jobID)
		}
	}

	for _, jobID := range emptyRooms {
		delete(h.Rooms, jobID)
		h.Logger.Info().Uint("jobId", jobID).Msg("Cleaned up empty room")
	}

	if len(emptyRooms) > 0 {
		h.Logger.Info().
			Int("cleanedRooms", len(emptyRooms)).
			Int("activeRooms", len(h.Rooms)).
			Msg("Room cleanup completed")
	}
}

// GetRoomStats returns statistics about active rooms
func (h *Hub) GetRoomStats() map[uint]int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := make(map[uint]int)
	for jobID, room := range h.Rooms {
		stats[jobID] = room.ClientCount()
	}
	return stats
}

// GetActiveUsersInRoom returns active users in a specific room
func (h *Hub) GetActiveUsersInRoom(jobID uint) []UserInfo {
	h.mu.RLock()
	room, exists := h.Rooms[jobID]
	h.mu.RUnlock()

	if !exists {
		return []UserInfo{}
	}

	return room.GetActiveUsers()
}

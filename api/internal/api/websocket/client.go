package websocket

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

type Client struct {
	ID           string
	UserID       uint
	Username     string
	JobID        uint
	Color        string
	Hub          *Hub
	Conn         *websocket.Conn
	Send         chan Message
	Processor    *MessageProcessor
	ProcessQueue chan Message
	Logger       zerolog.Logger
}

func NewClient(id string, userID uint, username string, jobID uint, hub *Hub, conn *websocket.Conn, processor *MessageProcessor, logger zerolog.Logger) *Client {
	client := &Client{
		ID:           id,
		UserID:       userID,
		Username:     username,
		JobID:        jobID,
		Color:        generateUserColor(userID),
		Hub:          hub,
		Conn:         conn,
		Send:         make(chan Message, 256),
		Processor:    processor,
		ProcessQueue: make(chan Message, 100),
		Logger:       logger,
	}

	// Start the sequential processor worker
	go client.processWorker()

	return client
}

func (c *Client) ReadPump() {
	defer func() {
		close(c.ProcessQueue) // Close the queue to stop the worker
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Logger.Error().Err(err).Str("clientId", c.ID).Msg("WebSocket read error")
			}
			break
		}

		var msg Message
		if err = json.Unmarshal(messageBytes, &msg); err != nil {
			c.Logger.Error().Err(err).Str("clientId", c.ID).Msg("Failed to unmarshal message")
			c.sendError("Invalid message format", err)
			continue
		}

		// Set metadata
		msg.UserID = c.UserID
		msg.Username = c.Username
		msg.JobID = c.JobID
		msg.Timestamp = time.Now()

		// Validate message
		if !c.validateMessage(&msg) {
			continue
		}

		// Fast path: Messages that don't need processing (cursor, chat, etc.)
		if !c.requiresProcessing(msg.Type) {
			c.Hub.Broadcast <- msg
			continue
		}

		// Slow path: Queue DB operations for sequential processing
		// This ensures operations are executed in order while not blocking ReadPump
		select {
		case c.ProcessQueue <- msg:
			// Successfully queued for processing
		default:
			// Queue is full - send error to client
			c.Logger.Warn().
				Str("type", string(msg.Type)).
				Msg("Process queue full, dropping message")
			c.sendError("Server is busy, please try again")
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			messageBytes, err := json.Marshal(message)
			if err != nil {
				c.Logger.Error().Err(err).Msg("Failed to marshal message")
				continue
			}

			w.Write(messageBytes)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				msg := <-c.Send
				msgBytes, _ := json.Marshal(msg)
				w.Write(msgBytes)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// validateMessage validates incoming messages
func (c *Client) validateMessage(msg *Message) bool {
	// Ensure message is for the correct job
	if msg.JobID != 0 && msg.JobID != c.JobID {
		c.sendError("Message job ID does not match connection job ID")
		return false
	}

	return true
}

// sendError sends an error message to the client
func (c *Client) sendError(errorMsg string, errs ...error) {
	c.Send <- NewErrorMessage(c.JobID, c.UserID, c.Username, errorMsg, errs...)
}

// processWorker processes messages from the queue sequentially
// This ensures DB operations maintain their order
func (c *Client) processWorker() {
	c.Logger.Debug().Str("clientId", c.ID).Msg("Process worker started")

	for msg := range c.ProcessQueue {
		// Process message (DB operation)
		if c.Processor != nil {
			processedMsg, err := c.Processor.ProcessMessage(&msg)
			if err != nil {
				c.Logger.Error().
					Err(err).
					Str("type", string(msg.Type)).
					Uint("userId", msg.UserID).
					Msg("Failed to process message")

				// Send error directly to this client only
				c.sendError(err.Error())
				continue
			}

			// Send processed message to hub for broadcasting
			c.Hub.Broadcast <- *processedMsg

			c.Logger.Debug().
				Str("type", string(msg.Type)).
				Uint("userId", msg.UserID).
				Msg("Message processed successfully")
		}
	}

	c.Logger.Debug().Str("clientId", c.ID).Msg("Process worker stopped")
}

// requiresProcessing checks if a message type requires database processing
func (c *Client) requiresProcessing(msgType MessageType) bool {
	switch msgType {
	case MessageTypeJobCreate, MessageTypeJobUpdate, MessageTypeJobDelete, MessageTypeJobExecute:
		return true
	default:
		return false
	}
}

// generateUserColor generates a consistent color for a user based on their ID
func generateUserColor(userID uint) string {
	colors := []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#FFA07A",
		"#98D8C8", "#F7DC6F", "#BB8FCE", "#85C1E2",
		"#F8B739", "#52B788", "#E76F51", "#2A9D8F",
	}
	return colors[userID%uint(len(colors))]
}

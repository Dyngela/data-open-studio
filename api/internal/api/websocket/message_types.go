package websocket

import (
	"time"
)

// Message is the base message structure
// Data field uses 'any' to allow different types through channels
type Message struct {
	Type      MessageType `json:"type"`
	JobID     uint        `json:"jobId,omitempty"`
	UserID    uint        `json:"userId"`
	Username  string      `json:"username"`
	Timestamp time.Time   `json:"timestamp"`
	Data      any         `json:"data"`
}

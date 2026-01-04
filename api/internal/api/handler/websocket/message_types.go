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

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Job operations
	MessageTypeJobUpdate  MessageType = "job_update"
	MessageTypeJobDelete  MessageType = "job_delete"
	MessageTypeJobCreate  MessageType = "job_create"
	MessageTypeJobExecute MessageType = "job_execute"
	MessageTypeJobGet     MessageType = "job_get"

	// User interactions
	MessageTypeCursorMove MessageType = "cursor_move"
	MessageTypeChat       MessageType = "chat"
	MessageTypeUserJoin   MessageType = "user_join"
	MessageTypeUserLeave  MessageType = "user_leave"

	// System messages
	MessageTypeError MessageType = "error"
	MessageTypePing  MessageType = "ping"
	MessageTypePong  MessageType = "pong"

	// Metadata operations
	MessageTypeMetadataCreate MessageType = "metadata_create"
	MessageTypeMetadataUpdate MessageType = "metadata_update"
	MessageTypeMetadataDelete MessageType = "metadata_delete"
	MessageTypeMetadataGet    MessageType = "metadata_get"
)

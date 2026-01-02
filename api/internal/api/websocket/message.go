package websocket

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Job operations
	MessageTypeJobUpdate  MessageType = "job_update"
	MessageTypeJobDelete  MessageType = "job_delete"
	MessageTypeJobCreate  MessageType = "job_create"
	MessageTypeJobExecute MessageType = "job_execute"

	// User interactions
	MessageTypeCursorMove MessageType = "cursor_move"
	MessageTypeChat       MessageType = "chat"
	MessageTypeUserJoin   MessageType = "user_join"
	MessageTypeUserLeave  MessageType = "user_leave"

	// System messages
	MessageTypeError MessageType = "error"
	MessageTypePing  MessageType = "ping"
	MessageTypePong  MessageType = "pong"
)

// JobUpdate represents a job update event
type JobUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Active      *bool   `json:"active,omitempty"`
	Nodes       any     `json:"nodes,omitempty"`
}

// UserInfo represents user information in the room
type UserInfo struct {
	UserID   uint   `json:"userId"`
	Username string `json:"username"`
	Color    string `json:"color"`
}

// ErrorMessage represents an error message
type ErrorMessage struct {
	Error string `json:"error"`
}

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
	MessageTypeJobUpdate   MessageType = "job_update"
	MessageTypeJobDelete   MessageType = "job_delete"
	MessageTypeJobCreate   MessageType = "job_create"
	MessageTypeJobExecute  MessageType = "job_execute"
	MessageTypeJobGet      MessageType = "job_get"
	MessageTypeJobProgress MessageType = "job_progress"

	// User interactions
	MessageTypeCursorMove MessageType = "cursor_move"
	MessageTypeChat       MessageType = "chat"
	MessageTypeUserJoin   MessageType = "user_join"
	MessageTypeUserLeave  MessageType = "user_leave"

	// System messages
	MessageTypeError MessageType = "error"
	MessageTypePing  MessageType = "ping"
	MessageTypePong  MessageType = "pong"

	// DB Metadata operations
	MessageTypeDbMetadataCreate  MessageType = "db_metadata_create"
	ResponseDbMetadataCreate     MessageType = "response_db_metadata_create"
	MessageTypeDbMetadataUpdate  MessageType = "db_metadata_update"
	ResponseDbMetadataUpdate     MessageType = "response_db_metadata_update"
	MessageTypeDbMetadataDelete  MessageType = "db_metadata_delete"
	ResponseDbMetadataDelete     MessageType = "response_db_metadata_delete"
	MessageTypeDbMetadataGet     MessageType = "db_metadata_get"
	ResponseDbMetadataGet        MessageType = "response_db_metadata_get"
	MessageTypeDbMetadataGetAll  MessageType = "db_metadata_get_all"
	ResponseDbMetadataGetAll     MessageType = "response_db_metadata_get_all"

	// SFTP Metadata operations
	MessageTypeSftpMetadataCreate  MessageType = "sftp_metadata_create"
	ResponseSftpMetadataCreate     MessageType = "response_sftp_metadata_create"
	MessageTypeSftpMetadataUpdate  MessageType = "sftp_metadata_update"
	ResponseSftpMetadataUpdate     MessageType = "response_sftp_metadata_update"
	MessageTypeSftpMetadataDelete  MessageType = "sftp_metadata_delete"
	ResponseSftpMetadataDelete     MessageType = "response_sftp_metadata_delete"
	MessageTypeSftpMetadataGet     MessageType = "sftp_metadata_get"
	ResponseSftpMetadataGet        MessageType = "response_sftp_metadata_get"
	MessageTypeSftpMetadataGetAll  MessageType = "sftp_metadata_get_all"
	ResponseSftpMetadataGetAll     MessageType = "response_sftp_metadata_get_all"

	// DB nodes operations
	MessageTypeDbNodeGuessDataModel MessageType = "db_node_guess_data_model"
	ResponseDbNodeGuessDataModel    MessageType = "response_db_node_guess_data_model"
)

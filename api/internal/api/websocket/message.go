package websocket

import (
	errors2 "errors"
	"time"
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
	Error         error  `json:"error"`
	CustomMessage string `json:"customMessage"`
}

// NewErrorMessage creates a new error message
func NewErrorMessage(jobID uint, userID uint, username string, errorText string, errors ...error) Message {
	return Message{
		Type:      MessageTypeError,
		JobID:     jobID,
		UserID:    userID,
		Username:  username,
		Timestamp: time.Now(),
		Data: ErrorMessage{
			Error:         errors2.Join(errors...),
			CustomMessage: errorText,
		},
	}
}

// NewUserJoinMessage creates a new user join message
func NewUserJoinMessage(jobID uint, userID uint, username string, userInfo UserInfo) Message {
	return Message{
		Type:      MessageTypeUserJoin,
		JobID:     jobID,
		UserID:    userID,
		Username:  username,
		Timestamp: time.Now(),
		Data:      userInfo,
	}
}

// NewUserLeaveMessage creates a new user leave message
func NewUserLeaveMessage(jobID uint, userID uint, username string, userInfo UserInfo) Message {
	return Message{
		Type:      MessageTypeUserLeave,
		JobID:     jobID,
		UserID:    userID,
		Username:  username,
		Timestamp: time.Now(),
		Data:      userInfo,
	}
}

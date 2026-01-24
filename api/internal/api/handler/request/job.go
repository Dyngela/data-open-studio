package request

import "api/internal/api/models"

type CreateJob struct {
	Name        string               `json:"name" validate:"required"`
	Description string               `json:"description"`
	FilePath    string               `json:"filePath"`
	OutputPath  string               `json:"outputPath"`
	Active      bool                 `json:"active"`
	Visibility  models.JobVisibility `json:"visibility"`           // public or private (default: private)
	SharedWith  []uint               `json:"sharedWith,omitempty"` // User IDs to share with
}
type UpdateJob struct {
	Name        *string               `json:"name,omitempty"`
	Description *string               `json:"description,omitempty"`
	FilePath    *string               `json:"filePath,omitempty"`
	OutputPath  *string               `json:"outputPath,omitempty"`
	Active      *bool                 `json:"active,omitempty"`
	Visibility  *models.JobVisibility `json:"visibility,omitempty"`
	SharedWith  []uint                `json:"sharedWith,omitempty"` // User IDs to share with (replaces existing)
	Nodes       []models.Node         `json:"nodes,omitempty"`
}

// ShareJob is for sharing/unsharing a job with users
type ShareJob struct {
	UserIDs []uint `json:"userIds" validate:"required"`
	Role    string `json:"role"` // viewer or editor (default: viewer)
}

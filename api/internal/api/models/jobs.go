package models

import "time"

type JobVisibility string

const (
	JobVisibilityPublic  JobVisibility = "public"
	JobVisibilityPrivate JobVisibility = "private"
)

type OwningJob string

const (
	Owner  OwningJob = "owner"
	Editor OwningJob = "editor"
	Viewer OwningJob = "viewer"
)

type Job struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	FilePath    string        `json:"filePath"` // Virtual folder path (e.g., "/projects/etl/")
	CreatorID   uint          `json:"creatorId"`
	Active      bool          `json:"active"`
	Visibility  JobVisibility `gorm:"default:private"`
	OutputPath  string        `json:"outputPath"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	Nodes       []Node        `gorm:"foreignKey:JobID" json:"nodes,omitempty"`

	// Users who have access to this job (for private jobs)
	SharedWith []User `gorm:"many2many:job_user_access;" json:"sharedWith,omitempty"`
}

// JobUserAccess is the junction table for job-user sharing
type JobUserAccess struct {
	JobID     uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"primaryKey"`
	Role      OwningJob `gorm:"default:viewer"` // viewer, editor
	CreatedAt time.Time `json:"createdAt"`
}

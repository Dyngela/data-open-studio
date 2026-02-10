package response

import (
	"api/internal/api/models"
	"time"
)

// SharedUser represents a user who has access to a job
type SharedUser struct {
	ID     uint             `json:"id"`
	Email  string           `json:"email"`
	Prenom string           `json:"prenom"`
	Nom    string           `json:"nom"`
	Role   models.OwningJob `json:"role"` // viewer or editor
}

// Job response without nodes (for listing)
type Job struct {
	ID          uint                 `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	FilePath    string               `json:"filePath"`
	CreatorID   uint                 `json:"creatorId"`
	Active      bool                 `json:"active"`
	Visibility  models.JobVisibility `json:"visibility"`
	OutputPath  string               `json:"outputPath"`
	CreatedAt   time.Time            `json:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt"`
}

// NotificationContact represents a user to notify on job failure
type NotificationContact struct {
	ID     uint   `json:"id"`
	Email  string `json:"email"`
	Prenom string `json:"prenom"`
	Nom    string `json:"nom"`
}

// JobWithNodes response with nodes included (for single job get)
type JobWithNodes struct {
	ID                   uint                 `json:"id"`
	Name                 string               `json:"name"`
	Description          string               `json:"description"`
	FilePath             string               `json:"filePath"`
	CreatorID            uint                 `json:"creatorId"`
	Active               bool                 `json:"active"`
	Visibility           models.JobVisibility `json:"visibility"`
	OutputPath           string               `json:"outputPath"`
	CreatedAt            time.Time            `json:"createdAt"`
	UpdatedAt            time.Time            `json:"updatedAt"`
	Nodes                []Node               `json:"nodes"`
	Connexions           []Connexion          `json:"connexions"`
	SharedUser           []SharedUser         `json:"sharedUser"`
	NotificationContacts []NotificationContact `json:"notificationContacts"`
}

type Connexion struct {
	SourceNodeId   int             `json:"sourceNodeId"`
	SourcePort     int             `json:"sourcePort"`
	SourcePortType models.PortType `json:"sourcePortType"`
	TargetNodeId   int             `json:"targetNodeId"`
	TargetPort     int             `json:"targetPort"`
	TargetPortType models.PortType `json:"targetPortType"`
}

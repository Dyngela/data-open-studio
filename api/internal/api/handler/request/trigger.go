package request

import "api/internal/api/models"

// CreateTrigger is the request for creating a new trigger
type CreateTrigger struct {
	Name            string                 `json:"name" validate:"required"`
	Description     string                 `json:"description"`
	Type            models.TriggerType     `json:"type" validate:"required,oneof=database email webhook"`
	PollingInterval int                    `json:"pollingInterval"` // seconds, default 60
	Config          models.TriggerConfig   `json:"config" validate:"required"`
}

// UpdateTrigger is the request for updating a trigger
type UpdateTrigger struct {
	Name            *string              `json:"name,omitempty"`
	Description     *string              `json:"description,omitempty"`
	PollingInterval *int                 `json:"pollingInterval,omitempty"`
	Config          *models.TriggerConfig `json:"config,omitempty"`
}

// CreateTriggerRule is the request for creating a trigger rule
type CreateTriggerRule struct {
	Name       string                  `json:"name"`
	Conditions models.RuleConditions   `json:"conditions" validate:"required"`
}

// UpdateTriggerRule is the request for updating a trigger rule
type UpdateTriggerRule struct {
	Name       *string                  `json:"name,omitempty"`
	Conditions *models.RuleConditions   `json:"conditions,omitempty"`
}

// LinkJob is the request for linking a job to a trigger
type LinkJob struct {
	JobID         uint `json:"jobId" validate:"required"`
	Priority      int  `json:"priority"`
	PassEventData bool `json:"passEventData"`
}

// UpdateJobLink is the request for updating a trigger-job link
type UpdateJobLink struct {
	Priority      *int  `json:"priority,omitempty"`
	Active        *bool `json:"active,omitempty"`
	PassEventData *bool `json:"passEventData,omitempty"`
}

// TestDatabaseConnection is the request for testing a database connection
type TestDatabaseConnection struct {
	Connection models.DBConnectionConfig `json:"connection" validate:"required"`
}

// IntrospectDatabase is the request for introspecting a database schema
type IntrospectDatabase struct {
	MetadataDatabaseID *uint                      `json:"metadataDatabaseId,omitempty"`
	Connection         *models.DBConnectionConfig `json:"connection,omitempty"`
}

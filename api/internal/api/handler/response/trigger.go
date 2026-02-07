package response

import (
	"api/internal/api/models"
	"time"
)

// Trigger is the response for a trigger (list view)
type Trigger struct {
	ID              uint                  `json:"id"`
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	Type            models.TriggerType    `json:"type"`
	Status          models.TriggerStatus  `json:"status"`
	CreatorID       uint                  `json:"creatorId"`
	PollingInterval int                   `json:"pollingInterval"`
	LastPolledAt    *time.Time            `json:"lastPolledAt,omitempty"`
	LastError       string                `json:"lastError,omitempty"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
	JobCount        int                   `json:"jobCount"`
}

// TriggerWithDetails is the response for a single trigger with full details
type TriggerWithDetails struct {
	ID              uint                  `json:"id"`
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	Type            models.TriggerType    `json:"type"`
	Status          models.TriggerStatus  `json:"status"`
	CreatorID       uint                  `json:"creatorId"`
	PollingInterval int                   `json:"pollingInterval"`
	LastPolledAt    *time.Time            `json:"lastPolledAt,omitempty"`
	LastError       string                `json:"lastError,omitempty"`
	Config          models.TriggerConfig  `json:"config"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
	Rules           []TriggerRule         `json:"rules"`
	Jobs            []TriggerJobLink      `json:"jobs"`
}

// TriggerRule is the response for a trigger rule
type TriggerRule struct {
	ID         uint                    `json:"id"`
	TriggerID  uint                    `json:"triggerId"`
	Name       string                  `json:"name"`
	Conditions models.RuleConditions   `json:"conditions"`
	CreatedAt  time.Time               `json:"createdAt"`
	UpdatedAt  time.Time               `json:"updatedAt"`
}

// TriggerJobLink is the response for a trigger-job link
type TriggerJobLink struct {
	ID            uint   `json:"id"`
	TriggerID     uint   `json:"triggerId"`
	JobID         uint   `json:"jobId"`
	JobName       string `json:"jobName"`
	Priority      int    `json:"priority"`
	Active        bool   `json:"active"`
	PassEventData bool   `json:"passEventData"`
}

// TriggerExecution is the response for a trigger execution record
type TriggerExecution struct {
	ID            uint                    `json:"id"`
	TriggerID     uint                    `json:"triggerId"`
	StartedAt     time.Time               `json:"startedAt"`
	FinishedAt    time.Time               `json:"finishedAt,omitempty"`
	Status        models.ExecutionStatus  `json:"status"`
	EventCount    int                     `json:"eventCount"`
	JobsTriggered int                     `json:"jobsTriggered"`
	Error         string                  `json:"error,omitempty"`
	EventSample   *string                 `json:"eventSample,omitempty"`
}

// DatabaseTable is the response for a table in schema introspection
type DatabaseTable struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

// DatabaseColumn is the response for a column in schema introspection
type DatabaseColumn struct {
	Name       string `json:"name"`
	DataType   string `json:"dataType"`
	IsNullable bool   `json:"isNullable"`
	IsPrimary  bool   `json:"isPrimary"`
}

// DatabaseIntrospection is the response for database schema introspection
type DatabaseIntrospection struct {
	Tables  []DatabaseTable  `json:"tables,omitempty"`
	Columns []DatabaseColumn `json:"columns,omitempty"`
}

// TestConnectionResult is the response for testing a database connection
type TestConnectionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Version string `json:"version,omitempty"`
}

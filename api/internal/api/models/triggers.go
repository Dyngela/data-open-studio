package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// TriggerType represents the type of trigger source
type TriggerType string

const (
	TriggerTypeDatabase TriggerType = "database"
	TriggerTypeEmail    TriggerType = "email"
	TriggerTypeWebhook  TriggerType = "webhook"
	TriggerTypeCron     TriggerType = "cron"
)

// TriggerStatus represents the status of a trigger
type TriggerStatus string

const (
	TriggerStatusActive   TriggerStatus = "active"
	TriggerStatusPaused   TriggerStatus = "paused"
	TriggerStatusError    TriggerStatus = "error"
	TriggerStatusDisabled TriggerStatus = "disabled"
)

// Trigger is the main trigger entity that watches for events
type Trigger struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"not null" json:"name"`
	Description string         `json:"description"`
	Type        TriggerType    `gorm:"not null;type:varchar(20)" json:"type"`
	Status      TriggerStatus  `gorm:"default:paused;type:varchar(20)" json:"status"`
	CreatorID   uint           `gorm:"not null" json:"creatorId"`
	Creator     User           `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Polling interval in seconds (default 60)
	PollingInterval int `gorm:"default:60" json:"pollingInterval"`

	// Last time this trigger was polled
	LastPolledAt *time.Time `json:"lastPolledAt,omitempty"`

	// Last error message if status is error
	LastError string `json:"lastError,omitempty"`

	// Type-specific configuration stored as JSON
	Config TriggerConfig `gorm:"type:jsonb" json:"config"`

	// Rules to filter events before triggering jobs
	Rules []TriggerRule `gorm:"foreignKey:TriggerID" json:"rules,omitempty"`

	// Jobs to execute when this trigger fires
	Jobs []TriggerJob `gorm:"foreignKey:TriggerID" json:"jobs,omitempty"`
}

// TriggerConfig holds type-specific configuration
type TriggerConfig struct {
	// Database trigger config
	Database *DatabaseTriggerConfig `json:"database,omitempty"`
	// Email trigger config
	Email *EmailTriggerConfig `json:"email,omitempty"`
	// Webhook trigger config
	Webhook *WebhookTriggerConfig `json:"webhook,omitempty"`
	// Cron trigger config
	Cron *CronTriggerConfig `json:"cron,omitempty"`
}

// Value implements driver.Valuer for GORM
func (tc TriggerConfig) Value() (driver.Value, error) {
	return json.Marshal(tc)
}

// Scan implements sql.Scanner for GORM
func (tc *TriggerConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan TriggerConfig: expected []byte")
	}
	return json.Unmarshal(bytes, tc)
}

// DatabaseTriggerConfig holds configuration for database polling triggers
type DatabaseTriggerConfig struct {
	// Reference to MetadataDatabase or inline connection config
	MetadataDatabaseID *uint `json:"metadataDatabaseId,omitempty"`

	// Inline connection config (if MetadataDatabaseID is not set)
	Connection *DBConnectionConfig `json:"connection,omitempty"`

	// Table to monitor
	TableName string `json:"tableName"`

	// Column to use as watermark (must be monotonically increasing)
	WatermarkColumn string `json:"watermarkColumn"`

	// Type of watermark column
	WatermarkType WatermarkType `json:"watermarkType"`

	// Last watermark value (stored as string, parsed based on type)
	LastWatermark string `json:"lastWatermark,omitempty"`

	// Optional: only select specific columns (empty = all columns)
	SelectColumns []string `json:"selectColumns,omitempty"`

	// Optional: additional WHERE clause conditions
	WhereClause string `json:"whereClause,omitempty"`

	// Batch size for polling (default 100)
	BatchSize int `json:"batchSize,omitempty"`
}

// WatermarkType represents the data type of the watermark column
type WatermarkType string

const (
	WatermarkTypeInt       WatermarkType = "int"
	WatermarkTypeTimestamp WatermarkType = "timestamp"
	WatermarkTypeUUID      WatermarkType = "uuid"
)

// EmailTriggerConfig holds configuration for email polling triggers
type EmailTriggerConfig struct {
	// Reference to MetadataEmail (alternative to inline credentials)
	MetadataEmailID *uint `json:"metadataEmailId,omitempty"`

	// IMAP server settings (inline, used if MetadataEmailID is nil)
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	UseTLS   bool   `json:"useTls"`

	// Folder to monitor (default: INBOX)
	Folder string `json:"folder,omitempty"`

	// Filter criteria
	FromAddress    string   `json:"fromAddress,omitempty"`
	ToAddress      string   `json:"toAddress,omitempty"`
	SubjectPattern string   `json:"subjectPattern,omitempty"` // regex pattern
	HasAttachment  *bool    `json:"hasAttachment,omitempty"`
	CCAddresses    []string `json:"ccAddresses,omitempty"`

	// Last UID processed
	LastUID uint32 `json:"lastUid,omitempty"`

	// Whether to mark emails as read after processing
	MarkAsRead bool `json:"markAsRead,omitempty"`
}

// WebhookTriggerConfig holds configuration for webhook triggers
type WebhookTriggerConfig struct {
	// Secret for validating webhook signatures
	Secret string `json:"secret,omitempty"`

	// Expected headers (for validation)
	RequiredHeaders map[string]string `json:"requiredHeaders,omitempty"`
}

// CronMode represents the scheduling mode for a cron trigger
type CronMode string

const (
	CronModeInterval CronMode = "interval"
	CronModeSchedule CronMode = "schedule"
)

// IntervalUnit represents the time unit for interval-based cron triggers
type IntervalUnit string

const (
	IntervalUnitMinutes IntervalUnit = "minutes"
	IntervalUnitHours   IntervalUnit = "hours"
	IntervalUnitDays    IntervalUnit = "days"
)

// ScheduleFrequency represents how often a scheduled cron trigger fires
type ScheduleFrequency string

const (
	ScheduleFrequencyDaily   ScheduleFrequency = "daily"
	ScheduleFrequencyWeekly  ScheduleFrequency = "weekly"
	ScheduleFrequencyMonthly ScheduleFrequency = "monthly"
)

// CronTriggerConfig holds configuration for cron-based triggers
type CronTriggerConfig struct {
	// Mode: "interval" (every X minutes/hours/days) or "schedule" (at specific time)
	Mode CronMode `json:"mode"`

	// Interval mode fields
	IntervalValue int          `json:"intervalValue,omitempty"` // e.g., 30
	IntervalUnit  IntervalUnit `json:"intervalUnit,omitempty"`  // "minutes", "hours", "days"

	// Schedule mode fields
	ScheduleFrequency ScheduleFrequency `json:"scheduleFrequency,omitempty"` // "daily", "weekly", "monthly"
	ScheduleTime      string            `json:"scheduleTime,omitempty"`      // "HH:MM" format
	ScheduleDayOfWeek *int              `json:"scheduleDayOfWeek,omitempty"` // 0=Sunday..6=Saturday (for weekly)
	ScheduleDayOfMonth *int             `json:"scheduleDayOfMonth,omitempty"` // 1-31 (for monthly)
}

// TriggerRule defines conditions that must be met for a trigger to fire
type TriggerRule struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	TriggerID uint           `gorm:"not null" json:"triggerId"`
	Name      string         `json:"name"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Rule conditions stored as JSON
	Conditions RuleConditions `gorm:"type:jsonb" json:"conditions"`
}

// RuleConditions defines the conditions for a rule
type RuleConditions struct {
	// All conditions must match (AND logic)
	All []RuleCondition `json:"all,omitempty"`
	// Any condition must match (OR logic)
	Any []RuleCondition `json:"any,omitempty"`
}

// Value implements driver.Valuer for GORM
func (rc RuleConditions) Value() (driver.Value, error) {
	return json.Marshal(rc)
}

// Scan implements sql.Scanner for GORM
func (rc *RuleConditions) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan RuleConditions: expected []byte")
	}
	return json.Unmarshal(bytes, rc)
}

// RuleCondition defines a single condition
type RuleCondition struct {
	// Field path in the event payload (e.g., "payload.status", "email.subject")
	Field string `json:"field"`

	// Operator for comparison
	Operator ConditionOperator `json:"operator"`

	// Value to compare against
	Value interface{} `json:"value"`
}

// ConditionOperator defines comparison operators
type ConditionOperator string

const (
	OperatorEquals      ConditionOperator = "eq"
	OperatorNotEquals   ConditionOperator = "neq"
	OperatorContains    ConditionOperator = "contains"
	OperatorStartsWith  ConditionOperator = "startsWith"
	OperatorEndsWith    ConditionOperator = "endsWith"
	OperatorGreaterThan ConditionOperator = "gt"
	OperatorLessThan    ConditionOperator = "lt"
	OperatorRegex       ConditionOperator = "regex"
	OperatorIn          ConditionOperator = "in"
	OperatorNotIn       ConditionOperator = "notIn"
	OperatorExists      ConditionOperator = "exists"
	OperatorNotExists   ConditionOperator = "notExists"
)

// TriggerJob links a trigger to a job that should be executed
type TriggerJob struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	TriggerID uint           `gorm:"not null" json:"triggerId"`
	JobID     uint           `gorm:"not null" json:"jobId"`
	Job       Job            `gorm:"foreignKey:JobID" json:"job,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Priority for execution order (lower = higher priority)
	Priority int `gorm:"default:0" json:"priority"`

	// Whether this job link is active
	Active bool `gorm:"default:true" json:"active"`

	// Optional: pass event data as job input parameters
	PassEventData bool `json:"passEventData"`
}

// TriggerExecution records each time a trigger fires
type TriggerExecution struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TriggerID  uint      `gorm:"not null;index" json:"triggerId"`
	StartedAt  time.Time `gorm:"not null" json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt,omitempty"`

	// Status of the execution
	Status ExecutionStatus `gorm:"type:varchar(20)" json:"status"`

	// Number of events detected in this execution
	EventCount int `json:"eventCount"`

	// Number of jobs triggered
	JobsTriggered int `json:"jobsTriggered"`

	// Error message if failed
	Error string `json:"error,omitempty"`

	// Sample of event data (first event)
	EventSample *string `gorm:"type:jsonb" json:"eventSample,omitempty"`
}

// ExecutionStatus represents the status of a trigger execution
type ExecutionStatus string

const (
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusNoEvents  ExecutionStatus = "no_events"
)

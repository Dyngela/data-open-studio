package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// DatasetStatus represents the current state of a dataset
type DatasetStatus string

const (
	DatasetStatusDraft DatasetStatus = "draft"
	DatasetStatusReady DatasetStatus = "ready"
	DatasetStatusError DatasetStatus = "error"
)

// Dataset defines a reusable data source backed by a SQL query against a MetadataDatabase
type Dataset struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	Name               string         `gorm:"not null" json:"name"`
	Description        string         `json:"description"`
	CreatorID          uint           `gorm:"not null" json:"creatorId"`
	Creator            User           `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	MetadataDatabaseID uint           `gorm:"not null" json:"metadataDatabaseId"`
	Query              string         `gorm:"type:text;not null" json:"query"`
	Schema             DatasetSchema  `gorm:"type:jsonb" json:"schema"`
	Status             DatasetStatus  `gorm:"default:draft;type:varchar(20)" json:"status"`
	LastRefreshedAt    *time.Time     `json:"lastRefreshedAt,omitempty"`
	LastError          string         `json:"lastError,omitempty"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

// DatasetSchema holds the auto-detected column schema
type DatasetSchema struct {
	Columns []DatasetColumn `json:"columns"`
}

// DatasetColumn describes a single column in the dataset
type DatasetColumn struct {
	Name     string `json:"name"`
	DataType string `json:"dataType"` // "string", "integer", "float", "date", "datetime", "boolean"
	Nullable bool   `json:"nullable"`
}

func (ds DatasetSchema) Value() (driver.Value, error) {
	return json.Marshal(ds)
}

func (ds *DatasetSchema) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan DatasetSchema: expected []byte")
	}
	return json.Unmarshal(bytes, ds)
}

// QueryFilter defines a column-level filter applied when querying a dataset
type QueryFilter struct {
	Column   string      `json:"column"`
	Operator string      `json:"operator"` // "eq", "neq", "gt", "lt", "gte", "lte", "like"
	Value    interface{} `json:"value"`
}

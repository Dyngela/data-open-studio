package response

import (
	"api/internal/api/models"
	"time"
)

// DatasetSummary is the response for a dataset in list views
type DatasetSummary struct {
	ID                 uint                  `json:"id"`
	Name               string                `json:"name"`
	Description        string                `json:"description"`
	CreatorID          uint                  `json:"creatorId"`
	MetadataDatabaseID uint                  `json:"metadataDatabaseId"`
	Status             models.DatasetStatus  `json:"status"`
	ColumnCount        int                   `json:"columnCount"`
	LastRefreshedAt    *time.Time            `json:"lastRefreshedAt,omitempty"`
	LastError          string                `json:"lastError,omitempty"`
	CreatedAt          time.Time             `json:"createdAt"`
	UpdatedAt          time.Time             `json:"updatedAt"`
}

// DatasetWithDetails is the full response for a single dataset
type DatasetWithDetails struct {
	ID                 uint                  `json:"id"`
	Name               string                `json:"name"`
	Description        string                `json:"description"`
	CreatorID          uint                  `json:"creatorId"`
	MetadataDatabaseID uint                  `json:"metadataDatabaseId"`
	Query              string                `json:"query"`
	Schema             models.DatasetSchema  `json:"schema"`
	Status             models.DatasetStatus  `json:"status"`
	LastRefreshedAt    *time.Time            `json:"lastRefreshedAt,omitempty"`
	LastError          string                `json:"lastError,omitempty"`
	CreatedAt          time.Time             `json:"createdAt"`
	UpdatedAt          time.Time             `json:"updatedAt"`
}

// DatasetPreviewResult holds sample rows from the dataset
type DatasetPreviewResult struct {
	Columns  []string                 `json:"columns"`
	Rows     []map[string]interface{} `json:"rows"`
	RowCount int                      `json:"rowCount"`
}

// DatasetQueryResult holds the full query result with optional filters applied
type DatasetQueryResult struct {
	Columns  []string                 `json:"columns"`
	Rows     []map[string]interface{} `json:"rows"`
	RowCount int                      `json:"rowCount"`
}

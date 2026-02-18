package request

// CreateDataset is the request for creating a new dataset
type CreateDataset struct {
	Name               string `json:"name" validate:"required"`
	Description        string `json:"description"`
	MetadataDatabaseID uint   `json:"metadataDatabaseId" validate:"required"`
	Query              string `json:"query" validate:"required"`
}

// UpdateDataset is the request for updating an existing dataset
type UpdateDataset struct {
	Name               *string `json:"name,omitempty"`
	Description        *string `json:"description,omitempty"`
	MetadataDatabaseID *uint   `json:"metadataDatabaseId,omitempty"`
	Query              *string `json:"query,omitempty"`
}

// DatasetPreview is the request for previewing dataset rows
type DatasetPreview struct {
	Limit int `json:"limit"` // defaults to 100, max 1000
}

// DatasetQueryFilter defines a column-level filter
type DatasetQueryFilter struct {
	Column   string      `json:"column" validate:"required"`
	Operator string      `json:"operator" validate:"required,oneof=eq neq gt lt gte lte like"`
	Value    interface{} `json:"value"`
}

// DatasetQuery is the request for querying a dataset with optional filters
type DatasetQuery struct {
	Filters []DatasetQueryFilter `json:"filters"`
	Limit   int                  `json:"limit"` // defaults to 1000, max 10000
}

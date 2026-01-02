package response

import "api/internal/api/models"

// NodeResponseDTO is a generic response for any node type
type NodeResponseDTO struct {
	ID         int             `json:"id"`
	Type       models.NodeType `json:"type"`
	Name       string          `json:"name"`
	Xpos       float32         `json:"xpos"`
	Ypos       float32         `json:"ypos"`
	InputPort  []models.Port   `json:"inputPort"`
	OutputPort []models.Port   `json:"outputPort"`

	// Type-specific fields (populated based on type)
	Query     *string `json:"query,omitempty"`
	Schema    *string `json:"schema,omitempty"`
	Table     *string `json:"table,omitempty"`
	Mode      *string `json:"mode,omitempty"`
	BatchSize *int    `json:"batchSize,omitempty"`
}

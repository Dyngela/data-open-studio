package response

import "api/internal/api/models"

// GuessSchemaResponse is returned from the guess-schema endpoint
type GuessSchemaResponse struct {
	NodeID     string             `json:"nodeId"`
	DataModels []models.DataModel `json:"dataModels"`
}

type Node struct {
	ID   int             `json:"id"`
	Type models.NodeType `json:"type"`
	Name string          `json:"name"`
	Xpos float32         `json:"xpos"`
	Ypos float32         `json:"ypos"`
	Data models.NodeData `json:"data"`
}

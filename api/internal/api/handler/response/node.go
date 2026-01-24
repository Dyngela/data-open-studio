package response

import "api/internal/api/models"

type Connection struct {
	EntryPortID  uint
	TargetPortID uint
}

// GuessSchemaResponse is returned from the guess-schema endpoint
type GuessSchemaResponse struct {
	NodeID     string             `json:"nodeId"`
	DataModels []models.DataModel `json:"dataModels"`
}

type Port struct {
	ID     uint   `json:"id"`
	Type   string `json:"type"`
	NodeID uint   `json:"nodeId"`
}

type Node struct {
	ID         int             `json:"id"`
	Type       models.NodeType `json:"type"`
	Name       string          `json:"name"`
	Xpos       float32         `json:"xpos"`
	Ypos       float32         `json:"ypos"`
	InputPort  []Port          `json:"inputPort"`
	OutputPort []Port          `json:"outputPort"`
	Data       models.NodeData `json:"data"`
	JobID      uint            `json:"jobId"`
}

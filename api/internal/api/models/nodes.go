package models

type NodeType string

const (
	NodeTypeDBInput  NodeType = "db_input"
	NodeTypeDBOutput NodeType = "db_output"
	NodeTypeMap      NodeType = "map"
)

type NodeConfig interface {
	NodeType() NodeType
	Validate() error
}

type BaseNode struct {
	ID         int      `json:"id"`
	Type       NodeType `json:"type"`
	Name       string   `json:"name"`
	Xpos       float32  `json:"xpos"`
	Ypos       float32  `json:"ypos"`
	InputPort  []Port   `json:"inputPort"`
	OutputPort []Port   `json:"outputPort"`
}

type DBInputConfig struct {
	BaseNode
	Query  string `json:"query"`
	Schema string `json:"schema"`
	Table  string `json:"table"`
}

func (c DBInputConfig) NodeType() NodeType { return NodeTypeDBInput }
func (c DBInputConfig) Validate() error    { return nil }

type DBOutputConfig struct {
	BaseNode
	Table     string `json:"table"`
	Mode      string `json:"mode"`
	BatchSize int    `json:"batchSize"`
}

func (c DBOutputConfig) NodeType() NodeType { return NodeTypeDBOutput }
func (c DBOutputConfig) Validate() error    { return nil }

type MapConfig struct {
	BaseNode
}

func (c MapConfig) NodeType() NodeType { return NodeTypeMap }
func (c MapConfig) Validate() error    { return nil }

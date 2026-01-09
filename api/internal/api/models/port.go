package models

type PortType string

const (
	PortTypeInput      PortType = "input"
	PortTypeOutput     PortType = "output"
	PortNodeFlowInput  PortType = "node_flow_input"
	PortNodeFlowOutput PortType = "node_flow_output"
)

type Port struct {
	ID     uint
	Type   PortType
	Node   Node `gorm:"foreignKey:NodeID"`
	NodeID uint
}

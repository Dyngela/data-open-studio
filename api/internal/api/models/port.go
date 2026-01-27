package models

type PortType string

const (
	PortTypeInput      PortType = "input"
	PortTypeOutput     PortType = "output"
	PortNodeFlowInput  PortType = "node_flow_input"
	PortNodeFlowOutput PortType = "node_flow_output"
)

type Port struct {
	ID              uint
	Type            PortType
	NodeID          uint // owning node (GORM foreign key for has-many)
	ConnectedNodeID uint // the node on the other end of the connection
	Node            Node
}

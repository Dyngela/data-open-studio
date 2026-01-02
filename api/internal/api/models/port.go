package models

type Connection struct {
	EntryPortID  uint `gorm:"primaryKey"`
	TargetPortID uint `gorm:"primaryKey"`
	JobID        uint `gorm:"primaryKey"`
}

type PortType string

const (
	PortTypeInput      PortType = "input"
	PortTypeOutput     PortType = "output"
	PortNodeFlowInput  PortType = "node_flow_input"
	PortNodeFlowOutput PortType = "node_flow_output"
)

type Port struct {
	ID   uint
	Type PortType
}

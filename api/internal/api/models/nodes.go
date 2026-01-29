package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type NodeData []byte

// Scan implements sql.Scanner interface
func (n *NodeData) Scan(value interface{}) error {
	if value == nil {
		*n = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*n = v
		return nil
	case string:
		*n = []byte(v)
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into NodeData", value)
	}
}

// Value implements driver.Valuer interface
func (n NodeData) Value() (driver.Value, error) {
	if n == nil {
		return nil, nil
	}
	return []byte(n), nil
}

// MarshalJSON implements json.Marshaler - returns raw JSON
func (n NodeData) MarshalJSON() ([]byte, error) {
	if n == nil {
		return []byte("null"), nil
	}
	return n, nil
}

// UnmarshalJSON implements json.Unmarshaler - stores raw JSON
func (n *NodeData) UnmarshalJSON(data []byte) error {
	if data == nil {
		*n = nil
		return nil
	}
	*n = data
	return nil
}

type NodeType string

const (
	NodeTypeStart    NodeType = "start"
	NodeTypeDBInput  NodeType = "db_input"
	NodeTypeDBOutput NodeType = "db_output"
	NodeTypeMap      NodeType = "map"
	NodeTypeLog      NodeType = "log"
)

type Node struct {
	ID int `json:"id"`
	// Type of the node. It has to be immutable
	Type NodeType
	Name string
	Xpos float32
	Ypos float32
	// Connections to other nodes.
	InputPort  []Port   `gorm:"foreignKey:NodeID"`
	OutputPort []Port   `gorm:"foreignKey:NodeID"`
	Data       NodeData `json:"data" gorm:"type:jsonb"`

	JobID uint `gorm:"index" json:"jobId"`
	Job   Job
}

// SetData serializes and stores typed config data
func (slf *Node) SetData(data any) error {
	// Validate data type matches node type
	switch slf.Type {
	case NodeTypeDBInput:
		if _, ok := data.(DBInputConfig); !ok {
			return errors.New("invalid data type for db_input node")
		}
	case NodeTypeDBOutput:
		if _, ok := data.(DBOutputConfig); !ok {
			return errors.New("invalid data type for db_output node")
		}
	case NodeTypeMap:
		if _, ok := data.(MapConfig); !ok {
			return errors.New("invalid data type for map node")
		}
	case NodeTypeLog:
		return errors.New("Log has no data, it just logs messages")
	default:
		return errors.New("unknown node type: " + string(slf.Type))
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	slf.Data = jsonData
	return nil
}

// GetTypedData deserializes the JSON data into the expected type
func GetTypedData[T any](node Node) (T, error) {
	var result T
	if node.Data == nil {
		return result, errors.New("node data is nil")
	}
	if err := json.Unmarshal(node.Data, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal data: %w", err)
	}
	return result, nil
}

func (slf Node) GetDBInputConfig() (DBInputConfig, error) {
	if slf.Type != NodeTypeDBInput {
		return DBInputConfig{}, errors.New("node is not a db_input type")
	}
	return GetTypedData[DBInputConfig](slf)
}

func (slf Node) GetDBOutputConfig() (DBOutputConfig, error) {
	if slf.Type != NodeTypeDBOutput {
		return DBOutputConfig{}, errors.New("node is not a db_output type")
	}
	return GetTypedData[DBOutputConfig](slf)
}

func (slf Node) GetMapConfig() (MapConfig, error) {
	if slf.Type != NodeTypeMap {
		return MapConfig{}, errors.New("node is not a map type")
	}
	return GetTypedData[MapConfig](slf)
}

func (slf Node) GetLogConfig() (NodeLogConfig, error) {
	if slf.Type != NodeTypeLog {
		return NodeLogConfig{}, errors.New("node is not a log type")
	}
	return GetTypedData[NodeLogConfig](slf)
}

func (slf Node) GetNextFlowNodeIDs() []int {
	if len(slf.OutputPort) == 0 {
		return nil
	}
	var ids []int
	for _, conn := range slf.OutputPort {
		if conn.Type == PortNodeFlowOutput {
			ids = append(ids, int(conn.ConnectedNodeID))
		}
	}
	return ids
}

func (slf Node) GetPrevFlowNodeIDs() []int {
	if len(slf.InputPort) == 0 {
		return nil
	}
	var ids []int
	for _, conn := range slf.InputPort {
		if conn.Type == PortNodeFlowInput {
			ids = append(ids, int(conn.ConnectedNodeID))
		}
	}
	return ids
}

// GetDataInputNodeIDs returns IDs of nodes that provide data input to this node
func (slf Node) GetDataInputNodeIDs() []int {
	if len(slf.InputPort) == 0 {
		return nil
	}
	var ids []int
	for _, conn := range slf.InputPort {
		if conn.Type == PortTypeInput {
			ids = append(ids, int(conn.ConnectedNodeID))
		}
	}
	return ids
}

// GetDataOutputNodeIDs returns IDs of nodes that receive data output from this node
func (slf Node) GetDataOutputNodeIDs() []int {
	if len(slf.OutputPort) == 0 {
		return nil
	}
	var ids []int
	for _, conn := range slf.OutputPort {
		if conn.Type == PortTypeOutput {
			ids = append(ids, int(conn.ConnectedNodeID))
		}
	}
	return ids
}

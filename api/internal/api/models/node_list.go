package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type NodeList []NodeConfig

func (n *NodeList) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan NodeList: %v", value)
	}
	return n.UnmarshalJSON(bytes)
}

func (n NodeList) Value() (driver.Value, error) {
	return json.Marshal(n)
}

func (n *NodeList) UnmarshalJSON(data []byte) error {
	var raws []json.RawMessage
	if err := json.Unmarshal(data, &raws); err != nil {
		return err
	}

	*n = make([]NodeConfig, 0, len(raws))

	for _, raw := range raws {
		var base struct {
			Type NodeType `json:"type"`
		}
		if err := json.Unmarshal(raw, &base); err != nil {
			return err
		}

		var node NodeConfig
		switch base.Type {
		case NodeTypeDBInput:
			node = &DBInputConfig{}
		case NodeTypeDBOutput:
			node = &DBOutputConfig{}
		case NodeTypeMap:
			node = &MapConfig{}
		default:
			return fmt.Errorf("unknown node type: %s", base.Type)
		}

		if err := json.Unmarshal(raw, node); err != nil {
			return err
		}
		*n = append(*n, node)
	}

	return nil
}

func (n NodeList) MarshalJSON() ([]byte, error) {
	return json.Marshal([]NodeConfig(n))
}

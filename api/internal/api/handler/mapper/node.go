package mapper

import (
	"api/internal/api/handler/response"
	"api/internal/api/models"
)

type NodeMapper struct{}

// NodeConfigToResponse converts a NodeConfig to NodeResponseDTO
func (m *NodeMapper) NodeConfigToResponse(node models.NodeConfig) response.NodeResponseDTO {
	var baseNode models.BaseNode

	// Extract base fields based on concrete type
	switch n := node.(type) {
	case *models.DBInputConfig:
		baseNode = n.BaseNode
		return response.NodeResponseDTO{
			ID:         baseNode.ID,
			Type:       baseNode.Type,
			Name:       baseNode.Name,
			Xpos:       baseNode.Xpos,
			Ypos:       baseNode.Ypos,
			InputPort:  baseNode.InputPort,
			OutputPort: baseNode.OutputPort,
			Query:      &n.Query,
			Schema:     &n.Schema,
			Table:      &n.Table,
		}
	case *models.DBOutputConfig:
		baseNode = n.BaseNode
		return response.NodeResponseDTO{
			ID:         baseNode.ID,
			Type:       baseNode.Type,
			Name:       baseNode.Name,
			Xpos:       baseNode.Xpos,
			Ypos:       baseNode.Ypos,
			InputPort:  baseNode.InputPort,
			OutputPort: baseNode.OutputPort,
			Table:      &n.Table,
			Mode:       &n.Mode,
			BatchSize:  &n.BatchSize,
		}
	case *models.MapConfig:
		baseNode = n.BaseNode
		return response.NodeResponseDTO{
			ID:         baseNode.ID,
			Type:       baseNode.Type,
			Name:       baseNode.Name,
			Xpos:       baseNode.Xpos,
			Ypos:       baseNode.Ypos,
			InputPort:  baseNode.InputPort,
			OutputPort: baseNode.OutputPort,
		}
	default:
		// Fallback for unknown types
		return response.NodeResponseDTO{}
	}
}

// NodeListToResponse converts a NodeList to a slice of NodeResponseDTO
func (m *NodeMapper) NodeListToResponse(nodes models.NodeList) []response.NodeResponseDTO {
	responseNodes := make([]response.NodeResponseDTO, 0, len(nodes))
	for _, node := range nodes {
		responseNodes = append(responseNodes, m.NodeConfigToResponse(node))
	}
	return responseNodes
}

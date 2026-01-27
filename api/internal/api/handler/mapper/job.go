package mapper

import (
	"api/internal/api/handler/request"
	"api/internal/api/handler/response"
	"api/internal/api/models"
)

// JobMapper handles job-related mappings
// Note: ToJobResponse methods are implemented manually due to complex Nodes mapping
//
//go:generate go run ../../../../tools/dtomapper -type=JobMapper
type JobMapper interface {
	// Request mapping
	CreateJob(req request.CreateJob) models.Job

	// patch
	PatchJob(req request.UpdateJob) map[string]any

	// Response mapping (simple, without nodes)
	ToJobResponses(entities []models.Job) []response.Job
	ToJobResponse(j models.Job) response.Job
}

// ToJobResponseWithNodes converts a job model to response including nodes and shared users
func ToJobResponseWithNodes(j models.Job, accessList []models.JobUserAccess) response.JobWithNodes {
	resp := response.JobWithNodes{
		ID:          j.ID,
		Name:        j.Name,
		Description: j.Description,
		FilePath:    j.FilePath,
		CreatorID:   j.CreatorID,
		Active:      j.Active,
		Visibility:  j.Visibility,
		OutputPath:  j.OutputPath,
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   j.UpdatedAt,
		Nodes:       nil,
		Connexions:  nil,
		SharedUser:  nil,
	}

	if len(j.Nodes) > 0 {
		resp.Nodes = make([]response.Node, len(j.Nodes))
		for i, n := range j.Nodes {
			resp.Nodes[i] = toNodeResponse(n)
		}

		// Build node lookup for resolving target port indices
		nodeByID := make(map[int]models.Node, len(j.Nodes))
		for _, n := range j.Nodes {
			nodeByID[n.ID] = n
		}

		for _, n := range j.Nodes {
			for portIdx, p := range n.OutputPort {
				connType := toConnexionPortType(p.Type)
				targetPortIdx := findTargetPortIndex(nodeByID, p, n.ID)
				resp.Connexions = append(resp.Connexions, response.Connexion{
					SourceNodeId:   n.ID,
					SourcePort:     portIdx,
					SourcePortType: connType,
					TargetNodeId:   int(p.ConnectedNodeID),
					TargetPort:     targetPortIdx,
					TargetPortType: connType,
				})
			}
		}
	}

	// Map shared users with their roles
	if len(j.SharedWith) > 0 {
		resp.SharedUser = make([]response.SharedUser, len(j.SharedWith))
		for i, u := range j.SharedWith {
			role := models.Viewer // default
			for _, access := range accessList {
				if access.UserID == u.ID {
					role = access.Role
					break
				}
			}
			resp.SharedUser[i] = response.SharedUser{
				ID:     u.ID,
				Email:  u.Email,
				Prenom: u.Prenom,
				Nom:    u.Nom,
				Role:   role,
			}
		}
	}

	return resp
}

func toNodeResponse(n models.Node) response.Node {
	node := response.Node{
		ID:   n.ID,
		Type: n.Type,
		Name: n.Name,
		Xpos: n.Xpos,
		Ypos: n.Ypos,
		Data: n.Data,
	}

	return node
}

// toConnexionPortType maps backend port types to frontend-friendly values ("data" or "flow").
func toConnexionPortType(pt models.PortType) models.PortType {
	switch pt {
	case models.PortTypeOutput, models.PortTypeInput:
		return "data"
	case models.PortNodeFlowOutput, models.PortNodeFlowInput:
		return "flow"
	default:
		return pt
	}
}

// correspondingInputType returns the input port type that matches the given output port type.
func correspondingInputType(outputType models.PortType) models.PortType {
	switch outputType {
	case models.PortTypeOutput:
		return models.PortTypeInput
	case models.PortNodeFlowOutput:
		return models.PortNodeFlowInput
	default:
		return outputType
	}
}

// findTargetPortIndex finds the index of the matching input port on the target node.
func findTargetPortIndex(nodeByID map[int]models.Node, outputPort models.Port, sourceNodeID int) int {
	target, ok := nodeByID[int(outputPort.ConnectedNodeID)]
	if !ok {
		return 0
	}
	inputType := correspondingInputType(outputPort.Type)
	for idx, ip := range target.InputPort {
		if ip.ConnectedNodeID == uint(sourceNodeID) && ip.Type == inputType {
			return idx
		}
	}
	return 0
}

func JobWithNodeToModel(jwn request.UpdateJob) []models.Node {
	var nodes []models.Node
	if len(jwn.Nodes) > 0 {
		for _, n := range jwn.Nodes {
			nodes = append(nodes, models.Node{
				ID:   n.ID,
				Type: n.Type,
				Name: n.Name,
				Xpos: n.Xpos,
				Ypos: n.Ypos,
				Data: n.Data,
			})
		}

		// Rebuild ports from the flat connexion array
		nodeIdxByID := make(map[int]int, len(jwn.Nodes))
		for i, n := range jwn.Nodes {
			nodeIdxByID[n.ID] = i
		}

		for _, c := range jwn.Connexions {
			outputPortType := fromConnexionPortType(c.SourcePortType, false)
			inputPortType := fromConnexionPortType(c.TargetPortType, true)

			// Add output port to source node
			if idx, ok := nodeIdxByID[c.SourceNodeId]; ok {
				nodes[idx].OutputPort = append(nodes[idx].OutputPort, models.Port{
					Type:            outputPortType,
					ConnectedNodeID: uint(c.TargetNodeId),
				})
			}

			// Add input port to target node
			if idx, ok := nodeIdxByID[c.TargetNodeId]; ok {
				nodes[idx].InputPort = append(nodes[idx].InputPort, models.Port{
					Type:            inputPortType,
					ConnectedNodeID: uint(c.SourceNodeId),
				})
			}
		}
	}

	return nodes
}

// fromConnexionPortType converts frontend port types ("data"/"flow") back to model port types.
func fromConnexionPortType(pt models.PortType, isInput bool) models.PortType {
	switch pt {
	case "data":
		if isInput {
			return models.PortTypeInput
		}
		return models.PortTypeOutput
	case "flow":
		if isInput {
			return models.PortNodeFlowInput
		}
		return models.PortNodeFlowOutput
	default:
		return pt
	}
}

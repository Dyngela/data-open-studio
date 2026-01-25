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
	}

	if len(j.Nodes) > 0 {
		resp.Nodes = make([]response.Node, len(j.Nodes))
		for i, n := range j.Nodes {
			resp.Nodes[i] = toNodeResponse(n)
		}
	}

	// Map shared users with their roles
	if len(j.SharedWith) > 0 {
		resp.SharedWith = make([]response.SharedUser, len(j.SharedWith))
		for i, u := range j.SharedWith {
			role := models.Viewer // default
			for _, access := range accessList {
				if access.UserID == u.ID {
					role = access.Role
					break
				}
			}
			resp.SharedWith[i] = response.SharedUser{
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
		ID:    n.ID,
		Type:  n.Type,
		Name:  n.Name,
		Xpos:  n.Xpos,
		Ypos:  n.Ypos,
		Data:  n.Data,
		JobID: n.JobID,
	}

	if len(n.InputPort) > 0 {
		node.InputPort = make([]response.Port, len(n.InputPort))
		for i, p := range n.InputPort {
			node.InputPort[i] = response.Port{ID: p.ID, Type: string(p.Type), NodeID: p.NodeID}
		}
	}

	if len(n.OutputPort) > 0 {
		node.OutputPort = make([]response.Port, len(n.OutputPort))
		for i, p := range n.OutputPort {
			node.OutputPort[i] = response.Port{ID: p.ID, Type: string(p.Type), NodeID: p.NodeID}
		}
	}

	return node
}

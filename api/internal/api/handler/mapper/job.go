package mapper

import (
	"api/internal/api/handler/response"
	"api/internal/api/models"
)

type JobMapper struct {
	nodeMapper NodeMapper
}

// EntityToJobResponse converts a Job entity to JobResponseDTO
func (m *JobMapper) EntityToJobResponse(job models.Job) response.JobResponseDTO {
	// Convert nodes using the node mapper
	nodeResponses := m.nodeMapper.NodeListToResponse(job.Nodes)

	return response.JobResponseDTO{
		ID:          job.ID,
		Name:        job.Name,
		Description: job.Description,
		CreatorID:   job.CreatorID,
		Active:      job.Active,
		Nodes:       nodeResponses,
	}
}

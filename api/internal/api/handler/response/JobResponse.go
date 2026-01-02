package response

// JobResponseDTO represents a job response
type JobResponseDTO struct {
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	CreatorID   uint              `json:"creatorId"`
	Active      bool              `json:"active"`
	Nodes       []NodeResponseDTO `json:"nodes"`
}

package repo

import (
	"api"
	"api/internal/api/models"

	"gorm.io/gorm"
)

type JobRepository struct {
	Db *gorm.DB
}

func NewJobRepository() *JobRepository {
	return &JobRepository{Db: api.DB}
}

// FindByID retrieves a job by ID and its associated nodes and ports
func (slf *JobRepository) FindByID(id uint) (models.Job, error) {
	var job models.Job
	err := slf.Db.
		Preload("Nodes").
		Preload("Nodes.InputPort", "type IN ?", []string{"input", "node_flow_input"}).
		Preload("Nodes.OutputPort", "type IN ?", []string{"output", "node_flow_output"}).
		First(&job, id).Error
	return job, err
}

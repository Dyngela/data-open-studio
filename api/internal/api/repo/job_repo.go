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
		Preload("Nodes.InputPort").
		Preload("Nodes.OutputPort").
		First(&job, id).Error
	return job, err
}

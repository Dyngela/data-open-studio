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

// FindByID retrieves a job by ID
func (slf *JobRepository) FindByID(id uint) (models.Job, error) {
	var job models.Job
	err := slf.Db.First(&job, id).Error
	return job, err
}

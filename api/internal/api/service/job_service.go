package service

import (
	"api"
	"api/internal/api/handler/mapper"
	"api/internal/api/models"
	"api/internal/api/repo"
	"errors"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type JobService struct {
	jobRepo   *repo.JobRepository
	logger    zerolog.Logger
	jobMapper mapper.JobMapper
}

func NewJobService() *JobService {
	return &JobService{
		jobRepo: repo.NewJobRepository(),
		logger:  api.Logger,
	}
}

// FindJobByID retrieves the raw job model (for WebSocket operations)
func (slf *JobService) FindJobByID(id uint) (models.Job, error) {
	job, err := slf.jobRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slf.logger.Error().Uint("jobId", id).Msg("Job not found")
			return models.Job{}, errors.New("job not found")
		}
		slf.logger.Error().Err(err).Uint("jobId", id).Msg("Error getting job")
		return models.Job{}, err
	}

	return job, nil
}

func (slf *JobService) CreateJob(job models.Job) (models.Job, error) {
	err := slf.jobRepo.Db.Create(&job).Error
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error creating job")
		return models.Job{}, err
	}
	return job, nil
}

// Patch updates a job directly (for WebSocket operations)
func (slf *JobService) Patch(job map[string]any) error {
	if err := slf.jobRepo.Db.Updates(&job).Error; err != nil {
		return err
	}
	return nil
}

func (slf *JobService) DeleteJob(id uint) error {
	if err := slf.jobRepo.Db.Delete(id).Error; err != nil {
		slf.logger.Error().Err(err).Uint("jobId", id).Msg("Error deleting job")
		return err
	}
	return nil
}

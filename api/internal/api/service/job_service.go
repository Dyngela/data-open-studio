package service

import (
	"api"
	"api/internal/api/models"
	"api/internal/api/repo"
	"errors"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type JobService struct {
	jobRepo *repo.JobRepository
	logger  zerolog.Logger
}

func NewJobService() *JobService {
	return &JobService{
		jobRepo: repo.NewJobRepository(),
		logger:  api.Logger,
	}
}

// FindAllForUser retrieves all jobs visible to a user (public + owned + shared)
func (slf *JobService) FindAllForUser(userID uint) ([]models.Job, error) {
	var jobs []models.Job

	// Get jobs that are:
	// 1. Public
	// 2. Created by the user
	// 3. Shared with the user
	err := slf.jobRepo.Db.
		Distinct().
		Joins("LEFT JOIN job_user_access ON job_user_access.job_id = job.id").
		Where("job.visibility = ? OR job.creator_id = ? OR job_user_access.user_id = ?",
			models.JobVisibilityPublic, userID, userID).
		Find(&jobs).Error

	if err != nil {
		slf.logger.Error().Err(err).Uint("userID", userID).Msg("Error getting jobs for user")
		return nil, err
	}
	return jobs, nil
}

// FindByFilePathForUser retrieves jobs by virtual folder path visible to a user
func (slf *JobService) FindByFilePathForUser(filePath string, userID uint) ([]models.Job, error) {
	var jobs []models.Job

	err := slf.jobRepo.Db.
		Distinct().
		Joins("LEFT JOIN job_user_access ON job_user_access.job_id = job.id").
		Where("job.file_path = ?", filePath).
		Where("job.visibility = ? OR job.creator_id = ? OR job_user_access.user_id = ?",
			models.JobVisibilityPublic, userID, userID).
		Find(&jobs).Error

	if err != nil {
		slf.logger.Error().Err(err).Str("filePath", filePath).Msg("Error getting jobs by file path")
		return nil, err
	}
	return jobs, nil
}

// FindByID retrieves a job by ID with its nodes
func (slf *JobService) FindByID(id uint) (*models.Job, error) {
	job, err := slf.jobRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slf.logger.Error().Uint("jobId", id).Msg("Job not found")
			return nil, errors.New("job not found")
		}
		slf.logger.Error().Err(err).Uint("jobId", id).Msg("Error getting job")
		return nil, err
	}
	return &job, nil
}

// Create creates a new job with its nodes
func (slf *JobService) Create(job models.Job) (*models.Job, error) {
	err := slf.jobRepo.Db.Create(&job).Error
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error creating job")
		return nil, err
	}
	return &job, nil
}

// Update updates a job's fields (not nodes)
func (slf *JobService) Update(id uint, patch map[string]any) (*models.Job, error) {
	if err := slf.jobRepo.Db.Model(&models.Job{}).Where("id = ?", id).Updates(patch).Error; err != nil {
		slf.logger.Error().Err(err).Uint("jobId", id).Msg("Error updating job")
		return nil, err
	}
	return slf.FindByID(id)
}

// UpdateWithNodes updates a job and replaces its nodes
func (slf *JobService) UpdateWithNodes(id uint, patch map[string]any, nodes []models.Node) (*models.Job, error) {
	tx := slf.jobRepo.Db.Begin()

	// Update job fields
	if len(patch) > 0 {
		if err := tx.Model(&models.Job{}).Where("id = ?", id).Updates(patch).Error; err != nil {
			tx.Rollback()
			slf.logger.Error().Err(err).Uint("jobId", id).Msg("Error updating job")
			return nil, err
		}
	}

	// Delete existing nodes and create new ones
	if err := tx.Where("job_id = ?", id).Delete(&models.Node{}).Error; err != nil {
		tx.Rollback()
		slf.logger.Error().Err(err).Uint("jobId", id).Msg("Error deleting old nodes")
		return nil, err
	}

	// Set JobID for all nodes and create them
	for i := range nodes {
		nodes[i].JobID = id
		nodes[i].ID = 0 // Reset ID to allow auto-increment
	}

	if len(nodes) > 0 {
		if err := tx.Create(&nodes).Error; err != nil {
			tx.Rollback()
			slf.logger.Error().Err(err).Uint("jobId", id).Msg("Error creating new nodes")
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		slf.logger.Error().Err(err).Uint("jobId", id).Msg("Error committing transaction")
		return nil, err
	}

	return slf.FindByID(id)
}

// Delete removes a job and its nodes
func (slf *JobService) Delete(id uint) error {
	// Delete sharing records first
	if err := slf.jobRepo.Db.Where("job_id = ?", id).Delete(&models.JobUserAccess{}).Error; err != nil {
		slf.logger.Error().Err(err).Uint("jobId", id).Msg("Error deleting job access records")
		return err
	}

	// Nodes will be cascade deleted due to foreign key
	if err := slf.jobRepo.Db.Delete(&models.Job{}, id).Error; err != nil {
		slf.logger.Error().Err(err).Uint("jobId", id).Msg("Error deleting job")
		return err
	}
	return nil
}

// CanUserAccess checks if a user can access a job
func (slf *JobService) CanUserAccess(jobID, userID uint) (bool, string, error) {
	var job models.Job
	if err := slf.jobRepo.Db.First(&job, jobID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "", errors.New("job not found")
		}
		return false, "", err
	}

	// Owner has full access
	if job.CreatorID == userID {
		return true, "owner", nil
	}

	// Public jobs are accessible to all
	if job.Visibility == models.JobVisibilityPublic {
		return true, "viewer", nil
	}

	// Check if user has explicit access
	var access models.JobUserAccess
	err := slf.jobRepo.Db.Where("job_id = ? AND user_id = ?", jobID, userID).First(&access).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "", nil
		}
		return false, "", err
	}

	return true, access.Role, nil
}

// ShareJob shares a job with users
func (slf *JobService) ShareJob(jobID uint, userIDs []uint, role string) error {
	if role == "" {
		role = "viewer"
	}

	tx := slf.jobRepo.Db.Begin()

	for _, userID := range userIDs {
		access := models.JobUserAccess{
			JobID:  jobID,
			UserID: userID,
			Role:   role,
		}

		// Upsert: update role if exists, create if not
		if err := tx.Where("job_id = ? AND user_id = ?", jobID, userID).
			Assign(models.JobUserAccess{Role: role}).
			FirstOrCreate(&access).Error; err != nil {
			tx.Rollback()
			slf.logger.Error().Err(err).Uint("jobId", jobID).Uint("userId", userID).Msg("Error sharing job")
			return err
		}
	}

	return tx.Commit().Error
}

// UnshareJob removes users' access to a job
func (slf *JobService) UnshareJob(jobID uint, userIDs []uint) error {
	if err := slf.jobRepo.Db.Where("job_id = ? AND user_id IN ?", jobID, userIDs).
		Delete(&models.JobUserAccess{}).Error; err != nil {
		slf.logger.Error().Err(err).Uint("jobId", jobID).Msg("Error unsharing job")
		return err
	}
	return nil
}

// GetJobAccess retrieves the access list for a job
func (slf *JobService) GetJobAccess(jobID uint) ([]models.JobUserAccess, error) {
	var accessList []models.JobUserAccess
	if err := slf.jobRepo.Db.Where("job_id = ?", jobID).Find(&accessList).Error; err != nil {
		return nil, err
	}
	return accessList, nil
}

// UpdateJobSharing replaces the sharing list for a job
func (slf *JobService) UpdateJobSharing(jobID uint, userIDs []uint, role string) error {
	if role == "" {
		role = "viewer"
	}

	tx := slf.jobRepo.Db.Begin()

	// Delete existing sharing records
	if err := tx.Where("job_id = ?", jobID).Delete(&models.JobUserAccess{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Create new sharing records
	for _, userID := range userIDs {
		access := models.JobUserAccess{
			JobID:  jobID,
			UserID: userID,
			Role:   role,
		}
		if err := tx.Create(&access).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// FindByIDWithAccess retrieves a job with its shared users
func (slf *JobService) FindByIDWithAccess(id uint) (*models.Job, []models.JobUserAccess, error) {
	var job models.Job
	err := slf.jobRepo.Db.
		Preload("Nodes").
		Preload("Nodes.InputPort").
		Preload("Nodes.OutputPort").
		Preload("SharedWith").
		First(&job, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("job not found")
		}
		return nil, nil, err
	}

	accessList, err := slf.GetJobAccess(id)
	if err != nil {
		return nil, nil, err
	}

	return &job, accessList, nil
}

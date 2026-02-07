package repo

import (
	"api"
	"api/internal/api/models"
	"time"

	"gorm.io/gorm"
)

type TriggerRepository struct {
	Db *gorm.DB
}

func NewTriggerRepository() *TriggerRepository {
	return &TriggerRepository{Db: api.DB}
}

// FindByID retrieves a trigger by ID with rules and jobs
func (slf *TriggerRepository) FindByID(id uint) (models.Trigger, error) {
	var trigger models.Trigger
	err := slf.Db.
		Preload("Rules").
		Preload("Jobs").
		Preload("Jobs.Job").
		First(&trigger, id).Error
	return trigger, err
}

// FindByIDSimple retrieves a trigger by ID without preloading
func (slf *TriggerRepository) FindByIDSimple(id uint) (models.Trigger, error) {
	var trigger models.Trigger
	err := slf.Db.First(&trigger, id).Error
	return trigger, err
}

// FindAllByCreator retrieves all triggers for a user
func (slf *TriggerRepository) FindAllByCreator(creatorID uint) ([]models.Trigger, error) {
	var triggers []models.Trigger
	err := slf.Db.
		Where("creator_id = ?", creatorID).
		Preload("Jobs").
		Order("created_at DESC").
		Find(&triggers).Error
	return triggers, err
}

// FindAllActive retrieves all active triggers for polling
func (slf *TriggerRepository) FindAllActive() ([]models.Trigger, error) {
	var triggers []models.Trigger
	err := slf.Db.
		Where("status = ?", models.TriggerStatusActive).
		Preload("Rules").
		Preload("Jobs", "active = ?", true).
		Find(&triggers).Error
	return triggers, err
}

// FindActiveByType retrieves all active triggers of a specific type
func (slf *TriggerRepository) FindActiveByType(triggerType models.TriggerType) ([]models.Trigger, error) {
	var triggers []models.Trigger
	err := slf.Db.
		Where("status = ? AND type = ?", models.TriggerStatusActive, triggerType).
		Preload("Rules").
		Preload("Jobs", "active = ?", true).
		Find(&triggers).Error
	return triggers, err
}

// Create creates a new trigger
func (slf *TriggerRepository) Create(trigger *models.Trigger) error {
	return slf.Db.Create(trigger).Error
}

// Update updates an existing trigger
func (slf *TriggerRepository) Update(trigger *models.Trigger) error {
	return slf.Db.Save(trigger).Error
}

// UpdateStatus updates only the status and error fields
func (slf *TriggerRepository) UpdateStatus(id uint, status models.TriggerStatus, lastError string) error {
	return slf.Db.Model(&models.Trigger{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"last_error": lastError,
		}).Error
}

// UpdateLastPolled updates the last polled timestamp
func (slf *TriggerRepository) UpdateLastPolled(id uint, lastPolledAt time.Time) error {
	return slf.Db.Model(&models.Trigger{}).
		Where("id = ?", id).
		Update("last_polled_at", lastPolledAt).Error
}

// UpdateWatermark updates the database trigger watermark
func (slf *TriggerRepository) UpdateWatermark(id uint, watermark string) error {
	return slf.Db.Model(&models.Trigger{}).
		Where("id = ?", id).
		Update("config", gorm.Expr(
			"jsonb_set(config, '{database,lastWatermark}', to_jsonb(?::text))",
			watermark,
		)).Error
}

// UpdateEmailUID updates the email trigger last UID
func (slf *TriggerRepository) UpdateEmailUID(id uint, uid uint32) error {
	return slf.Db.Model(&models.Trigger{}).
		Where("id = ?", id).
		Update("config", gorm.Expr(
			"jsonb_set(config, '{email,lastUid}', to_jsonb(?::int))",
			uid,
		)).Error
}

// Delete soft-deletes a trigger
func (slf *TriggerRepository) Delete(id uint) error {
	return slf.Db.Delete(&models.Trigger{}, id).Error
}

// AddRule adds a rule to a trigger
func (slf *TriggerRepository) AddRule(rule *models.TriggerRule) error {
	return slf.Db.Create(rule).Error
}

// UpdateRule updates an existing rule
func (slf *TriggerRepository) UpdateRule(rule *models.TriggerRule) error {
	return slf.Db.Save(rule).Error
}

// DeleteRule deletes a rule
func (slf *TriggerRepository) DeleteRule(ruleID uint) error {
	return slf.Db.Delete(&models.TriggerRule{}, ruleID).Error
}

// AddJob links a job to a trigger
func (slf *TriggerRepository) AddJob(triggerJob *models.TriggerJob) error {
	return slf.Db.Create(triggerJob).Error
}

// RemoveJob removes a job link from a trigger
func (slf *TriggerRepository) RemoveJob(triggerID, jobID uint) error {
	return slf.Db.
		Where("trigger_id = ? AND job_id = ?", triggerID, jobID).
		Delete(&models.TriggerJob{}).Error
}

// UpdateJobLink updates a trigger-job link
func (slf *TriggerRepository) UpdateJobLink(triggerJob *models.TriggerJob) error {
	return slf.Db.Save(triggerJob).Error
}

// CreateExecution creates a new trigger execution record
func (slf *TriggerRepository) CreateExecution(exec *models.TriggerExecution) error {
	return slf.Db.Create(exec).Error
}

// UpdateExecution updates an execution record
func (slf *TriggerRepository) UpdateExecution(exec *models.TriggerExecution) error {
	return slf.Db.Save(exec).Error
}

// GetRecentExecutions retrieves recent executions for a trigger
func (slf *TriggerRepository) GetRecentExecutions(triggerID uint, limit int) ([]models.TriggerExecution, error) {
	var executions []models.TriggerExecution
	err := slf.Db.
		Where("trigger_id = ?", triggerID).
		Order("started_at DESC").
		Limit(limit).
		Find(&executions).Error
	return executions, err
}

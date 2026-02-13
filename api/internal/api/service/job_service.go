package service

import (
	"api"
	"api/internal/api/models"
	"api/internal/api/repo"
	"api/internal/gen"
	"api/internal/gen/lib"
	"api/pkg"
	"errors"
	"fmt"
	"time"

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

// UpdateWithNodes updates a job and upserts its nodes (preserving existing IDs)
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

	// Get existing node IDs for this job
	var existingIDs []int
	if err := tx.Model(&models.Node{}).Where("job_id = ?", id).Pluck("id", &existingIDs).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	existingSet := make(map[int]bool, len(existingIDs))
	for _, nid := range existingIDs {
		existingSet[nid] = true
	}

	// Save original state per node (ports + original ID)
	type nodeState struct {
		originalID int
		input      []models.Port
		output     []models.Port
		isExisting bool
	}
	states := make([]nodeState, len(nodes))
	keepIDs := make(map[int]bool)

	for i := range nodes {
		isExisting := nodes[i].ID > 0 && existingSet[nodes[i].ID]
		states[i] = nodeState{
			originalID: nodes[i].ID,
			input:      nodes[i].InputPort,
			output:     nodes[i].OutputPort,
			isExisting: isExisting,
		}
		nodes[i].InputPort = nil
		nodes[i].OutputPort = nil
		nodes[i].JobID = id
		if isExisting {
			keepIDs[nodes[i].ID] = true
		}
	}

	// Delete nodes that were removed by the user
	var removeIDs []int
	for _, nid := range existingIDs {
		if !keepIDs[nid] {
			removeIDs = append(removeIDs, nid)
		}
	}
	if len(removeIDs) > 0 {
		if err := tx.Where("node_id IN ?", removeIDs).Delete(&models.Port{}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := tx.Where("id IN ?", removeIDs).Delete(&models.Node{}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Upsert nodes: update existing in place, create new ones
	idMap := make(map[int]int, len(nodes)) // originalID → finalID
	for i := range nodes {
		if states[i].isExisting {
			// Update in place — preserves the ID
			if err := tx.Model(&models.Node{}).Where("id = ?", nodes[i].ID).Updates(map[string]any{
				"type": nodes[i].Type,
				"name": nodes[i].Name,
				"xpos": nodes[i].Xpos,
				"ypos": nodes[i].Ypos,
				"data": nodes[i].Data,
			}).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
			idMap[states[i].originalID] = nodes[i].ID
		} else {
			// New node — create to get auto-increment ID
			oldID := nodes[i].ID
			nodes[i].ID = 0
			if err := tx.Create(&nodes[i]).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
			idMap[oldID] = nodes[i].ID
		}
	}

	resolveID := func(oldID int) uint {
		if finalID, ok := idMap[oldID]; ok {
			return uint(finalID)
		}
		return uint(oldID)
	}

	// Delete all ports for kept nodes (will recreate below)
	if len(keepIDs) > 0 {
		keptIDs := make([]int, 0, len(keepIDs))
		for nid := range keepIDs {
			keptIDs = append(keptIDs, nid)
		}
		if err := tx.Where("node_id IN ?", keptIDs).Delete(&models.Port{}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Recreate all ports with correct node IDs
	var allPorts []models.Port
	for i := range nodes {
		finalNodeID := resolveID(states[i].originalID)
		for _, p := range states[i].output {
			allPorts = append(allPorts, models.Port{
				Type:            p.Type,
				NodeID:          finalNodeID,
				ConnectedNodeID: resolveID(int(p.ConnectedNodeID)),
			})
		}
		for _, p := range states[i].input {
			allPorts = append(allPorts, models.Port{
				Type:            p.Type,
				NodeID:          finalNodeID,
				ConnectedNodeID: resolveID(int(p.ConnectedNodeID)),
			})
		}
	}

	if len(allPorts) > 0 {
		if err := tx.Create(&allPorts).Error; err != nil {
			tx.Rollback()
			slf.logger.Error().Err(err).Uint("jobId", id).Msg("Error creating ports")
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
func (slf *JobService) CanUserAccess(jobID, userID uint) (bool, models.OwningJob, error) {
	var job models.Job
	if err := slf.jobRepo.Db.First(&job, jobID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "", errors.New("job not found")
		}
		return false, "", err
	}

	// Owner has full access
	if job.CreatorID == userID {
		return true, models.Owner, nil
	}

	// Public jobs are accessible to all
	if job.Visibility == models.JobVisibilityPublic {
		return true, models.Viewer, nil
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
func (slf *JobService) ShareJob(jobID uint, userIDs []uint, role models.OwningJob) error {
	if role == "" {
		role = models.Viewer
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
func (slf *JobService) UpdateJobSharing(jobID uint, userIDs []uint, role models.OwningJob) error {
	if role == "" {
		role = models.Viewer
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

// AddNotificationContact adds a user as a notification contact for a job
func (slf *JobService) AddNotificationContact(jobID, userID uint) error {
	contact := models.JobNotificationContact{
		JobID:  jobID,
		UserID: userID,
	}
	return slf.jobRepo.Db.FirstOrCreate(&contact, "job_id = ? AND user_id = ?", jobID, userID).Error
}

// RemoveNotificationContact removes a user from a job's notification contacts
func (slf *JobService) RemoveNotificationContact(jobID, userID uint) error {
	return slf.jobRepo.Db.Where("job_id = ? AND user_id = ?", jobID, userID).
		Delete(&models.JobNotificationContact{}).Error
}

// GetNotificationContacts retrieves users to notify for a job
func (slf *JobService) GetNotificationContacts(jobID uint) ([]models.User, error) {
	var users []models.User
	err := slf.jobRepo.Db.
		Joins("JOIN job_notification_contact ON job_notification_contact.user_id = users.id").
		Where("job_notification_contact.job_id = ?", jobID).
		Find(&users).Error
	return users, err
}

// FindByIDWithAccess retrieves a job with its shared users
func (slf *JobService) FindByIDWithAccess(id uint) (*models.Job, []models.JobUserAccess, error) {
	var job models.Job
	err := slf.jobRepo.Db.
		Preload("Nodes").
		Preload("Nodes.InputPort", "type IN ?", []string{"input", "node_flow_input"}).
		Preload("Nodes.OutputPort", "type IN ?", []string{"output", "node_flow_output"}).
		Preload("SharedWith").
		Preload("NotifyUsers").
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

func (slf *JobService) Execute(id uint) error {
	job, err := slf.jobRepo.FindByID(id)
	if err != nil {
		return err
	}
	executer := gen.NewJobExecution(&job)
	err = executer.Run()

	slf.logger.Info().Msgf("%v", err)
	//slf.logger.Info().Msgf("steps: %v", executer.Steps)
	//slf.logger.Info().Msgf("context: %v", executer.Context)

	// Notify frontend via NATS that the job is done
	slf.notifyJobDone(id, err, executer.Logs, executer.Stats)

	return err
}

// notifyJobDone publishes a final progress message (nodeId=0) so the frontend knows the job ended.
// On failure, sends email notifications to configured contacts.
func (slf *JobService) notifyJobDone(jobID uint, jobErr error, logs string, stats gen.DockerStats) {
	natsURL := api.GetEnv("NATS_URL", "nats://localhost:4222")
	tenantID := api.GetEnv("TENANT_ID", "default")

	reporter := lib.NewProgressReporter(natsURL, tenantID, jobID)
	defer reporter.Close()

	progress := reporter.ReportFunc()
	if jobErr != nil {
		progress(lib.NewProgress(0, "Pipeline", lib.StatusFailed, 0, jobErr.Error()))
		slf.sendFailureEmails(jobID, jobErr, logs, stats)
	} else {
		progress(lib.NewProgress(0, "Pipeline", lib.StatusCompleted, 0, "Pipeline completed successfully"))
	}
}

// sendFailureEmails sends an email to all notification contacts for the given job.
func (slf *JobService) sendFailureEmails(jobID uint, jobErr error, logs string, stats gen.DockerStats) {
	contacts, err := slf.GetNotificationContacts(jobID)
	if err != nil {
		slf.logger.Error().Err(err).Uint("jobID", jobID).Msg("Failed to get notification contacts for failure email")
		return
	}
	if len(contacts) == 0 {
		return
	}

	job, findErr := slf.jobRepo.FindByID(jobID)
	jobName := fmt.Sprintf("Job #%d", jobID)
	if findErr == nil {
		jobName = job.Name
	}

	truncatedLogs := logs
	if len(truncatedLogs) > 50000 {
		truncatedLogs = truncatedLogs[:50000] + "\n\n... (logs truncated)"
	}

	recipients := make([]string, len(contacts))
	for i, u := range contacts {
		recipients[i] = u.Email
	}

	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; color: #333;">
  <h2 style="color: #d32f2f;">Job en échec : %s</h2>
  <table style="border-collapse: collapse; margin-bottom: 16px;">
    <tr><td style="padding: 4px 12px; font-weight: bold;">Job ID</td><td style="padding: 4px 12px;">%d</td></tr>
    <tr><td style="padding: 4px 12px; font-weight: bold;">Date</td><td style="padding: 4px 12px;">%s</td></tr>
    <tr><td style="padding: 4px 12px; font-weight: bold;">Erreur</td><td style="padding: 4px 12px; color: #d32f2f;">%s</td></tr>
    <tr><td style="padding: 4px 12px; font-weight: bold;">CPU</td><td style="padding: 4px 12px;">%s</td></tr>
    <tr><td style="padding: 4px 12px; font-weight: bold;">Mémoire</td><td style="padding: 4px 12px;">%s</td></tr>
  </table>
  <h3>Logs</h3>
  <pre style="background: #f5f5f5; padding: 12px; border-radius: 4px; overflow-x: auto; font-size: 12px; max-height: 600px; overflow-y: auto;">%s</pre>
</body>
</html>`,
		jobName,
		jobID,
		time.Now().Format("2006-01-02 15:04:05"),
		jobErr.Error(),
		stats.CPUPercent,
		stats.MemUsage,
		truncatedLogs,
	)

	msg := pkg.EmailMessage{
		To:          recipients,
		CC:          nil,
		BCC:         nil,
		Subject:     fmt.Sprintf("[Data Open Studio] Job en échec : %s", jobName),
		Body:        body,
		IsHTML:      true,
		Attachments: nil,
	}

	if err := pkg.SendEmail(msg); err != nil {
		slf.logger.Error().Err(err).Uint("jobID", jobID).Msg("Failed to send failure notification email")
	} else {
		slf.logger.Info().Uint("jobID", jobID).Int("recipients", len(recipients)).Msg("Failure notification email sent")
	}
}

func (slf *JobService) Stop(id uint) error {
	return gen.DockerStop(id, slf.logger)
}

func (slf *JobService) PrintCode(id uint) (string, any, error) {
	job, err := slf.jobRepo.FindByID(id)
	if err != nil {
		return "", nil, err
	}
	executer := gen.NewJobExecution(&job)
	return executer.LogDebug()
}

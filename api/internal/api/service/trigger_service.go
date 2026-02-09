package service

import (
	"api"
	"api/internal/api/models"
	"api/internal/api/repo"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/emersion/go-imap/v2/imapclient"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type TriggerService struct {
	triggerRepo *repo.TriggerRepository
	jobService  *JobService
	logger      zerolog.Logger
}

func NewTriggerService() *TriggerService {
	return &TriggerService{
		triggerRepo: repo.NewTriggerRepository(),
		jobService:  NewJobService(),
		logger:      api.Logger,
	}
}

// FindAllForUser retrieves all triggers for a user
func (slf *TriggerService) FindAllForUser(userID uint) ([]models.Trigger, error) {
	triggers, err := slf.triggerRepo.FindAllByCreator(userID)
	if err != nil {
		slf.logger.Error().Err(err).Uint("userID", userID).Msg("Error getting triggers for user")
		return nil, err
	}
	return triggers, nil
}

// FindByID retrieves a trigger by ID with rules and jobs
func (slf *TriggerService) FindByID(id uint) (*models.Trigger, error) {
	trigger, err := slf.triggerRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slf.logger.Error().Uint("triggerId", id).Msg("Trigger not found")
			return nil, errors.New("trigger not found")
		}
		slf.logger.Error().Err(err).Uint("triggerId", id).Msg("Error getting trigger")
		return nil, err
	}
	return &trigger, nil
}

// Create creates a new trigger
func (slf *TriggerService) Create(trigger models.Trigger) (*models.Trigger, error) {
	// Validate trigger type and config
	if err := slf.validateTriggerConfig(&trigger); err != nil {
		return nil, err
	}

	// Set default values
	if trigger.PollingInterval == 0 {
		trigger.PollingInterval = 60 // 1 minute default
	}
	if trigger.Status == "" {
		trigger.Status = models.TriggerStatusPaused
	}

	if err := slf.triggerRepo.Create(&trigger); err != nil {
		slf.logger.Error().Err(err).Msg("Error creating trigger")
		return nil, err
	}
	return &trigger, nil
}

// Update updates a trigger's fields
func (slf *TriggerService) Update(id uint, patch map[string]interface{}) (*models.Trigger, error) {
	// Get existing trigger
	_, err := slf.triggerRepo.FindByIDSimple(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("trigger not found")
		}
		return nil, err
	}

	if err := slf.triggerRepo.Db.Model(&models.Trigger{}).Where("id = ?", id).Updates(patch).Error; err != nil {
		slf.logger.Error().Err(err).Uint("triggerId", id).Msg("Error updating trigger")
		return nil, err
	}

	return slf.FindByID(id)
}

// UpdateConfig updates the trigger configuration
func (slf *TriggerService) UpdateConfig(id uint, config models.TriggerConfig) (*models.Trigger, error) {
	trigger, err := slf.triggerRepo.FindByIDSimple(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("trigger not found")
		}
		return nil, err
	}

	trigger.Config = config
	if err := slf.validateTriggerConfig(&trigger); err != nil {
		return nil, err
	}

	if err := slf.triggerRepo.Update(&trigger); err != nil {
		slf.logger.Error().Err(err).Uint("triggerId", id).Msg("Error updating trigger config")
		return nil, err
	}

	return slf.FindByID(id)
}

// Delete removes a trigger
func (slf *TriggerService) Delete(id uint) error {
	// First check if trigger exists
	_, err := slf.triggerRepo.FindByIDSimple(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("trigger not found")
		}
		return err
	}

	if err := slf.triggerRepo.Delete(id); err != nil {
		slf.logger.Error().Err(err).Uint("triggerId", id).Msg("Error deleting trigger")
		return err
	}
	return nil
}

// Activate activates a trigger (starts polling)
func (slf *TriggerService) Activate(id uint) (*models.Trigger, error) {
	trigger, err := slf.triggerRepo.FindByIDSimple(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("trigger not found")
		}
		return nil, err
	}

	// Validate config before activating
	if err := slf.validateTriggerConfig(&trigger); err != nil {
		return nil, err
	}

	// For database triggers, initialize watermark to current max so we only catch new events
	if trigger.Type == models.TriggerTypeDatabase && trigger.Config.Database != nil {
		if err := slf.initializeWatermark(&trigger); err != nil {
			slf.logger.Warn().Err(err).Uint("triggerId", id).Msg("Could not initialize watermark, will use existing value")
		}
	}

	// For email triggers, initialize last UID via IMAP so only new messages are processed
	if trigger.Type == models.TriggerTypeEmail && trigger.Config.Email != nil {
		if err := slf.initializeEmailUID(&trigger); err != nil {
			slf.logger.Warn().Err(err).Uint("triggerId", id).Msg("Could not initialize email UID, will use existing value")
		}
	}

	trigger.Status = models.TriggerStatusActive
	trigger.LastError = ""
	if err := slf.triggerRepo.Update(&trigger); err != nil {
		return nil, err
	}

	return slf.FindByID(id)
}

// initializeWatermark queries the source DB for the current max watermark value
// so the trigger only fires for events that happen after activation
func (slf *TriggerService) initializeWatermark(trigger *models.Trigger) error {
	cfg := trigger.Config.Database

	connCfg, err := slf.resolveConnection(cfg)
	if err != nil {
		return err
	}

	db, err := sql.Open(connCfg.GetDriverName(), connCfg.BuildConnectionString())
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(10 * time.Second)
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping: %w", err)
	}

	query := fmt.Sprintf("SELECT MAX(%s) FROM %s", cfg.WatermarkColumn, cfg.TableName)
	var maxVal interface{}
	if err := db.QueryRow(query).Scan(&maxVal); err != nil {
		return fmt.Errorf("failed to query max watermark: %w", err)
	}

	if maxVal == nil {
		// Table is empty, keep watermark as-is (zero value is fine)
		return nil
	}

	var wmStr string
	switch cfg.WatermarkType {
	case models.WatermarkTypeInt:
		wmStr = fmt.Sprintf("%v", maxVal)
	case models.WatermarkTypeTimestamp:
		if t, ok := maxVal.(time.Time); ok {
			wmStr = t.Format(time.RFC3339)
		} else {
			wmStr = fmt.Sprintf("%v", maxVal)
		}
	default:
		wmStr = fmt.Sprintf("%v", maxVal)
	}

	cfg.LastWatermark = wmStr
	trigger.Config.Database = cfg
	slf.logger.Info().Uint("triggerId", trigger.ID).Str("watermark", wmStr).Msg("Initialized watermark to current max")
	return nil
}

// resolveConnection builds a DBConnectionConfig from a database trigger config
func (slf *TriggerService) resolveConnection(cfg *models.DatabaseTriggerConfig) (models.DBConnectionConfig, error) {
	if cfg.MetadataDatabaseID != nil {
		var meta models.MetadataDatabase
		if err := slf.triggerRepo.Db.First(&meta, *cfg.MetadataDatabaseID).Error; err != nil {
			return models.DBConnectionConfig{}, fmt.Errorf("failed to load database metadata: %w", err)
		}
		return models.DBConnectionConfig{
			Type:     meta.DbType,
			Host:     meta.Host,
			Port:     meta.Port,
			Database: meta.DatabaseName,
			Username: meta.User,
			Password: meta.Password,
			SSLMode:  meta.SSLMode,
		}, nil
	}
	if cfg.Connection != nil {
		return *cfg.Connection, nil
	}
	return models.DBConnectionConfig{}, errors.New("no database connection configured")
}

// initializeEmailUID queries the IMAP server for the current max UID
// so the trigger only fires for emails received after activation
func (slf *TriggerService) initializeEmailUID(trigger *models.Trigger) error {
	cfg := trigger.Config.Email

	host, port, username, password, useTLS, err := slf.resolveEmailCredentials(cfg)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	var client *imapclient.Client
	if useTLS {
		client, err = imapclient.DialTLS(addr, &imapclient.Options{
			TLSConfig: &tls.Config{ServerName: host},
		})
	} else {
		client, err = imapclient.DialInsecure(addr, nil)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP: %w", err)
	}
	defer client.Close()

	if err := client.Login(username, password).Wait(); err != nil {
		return fmt.Errorf("IMAP login failed: %w", err)
	}

	folder := cfg.Folder
	if folder == "" {
		folder = "INBOX"
	}

	selectData, err := client.Select(folder, nil).Wait()
	if err != nil {
		return fmt.Errorf("failed to select folder: %w", err)
	}

	if selectData.UIDNext > 0 {
		cfg.LastUID = uint32(selectData.UIDNext - 1)
		trigger.Config.Email = cfg
		slf.logger.Info().Uint("triggerId", trigger.ID).Uint32("lastUID", cfg.LastUID).Msg("Initialized email UID to current max")
	}

	return nil
}

// resolveEmailCredentials resolves IMAP credentials from MetadataEmailID or inline config
func (slf *TriggerService) resolveEmailCredentials(cfg *models.EmailTriggerConfig) (host string, port int, username, password string, useTLS bool, err error) {
	if cfg.MetadataEmailID != nil {
		var meta models.MetadataEmail
		if err := slf.triggerRepo.Db.First(&meta, *cfg.MetadataEmailID).Error; err != nil {
			return "", 0, "", "", false, fmt.Errorf("failed to load email metadata: %w", err)
		}
		return meta.ImapHost, meta.ImapPort, meta.Username, meta.Password, meta.UseTLS, nil
	}
	return cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.UseTLS, nil
}

// Pause pauses a trigger (stops polling)
func (slf *TriggerService) Pause(id uint) (*models.Trigger, error) {
	if err := slf.triggerRepo.UpdateStatus(id, models.TriggerStatusPaused, ""); err != nil {
		return nil, err
	}
	return slf.FindByID(id)
}

// AddRule adds a rule to a trigger
func (slf *TriggerService) AddRule(triggerID uint, rule models.TriggerRule) (*models.TriggerRule, error) {
	// Verify trigger exists
	_, err := slf.triggerRepo.FindByIDSimple(triggerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("trigger not found")
		}
		return nil, err
	}

	rule.TriggerID = triggerID
	if err := slf.triggerRepo.AddRule(&rule); err != nil {
		slf.logger.Error().Err(err).Uint("triggerId", triggerID).Msg("Error adding rule")
		return nil, err
	}
	return &rule, nil
}

// UpdateRule updates a trigger rule
func (slf *TriggerService) UpdateRule(rule models.TriggerRule) (*models.TriggerRule, error) {
	if err := slf.triggerRepo.UpdateRule(&rule); err != nil {
		slf.logger.Error().Err(err).Uint("ruleId", rule.ID).Msg("Error updating rule")
		return nil, err
	}
	return &rule, nil
}

// DeleteRule removes a rule from a trigger
func (slf *TriggerService) DeleteRule(ruleID uint) error {
	return slf.triggerRepo.DeleteRule(ruleID)
}

// LinkJob links a job to a trigger
func (slf *TriggerService) LinkJob(triggerID, jobID uint, priority int, passEventData bool) (*models.TriggerJob, error) {
	// Verify trigger exists
	_, err := slf.triggerRepo.FindByIDSimple(triggerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("trigger not found")
		}
		return nil, err
	}

	// Verify job exists
	_, err = slf.jobService.FindByID(jobID)
	if err != nil {
		return nil, err
	}

	triggerJob := models.TriggerJob{
		TriggerID:     triggerID,
		JobID:         jobID,
		Priority:      priority,
		Active:        true,
		PassEventData: passEventData,
	}

	if err := slf.triggerRepo.AddJob(&triggerJob); err != nil {
		slf.logger.Error().Err(err).Uint("triggerId", triggerID).Uint("jobId", jobID).Msg("Error linking job")
		return nil, err
	}
	return &triggerJob, nil
}

// UnlinkJob removes a job from a trigger
func (slf *TriggerService) UnlinkJob(triggerID, jobID uint) error {
	return slf.triggerRepo.RemoveJob(triggerID, jobID)
}

// GetRecentExecutions retrieves recent executions for a trigger
func (slf *TriggerService) GetRecentExecutions(triggerID uint, limit int) ([]models.TriggerExecution, error) {
	if limit <= 0 {
		limit = 20
	}
	return slf.triggerRepo.GetRecentExecutions(triggerID, limit)
}

// CanUserAccess checks if a user can access a trigger
func (slf *TriggerService) CanUserAccess(triggerID, userID uint) (bool, error) {
	trigger, err := slf.triggerRepo.FindByIDSimple(triggerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("trigger not found")
		}
		return false, err
	}
	return trigger.CreatorID == userID, nil
}

// validateTriggerConfig validates the trigger configuration based on type
func (slf *TriggerService) validateTriggerConfig(trigger *models.Trigger) error {
	switch trigger.Type {
	case models.TriggerTypeDatabase:
		if trigger.Config.Database == nil {
			return errors.New("database trigger requires database configuration")
		}
		cfg := trigger.Config.Database
		if cfg.TableName == "" {
			return errors.New("table name is required for database trigger")
		}
		if cfg.WatermarkColumn == "" {
			return errors.New("watermark column is required for database trigger")
		}
		if cfg.WatermarkType == "" {
			return errors.New("watermark type is required for database trigger")
		}
		if cfg.MetadataDatabaseID == nil && cfg.Connection == nil {
			return errors.New("database connection is required for database trigger")
		}

	case models.TriggerTypeEmail:
		if trigger.Config.Email == nil {
			return errors.New("email trigger requires email configuration")
		}
		cfg := trigger.Config.Email
		if cfg.MetadataEmailID == nil {
			// Inline credentials required
			if cfg.Host == "" {
				return errors.New("IMAP host is required for email trigger")
			}
			if cfg.Username == "" {
				return errors.New("username is required for email trigger")
			}
			if cfg.Password == "" {
				return errors.New("password is required for email trigger")
			}
		}

	case models.TriggerTypeWebhook:
		// Webhook triggers have minimal config requirements
		if trigger.Config.Webhook == nil {
			trigger.Config.Webhook = &models.WebhookTriggerConfig{}
		}

	case models.TriggerTypeCron:
		if trigger.Config.Cron == nil {
			return errors.New("cron trigger requires cron configuration")
		}
		cfg := trigger.Config.Cron
		switch cfg.Mode {
		case models.CronModeInterval:
			if cfg.IntervalValue <= 0 {
				return errors.New("interval value must be greater than 0")
			}
			switch cfg.IntervalUnit {
			case models.IntervalUnitMinutes, models.IntervalUnitHours, models.IntervalUnitDays:
				// valid
			default:
				return errors.New("interval unit must be minutes, hours, or days")
			}
		case models.CronModeSchedule:
			switch cfg.ScheduleFrequency {
			case models.ScheduleFrequencyDaily, models.ScheduleFrequencyWeekly, models.ScheduleFrequencyMonthly:
				// valid
			default:
				return errors.New("schedule frequency must be daily, weekly, or monthly")
			}
			if cfg.ScheduleTime == "" {
				return errors.New("schedule time is required (HH:MM format)")
			}
			if cfg.ScheduleFrequency == models.ScheduleFrequencyWeekly && cfg.ScheduleDayOfWeek == nil {
				return errors.New("day of week is required for weekly schedule")
			}
			if cfg.ScheduleFrequency == models.ScheduleFrequencyMonthly && cfg.ScheduleDayOfMonth == nil {
				return errors.New("day of month is required for monthly schedule")
			}
		default:
			return errors.New("cron mode must be interval or schedule")
		}

	default:
		return errors.New("invalid trigger type")
	}

	return nil
}

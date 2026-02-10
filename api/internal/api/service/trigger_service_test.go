package service

import (
	"api"
	"api/internal/api/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTriggerTestDB(t *testing.T) {
	api.InitConfig("../../../.env.test")

	err := api.DB.AutoMigrate(
		&models.User{},
		&models.Job{},
		&models.Node{},
		&models.Port{},
		&models.JobUserAccess{},
		&models.Trigger{},
		&models.TriggerRule{},
		&models.TriggerJob{},
		&models.TriggerExecution{},
	)
	require.NoError(t, err, "Failed to migrate trigger-related tables")
}

func cleanupTrigger(t *testing.T, id uint) {
	if id > 0 {
		api.DB.Unscoped().Where("trigger_id = ?", id).Delete(&models.TriggerRule{})
		api.DB.Unscoped().Where("trigger_id = ?", id).Delete(&models.TriggerJob{})
		api.DB.Unscoped().Where("trigger_id = ?", id).Delete(&models.TriggerExecution{})
		api.DB.Unscoped().Delete(&models.Trigger{}, id)
	}
}

// ============ Trigger CRUD Tests ============

func TestTrigger_Create_Database(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	metaDBID := uint(1)
	trigger := models.Trigger{
		Name:        "DB Trigger",
		Description: "Monitors a database table",
		Type:        models.TriggerTypeDatabase,
		CreatorID:   user.ID,
		Config: models.TriggerConfig{
			Database: &models.DatabaseTriggerConfig{
				MetadataDatabaseID: &metaDBID,
				TableName:          "orders",
				WatermarkColumn:    "id",
				WatermarkType:      models.WatermarkTypeInt,
			},
		},
	}

	created, err := service.Create(trigger)
	require.NoError(t, err, "Failed to create database trigger")
	require.NotNil(t, created)
	require.NotZero(t, created.ID)
	defer cleanupTrigger(t, created.ID)

	assert.Equal(t, "DB Trigger", created.Name)
	assert.Equal(t, models.TriggerTypeDatabase, created.Type)
	assert.Equal(t, models.TriggerStatusPaused, created.Status)
	assert.Equal(t, 60, created.PollingInterval)
}

func TestTrigger_Create_Cron_Interval(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Cron Interval Trigger",
		Type:      models.TriggerTypeCron,
		CreatorID: user.ID,
		Config: models.TriggerConfig{
			Cron: &models.CronTriggerConfig{
				Mode:          models.CronModeInterval,
				IntervalValue: 30,
				IntervalUnit:  models.IntervalUnitMinutes,
			},
		},
	}

	created, err := service.Create(trigger)
	require.NoError(t, err, "Failed to create cron interval trigger")
	require.NotNil(t, created)
	defer cleanupTrigger(t, created.ID)

	assert.Equal(t, models.TriggerTypeCron, created.Type)
}

func TestTrigger_Create_DefaultValues(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Default Values Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}

	created, err := service.Create(trigger)
	require.NoError(t, err)
	defer cleanupTrigger(t, created.ID)

	assert.Equal(t, 60, created.PollingInterval, "Default polling interval should be 60")
	assert.Equal(t, models.TriggerStatusPaused, created.Status, "Default status should be paused")
}

func TestTrigger_FindByID(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Find Me Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}

	created, err := service.Create(trigger)
	require.NoError(t, err)
	defer cleanupTrigger(t, created.ID)

	found, err := service.FindByID(created.ID)
	require.NoError(t, err, "Failed to find trigger by ID")
	require.NotNil(t, found)

	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Find Me Trigger", found.Name)
}

func TestTrigger_FindByID_NotFound(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	_, err := service.FindByID(99999)
	require.Error(t, err)
	assert.Equal(t, "trigger not found", err.Error())
}

func TestTrigger_Update(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:        "Old Trigger Name",
		Description: "Old description",
		Type:        models.TriggerTypeWebhook,
		CreatorID:   user.ID,
		Config:      models.TriggerConfig{},
	}

	created, err := service.Create(trigger)
	require.NoError(t, err)
	defer cleanupTrigger(t, created.ID)

	patch := map[string]interface{}{
		"name":        "New Trigger Name",
		"description": "New description",
	}

	updated, err := service.Update(created.ID, patch)
	require.NoError(t, err, "Failed to update trigger")
	require.NotNil(t, updated)

	assert.Equal(t, "New Trigger Name", updated.Name)
	assert.Equal(t, "New description", updated.Description)
}

func TestTrigger_Delete(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Delete Me Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}

	created, err := service.Create(trigger)
	require.NoError(t, err)

	err = service.Delete(created.ID)
	require.NoError(t, err, "Failed to delete trigger")

	_, err = service.FindByID(created.ID)
	require.Error(t, err, "Should not find deleted trigger")
}

func TestTrigger_Pause(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Pause Test Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user.ID,
		Status:    models.TriggerStatusActive,
		Config:    models.TriggerConfig{},
	}

	created, err := service.Create(trigger)
	require.NoError(t, err)
	defer cleanupTrigger(t, created.ID)

	paused, err := service.Pause(created.ID)
	require.NoError(t, err, "Failed to pause trigger")
	require.NotNil(t, paused)

	assert.Equal(t, models.TriggerStatusPaused, paused.Status)
}

func TestTrigger_FindAllForUser(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user1 := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user1.ID)

	user2 := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user2.ID)

	t1 := models.Trigger{
		Name:      "User1 Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user1.ID,
		Config:    models.TriggerConfig{},
	}
	created1, err := service.Create(t1)
	require.NoError(t, err)
	defer cleanupTrigger(t, created1.ID)

	t2 := models.Trigger{
		Name:      "User2 Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user2.ID,
		Config:    models.TriggerConfig{},
	}
	created2, err := service.Create(t2)
	require.NoError(t, err)
	defer cleanupTrigger(t, created2.ID)

	triggers, err := service.FindAllForUser(user1.ID)
	require.NoError(t, err)

	for _, trig := range triggers {
		assert.Equal(t, user1.ID, trig.CreatorID, "Should only return user1's triggers")
	}
}

// ============ Validation Tests ============

func TestTrigger_Validate_Database_MissingConfig(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Bad DB Trigger",
		Type:      models.TriggerTypeDatabase,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database trigger requires database configuration")
}

func TestTrigger_Validate_Database_MissingTable(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	metaDBID := uint(1)
	trigger := models.Trigger{
		Name:      "No Table Trigger",
		Type:      models.TriggerTypeDatabase,
		CreatorID: user.ID,
		Config: models.TriggerConfig{
			Database: &models.DatabaseTriggerConfig{
				MetadataDatabaseID: &metaDBID,
				TableName:          "",
				WatermarkColumn:    "id",
				WatermarkType:      models.WatermarkTypeInt,
			},
		},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "table name is required")
}

func TestTrigger_Validate_Database_MissingWatermark(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	metaDBID := uint(1)
	trigger := models.Trigger{
		Name:      "No Watermark Trigger",
		Type:      models.TriggerTypeDatabase,
		CreatorID: user.ID,
		Config: models.TriggerConfig{
			Database: &models.DatabaseTriggerConfig{
				MetadataDatabaseID: &metaDBID,
				TableName:          "orders",
				WatermarkColumn:    "",
				WatermarkType:      models.WatermarkTypeInt,
			},
		},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "watermark column is required")
}

func TestTrigger_Validate_Database_MissingConnection(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "No Connection Trigger",
		Type:      models.TriggerTypeDatabase,
		CreatorID: user.ID,
		Config: models.TriggerConfig{
			Database: &models.DatabaseTriggerConfig{
				TableName:       "orders",
				WatermarkColumn: "id",
				WatermarkType:   models.WatermarkTypeInt,
				// No MetadataDatabaseID and no Connection
			},
		},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database connection is required")
}

func TestTrigger_Validate_Email_MissingConfig(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Bad Email Trigger",
		Type:      models.TriggerTypeEmail,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email trigger requires email configuration")
}

func TestTrigger_Validate_Email_InlineMissingHost(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Email No Host Trigger",
		Type:      models.TriggerTypeEmail,
		CreatorID: user.ID,
		Config: models.TriggerConfig{
			Email: &models.EmailTriggerConfig{
				Host:     "",
				Username: "test",
				Password: "pass",
			},
		},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IMAP host is required")
}

func TestTrigger_Validate_Cron_MissingConfig(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Bad Cron Trigger",
		Type:      models.TriggerTypeCron,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cron trigger requires cron configuration")
}

func TestTrigger_Validate_Cron_InvalidIntervalValue(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Bad Interval Value",
		Type:      models.TriggerTypeCron,
		CreatorID: user.ID,
		Config: models.TriggerConfig{
			Cron: &models.CronTriggerConfig{
				Mode:          models.CronModeInterval,
				IntervalValue: 0,
				IntervalUnit:  models.IntervalUnitMinutes,
			},
		},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "interval value must be greater than 0")
}

func TestTrigger_Validate_Cron_InvalidIntervalUnit(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Bad Interval Unit",
		Type:      models.TriggerTypeCron,
		CreatorID: user.ID,
		Config: models.TriggerConfig{
			Cron: &models.CronTriggerConfig{
				Mode:          models.CronModeInterval,
				IntervalValue: 10,
				IntervalUnit:  "invalid",
			},
		},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "interval unit must be minutes, hours, or days")
}

func TestTrigger_Validate_Cron_InvalidScheduleFrequency(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Bad Schedule Freq",
		Type:      models.TriggerTypeCron,
		CreatorID: user.ID,
		Config: models.TriggerConfig{
			Cron: &models.CronTriggerConfig{
				Mode:              models.CronModeSchedule,
				ScheduleFrequency: "invalid",
				ScheduleTime:      "10:00",
			},
		},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schedule frequency must be daily, weekly, or monthly")
}

func TestTrigger_Validate_Cron_MissingScheduleTime(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "No Schedule Time",
		Type:      models.TriggerTypeCron,
		CreatorID: user.ID,
		Config: models.TriggerConfig{
			Cron: &models.CronTriggerConfig{
				Mode:              models.CronModeSchedule,
				ScheduleFrequency: models.ScheduleFrequencyDaily,
				ScheduleTime:      "",
			},
		},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schedule time is required")
}

func TestTrigger_Validate_Cron_WeeklyMissingDay(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Weekly No Day",
		Type:      models.TriggerTypeCron,
		CreatorID: user.ID,
		Config: models.TriggerConfig{
			Cron: &models.CronTriggerConfig{
				Mode:              models.CronModeSchedule,
				ScheduleFrequency: models.ScheduleFrequencyWeekly,
				ScheduleTime:      "10:00",
				// ScheduleDayOfWeek is nil
			},
		},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "day of week is required")
}

func TestTrigger_Validate_Cron_MonthlyMissingDay(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Monthly No Day",
		Type:      models.TriggerTypeCron,
		CreatorID: user.ID,
		Config: models.TriggerConfig{
			Cron: &models.CronTriggerConfig{
				Mode:               models.CronModeSchedule,
				ScheduleFrequency:  models.ScheduleFrequencyMonthly,
				ScheduleTime:       "10:00",
				// ScheduleDayOfMonth is nil
			},
		},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "day of month is required")
}

func TestTrigger_Validate_InvalidType(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Unknown Type",
		Type:      "unknown",
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}

	_, err := service.Create(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid trigger type")
}

func TestTrigger_Validate_Webhook_NilConfig(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Webhook Default Config",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{}, // Webhook is nil
	}

	created, err := service.Create(trigger)
	require.NoError(t, err, "Webhook with nil config should create default config")
	defer cleanupTrigger(t, created.ID)

	assert.NotNil(t, created.Config.Webhook, "Should have created default webhook config")
}

// ============ Rule Tests ============

func TestTrigger_AddRule(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Rule Test Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}
	created, err := service.Create(trigger)
	require.NoError(t, err)
	defer cleanupTrigger(t, created.ID)

	rule := models.TriggerRule{
		Name: "Test Rule",
		Conditions: models.RuleConditions{
			All: []models.RuleCondition{
				{Field: "status", Operator: models.OperatorEquals, Value: "active"},
			},
		},
	}

	createdRule, err := service.AddRule(created.ID, rule)
	require.NoError(t, err, "Failed to add rule")
	require.NotNil(t, createdRule)
	require.NotZero(t, createdRule.ID)
	assert.Equal(t, created.ID, createdRule.TriggerID)
	assert.Equal(t, "Test Rule", createdRule.Name)
}

func TestTrigger_DeleteRule(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Delete Rule Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}
	created, err := service.Create(trigger)
	require.NoError(t, err)
	defer cleanupTrigger(t, created.ID)

	rule := models.TriggerRule{
		Name: "Delete Me Rule",
		Conditions: models.RuleConditions{
			All: []models.RuleCondition{
				{Field: "x", Operator: models.OperatorEquals, Value: "y"},
			},
		},
	}

	createdRule, err := service.AddRule(created.ID, rule)
	require.NoError(t, err)

	err = service.DeleteRule(createdRule.ID)
	require.NoError(t, err, "Failed to delete rule")
}

// ============ Job Linking Tests ============

func TestTrigger_LinkJob(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()
	jobService := NewJobService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Link Job Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}
	createdTrigger, err := service.Create(trigger)
	require.NoError(t, err)
	defer cleanupTrigger(t, createdTrigger.ID)

	job := models.Job{
		Name:       "Linked Job",
		CreatorID:  user.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	createdJob, err := jobService.Create(job)
	require.NoError(t, err)
	defer cleanupJob(t, createdJob.ID)

	link, err := service.LinkJob(createdTrigger.ID, createdJob.ID, 1, true)
	require.NoError(t, err, "Failed to link job to trigger")
	require.NotNil(t, link)

	assert.Equal(t, createdTrigger.ID, link.TriggerID)
	assert.Equal(t, createdJob.ID, link.JobID)
	assert.Equal(t, 1, link.Priority)
	assert.True(t, link.Active)
	assert.True(t, link.PassEventData)
}

func TestTrigger_UnlinkJob(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()
	jobService := NewJobService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Unlink Job Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}
	createdTrigger, err := service.Create(trigger)
	require.NoError(t, err)
	defer cleanupTrigger(t, createdTrigger.ID)

	job := models.Job{
		Name:       "Unlink Job",
		CreatorID:  user.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	createdJob, err := jobService.Create(job)
	require.NoError(t, err)
	defer cleanupJob(t, createdJob.ID)

	_, err = service.LinkJob(createdTrigger.ID, createdJob.ID, 0, false)
	require.NoError(t, err)

	err = service.UnlinkJob(createdTrigger.ID, createdJob.ID)
	require.NoError(t, err, "Failed to unlink job from trigger")
}

func TestTrigger_LinkJob_TriggerNotFound(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	_, err := service.LinkJob(99999, 1, 0, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trigger not found")
}

func TestTrigger_LinkJob_JobNotFound(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Link Missing Job",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}
	createdTrigger, err := service.Create(trigger)
	require.NoError(t, err)
	defer cleanupTrigger(t, createdTrigger.ID)

	_, err = service.LinkJob(createdTrigger.ID, 99999, 0, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")
}

// ============ Access Control Tests ============

func TestTrigger_CanUserAccess_Creator(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Creator Access Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}
	created, err := service.Create(trigger)
	require.NoError(t, err)
	defer cleanupTrigger(t, created.ID)

	canAccess, err := service.CanUserAccess(created.ID, user.ID)
	require.NoError(t, err)
	assert.True(t, canAccess)
}

func TestTrigger_CanUserAccess_NonCreator(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	owner := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, owner.ID)

	other := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, other.ID)

	trigger := models.Trigger{
		Name:      "Non-Creator Access Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: owner.ID,
		Config:    models.TriggerConfig{},
	}
	created, err := service.Create(trigger)
	require.NoError(t, err)
	defer cleanupTrigger(t, created.ID)

	canAccess, err := service.CanUserAccess(created.ID, other.ID)
	require.NoError(t, err)
	assert.False(t, canAccess)
}

func TestTrigger_GetRecentExecutions(t *testing.T) {
	setupTriggerTestDB(t)

	service := NewTriggerService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	trigger := models.Trigger{
		Name:      "Execution Trigger",
		Type:      models.TriggerTypeWebhook,
		CreatorID: user.ID,
		Config:    models.TriggerConfig{},
	}
	created, err := service.Create(trigger)
	require.NoError(t, err)
	defer cleanupTrigger(t, created.ID)

	// Default limit should be 20
	executions, err := service.GetRecentExecutions(created.ID, 0)
	require.NoError(t, err)
	assert.NotNil(t, executions)

	// Custom limit
	executions, err = service.GetRecentExecutions(created.ID, 5)
	require.NoError(t, err)
	assert.NotNil(t, executions)
}

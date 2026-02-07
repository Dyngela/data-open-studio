package service

import (
	"api"
	"api/internal/api/models"
	"api/internal/api/repo"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

// TriggerPollerService manages polling for all active triggers
type TriggerPollerService struct {
	triggerRepo *repo.TriggerRepository
	jobService  *JobService
	logger      zerolog.Logger

	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	workerPool chan struct{}

	// Configuration
	maxWorkers     int
	dispatchPeriod time.Duration
}

// NewTriggerPollerService creates a new poller service
func NewTriggerPollerService(maxWorkers int) *TriggerPollerService {
	ctx, cancel := context.WithCancel(context.Background())
	return &TriggerPollerService{
		triggerRepo:    repo.NewTriggerRepository(),
		jobService:     NewJobService(),
		logger:         api.Logger,
		ctx:            ctx,
		cancel:         cancel,
		workerPool:     make(chan struct{}, maxWorkers),
		maxWorkers:     maxWorkers,
		dispatchPeriod: 10 * time.Second, // Check for work every 10 seconds
	}
}

// Start begins the polling dispatcher
func (slf *TriggerPollerService) Start() {
	slf.logger.Info().Int("maxWorkers", slf.maxWorkers).Msg("Starting trigger poller service")
	go slf.dispatcher()
}

// Stop gracefully shuts down the poller
func (slf *TriggerPollerService) Stop() {
	slf.logger.Info().Msg("Stopping trigger poller service")
	slf.cancel()
	slf.wg.Wait()
	slf.logger.Info().Msg("Trigger poller service stopped")
}

// dispatcher periodically checks for triggers that need polling
func (slf *TriggerPollerService) dispatcher() {
	defer func() {
		if r := recover(); r != nil {
			slf.logger.Error().Interface("panic", r).Msg("Trigger dispatcher panicked, restarting")
			go slf.dispatcher()
		}
	}()

	// Poll immediately on startup to pick up active triggers
	slf.logger.Info().Msg("Trigger dispatcher running initial poll")
	slf.dispatchWork()

	ticker := time.NewTicker(slf.dispatchPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-slf.ctx.Done():
			return
		case <-ticker.C:
			slf.dispatchWork()
		}
	}
}

// dispatchWork finds triggers ready for polling and dispatches workers
func (slf *TriggerPollerService) dispatchWork() {
	triggers, err := slf.triggerRepo.FindAllActive()
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error fetching active triggers")
		return
	}

	if len(triggers) == 0 {
		slf.logger.Debug().Msg("No active triggers found")
		return
	}

	slf.logger.Debug().Int("count", len(triggers)).Msg("Active triggers found")

	now := time.Now()
	for _, trigger := range triggers {
		// Check if trigger is due for polling
		if !slf.isDueForPolling(trigger, now) {
			continue
		}

		slf.logger.Info().Uint("triggerId", trigger.ID).Str("name", trigger.Name).Msg("Dispatching trigger poll")

		// Try to acquire a worker slot
		select {
		case slf.workerPool <- struct{}{}:
			slf.wg.Add(1)
			go slf.pollTrigger(trigger)
		default:
			slf.logger.Warn().Uint("triggerId", trigger.ID).Msg("Workers busy, skipping trigger")
		}
	}
}

// isDueForPolling checks if a trigger should be polled now
func (slf *TriggerPollerService) isDueForPolling(trigger models.Trigger, now time.Time) bool {
	if trigger.LastPolledAt == nil {
		return true
	}
	nextPoll := trigger.LastPolledAt.Add(time.Duration(trigger.PollingInterval) * time.Second)
	return now.After(nextPoll)
}

// pollTrigger executes polling for a single trigger
func (slf *TriggerPollerService) pollTrigger(trigger models.Trigger) {
	defer func() {
		<-slf.workerPool // Release worker slot
		slf.wg.Done()
	}()

	startTime := time.Now()
	execution := models.TriggerExecution{
		TriggerID: trigger.ID,
		StartedAt: startTime,
		Status:    models.ExecutionStatusRunning,
	}
	_ = slf.triggerRepo.CreateExecution(&execution)

	// Update last polled time
	_ = slf.triggerRepo.UpdateLastPolled(trigger.ID, startTime)

	var events []map[string]interface{}
	var err error

	// Poll based on trigger type
	switch trigger.Type {
	case models.TriggerTypeDatabase:
		events, err = slf.pollDatabase(trigger)
	case models.TriggerTypeEmail:
		events, err = slf.pollEmail(trigger)
	default:
		err = fmt.Errorf("unsupported trigger type: %s", trigger.Type)
	}

	// Update execution record
	execution.FinishedAt = time.Now()
	execution.EventCount = len(events)

	if err != nil {
		execution.Status = models.ExecutionStatusFailed
		execution.Error = err.Error()
		_ = slf.triggerRepo.UpdateStatus(trigger.ID, models.TriggerStatusError, err.Error())
		slf.logger.Error().Err(err).Uint("triggerId", trigger.ID).Msg("Error polling trigger")
	} else if len(events) == 0 {
		execution.Status = models.ExecutionStatusNoEvents
	} else {
		execution.Status = models.ExecutionStatusCompleted

		// Store sample of first event
		if sample, err := json.Marshal(events[0]); err == nil {
			s := string(sample)
			execution.EventSample = &s
		}

		// Process events through rules and trigger jobs
		matchedEvents := slf.filterEventsByRules(events, trigger.Rules)
		if len(matchedEvents) > 0 {
			execution.JobsTriggered = slf.triggerJobs(trigger, matchedEvents)
		}
	}

	_ = slf.triggerRepo.UpdateExecution(&execution)
}

// pollDatabase polls a database for new records
func (slf *TriggerPollerService) pollDatabase(trigger models.Trigger) ([]map[string]interface{}, error) {
	cfg := trigger.Config.Database
	if cfg == nil {
		return nil, fmt.Errorf("no database configuration")
	}

	// Get connection config
	var connCfg models.DBConnectionConfig
	if cfg.MetadataDatabaseID != nil {
		// Load from metadata
		var meta models.MetadataDatabase
		if err := slf.triggerRepo.Db.First(&meta, *cfg.MetadataDatabaseID).Error; err != nil {
			return nil, fmt.Errorf("failed to load database metadata: %w", err)
		}
		connCfg = models.DBConnectionConfig{
			Type:     meta.DbType,
			Host:     meta.Host,
			Port:     meta.Port,
			Database: meta.DatabaseName,
			Username: meta.User,
			Password: meta.Password,
			SSLMode:  meta.SSLMode,
		}
	} else if cfg.Connection != nil {
		connCfg = *cfg.Connection
	} else {
		return nil, fmt.Errorf("no database connection configured")
	}

	// Connect to database
	db, err := sql.Open(connCfg.GetDriverName(), connCfg.BuildConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Set connection timeout
	db.SetConnMaxLifetime(30 * time.Second)
	db.SetMaxOpenConns(1)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Build query
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	// Validate table name to prevent SQL injection
	if !isValidIdentifier(cfg.TableName) {
		return nil, fmt.Errorf("invalid table name: %s", cfg.TableName)
	}
	if !isValidIdentifier(cfg.WatermarkColumn) {
		return nil, fmt.Errorf("invalid watermark column: %s", cfg.WatermarkColumn)
	}

	// Build SELECT clause
	selectClause := "*"
	if len(cfg.SelectColumns) > 0 {
		for _, col := range cfg.SelectColumns {
			if !isValidIdentifier(col) {
				return nil, fmt.Errorf("invalid column name: %s", col)
			}
		}
		selectClause = strings.Join(cfg.SelectColumns, ", ")
		// Ensure watermark column is included
		if !containsColumn(cfg.SelectColumns, cfg.WatermarkColumn) {
			selectClause = cfg.WatermarkColumn + ", " + selectClause
		}
	}

	// Build WHERE clause
	whereClause := fmt.Sprintf("%s > $1", cfg.WatermarkColumn)
	if cfg.WhereClause != "" {
		whereClause = fmt.Sprintf("(%s) AND (%s)", whereClause, cfg.WhereClause)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s ORDER BY %s ASC LIMIT %d",
		selectClause, cfg.TableName, whereClause, cfg.WatermarkColumn, batchSize,
	)

	// Parse last watermark
	lastWatermark := parseWatermark(cfg.LastWatermark, cfg.WatermarkType)

	// Execute query
	rows, err := db.Query(query, lastWatermark)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Scan results into maps
	var events []map[string]interface{}
	var newWatermark interface{}

	for rows.Next() {
		// Create interface slice for scanning
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		// Build event map
		event := make(map[string]interface{})
		for i, col := range cols {
			event[col] = sanitizeValue(values[i])
		}
		events = append(events, event)

		// Track watermark
		if wm, ok := event[cfg.WatermarkColumn]; ok {
			newWatermark = wm
		}
	}

	// Update watermark if we got events
	if newWatermark != nil {
		wmStr := formatWatermark(newWatermark, cfg.WatermarkType)
		_ = slf.triggerRepo.UpdateWatermark(trigger.ID, wmStr)
	}

	return events, nil
}

// pollEmail polls an email inbox for new messages
func (slf *TriggerPollerService) pollEmail(trigger models.Trigger) ([]map[string]interface{}, error) {
	cfg := trigger.Config.Email
	if cfg == nil {
		return nil, fmt.Errorf("no email configuration")
	}

	// TODO: Implement IMAP polling
	// This requires an IMAP client library like github.com/emersion/go-imap
	// For now, return a placeholder error
	return nil, fmt.Errorf("email polling not yet implemented")
}

// filterEventsByRules filters events based on trigger rules
func (slf *TriggerPollerService) filterEventsByRules(events []map[string]interface{}, rules []models.TriggerRule) []map[string]interface{} {
	if len(rules) == 0 {
		return events // No rules means all events match
	}

	var matched []map[string]interface{}
	for _, event := range events {
		if slf.eventMatchesRules(event, rules) {
			matched = append(matched, event)
		}
	}
	return matched
}

// eventMatchesRules checks if an event matches all rules
func (slf *TriggerPollerService) eventMatchesRules(event map[string]interface{}, rules []models.TriggerRule) bool {
	for _, rule := range rules {
		if !slf.eventMatchesRule(event, rule) {
			return false
		}
	}
	return true
}

// eventMatchesRule checks if an event matches a single rule
func (slf *TriggerPollerService) eventMatchesRule(event map[string]interface{}, rule models.TriggerRule) bool {
	conditions := rule.Conditions

	// Check ALL conditions (must all match)
	if len(conditions.All) > 0 {
		for _, cond := range conditions.All {
			if !slf.checkCondition(event, cond) {
				return false
			}
		}
	}

	// Check ANY conditions (at least one must match)
	if len(conditions.Any) > 0 {
		anyMatch := false
		for _, cond := range conditions.Any {
			if slf.checkCondition(event, cond) {
				anyMatch = true
				break
			}
		}
		if !anyMatch {
			return false
		}
	}

	return true
}

// checkCondition checks a single condition against an event
func (slf *TriggerPollerService) checkCondition(event map[string]interface{}, cond models.RuleCondition) bool {
	// Get field value using dot notation
	value := getFieldValue(event, cond.Field)

	switch cond.Operator {
	case models.OperatorEquals:
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", cond.Value)
	case models.OperatorNotEquals:
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", cond.Value)
	case models.OperatorContains:
		return strings.Contains(fmt.Sprintf("%v", value), fmt.Sprintf("%v", cond.Value))
	case models.OperatorStartsWith:
		return strings.HasPrefix(fmt.Sprintf("%v", value), fmt.Sprintf("%v", cond.Value))
	case models.OperatorEndsWith:
		return strings.HasSuffix(fmt.Sprintf("%v", value), fmt.Sprintf("%v", cond.Value))
	case models.OperatorGreaterThan:
		return compareNumbers(value, cond.Value) > 0
	case models.OperatorLessThan:
		return compareNumbers(value, cond.Value) < 0
	case models.OperatorRegex:
		pattern := fmt.Sprintf("%v", cond.Value)
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false
		}
		return re.MatchString(fmt.Sprintf("%v", value))
	case models.OperatorExists:
		return value != nil
	case models.OperatorNotExists:
		return value == nil
	case models.OperatorIn:
		return valueInList(value, cond.Value)
	case models.OperatorNotIn:
		return !valueInList(value, cond.Value)
	default:
		return false
	}
}

// triggerJobs executes linked jobs for matched events
func (slf *TriggerPollerService) triggerJobs(trigger models.Trigger, events []map[string]interface{}) int {
	triggered := 0
	for _, tj := range trigger.Jobs {
		if !tj.Active {
			continue
		}

		// Execute job asynchronously
		go func(jobID uint, passEventData bool, eventData []map[string]interface{}) {
			err := slf.jobService.Execute(jobID)
			if err != nil {
				slf.logger.Error().Err(err).Uint("jobId", jobID).Msg("Failed to execute triggered job")
			}
		}(tj.JobID, tj.PassEventData, events)

		triggered++
	}
	return triggered
}

// Helper functions

func isValidIdentifier(s string) bool {
	if s == "" || len(s) > 128 {
		return false
	}
	// Only allow alphanumeric, underscore, and schema.table format
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`, s)
	return matched
}

func containsColumn(columns []string, target string) bool {
	for _, col := range columns {
		if strings.EqualFold(col, target) {
			return true
		}
	}
	return false
}

func parseWatermark(value string, wmType models.WatermarkType) interface{} {
	if value == "" {
		switch wmType {
		case models.WatermarkTypeInt:
			return 0
		case models.WatermarkTypeTimestamp:
			return time.Time{}
		default:
			return ""
		}
	}

	switch wmType {
	case models.WatermarkTypeInt:
		i, _ := strconv.ParseInt(value, 10, 64)
		return i
	case models.WatermarkTypeTimestamp:
		t, _ := time.Parse(time.RFC3339, value)
		return t
	default:
		return value
	}
}

func formatWatermark(value interface{}, wmType models.WatermarkType) string {
	switch wmType {
	case models.WatermarkTypeInt:
		return fmt.Sprintf("%d", value)
	case models.WatermarkTypeTimestamp:
		if t, ok := value.(time.Time); ok {
			return t.Format(time.RFC3339)
		}
		return fmt.Sprintf("%v", value)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func sanitizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case []byte:
		return string(val)
	case nil:
		return nil
	default:
		return val
	}
}

func getFieldValue(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}
	return current
}

func compareNumbers(a, b interface{}) int {
	aFloat := toFloat64(a)
	bFloat := toFloat64(b)
	if aFloat < bFloat {
		return -1
	} else if aFloat > bFloat {
		return 1
	}
	return 0
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

func valueInList(value, list interface{}) bool {
	valStr := fmt.Sprintf("%v", value)
	switch l := list.(type) {
	case []interface{}:
		for _, item := range l {
			if fmt.Sprintf("%v", item) == valStr {
				return true
			}
		}
	case []string:
		for _, item := range l {
			if item == valStr {
				return true
			}
		}
	}
	return false
}

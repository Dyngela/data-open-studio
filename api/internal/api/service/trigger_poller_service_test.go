package service

import (
	"api/internal/api/models"
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestPollerService creates a minimal TriggerPollerService for unit tests
// that don't need DB access. Only logger and context fields are initialized.
func newTestPollerService() *TriggerPollerService {
	ctx, cancel := context.WithCancel(context.Background())
	return &TriggerPollerService{
		logger:         zerolog.Nop(),
		ctx:            ctx,
		cancel:         cancel,
		workerPool:     make(chan struct{}, 1),
		maxWorkers:     1,
		dispatchPeriod: 10 * time.Second,
	}
}

// ============ Helper Function Tests ============

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid simple", "users", true},
		{"valid with underscore", "user_name", true},
		{"valid schema.table", "public.users", true},
		{"valid starts with underscore", "_hidden", true},
		{"empty string", "", false},
		{"too long", string(make([]byte, 129)), false},
		{"starts with number", "1table", false},
		{"contains spaces", "user name", false},
		{"contains dash", "user-name", false},
		{"sql injection semicolon", "users; DROP TABLE", false},
		{"sql injection quotes", "users'--", false},
		{"double dot", "public.schema.table", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidIdentifier(tt.input))
		})
	}
}

func TestContainsColumn(t *testing.T) {
	columns := []string{"id", "name", "email"}

	assert.True(t, containsColumn(columns, "id"))
	assert.True(t, containsColumn(columns, "name"), "Should be case-insensitive")
	assert.True(t, containsColumn(columns, "NAME"), "Should be case-insensitive")
	assert.False(t, containsColumn(columns, "phone"))
	assert.False(t, containsColumn(nil, "id"))
}

func TestParseWatermark(t *testing.T) {
	// Int type
	result := parseWatermark("42", models.WatermarkTypeInt)
	assert.Equal(t, int64(42), result)

	// Empty int defaults to 0
	result = parseWatermark("", models.WatermarkTypeInt)
	assert.Equal(t, 0, result)

	// Timestamp type
	ts := "2025-01-15T10:30:00Z"
	result = parseWatermark(ts, models.WatermarkTypeTimestamp)
	parsed, ok := result.(time.Time)
	require.True(t, ok)
	assert.Equal(t, 2025, parsed.Year())

	// Empty timestamp defaults to zero time
	result = parseWatermark("", models.WatermarkTypeTimestamp)
	zeroTime, ok := result.(time.Time)
	require.True(t, ok)
	assert.True(t, zeroTime.IsZero())

	// Default type returns string
	result = parseWatermark("abc", "other")
	assert.Equal(t, "abc", result)

	// Empty default returns empty string
	result = parseWatermark("", "other")
	assert.Equal(t, "", result)
}

func TestFormatWatermark(t *testing.T) {
	// Int type
	assert.Equal(t, "42", formatWatermark(42, models.WatermarkTypeInt))

	// Timestamp with time.Time
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	result := formatWatermark(ts, models.WatermarkTypeTimestamp)
	assert.Equal(t, "2025-01-15T10:30:00Z", result)

	// Timestamp with non-time value
	result = formatWatermark("2025-01-15", models.WatermarkTypeTimestamp)
	assert.Equal(t, "2025-01-15", result)

	// Default type
	assert.Equal(t, "hello", formatWatermark("hello", "other"))
}

func TestSanitizeValue(t *testing.T) {
	// []byte → string
	assert.Equal(t, "hello", sanitizeValue([]byte("hello")))

	// nil stays nil
	assert.Nil(t, sanitizeValue(nil))

	// string passes through
	assert.Equal(t, "test", sanitizeValue("test"))

	// int passes through
	assert.Equal(t, 42, sanitizeValue(42))
}

func TestGetFieldValue(t *testing.T) {
	data := map[string]interface{}{
		"name": "John",
		"address": map[string]interface{}{
			"city": "Paris",
		},
	}

	// Simple key
	assert.Equal(t, "John", getFieldValue(data, "name"))

	// Nested dot notation
	assert.Equal(t, "Paris", getFieldValue(data, "address.city"))

	// Missing key
	assert.Nil(t, getFieldValue(data, "missing"))

	// Dot path through non-map
	assert.Nil(t, getFieldValue(data, "name.something"))
}

func TestCompareNumbers(t *testing.T) {
	assert.Equal(t, -1, compareNumbers(1, 2))
	assert.Equal(t, 1, compareNumbers(5, 3))
	assert.Equal(t, 0, compareNumbers(4, 4))
	assert.Equal(t, -1, compareNumbers("1.5", "2.5"))
}

func TestToFloat64(t *testing.T) {
	assert.Equal(t, float64(5), toFloat64(5))
	assert.Equal(t, float64(10), toFloat64(int64(10)))
	assert.Equal(t, 3.14, toFloat64(3.14))
	assert.Equal(t, 2.5, toFloat64("2.5"))
	assert.Equal(t, float64(0), toFloat64(true)) // unsupported → 0
}

func TestValueInList(t *testing.T) {
	// []interface{} list
	list := []interface{}{"apple", "banana", "cherry"}
	assert.True(t, valueInList("banana", list))
	assert.False(t, valueInList("grape", list))

	// []string list
	strList := []string{"one", "two", "three"}
	assert.True(t, valueInList("two", strList))
	assert.False(t, valueInList("four", strList))

	// Empty list
	assert.False(t, valueInList("x", []interface{}{}))

	// Unsupported list type
	assert.False(t, valueInList("x", "not-a-list"))
}

// ============ Rule Engine Tests ============

func TestCheckCondition_Equals(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"status": "active"}

	// Match
	assert.True(t, svc.checkCondition(event, models.RuleCondition{
		Field: "status", Operator: models.OperatorEquals, Value: "active",
	}))
	// No match
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "status", Operator: models.OperatorEquals, Value: "inactive",
	}))
}

func TestCheckCondition_NotEquals(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"status": "active"}

	assert.True(t, svc.checkCondition(event, models.RuleCondition{
		Field: "status", Operator: models.OperatorNotEquals, Value: "inactive",
	}))
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "status", Operator: models.OperatorNotEquals, Value: "active",
	}))
}

func TestCheckCondition_Contains(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"message": "Hello World"}

	assert.True(t, svc.checkCondition(event, models.RuleCondition{
		Field: "message", Operator: models.OperatorContains, Value: "World",
	}))
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "message", Operator: models.OperatorContains, Value: "Goodbye",
	}))
}

func TestCheckCondition_StartsWith(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"path": "/api/v1/users"}

	assert.True(t, svc.checkCondition(event, models.RuleCondition{
		Field: "path", Operator: models.OperatorStartsWith, Value: "/api",
	}))
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "path", Operator: models.OperatorStartsWith, Value: "/web",
	}))
}

func TestCheckCondition_EndsWith(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"file": "report.pdf"}

	assert.True(t, svc.checkCondition(event, models.RuleCondition{
		Field: "file", Operator: models.OperatorEndsWith, Value: ".pdf",
	}))
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "file", Operator: models.OperatorEndsWith, Value: ".csv",
	}))
}

func TestCheckCondition_GreaterThan(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"count": 10}

	assert.True(t, svc.checkCondition(event, models.RuleCondition{
		Field: "count", Operator: models.OperatorGreaterThan, Value: 5,
	}))
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "count", Operator: models.OperatorGreaterThan, Value: 20,
	}))
}

func TestCheckCondition_LessThan(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"count": 3}

	assert.True(t, svc.checkCondition(event, models.RuleCondition{
		Field: "count", Operator: models.OperatorLessThan, Value: 10,
	}))
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "count", Operator: models.OperatorLessThan, Value: 1,
	}))
}

func TestCheckCondition_Regex(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"email": "test@example.com"}

	// Valid pattern match
	assert.True(t, svc.checkCondition(event, models.RuleCondition{
		Field: "email", Operator: models.OperatorRegex, Value: `^test@.*\.com$`,
	}))
	// Valid pattern no match
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "email", Operator: models.OperatorRegex, Value: `^admin@`,
	}))
	// Invalid regex returns false
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "email", Operator: models.OperatorRegex, Value: `[invalid`,
	}))
}

func TestCheckCondition_Exists(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"name": "John", "age": nil}

	// Non-nil value exists
	assert.True(t, svc.checkCondition(event, models.RuleCondition{
		Field: "name", Operator: models.OperatorExists,
	}))
	// nil value does not exist
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "age", Operator: models.OperatorExists,
	}))
	// Missing key does not exist
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "missing", Operator: models.OperatorExists,
	}))
}

func TestCheckCondition_NotExists(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"name": "John"}

	assert.True(t, svc.checkCondition(event, models.RuleCondition{
		Field: "missing", Operator: models.OperatorNotExists,
	}))
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "name", Operator: models.OperatorNotExists,
	}))
}

func TestCheckCondition_In(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"color": "red"}

	assert.True(t, svc.checkCondition(event, models.RuleCondition{
		Field: "color", Operator: models.OperatorIn, Value: []interface{}{"red", "blue", "green"},
	}))
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "color", Operator: models.OperatorIn, Value: []interface{}{"yellow", "purple"},
	}))
}

func TestCheckCondition_NotIn(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"status": "active"}

	assert.True(t, svc.checkCondition(event, models.RuleCondition{
		Field: "status", Operator: models.OperatorNotIn, Value: []interface{}{"deleted", "archived"},
	}))
	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "status", Operator: models.OperatorNotIn, Value: []interface{}{"active", "pending"},
	}))
}

func TestCheckCondition_UnknownOperator(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"x": "y"}

	assert.False(t, svc.checkCondition(event, models.RuleCondition{
		Field: "x", Operator: "unknownOp", Value: "y",
	}))
}

func TestEventMatchesRule_AllConditions(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"status": "active", "count": 5}

	// All match
	rule := models.TriggerRule{
		Conditions: models.RuleConditions{
			All: []models.RuleCondition{
				{Field: "status", Operator: models.OperatorEquals, Value: "active"},
				{Field: "count", Operator: models.OperatorGreaterThan, Value: 3},
			},
		},
	}
	assert.True(t, svc.eventMatchesRule(event, rule))

	// One fails
	rule.Conditions.All[1] = models.RuleCondition{
		Field: "count", Operator: models.OperatorGreaterThan, Value: 10,
	}
	assert.False(t, svc.eventMatchesRule(event, rule))
}

func TestEventMatchesRule_AnyConditions(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"status": "active", "priority": "low"}

	// One matches
	rule := models.TriggerRule{
		Conditions: models.RuleConditions{
			Any: []models.RuleCondition{
				{Field: "status", Operator: models.OperatorEquals, Value: "inactive"},
				{Field: "priority", Operator: models.OperatorEquals, Value: "low"},
			},
		},
	}
	assert.True(t, svc.eventMatchesRule(event, rule))

	// None match
	rule.Conditions.Any = []models.RuleCondition{
		{Field: "status", Operator: models.OperatorEquals, Value: "inactive"},
		{Field: "priority", Operator: models.OperatorEquals, Value: "high"},
	}
	assert.False(t, svc.eventMatchesRule(event, rule))
}

func TestEventMatchesRule_AllAndAny(t *testing.T) {
	svc := newTestPollerService()
	event := map[string]interface{}{"status": "active", "type": "order", "priority": "high"}

	// Both All and Any must pass
	rule := models.TriggerRule{
		Conditions: models.RuleConditions{
			All: []models.RuleCondition{
				{Field: "status", Operator: models.OperatorEquals, Value: "active"},
			},
			Any: []models.RuleCondition{
				{Field: "type", Operator: models.OperatorEquals, Value: "order"},
				{Field: "type", Operator: models.OperatorEquals, Value: "invoice"},
			},
		},
	}
	assert.True(t, svc.eventMatchesRule(event, rule))

	// All passes but Any fails
	rule.Conditions.Any = []models.RuleCondition{
		{Field: "type", Operator: models.OperatorEquals, Value: "payment"},
	}
	assert.False(t, svc.eventMatchesRule(event, rule))
}

func TestFilterEventsByRules_NoRules(t *testing.T) {
	svc := newTestPollerService()

	events := []map[string]interface{}{
		{"a": 1}, {"b": 2},
	}

	result := svc.filterEventsByRules(events, nil)
	assert.Len(t, result, 2, "No rules should return all events")
}

func TestFilterEventsByRules_WithRules(t *testing.T) {
	svc := newTestPollerService()

	events := []map[string]interface{}{
		{"status": "active", "name": "A"},
		{"status": "inactive", "name": "B"},
		{"status": "active", "name": "C"},
	}

	rules := []models.TriggerRule{
		{
			Conditions: models.RuleConditions{
				All: []models.RuleCondition{
					{Field: "status", Operator: models.OperatorEquals, Value: "active"},
				},
			},
		},
	}

	result := svc.filterEventsByRules(events, rules)
	assert.Len(t, result, 2, "Should filter to only active events")
	assert.Equal(t, "A", result[0]["name"])
	assert.Equal(t, "C", result[1]["name"])
}

// ============ Scheduling Logic Tests ============

func TestIsDueForPolling_NeverPolled(t *testing.T) {
	svc := newTestPollerService()

	trigger := models.Trigger{
		LastPolledAt:    nil,
		PollingInterval: 60,
		Type:            models.TriggerTypeDatabase,
	}

	assert.True(t, svc.isDueForPolling(trigger, time.Now()), "Never-polled trigger should be due")
}

func TestIsDueForPolling_NotYetDue(t *testing.T) {
	svc := newTestPollerService()

	now := time.Now()
	lastPolled := now.Add(-30 * time.Second) // 30 seconds ago
	trigger := models.Trigger{
		LastPolledAt:    &lastPolled,
		PollingInterval: 60, // 60 second interval
		Type:            models.TriggerTypeDatabase,
	}

	assert.False(t, svc.isDueForPolling(trigger, now), "Should not be due yet")
}

func TestIsDueForPolling_Due(t *testing.T) {
	svc := newTestPollerService()

	now := time.Now()
	lastPolled := now.Add(-120 * time.Second) // 2 minutes ago
	trigger := models.Trigger{
		LastPolledAt:    &lastPolled,
		PollingInterval: 60, // 60 second interval
		Type:            models.TriggerTypeDatabase,
	}

	assert.True(t, svc.isDueForPolling(trigger, now), "Should be due")
}

func TestIsDueForPolling_CronInterval(t *testing.T) {
	svc := newTestPollerService()

	now := time.Now()
	lastPolled := now.Add(-35 * time.Minute) // 35 minutes ago
	trigger := models.Trigger{
		LastPolledAt: &lastPolled,
		Type:         models.TriggerTypeCron,
		Config: models.TriggerConfig{
			Cron: &models.CronTriggerConfig{
				Mode:          models.CronModeInterval,
				IntervalValue: 30,
				IntervalUnit:  models.IntervalUnitMinutes,
			},
		},
	}

	assert.True(t, svc.isDueForPolling(trigger, now), "Should be due after 35min with 30min interval")

	// Not yet due
	lastPolled = now.Add(-20 * time.Minute)
	trigger.LastPolledAt = &lastPolled
	assert.False(t, svc.isDueForPolling(trigger, now), "Should not be due after 20min with 30min interval")
}

func TestIsDueForPolling_CronSchedule_NeverPolled(t *testing.T) {
	svc := newTestPollerService()

	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC) // 14:00
	trigger := models.Trigger{
		LastPolledAt: nil,
		Type:         models.TriggerTypeCron,
		Config: models.TriggerConfig{
			Cron: &models.CronTriggerConfig{
				Mode:              models.CronModeSchedule,
				ScheduleFrequency: models.ScheduleFrequencyDaily,
				ScheduleTime:      "10:00",
			},
		},
	}

	// Scheduled at 10:00, now is 14:00 — should be due (first run, time has passed)
	assert.True(t, svc.isDueForPolling(trigger, now))
}

func TestCronIntervalDuration_Minutes(t *testing.T) {
	svc := newTestPollerService()

	cfg := &models.CronTriggerConfig{IntervalValue: 15, IntervalUnit: models.IntervalUnitMinutes}
	assert.Equal(t, 15*time.Minute, svc.cronIntervalDuration(cfg))
}

func TestCronIntervalDuration_Hours(t *testing.T) {
	svc := newTestPollerService()

	cfg := &models.CronTriggerConfig{IntervalValue: 2, IntervalUnit: models.IntervalUnitHours}
	assert.Equal(t, 2*time.Hour, svc.cronIntervalDuration(cfg))
}

func TestCronIntervalDuration_Days(t *testing.T) {
	svc := newTestPollerService()

	cfg := &models.CronTriggerConfig{IntervalValue: 3, IntervalUnit: models.IntervalUnitDays}
	assert.Equal(t, 3*24*time.Hour, svc.cronIntervalDuration(cfg))
}

func TestCronIntervalDuration_Default(t *testing.T) {
	svc := newTestPollerService()

	cfg := &models.CronTriggerConfig{IntervalValue: 5, IntervalUnit: "unknown"}
	assert.Equal(t, 5*time.Minute, svc.cronIntervalDuration(cfg), "Unknown unit defaults to minutes")
}

func TestComputeNextScheduledTime_Daily(t *testing.T) {
	svc := newTestPollerService()

	now := time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC) // 09:00

	cfg := &models.CronTriggerConfig{
		Mode:              models.CronModeSchedule,
		ScheduleFrequency: models.ScheduleFrequencyDaily,
		ScheduleTime:      "10:00",
	}

	// Not fired today — should be today at 10:00
	result := svc.computeNextScheduledTime(time.Time{}, cfg, now)
	assert.Equal(t, 10, result.Hour())
	assert.Equal(t, 15, result.Day())

	// Already fired today — should be tomorrow
	lastPolled := time.Date(2025, 6, 15, 10, 5, 0, 0, time.UTC)
	result = svc.computeNextScheduledTime(lastPolled, cfg, now)
	assert.Equal(t, 16, result.Day())
}

func TestComputeNextScheduledTime_Weekly(t *testing.T) {
	svc := newTestPollerService()

	// June 15, 2025 is a Sunday (weekday 0)
	now := time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC)

	wednesday := 3
	cfg := &models.CronTriggerConfig{
		Mode:              models.CronModeSchedule,
		ScheduleFrequency: models.ScheduleFrequencyWeekly,
		ScheduleTime:      "08:00",
		ScheduleDayOfWeek: &wednesday,
	}

	// Should be Wednesday June 18
	result := svc.computeNextScheduledTime(time.Time{}, cfg, now)
	assert.Equal(t, time.Wednesday, result.Weekday())
	assert.Equal(t, 18, result.Day())
}

func TestComputeNextScheduledTime_Monthly(t *testing.T) {
	svc := newTestPollerService()

	now := time.Date(2025, 6, 10, 9, 0, 0, 0, time.UTC)

	day15 := 15
	cfg := &models.CronTriggerConfig{
		Mode:               models.CronModeSchedule,
		ScheduleFrequency:  models.ScheduleFrequencyMonthly,
		ScheduleTime:       "12:00",
		ScheduleDayOfMonth: &day15,
	}

	result := svc.computeNextScheduledTime(time.Time{}, cfg, now)
	assert.Equal(t, 15, result.Day())
	assert.Equal(t, time.June, result.Month())
}

func TestComputeNextScheduledTime_WeeklyMissingDay(t *testing.T) {
	svc := newTestPollerService()

	now := time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC)

	cfg := &models.CronTriggerConfig{
		Mode:              models.CronModeSchedule,
		ScheduleFrequency: models.ScheduleFrequencyWeekly,
		ScheduleTime:      "08:00",
		// ScheduleDayOfWeek is nil
	}

	result := svc.computeNextScheduledTime(time.Time{}, cfg, now)
	assert.True(t, result.IsZero(), "Should return zero time when day of week is missing")
}

func TestParseScheduleTime(t *testing.T) {
	svc := newTestPollerService()

	hour, minute := svc.parseScheduleTime("14:30")
	assert.Equal(t, 14, hour)
	assert.Equal(t, 30, minute)

	hour, minute = svc.parseScheduleTime("00:00")
	assert.Equal(t, 0, hour)
	assert.Equal(t, 0, minute)
}

// ============ Cron Poll Tests ============

func TestPollCron_Interval(t *testing.T) {
	svc := newTestPollerService()

	trigger := models.Trigger{
		ID:   1,
		Type: models.TriggerTypeCron,
		Config: models.TriggerConfig{
			Cron: &models.CronTriggerConfig{
				Mode:          models.CronModeInterval,
				IntervalValue: 15,
				IntervalUnit:  models.IntervalUnitMinutes,
			},
		},
	}

	events, err := svc.pollCron(trigger)
	require.NoError(t, err)
	require.Len(t, events, 1)

	assert.Equal(t, "cron", events[0]["type"])
	assert.Equal(t, "interval", events[0]["mode"])
	assert.Equal(t, 15, events[0]["intervalValue"])
	assert.Equal(t, "minutes", events[0]["intervalUnit"])
	assert.NotEmpty(t, events[0]["timestamp"])
}

func TestPollCron_Schedule(t *testing.T) {
	svc := newTestPollerService()

	trigger := models.Trigger{
		ID:   2,
		Type: models.TriggerTypeCron,
		Config: models.TriggerConfig{
			Cron: &models.CronTriggerConfig{
				Mode:              models.CronModeSchedule,
				ScheduleFrequency: models.ScheduleFrequencyDaily,
				ScheduleTime:      "09:00",
			},
		},
	}

	events, err := svc.pollCron(trigger)
	require.NoError(t, err)
	require.Len(t, events, 1)

	assert.Equal(t, "cron", events[0]["type"])
	assert.Equal(t, "schedule", events[0]["mode"])
	assert.Equal(t, "daily", events[0]["scheduleFrequency"])
	assert.Equal(t, "09:00", events[0]["scheduleTime"])
}

func TestPollCron_NilConfig(t *testing.T) {
	svc := newTestPollerService()

	trigger := models.Trigger{
		ID:     3,
		Type:   models.TriggerTypeCron,
		Config: models.TriggerConfig{},
	}

	_, err := svc.pollCron(trigger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no cron configuration")
}

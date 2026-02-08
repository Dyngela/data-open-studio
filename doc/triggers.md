# Trigger System

Triggers automate job execution by polling data sources for changes. Located in `api/internal/api/service/trigger_service.go` and `trigger_poller_service.go`.

## Trigger Types

| Type | Source | Polling Method | Watermark |
|------|--------|---------------|-----------|
| `database` | SQL table | Query rows > last watermark | int / timestamp / uuid column |
| `email` | IMAP inbox | Search UID > last UID | IMAP UID (uint32) |
| `webhook` | HTTP endpoint | (not polled, push-based) | N/A |

## Data Model (`models/triggers.go`)

### Trigger
```go
type Trigger struct {
    ID              uint
    Name            string
    Description     string
    Type            TriggerType       // "database" | "email" | "webhook"
    Status          TriggerStatus     // "active" | "paused" | "error" | "disabled"
    CreatorID       uint
    PollingInterval int               // seconds, default: 60
    LastPolledAt    *time.Time
    LastError       string
    Config          TriggerConfig     // JSONB - polymorphic
    Rules           []TriggerRule     // HasMany
    Jobs            []TriggerJob      // HasMany
}
```

### TriggerConfig (JSONB)
```go
type TriggerConfig struct {
    Database *DatabaseTriggerConfig
    Email    *EmailTriggerConfig
    Webhook  *WebhookTriggerConfig
}
```

### DatabaseTriggerConfig
```go
type DatabaseTriggerConfig struct {
    MetadataDatabaseID *uint               // Reference to saved connection
    Connection         *DBConnectionConfig // OR inline connection
    TableName          string              // required
    WatermarkColumn    string              // required
    WatermarkType      WatermarkType       // "int" | "timestamp" | "uuid"
    LastWatermark      string              // Current position (updated by poller)
    SelectColumns      []string            // Optional column filter
    WhereClause        string              // Optional WHERE filter
    BatchSize          int                 // Default: 100
}
```

### EmailTriggerConfig
```go
type EmailTriggerConfig struct {
    MetadataEmailID *uint    // Reference to saved connection
    // OR inline credentials:
    Host            string
    Port            int
    Username        string
    Password        string
    UseTLS          bool

    Folder          string   // Default: "INBOX"
    FromAddress     string   // Filter: sender email
    ToAddress       string   // Filter: recipient email
    SubjectPattern  string   // Filter: regex pattern
    HasAttachment   *bool    // Filter: has attachment
    CCAddresses     []string // Filter: CC contains
    LastUID         uint32   // Current position (updated by poller)
    MarkAsRead      bool     // Mark processed emails as read
}
```

### TriggerRule
```go
type TriggerRule struct {
    ID         uint
    TriggerID  uint
    Name       string
    Conditions RuleConditions // JSONB
}

type RuleConditions struct {
    All []RuleCondition   // AND logic
    Any []RuleCondition   // OR logic
}

type RuleCondition struct {
    Field    string             // Dot-notation path: "email.subject", "payload.status"
    Operator ConditionOperator  // eq | neq | contains | startsWith | endsWith | gt | lt | regex | in | notIn | exists | notExists
    Value    interface{}
}
```

### TriggerJob (junction)
```go
type TriggerJob struct {
    ID            uint
    TriggerID     uint
    JobID         uint
    Priority      int    // Lower = higher priority, default: 0
    Active        bool   // Default: true
    PassEventData bool   // Send event data to job
}
```

### TriggerExecution (audit)
```go
type TriggerExecution struct {
    ID            uint
    TriggerID     uint
    StartedAt     time.Time
    FinishedAt    *time.Time
    Status        ExecutionStatus  // "running" | "completed" | "failed" | "no_events"
    EventCount    int
    JobsTriggered int
    Error         string
    EventSample   *string          // First event as JSONB
}
```

## Trigger Lifecycle

### Creation
1. User creates trigger via API with config
2. `TriggerService.Create()` validates config
3. Status set to `paused`

### Activation
1. User calls `POST /triggers/:id/activate`
2. `TriggerService.Activate()`:
   - For database: `initializeWatermark()` - queries `MAX(watermarkColumn)` to set starting position
   - For email: `initializeEmailUID()` - queries max UID in folder to set starting position
3. Status set to `active`
4. Poller picks it up on next dispatch cycle

### Polling (Active)
See Poller Service below.

### Pause
1. User calls `POST /triggers/:id/pause`
2. Status set to `paused`
3. Poller stops polling this trigger

## Poller Service (`trigger_poller_service.go`)

### Architecture
```
TriggerPollerService
    |
    +-- dispatcher goroutine (runs every 10s)
    |       |
    |       +-- dispatchWork()
    |               |
    |               +-- For each due trigger:
    |                       |
    |                       +-- worker goroutine (bounded by maxWorkers)
    |                               |
    |                               +-- pollTrigger()
    |                                       |
    |                                       +-- pollDatabase() OR pollEmail()
    |                                       +-- filterEventsByRules()
    |                                       +-- triggerJobs()
```

### Configuration
```go
type TriggerPollerService struct {
    maxWorkers     int            // Default: 10
    dispatchPeriod time.Duration  // Default: 10 seconds
    workerPool     chan struct{}  // Semaphore for worker limit
}
```

### Dispatch Cycle
1. `dispatcher()` runs in a goroutine, loops every `dispatchPeriod`
2. `dispatchWork()` fetches all active triggers
3. For each trigger: check if `isDueForPolling(trigger, now)`
   - Due if: `LastPolledAt == nil` OR `now >= LastPolledAt + PollingInterval`
4. Acquire worker slot from `workerPool` channel
5. Launch `pollTrigger()` in goroutine

### Database Polling (`pollDatabase`)
1. Resolve connection (from MetadataID or inline config)
2. Build query: `SELECT {columns} FROM {table} WHERE {watermarkColumn} > {lastWatermark} ORDER BY {watermarkColumn} LIMIT {batchSize}`
3. Execute query against source database
4. Convert rows to `[]map[string]interface{}`
5. Update `lastWatermark` to max value from results (JSONB update in DB)
6. Return events

### Email Polling (`pollEmail`)
1. Resolve IMAP credentials (from MetadataID or inline)
2. Connect to IMAP server (with TLS if configured)
3. Select folder (default: INBOX)
4. Search: `UID > lastUID`
5. Fetch matching messages (envelope + body)
6. For each message:
   - Extract: Subject, From, To, CC, Date, Body, HasAttachment
   - Apply filters: `emailMatchesFilters()`:
     - FromAddress: exact match on sender
     - ToAddress: exact match on recipient
     - SubjectPattern: regex match on subject
     - HasAttachment: boolean check
     - CCAddresses: checks CC contains any
   - Convert to event map
7. Update `lastUID` to max UID from results
8. Optionally mark as read (`\Seen` flag via IMAP STORE)
9. Return events

### Rule Filtering (`filterEventsByRules`)
Applied after polling, before triggering jobs.

```
For each event:
    For each rule:
        Match ALL conditions (AND) AND ANY conditions (OR)
        If rule matches -> keep event
    If no rules defined -> keep all events
```

Condition matching (`checkCondition`):
| Operator | Logic |
|----------|-------|
| eq | `value == condValue` |
| neq | `value != condValue` |
| contains | `strings.Contains(value, condValue)` |
| startsWith | `strings.HasPrefix(value, condValue)` |
| endsWith | `strings.HasSuffix(value, condValue)` |
| gt | `compareNumbers(value, condValue) > 0` |
| lt | `compareNumbers(value, condValue) < 0` |
| regex | `regexp.MatchString(condValue, value)` |
| in | `valueInList(value, condValue)` |
| notIn | `!valueInList(value, condValue)` |
| exists | `value != nil` |
| notExists | `value == nil` |

Field paths use dot notation: `getFieldValue("email.subject", event)` traverses nested maps.

### Job Triggering (`triggerJobs`)
1. Get linked jobs (active, sorted by priority)
2. For each job: call `jobService.Execute(job.JobID)` (async)
3. Return count of triggered jobs

### Execution Tracking
Each poll creates a `TriggerExecution` record:
- Status: `running` -> `completed` / `failed` / `no_events`
- Tracks: eventCount, jobsTriggered, error, eventSample (first event)

## Error Handling

- If poll fails: trigger status set to `error`, lastError updated
- If poll succeeds with 0 events: execution recorded as `no_events`
- Dispatcher uses `recover()` to handle panics
- Individual worker failures don't crash the poller

## Repository JSONB Updates

Watermark and email UID are updated via PostgreSQL JSONB path operations:
```go
// UpdateWatermark
DB.Exec(`UPDATE trigger SET config = jsonb_set(config, '{database,lastWatermark}', $1) WHERE id = $2`)

// UpdateEmailUID
DB.Exec(`UPDATE trigger SET config = jsonb_set(config, '{email,lastUid}', $1) WHERE id = $2`)
```

## Frontend Integration

### Create Trigger Wizard (4 steps)
1. **Type**: Name, description, type selection (database/email/webhook), polling interval
2. **Configuration**:
   - Database: Select saved connection -> load tables -> select table + watermark column + type + batch size
   - Email: Select saved email connection -> configure folder + filters (from, to, subject pattern, mark as read)
3. **Jobs**: Select jobs to link
4. **Summary**: Review all settings

### Sidebar Details (4 tabs)
1. **Configuration**: Shows all config fields
2. **Rules**: Add/edit/delete filter rules
3. **Jobs**: Link/unlink jobs
4. **History**: Recent execution records with status

### Key Signals in Triggers Component
```typescript
selectedDbConnection = signal<DbMetadata | null>(null)
selectedEmailConnection = signal<EmailMetadata | null>(null)
triggerForm: FormGroup       // name, description, type, pollingInterval
tableForm: FormGroup         // tableName, watermarkColumn, watermarkType, batchSize
emailForm: FormGroup         // folder, fromAddress, toAddress, subjectPattern, markAsRead
```

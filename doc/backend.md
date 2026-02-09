# Backend Systems

All backend code is in `api/`. Entry point: `api/cmd/main.go`.

## Layered Architecture

```
HTTP Request
    |
    v
Handler (endpoints/)    -- Route handling, request parsing, response mapping
    |
    v
Service (service/)      -- Business logic, validation, transactions
    |
    v
Repository (repo/)      -- Thin GORM wrappers
    |
    v
Model (models/)         -- GORM database entities
```

## Models (`internal/api/models/`)

### User (`user.go`)
```go
type User struct {
    ID           uint          // primaryKey
    Email        string        // uniqueIndex, required
    Password     string        // required (bcrypt hashed)
    Prenom       string        // required (first name)
    Nom          string        // required (last name)
    Role         AppRole       // "user" | "admin", default: "user"
    Actif        bool          // default: true
    RefreshToken string        // text type
    CreatedAt    time.Time
    UpdatedAt    time.Time
    DeletedAt    gorm.DeletedAt // soft delete, indexed
}
// Table: "users"
```

### Metadata Domain

**MetadataDatabase** (`metadata.go`):
| Field | Type | Notes |
|-------|------|-------|
| ID | uint | primaryKey |
| Host | string | |
| Port | int | |
| User | string | |
| Password | string | |
| DatabaseName | string | |
| SSLMode | string | |
| DbType | DBType | `postgres` / `sqlserver` / `mysql` |
| Extra | string | |

**MetadataSftp** (`metadata.go`):
| Field | Type | Notes |
|-------|------|-------|
| ID, Host, Port, User, Password | - | Standard |
| PrivateKey | string | SSH key |
| BasePath | string | Root directory |
| Extra | string | |

**MetadataEmail** (`metadata_email.go`):
| Field | Type | Notes |
|-------|------|-------|
| ID | uint | primaryKey |
| Name | string | Display name |
| ImapHost | string | IMAP server |
| ImapPort | int | Default: 993 |
| SmtpHost | string | SMTP server |
| SmtpPort | int | Default: 587 |
| Username | string | |
| Password | string | |
| UseTLS | bool | Default: true |
| Extra | string | |

### Job Domain (`jobs.go`)

**Job**:
| Field | Type | Notes |
|-------|------|-------|
| ID | uint | primaryKey |
| Name | string | |
| Description | string | |
| FilePath | string | Virtual path (folder structure) |
| CreatorID | uint | FK to User |
| Active | bool | |
| Visibility | JobVisibility | `public` / `private` |
| OutputPath | string | Generated code output |
| Nodes | []Node | HasMany, FK: JobID |
| SharedWith | []User | Many2Many via job_user_access |

**JobUserAccess** (junction):
| Field | Type | Notes |
|-------|------|-------|
| JobID | uint | composite PK |
| UserID | uint | composite PK |
| Role | OwningJob | `owner` / `editor` / `viewer` |

### Node Domain (`nodes.go`, `port.go`)

**Node**:
| Field | Type | Notes |
|-------|------|-------|
| ID | uint | primaryKey |
| Type | NodeType | `start` / `db_input` / `db_output` / `map` / `log` / `email_output` |
| Name | string | |
| Xpos | float64 | Canvas X position |
| Ypos | float64 | Canvas Y position |
| Data | NodeData | JSONB - polymorphic config |
| JobID | uint | FK, indexed |
| InputPort | []Port | HasMany |
| OutputPort | []Port | HasMany |

**Port**:
| Field | Type | Notes |
|-------|------|-------|
| ID | uint | |
| Type | PortType | `input` / `output` / `node_flow_input` / `node_flow_output` |
| NodeID | uint | FK |
| ConnectedNodeID | *uint | FK to connected Port |

**Node helper methods**: `GetDBInputConfig()`, `GetDBOutputConfig()`, `GetMapConfig()`, `GetLogConfig()`, `GetEmailOutputConfig()`, `GetNextFlowNodeIDs()`, `GetPrevFlowNodeIDs()`, `GetDataInputNodeIDs()`, `GetDataOutputNodeIDs()`.

### Node Config Models

**DBInputConfig** (`node_db_input_config.go`):
- Query, DbSchema, QueryWithSchema, BatchSize
- Connection (DBConnectionConfig)
- DataModels ([]DataModel - column schema)
- Methods: `Validate()`, `EnforceSchema()`, `FillDataModels()` (executes query to detect types)

**DBOutputConfig** (`node_db_output_config.go`):
- Table, Mode (`insert` / `update` / `merge` / `delete` / `truncate`)
- BatchSize, DbSchema, Connection, DataModels
- Methods: `FillDataModels()`

**MapConfig** (`node_map_config.go`):
- Inputs ([]InputFlow), Outputs ([]OutputFlow)
- Join (*JoinConfig) - join type: `inner` / `left` / `right` / `full` / `cross` / `union`
- Methods: `GetInputByName()`, `GetOutputByName()`, `HasMultipleInputs()`

**EmailOutputConfig** (`node_email_output_config.go`):
- MetadataEmailID (*uint) or inline SMTP (SmtpHost, SmtpPort, Username, Password, UseTLS)
- To, CC, BCC ([]string), Subject, Body, IsHTML

**DBConnectionConfig** (`db_conn_config.go`):
- Type (DBType), Host, Port, Database, Username, Password, SSLMode, Extra, DSN
- Methods: `BuildConnectionString()`, `GetDriverName()`, `GetImportPath()`

**DataModel** (`db_data_model.go`):
- Name, Type, GoType, Nullable, Length, Precision, Scale
- Methods: `GoFieldName()`, `GoFieldType()`, `GoScanType()`

### Trigger Domain (`triggers.go`)

See [triggers.md](triggers.md) for full details.

## Repositories (`internal/api/repo/`)

All repos are thin GORM wrappers with a `Db *gorm.DB` field.

### UserRepository (`user_repo.go`)
```
FindByEmail(email) -> (User, error)
FindByID(id) -> (User, error)
Create(user) -> error
Update(user) -> error
Delete(id) -> error
ExistsByEmail(email) -> (bool, error)
GetAll() -> ([]User, error)
```

### JobRepository (`job_repo.go`)
```
FindByID(id) -> (Job, error)    // Preloads: Nodes, InputPort, OutputPort
```

### TriggerRepository (`trigger_repo.go`)
```
FindByID(id) -> (Trigger, error)                    // Full preload
FindByIDSimple(id) -> (Trigger, error)               // No preload
FindAllByCreator(creatorID) -> ([]Trigger, error)
FindAllActive() -> ([]Trigger, error)
FindActiveByType(type) -> ([]Trigger, error)
Create(trigger) / Update(trigger) / Delete(id)
UpdateStatus(id, status, lastError)
UpdateLastPolled(id, lastPolledAt)
UpdateWatermark(id, watermark)                        // JSONB path update
UpdateEmailUID(id, uid)                               // JSONB path update
AddRule / UpdateRule / DeleteRule
AddJob / RemoveJob / UpdateJobLink
CreateExecution / UpdateExecution / GetRecentExecutions
```

### MetadataRepository, SftpMetadataRepository, EmailMetadataRepository
Minimal wrappers - services use GORM directly through them.

## Services (`internal/api/service/`)

### UserService
- `Register(dto) -> AuthResponse` - Hash password, create user, generate tokens
- `Login(dto) -> AuthResponse` - Validate credentials, generate tokens
- `GetByID(id) -> UserResponse`
- `RefreshToken(refreshToken) -> AuthResponse`

### MetadataService / SftpMetadataService / EmailMetadataService
All follow the same CRUD pattern:
```
FindAll() -> ([]Model, error)
FindByID(id) -> (*Model, error)
Create(model) -> (*Model, error)
Update(id, patch map[string]any) -> (*Model, error)
Delete(id) -> error
```

### JobService
- CRUD: `FindAllForUser`, `FindByID`, `Create`, `Update`, `UpdateWithNodes` (transactional), `Delete`
- Access control: `CanUserAccess`, `ShareJob`, `UnshareJob`, `GetJobAccess`
- Execution: `Execute(id)` (async via gen.JobExecution), `Stop(id)`, `PrintCode(id)`
- Notification: `notifyJobDone(jobID, err)` via NATS

### TriggerService
- CRUD + lifecycle: `Create`, `Update`, `Delete`, `Activate`, `Pause`
- Rules: `AddRule`, `UpdateRule`, `DeleteRule`
- Jobs: `LinkJob`, `UnlinkJob`
- History: `GetRecentExecutions`
- Internal: `validateTriggerConfig`, `initializeWatermark`, `initializeEmailUID`, `resolveConnection`

### TriggerPollerService
Background service - see [triggers.md](triggers.md).

### MailService
```
SendWithMetadata(metadataID, msg) -> error     // Load creds from DB
SendWithInline(host, port, user, pass, tls, msg) -> error
TestSMTPConnection(host, port, user, pass, tls) -> error
TestIMAPConnection(host, port, user, pass, tls) -> error
```
Uses `wneessen/go-mail` for SMTP, `emersion/go-imap/v2` for IMAP.

## Handlers/Endpoints (`internal/api/handler/endpoints/`)

### Auth Routes (`/api/v1/auth`)
| Method | Path | Handler | Auth |
|--------|------|---------|------|
| POST | /auth/register | register | No |
| POST | /auth/login | login | No |
| POST | /auth/refresh | refreshToken | No |
| GET | /me | getMe | Yes |

### Metadata Routes (`/api/v1/metadata`)
All routes require AuthMiddleware.

| Method | Path | Handler |
|--------|------|---------|
| GET | /db | getAll |
| GET | /db/:id | getByID |
| POST | /db | create |
| PUT | /db/:id | update |
| DELETE | /db/:id | delete |
| POST | /db/test-connection | testConnection |
| GET | /sftp | getAll |
| GET | /sftp/:id | getByID |
| POST | /sftp | create |
| PUT | /sftp/:id | update |
| DELETE | /sftp/:id | delete |
| GET | /email | getAll |
| GET | /email/:id | getByID |
| POST | /email | create |
| PUT | /email/:id | update |
| DELETE | /email/:id | delete |
| POST | /email/test-connection | testConnection |

### Job Routes (`/api/v1/jobs`)
| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | /jobs | getAll | Optional `?filePath=` filter |
| GET | /jobs/:id | getByID | |
| POST | /jobs | create | |
| PUT | /jobs/:id | update | |
| DELETE | /jobs/:id | delete | |
| POST | /jobs/:id/share | share | |
| DELETE | /jobs/:id/share | unshare | |
| POST | /jobs/:id/execute | execute | Async |
| POST | /jobs/:id/stop | stop | |
| POST | /jobs/:id/print-code | printCode | Returns generated Go source |

### Trigger Routes (`/api/v1/triggers`)
| Method | Path | Handler |
|--------|------|---------|
| GET | /triggers | getAll |
| GET | /triggers/:id | getByID |
| POST | /triggers | create |
| PUT | /triggers/:id | update |
| DELETE | /triggers/:id | delete |
| POST | /triggers/:id/activate | activate |
| POST | /triggers/:id/pause | pause |
| POST | /triggers/:id/rules | addRule |
| PUT | /triggers/:id/rules/:ruleId | updateRule |
| DELETE | /triggers/:id/rules/:ruleId | deleteRule |
| POST | /triggers/:id/jobs | linkJob |
| DELETE | /triggers/:id/jobs/:jobId | unlinkJob |
| GET | /triggers/:id/executions | getExecutions |

### SQL Routes (`/api/v1/sql`)
| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| POST | /guess-query | guessQuery | AI-powered query generation |
| POST | /optimize-query | optimizeQuery | AI-powered optimization |
| POST | /introspect/test-connection | testConnection | |
| POST | /introspect/tables | getTables | |
| POST | /introspect/columns | getColumns | |

### DB Node Routes (`/api/v1/db-node`)
| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| POST | /guess-schema | guessSchema | Execute query to detect column types |

## Middleware (`internal/api/handler/middleware/`)

### AuthMiddleware
- Extracts `Authorization: Bearer <token>` header
- Validates JWT with `pkg.ValidateToken()`
- Sets context keys: `userID`, `userEmail`, `userRole`, `username`
- Returns 401 on missing/invalid token

### RequireRole
- Checks `userRole` from context against allowed roles
- Returns 403 if insufficient

## Mapper System (`internal/api/handler/mapper/`)

### Pattern
1. Define interface in `mapper/<name>.go` with mapping methods
2. Add `//go:generate` directive for `dtomapper` tool
3. Run `go generate` to produce `<name>mapper.impl.generated.go`
4. Override generated methods manually where needed

### MetadataMapper (`metadata.go`)
```go
type MetadataMapper interface {
    // DB Metadata
    ToMetadataResponses([]MetadataDatabase) []response.Metadata
    ToMetadataResponse(MetadataDatabase) response.Metadata
    CreateDbMetadata(request.CreateMetadata) MetadataDatabase
    PatchDbMetadata(request.UpdateMetadata) map[string]any

    // SFTP Metadata
    ToSftpMetadataResponses([]MetadataSftp) []response.SftpMetadata
    ToSftpMetadataResponse(MetadataSftp) response.SftpMetadata
    CreateSftpMetadata(request.CreateSftpMetadata) MetadataSftp
    PatchSftpMetadata(request.UpdateSftpMetadata) map[string]any

    // Email Metadata
    ToEmailMetadataResponses([]MetadataEmail) []response.EmailMetadata
    ToEmailMetadataResponse(MetadataEmail) response.EmailMetadata
    CreateEmailMetadata(request.CreateEmailMetadata) MetadataEmail
    PatchEmailMetadata(request.UpdateEmailMetadata) map[string]any
}
```

### TriggerMapper (`triggermapper.go`)
Manually implemented (not generated). Maps between Trigger models and response DTOs.

### Other Mappers
- **UserMapper** (`user.go`) - generated
- **JobMapper** (`job.go`) - generated
- **NodeMapper** (`node.go`) - generated

## Request/Response DTOs

### Request DTOs (`handler/request/`)

**Auth**: RegisterDTO, LoginDTO, RefreshTokenDTO, UpdateUser
**Metadata**: CreateMetadata, UpdateMetadata, CreateSftpMetadata, UpdateSftpMetadata, CreateEmailMetadata, UpdateEmailMetadata
**Trigger**: CreateTrigger, UpdateTrigger, CreateTriggerRule, UpdateTriggerRule, LinkJob, UpdateJobLink
**SQL**: GuessQueryRequest, OptimizeQueryRequest, IntrospectDatabase, TestDatabaseConnection, GuessSchemaRequest

### Response DTOs (`handler/response/`)

**Auth**: AuthResponse (token + refreshToken + user)
**Metadata**: Metadata (DB), SftpMetadata, EmailMetadata, TestConnectionResult, TestEmailConnectionResult, DeleteResponse
**Trigger**: Trigger, TriggerWithDetails, TriggerRule, TriggerJobLink, TriggerExecution
**Job**: Job, JobWithNodes (includes Nodes, Connexions, SharedUser)
**SQL**: GuessQueryResponse, OptimizeQueryResponse, DatabaseIntrospection, GuessSchemaResponse

## Adding a New CRUD Entity (Pattern)

1. Create model in `models/`
2. Create repo in `repo/` (thin GORM wrapper)
3. Create service in `service/` (CRUD methods using repo)
4. Create request DTOs in `handler/request/`
5. Create response DTOs in `handler/response/`
6. Add mapper methods to interface in `handler/mapper/`
7. Run `go generate` or implement mapper manually
8. Create handler in `handler/endpoints/` with route registration
9. Add model to `DB.AutoMigrate()` in `cmd/main.go`
10. Call handler init function in `initAPI()` in `cmd/main.go`

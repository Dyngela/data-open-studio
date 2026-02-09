# Frontend Systems

Angular 21 SPA in `front/`. All components are standalone (no NgModules).

## Bootstrap

**`src/main.ts`** - Bootstraps `App` with `appConfig`.

**`src/app/app.config.ts`** - Providers:
- `provideRouter(routes)` - route config
- `provideHttpClient(withInterceptors([authInterceptor, tokenRefreshInterceptor]))`
- `provideAnimationsAsync()`
- `providePrimeNG()` - Aura theme, ripple enabled
- `PrimeLocaleService` - FR/EN locale
- `MessageService`, `ConfirmationService` - PrimeNG services
- `provideAppInitializer()` - Sets locale from localStorage/navigator.language

**`src/app/app.ts`** - Root component:
- Shows navbar + router-outlet when authenticated
- Shows auth router-outlet when not authenticated
- Navbar: logo, links (Playground, Triggers, Jobs, Settings), user menu, logout
- Calls `authService.initializeAuth()` in `ngAfterViewInit`

## Routes (`src/app/app.routes.ts`)

| Path | Component | Guard | Load |
|------|-----------|-------|------|
| `/auth/login` | Login | guestGuard | Eager |
| `/auth/register` | Register | guestGuard | Eager |
| `/` | redirect -> `/jobs` | - | - |
| `/playground` | Playground | authGuard | Eager |
| `/playground/:id` | Playground | authGuard | Eager |
| `/triggers` | Triggers | authGuard | Eager |
| `/settings` | Settings (container) | authGuard | Eager |
| `/settings/db` | DbMetadataList | authGuard | Lazy |
| `/settings/sftp` | SftpMetadataList | authGuard | Lazy |
| `/settings/email` | EmailMetadataList | authGuard | Lazy |
| `/jobs` | Jobs | authGuard | Eager |
| `**` | redirect -> `/jobs` | - | - |

## Signal-Based API Pattern

The entire frontend uses a **signals-based reactive pattern** instead of RxJS observables in components.

### ApiResult (GET requests)
```typescript
interface ApiResult<T> {
    data: Signal<T | null>;       // Response data
    error: Signal<ApiError | null>;
    isLoading: Signal<boolean>;
    refresh: () => void;          // Re-fetch
}
```
Usage: `result = this.api.get<T>(path)` — returns immediately, signals update async.

### ApiMutation (POST/PUT/DELETE)
```typescript
interface ApiMutation<TResponse, TBody> {
    data: Signal<TResponse | null>;
    execute: (body: TBody) => void;   // Trigger the request
    isLoading: Signal<boolean>;
    error: Signal<ApiError | null>;
    success: Signal<MutationSuccess<TResponse> | null>;
    reset: () => void;
}
```
Usage: `mutation = service.createFoo(onSuccess, onError)` then `mutation.execute(body)`.

### BaseApiService (`core/services/base-api.service.ts`)
```typescript
get<T>(path, criteria?, options?): ApiResult<T>
post<TRes, TBody>(path, onSuccess?, onError?, options?): ApiMutation<TRes, TBody>
put<TRes, TBody>(path, onSuccess?, onError?, options?): ApiMutation<TRes, TBody>
delete<TRes, TBody>(path, onSuccess?, onError?, options?): ApiMutation<TRes, TBody>
```

## API Services (`core/api/`)

### AuthService (`auth.service.ts`)
- `register()` -> ApiMutation<AuthResponse, RegisterDto>
- `login()` -> ApiMutation<AuthResponse, LoginDto>
- `logout()` -> void (clears cookies)
- `refreshToken()` -> Observable<AuthResponse>
- `getAccessToken()` / `getRefreshToken()` -> string | null
- `initializeAuth()` -> void (decode token, restore user)
- Signals: `currentUser`, `isAuthenticated` (readonly)

### MetadataService (`metadata.service.ts`)
Three resource groups with identical CRUD pattern:

**Database** (`/metadata/db`):
- `getAllDb()`, `getDbById(id)`, `createDb()`, `updateDb(id)`, `deleteDb(id)`, `testDbConnection()`

**SFTP** (`/metadata/sftp`):
- `getAllSftp()`, `getSftpById(id)`, `createSftp()`, `updateSftp(id)`, `deleteSftp(id)`

**Email** (`/metadata/email`):
- `getAllEmail()`, `getEmailById(id)`, `createEmail()`, `updateEmail(id)`, `deleteEmail(id)`, `testEmailConnection()`

### TriggerService (`trigger.service.ts`)
- `getAll()`, `getById(id)`, `create()`, `update(id)`, `deleteTrigger(id)`
- `activate(id)`, `pause(id)`
- `addRule(triggerId)`, `updateRule(triggerId, ruleId)`, `deleteRule(triggerId, ruleId)`
- `linkJob(triggerId)`, `unlinkJob(triggerId, jobId)`
- `getExecutions(triggerId, limit)`

### JobService (`job.service.ts`)
- `getAll(filePath?)`, `getById(id)`, `create()`, `update(id)`, `deleteJob(id)`
- `share(id)`, `unshare(id)`
- `execute(id)`, `stop(id)`, `printCode(id)`

### SqlService (`sql.service.ts`)
- `guessQuery()` - AI-powered SQL generation
- `optimizeQuery()` - AI-powered optimization
- `testConnection()`, `getTables()`, `getColumns()` - Database introspection

### DbNodeService (`db-node.service.ts`)
- `guessSchema()` - Execute query to detect column types

## Type Definitions (`core/api/*.type.ts`)

### auth.type.ts
- `User { id, email, prenom, nom, role }`
- `UserRole` enum: admin, user
- `RegisterDto`, `LoginDto`, `AuthResponse`

### metadata.type.ts
- `DbType` enum: postgres, sqlserver, mysql
- `DbMetadata`, `CreateDbMetadataRequest`, `UpdateDbMetadataRequest`
- `SftpMetadata`, `CreateSftpMetadataRequest`, `UpdateSftpMetadataRequest`
- `EmailMetadata`, `CreateEmailMetadataRequest`, `UpdateEmailMetadataRequest`
- `TestConnectionResult { success, message, version? }`
- `TestEmailConnectionResult { imapSuccess, imapMessage, smtpSuccess, smtpMessage }`
- `DataModel { name, type, goType, nullable, length?, precision?, scale? }`
- `DeleteResponse { id, deleted }`

### trigger.type.ts
- Types: `TriggerType`, `TriggerStatus`, `WatermarkType`, `ExecutionStatus`, `ConditionOperator`
- `DatabaseTriggerConfig { metadataDatabaseId?, tableName, watermarkColumn, watermarkType, ... }`
- `EmailTriggerConfig { metadataEmailId?, folder?, fromAddress?, subjectPattern?, markAsRead?, lastUid?, ... }`
- `WebhookTriggerConfig { secret?, requiredHeaders? }`
- `TriggerConfig { database?, email?, webhook? }`
- `Trigger`, `TriggerWithDetails`, `TriggerRule`, `TriggerJobLink`, `TriggerExecution`
- Request types: `CreateTriggerRequest`, `UpdateTriggerRequest`, `CreateTriggerRuleRequest`, `LinkJobRequest`

### job.type.ts
- `Node { id, type, name, xpos, ypos, data }`
- `Job { id, name, description, filePath, creatorId, active, visibility, ... }`
- `JobWithNodes extends Job { nodes, connexions? }`
- `SharedUser { id, email, prenom, nom, role }`
- `PrintCode { code, steps }`
- `Connection { sourceNodeId, sourcePort, sourcePortType, targetNodeId, targetPort, targetPortType }`

### sql.type.ts
- `DatabaseTable { schema, name }`
- `DatabaseColumn { name, dataType, isNullable, isPrimary }`
- `DatabaseIntrospection { tables?, columns? }`
- `GuessQueryRequest`, `OptimizeQueryRequest`, etc.

## Guards & Interceptors

### authGuard (`core/guards/auth.guard.ts`)
- Redirects to `/auth/login?returnUrl=...` when not authenticated

### guestGuard
- Redirects to `/` when already authenticated

### authInterceptor (`core/interceptors/auth.interceptor.ts`)
- Adds `Authorization: Bearer <token>` to all requests

### tokenRefreshInterceptor (`core/interceptors/token-refresh.interceptor.ts`)
- Catches 401 errors
- Refreshes token via AuthService
- Queues pending requests using BehaviorSubject
- Retries failed requests with new token
- Redirects to login if refresh fails

## Core Services

### MetadataLocalService (`core/services/metadata.local.service.ts`)
- Eagerly loads DB and SFTP metadata on auth
- Provides cached `ApiResult` signals for immediate access

### LayoutService (`core/services/layout-service.ts`)
- Playground UI state: bottom bar height, active tab, sidebar width, viewport
- Node modal management: `openNodeModal(nodeId)`, `closeModal()`
- Sidebar toggling, resize handling

### LoadingService (`core/services/loading.service.ts`)
- Global HTTP loading state signal

### TokenRefreshScheduler (`core/services/token-refresh-scheduler.service.ts`)
- Checks JWT expiration every 3 minutes
- Auto-refreshes 5 minutes before expiry

### JobRealtimeService (`core/services/base-ws.service.ts`)
- WebSocket connection to `/api/ws?token=<jwt>`
- Subscribe to job progress: `subscribeToJob(jobId)`
- Events: `onProgress(listener)` receives `ProgressEvent { nodeId, nodeName, status, rowCount, message, jobId }`
- Connection states: disconnected, connecting, connected
- Auto-reconnect with 3s delay

## View Components

### Playground (`views/graph/playground/`)
The node graph canvas editor. Key features:
- SVG-based rendering with pan/zoom
- Drag & drop node creation from sidebar
- Node dragging with collision detection
- Connection creation by clicking ports
- Right-click context menus
- Node modal dialogs for configuration
- Real-time job progress via WebSocket
- Save/Execute/Stop actions
- Job loading from route param (`:id`)

Key signals: `nodes`, `connections`, `selectedNodeId`, `panOffset`, `zoom`, `tempConnection`, `currentJobId`

### Triggers (`views/triggers/triggers/`)
Trigger management with create/edit wizard (4 steps):
1. **Type** - Name, description, trigger type selection, polling interval
2. **Configuration** - Database: connection + table + watermark. Email: connection + filters
3. **Jobs** - Select jobs to link
4. **Summary** - Review before creation

Sidebar details view with tabs: Configuration, Rules, Jobs, History (executions)

Key signals: `triggers`, `selectedTrigger`, `currentStep`, `selectedDbConnection`, `selectedEmailConnection`

### Settings (`views/settings/`)
Container with tab navigation. Three sub-pages:
- **DbMetadataList** - Database connection CRUD with test connection
- **SftpMetadataList** - SFTP connection CRUD
- **EmailMetadataList** - Email (IMAP/SMTP) connection CRUD with dual test (IMAP + SMTP)

All follow identical pattern: table + modal form + test connection button.

### Jobs (`views/jobs/jobs/`)
File tree browser for jobs:
- Tree view with folders based on `filePath`
- Create modal with folder selection
- Job sharing (user autocomplete)
- Context menu actions (create folder, rename, delete)
- Open job navigates to `/playground/:id`

### Authentication (`views/authentication/`)
- **Login** - Email + password form, redirects to returnUrl or `/`
- **Register** - Email, name, password with confirmation, auto-login on success

## Node System (`nodes/`)

### Node Definitions
Each node type has a definition file exporting a `NodeDefinition<TConfig>`:

| Node | ID | API Type | Data In | Data Out | Flow In | Flow Out |
|------|-----|----------|---------|----------|---------|----------|
| Start | `start` | `start` | No | No | No | Yes |
| DB Input | `db-input` | `db_input` | No | Yes | Yes | Yes |
| Transform | `transform` | `map` | Yes | Yes | Yes | Yes |
| Log | `log` | `log` | Yes | No | Yes | No |
| Output | `output` | `db_output` | Yes | No | Yes | No |

### Node Modals
Each node type has a modal component for configuration:
- **StartModal** - Minimal (just close)
- **DbInputModal** - Connection selection, SQL query editor, schema guessing
- **TransformModal** - Column mapping (direct/library/custom), join configuration
- **LogModal** - Input schema display

### NodeRegistry (`nodes/node-registry.service.ts`)
- Singleton service
- `getNodeTypes()`, `getNodeTypeById(id)`, `getApiType(nodeTypeId)`, `getNodeTypeFromApiType(apiType)`

### Graph Services (`core/nodes-services/`)

**NodeGraphService** - Manages node instances and connections:
- `createNode(type, position)`, `deleteNode(nodeId)`, `updateNodeConfig/Position/Status`
- `createConnection(source, target)`, `deleteConnection(connection)`
- `loadFromJob(job)`, `toApiNodes(jobId)` - Serialize/deserialize
- `calculatePortPosition()`, `getConnectionPath()` - SVG rendering
- `findNonOverlappingPosition()`, `resolveCollision()` - Layout
- NODE_DIMENSIONS: width=180, headerPadding=12, bodyPadding=16, portSize=16

**JobStateService** - Manages node configs and schemas:
- `setNodeConfig(nodeId, config)`, `getNodeConfig<T>(nodeId)`
- `getOutputSchema(nodeId)` - Extract DataModel[] from config
- `getUpstreamSchemas(nodeId)` - Trace backward through connections to find input schemas
- `schemaVersion` signal - Incremented on config changes

**ConnectionService** - Database connection management (not graph connections):
- `getConnections()`, `getConnectionById(id)`, `addConnection()`, `deleteConnection()`, `updateConnection()`

## UI Components (`ui/`)

Reusable standalone components, all prefixed with `Kui`:

| Component | Selector | Purpose |
|-----------|----------|---------|
| KuiInputText | kui-input-text | Text input with validation |
| KuiInputPassword | kui-input-password | Password input |
| KuiInputTextArea | kui-input-textarea | Textarea |
| KuiInputNumber | kui-input-number | Number input |
| KuiSlider | kui-slider | Range slider |
| KuiSwitch | kui-switch | Toggle switch |
| KuiAutocomplete | kui-autocomplete | Autocomplete with filtering |
| KuiDatePicker | kui-date-picker | Date selection |
| KuiSelect | kui-select | Single select dropdown |
| KuiMultiselect | kui-multiselect | Multi-select |
| KuiFileUpload | kui-file-upload | File upload |
| KuiSubmitButton | kui-submit-button | Submit with loading state |
| KuiTable | kui-table | Data table |
| KuiGlobalLoadingSpinner | - | Full-screen loader |
| KuiLocalLoadingSpinner | - | Inline loader |

## Environment Configuration

**Development** (`src/environments/environment.ts`):
```typescript
export const environment = {
    production: false,
    baseUrl: 'http://localhost:8080/api/v1',
    apiVersion: 'v1',
    wsUrl: 'ws://localhost:8081/ws',
};
```

**Production** (`src/environments/environment.prod.ts`):
```typescript
export const environment = {
    production: true,
    baseUrl: 'https://api.votre-domaine.com/api',
    apiVersion: 'v1',
    wsUrl: 'wss://api.votre-domaine.com/ws',
};
```

## Adding a New Frontend Feature (Pattern)

1. Define types in `core/api/<name>.type.ts`
2. Create service extending `BaseApiService` in `core/api/<name>.service.ts`
3. Create component in `views/` (standalone, with signals)
4. Add route in `app.routes.ts` (lazy load if settings sub-page)
5. Use `ApiResult` for GET data, `ApiMutation` for mutations
6. Use `signal()` for local state, `computed()` for derived state

# Infrastructure & Configuration

## Docker Compose (`api/docker-compose.yml`)

### Services

| Service | Image | Container | Port | Purpose |
|---------|-------|-----------|------|---------|
| postgres | postgres:18 | data-open-studio-db | ${DB_PORT}:5432 | Main application database |
| postgres-test | postgres:18 | data-open-studio-pg-test | 5434:5432 | Test database (testuser/testpass/testdb) |
| sqlserver | mcr.microsoft.com/mssql/server:2022-latest | data-open-studio-sqlserver | 1433:1433 | SQL Server dev instance (SA/TestPass123!) |
| nats | nats:2.10-alpine | data-open-studio-nats | 4222, 8222 | Message broker for job progress |
| redis | redis:8.4-alpine | data-open-studio-redis | 6379:6379 | Cache |

All services have healthchecks configured.

**Volumes**: postgres_data, postgres_test_data, sqlserver_data, redis_data

**Init scripts**:
- `./init/postgres-main/` - Main DB initialization
- `./init/postgres-test/` - Test DB initialization
- `./init/sqlserver/` - SQL Server initialization (bash script)

### Starting Infrastructure
```bash
cd api
docker-compose up -d
```

## Backend Configuration

### Environment Variables (`api/.env`)

```bash
# Server
RUN_MODE=dev                          # dev or prod
API_PORT=:8080

# Main Database
DB_HOSTNAME=localhost
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=yourpassword
DB_NAME=data_open_studio
DB_SSL_MODE=disable

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRATION_MINUTES=60
JWT_REFRESH_EXPIRATION_DAYS=30

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Ollama (AI)
OLLAMA_HOST=http://localhost:11434
OLLAMA_MSG_LIMIT=10
```

### Config Loading (`api/config.go`)

```go
type AppConfig struct {
    Mode               string
    ApiPort            string
    OllamaHost         string
    OllamaMessageLimit int
    LogConfig          struct { Enabled bool; QueueName string }
    MainDatabase       struct { Host, Port, User, Password, DatabaseName, SSLMode string }
    JWTConfig          struct { Secret string; Expiration int; RefreshExpiration int }
    RedisConfig        struct { Host, Port, Password string; DB int }
}
```

`InitConfig(envfile)`:
1. Loads `.env` file with `godotenv`
2. Reads all environment variables
3. Connects to PostgreSQL (GORM)
4. Initializes logger (zerolog)
5. Connects to Redis

### Global Instances (`api/global.go`)

```go
var (
    DB     *gorm.DB          // PostgreSQL connection
    Logger zerolog.Logger     // Structured logger
    Redis  *redis.Client     // Redis client
)
```

### GORM Configuration
- `SingularTable: true` - No table name pluralization
- `FullSaveAssociations: true`
- `CreateBatchSize: 1000`
- Connection pool: MaxIdleConns=10, MaxOpenConns=10, ConnMaxLifetime=1h

### Auto-Migration (dev mode only)
In `cmd/main.go`, when `RUN_MODE=dev`:
```go
DB.AutoMigrate(
    &models.User{},
    &models.Job{},
    &models.Node{},
    &models.Port{},
    &models.MetadataDatabase{},
    &models.MetadataSftp{},
    &models.MetadataEmail{},
    &models.JobUserAccess{},
    &models.Trigger{},
    &models.TriggerRule{},
    &models.TriggerJob{},
    &models.TriggerExecution{},
)
```

## Server Startup (`cmd/main.go`)

```
1. InitConfig(".env")
2. Auto-migrate (dev mode)
3. Setup graceful shutdown (SIGINT, SIGTERM)
4. Create Gin router with CORS
5. Initialize API routes (initAPI)
6. Start TriggerPollerService (10 workers)
7. Run server with graceful shutdown
```

### Route Registration (`initAPI`)
```go
func initAPI(router *graceful.Graceful) {
    endpoints.AuthHandler(router)
    endpoints.DbMetadataHandler(router)
    endpoints.DbNodeHandler(router)
    endpoints.JobHandler(router)
    endpoints.SqlHandler(router)
    endpoints.TriggerHandler(router)
}
```

### CORS Configuration
- AllowOrigins: `["*"]`
- AllowMethods: GET, POST, PUT, DELETE, PATCH, OPTIONS
- AllowHeaders: Origin, Content-Type, Authorization
- AllowCredentials: true
- MaxAge: 12 hours

## Go Dependencies (`api/go.mod`)

### Core Framework
| Package | Purpose |
|---------|---------|
| github.com/gin-gonic/gin | HTTP framework |
| github.com/gin-contrib/cors | CORS middleware |
| github.com/gin-contrib/graceful | Graceful shutdown |
| gorm.io/gorm | ORM |
| gorm.io/driver/postgres | GORM PostgreSQL adapter |

### Database Drivers
| Package | Purpose |
|---------|---------|
| github.com/jackc/pgx/v5 | PostgreSQL driver |
| github.com/lib/pq | PostgreSQL driver (alternative) |
| github.com/go-sql-driver/mysql | MySQL driver |
| github.com/denisenkom/go-mssqldb | SQL Server driver |

### Auth & Security
| Package | Purpose |
|---------|---------|
| github.com/golang-jwt/jwt/v5 | JWT token handling |
| golang.org/x/crypto | Password hashing (bcrypt) |

### Messaging & Real-Time
| Package | Purpose |
|---------|---------|
| github.com/nats-io/nats.go | NATS messaging |
| github.com/gorilla/websocket | WebSocket support |
| github.com/redis/go-redis/v9 | Redis client |

### Email
| Package | Purpose |
|---------|---------|
| github.com/emersion/go-imap/v2 | IMAP v2 client |
| github.com/emersion/go-message | Email message parsing |
| github.com/wneessen/go-mail | SMTP email sending |

### Utilities
| Package | Purpose |
|---------|---------|
| github.com/rs/zerolog | Structured logging |
| github.com/joho/godotenv | .env file loading |
| github.com/go-playground/validator/v10 | Struct validation |
| github.com/google/uuid | UUID generation |
| github.com/blastrain/vitess-sqlparser | SQL parsing |
| github.com/stretchr/testify | Testing |

## Frontend Configuration

### Angular Build (`front/angular.json`)
- Builder: `@angular/build:application` (Vite-based)
- Assets: public folder
- Styles: `src/styles.css` (Tailwind entry)
- Budget limits:
  - Initial bundle: 500kB warning, 1MB error
  - Component styles: 4kB warning, 8kB error

### Frontend Dependencies (`front/package.json`)

| Package | Version | Purpose |
|---------|---------|---------|
| @angular/* | 21.0.6 | Framework |
| primeng | 21.0.2 | UI components |
| primeicons | 7.0.0 | Icon set |
| tailwindcss | 4.1.18 | Utility CSS |
| rxjs | 7.8.0 | Reactive extensions |
| jwt-decode | 4.0.0 | JWT decoding |
| ngx-cookie-service | 21.1.0 | Cookie management |

### Scripts
```bash
npm start      # ng serve (dev server on :4200)
npm run build  # ng build (production)
npm test       # ng test (Vitest)
npm run test:e2e  # Playwright E2E
```

## Development Workflow

### Starting Everything
```bash
# 1. Start infrastructure
cd api && docker-compose up -d

# 2. Start backend
cd api && go run cmd/main.go

# 3. Start frontend
cd front && ng serve
```

### Access Points
| Service | URL |
|---------|-----|
| Frontend | http://localhost:4200 |
| API | http://localhost:8080/api/v1 |
| WebSocket | ws://localhost:8081/ws |
| NATS Monitor | http://localhost:8222 |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |

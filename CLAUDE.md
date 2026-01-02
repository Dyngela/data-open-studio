# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Data Open Studio is a full-stack application with:
- **Backend API**: Go (Gin framework) with PostgreSQL database
- **Frontend**: Angular 21 with PrimeNG UI components and Tailwind CSS
- **Database**: PostgreSQL 18 (containerized)

## Development Commands

### Backend (Go API)

```bash
# From /api directory
cd api

# Run the API server
go run cmd/main.go

# Build the API
go build -o bin/api cmd/main.go

# Install dependencies
go mod download
go mod tidy
```

### Frontend (Angular)

```bash
# From /front directory
cd front

# Install dependencies
npm install

# Development server (http://localhost:4200)
ng serve

# Production build
ng build

# Run tests (Vitest)
ng test

# Watch mode for development
ng build --watch --configuration development
```

### Database

```bash
# Start PostgreSQL container
docker-compose up -d

# Stop database
docker-compose down

# View logs
docker-compose logs -f postgres
```

## Architecture

### Backend Structure (Go)

The Go API follows a layered architecture pattern:

```
api/
├── cmd/main.go              # Entry point, server setup, CORS, graceful shutdown
├── config.go                # Configuration loading from .env
├── global.go                # Global DB and Logger instances
├── pkg/                     # Shared utilities (JWT, Gin helpers)
│   ├── jwt.go              # JWT token generation/validation
│   └── gin-parser.go       # Request parsing and validation
└── internal/api/
    ├── models/             # GORM database models
    ├── repo/               # Data access layer (repositories)
    ├── service/            # Business logic layer
    └── handler/
        ├── endpoints/      # Route handlers (controllers)
        ├── middleware/     # Auth middleware, role checks
        ├── request/        # Request DTOs
        ├── response/       # Response DTOs
        └── mapper/         # Entity-DTO conversion
```

**Key patterns**:
- **Layered architecture**: Handler → Service → Repository → Model
- **DTO pattern**: Separate request/response DTOs from database models
- **Dependency injection**: Services are created in handlers and passed dependencies
- **Middleware**: JWT authentication and role-based authorization
- **Graceful shutdown**: Uses `gin-contrib/graceful` for clean server shutdown
- **Auto-migration**: In dev mode (`RUN_MODE=dev`), GORM auto-migrates models on startup

**Database**: GORM is configured with:
- Singular table names (no pluralization)
- Soft deletes via `DeletedAt` field
- Connection pooling (max 10 idle/open connections)

### Frontend Structure (Angular)

The Angular app uses modern Angular 21 patterns:

```
front/src/
├── main.ts                 # Bootstrap application
├── app/
│   ├── app.ts             # Root component
│   ├── app.config.ts      # App configuration (providers, interceptors)
│   └── app.routes.ts      # Route definitions
└── core/                  # Core services and infrastructure
    ├── api/               # API service layer
    │   ├── base-api.service.ts    # Abstract base with HTTP methods
    │   └── auth.service.ts        # Auth service extending base
    ├── interceptors/      # HTTP interceptors
    │   ├── auth.interceptor.ts           # Adds JWT to requests
    │   └── token-refresh.interceptor.ts  # Auto-refresh expired tokens
    ├── guards/            # Route guards
    ├── models/            # TypeScript interfaces/types
    └── services/          # App-wide services
```

**Key patterns**:
- **Signals**: Modern reactive state management (preferred over RxJS subjects)
- **Standalone components**: No NgModules, components are standalone
- **Service inheritance**: All API services extend `BaseApiService`
- **Interceptors**: Automatic JWT injection and token refresh
- **Type safety**: Strong typing with interfaces for all API models
- **API versioning**: Built into `BaseApiService.buildUrl()` using `environment.apiVersion`

See `front/src/core/API_USAGE.md` for complete API service patterns and examples.

### API Integration

**Frontend → Backend flow**:
1. Frontend calls service method (e.g., `authService.login()`)
2. `BaseApiService` builds URL: `${apiUrl}/${apiVersion}/${endpoint}`
3. `authInterceptor` adds JWT token to headers
4. Backend receives at `/api/v1/{endpoint}`
5. Gin middleware validates JWT (if protected route)
6. Handler → Service → Repository → Database
7. Response flows back through DTOs/mappers

**Authentication**:
- JWT-based with access + refresh tokens
- Access token in `Authorization: Bearer {token}` header
- Refresh token stored separately, used for token renewal
- Frontend auto-refreshes tokens via `token-refresh.interceptor.ts`
- Backend validates tokens in `middleware/auth_middleware.go`

## Configuration

### Backend (.env)

Required environment variables in `/api/.env`:

```bash
RUN_MODE=dev                    # dev or prod
API_PORT=:8080

DB_HOSTNAME=localhost
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=yourpassword
DB_NAME=data_open_studio
DB_SSL_MODE=disable

JWT_SECRET=your-secret-key
JWT_EXPIRATION_MINUTES=60
JWT_REFRESH_EXPIRATION_DAYS=30
```

### Frontend (environment.ts)

Located in `front/src/environments/`:

```typescript
export const environment = {
  production: false,
  apiUrl: 'http://localhost:8080/api',
  apiVersion: 'v1'
};
```

## Adding New Features

### Adding a New API Endpoint

1. **Create model** in `api/internal/api/models/`
2. **Create repository** in `api/internal/api/repo/` for data access
3. **Create service** in `api/internal/api/service/` for business logic
4. **Create request/response DTOs** in `api/internal/api/handler/request/` and `response/`
5. **Create mapper** in `api/internal/api/handler/mapper/` for entity-DTO conversion
6. **Create handler** in `api/internal/api/handler/endpoints/`
7. **Register routes** in handler's init function
8. **Update migrations** by adding model to `DB.AutoMigrate()` in `cmd/main.go`

### Adding a New Angular Service

1. **Define models** in `front/src/core/models/`
2. **Create service** extending `BaseApiService` in `front/src/core/api/`
3. **Use inherited methods**: `getList()`, `getWithResponse()`, `postWithResponse()`, etc.
4. **Inject and use** in components with `inject()` function

See `front/src/core/API_USAGE.md` for detailed examples.

## Testing

### Backend
Currently no test framework configured. To add tests, use Go's built-in `testing` package or a framework like Testify.

### Frontend
- Test runner: **Vitest**
- Run tests: `ng test` from `/front` directory
- Test files: `*.spec.ts` alongside source files

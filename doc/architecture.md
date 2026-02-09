# Data Open Studio - Architecture Overview

## What is Data Open Studio?

Data Open Studio is a visual data pipeline builder. Users design ETL (Extract-Transform-Load) workflows by connecting nodes on a graph canvas, then execute them as generated Go programs. The system also supports automated triggers that poll data sources and fire jobs when new data arrives.

## Tech Stack

| Layer | Technology | Version       |
|-------|-----------|---------------|
| Backend API | Go + Gin | Go 1.25       |
| Database | PostgreSQL | 18            |
| ORM | GORM | v2            |
| Frontend | Angular | 21            |
| UI Components | PrimeNG | 21            |
| CSS | Tailwind CSS | 4             |
| Message Broker | NATS | 2.10          |
| Cache | Redis | 8.4           |
| WebSocket | gorilla/websocket | -             |
| Auth | JWT | golang-jwt/v5 |
| Test DB | SQL Server 2022 | -             |

## High-Level Architecture

```
                        +-------------------+
                        |   Angular 21 SPA  |
                        |   (PrimeNG UI)    |
                        +--------+----------+
                                 |
                    HTTP REST + WebSocket
                                 |
                        +--------v----------+
                        |    Gin HTTP API    |
                        |   (port 8080)     |
                        +--------+----------+
                                 |
              +------------------+------------------+
              |                  |                  |
     +--------v------+  +-------v-------+  +-------v--------+
     |   PostgreSQL  |  |     NATS      |  |     Redis       |
     |  (main DB)    |  | (job progress)|  |   (cache)       |
     +---------------+  +-------+-------+  +----------------+
                                |
                        +-------v-------+
                        |  WebSocket    |
                        |  Hub          |
                        +---------------+
```

## Core Systems

### 1. Node Graph & Code Generation (`doc/codegen.md`)
The visual pipeline editor. Users connect nodes (DB Input, Transform, DB Output, Log, Email Output) on a canvas. When executed, the backend traverses the graph and generates a standalone Go program that runs the data pipeline.

### 2. Trigger System (`doc/triggers.md`)
Automated polling that watches data sources (database tables, email inboxes) for changes and fires linked jobs. Supports watermark-based database polling and IMAP UID-based email polling with configurable filters and rules.

### 3. Real-Time Progress (`doc/realtime.md`)
When a job executes, generated Go code publishes progress events to NATS. A bridge subscribes to NATS and forwards events through a WebSocket hub to connected browser clients, enabling live progress bars per node.

### 4. Metadata System (part of `doc/backend.md`)
CRUD for external connection credentials (Database, SFTP, Email). Used by nodes and triggers to reference saved connections instead of embedding credentials.

### 5. Authentication & Authorization (part of `doc/backend.md`)
JWT-based auth with access + refresh tokens. Role-based access (admin/user). Job-level sharing with owner/editor/viewer roles.

## Project Directory Structure

```
data-open-studio/
+-- api/                          # Go backend
|   +-- cmd/main.go               # Entry point
|   +-- config.go                 # Config loading from .env
|   +-- global.go                 # Global DB, Logger, Redis
|   +-- pkg/                      # Shared utilities
|   |   +-- jwt.go                # JWT generation/validation
|   |   +-- gin-parser.go         # Request parsing
|   |   +-- mail.go               # (deprecated, moved to service)
|   +-- internal/
|   |   +-- api/
|   |   |   +-- models/           # GORM models (15+ files)
|   |   |   +-- repo/             # Data access layer
|   |   |   +-- service/          # Business logic
|   |   |   +-- handler/
|   |   |       +-- endpoints/    # Route handlers
|   |   |       +-- middleware/   # Auth middleware
|   |   |       +-- request/      # Request DTOs
|   |   |       +-- response/     # Response DTOs
|   |   |       +-- mapper/       # Model <-> DTO mappers
|   |   +-- gen/                  # Code generation engine
|   |   |   +-- generator.go      # Registry + interfaces
|   |   |   +-- template_data.go  # Shared data structures
|   |   |   +-- node_*.go         # Per-node generators
|   |   |   +-- templates/        # Go text templates
|   |   |   +-- lib/              # Runtime lib for generated code
|   |   +-- realtime/             # WebSocket + NATS
|   +-- docker-compose.yml        # Infrastructure services
|   +-- go.mod / go.sum
|
+-- front/                        # Angular frontend
|   +-- src/
|   |   +-- app/                  # App bootstrap, routes
|   |   +-- core/                 # Services, guards, interceptors
|   |   |   +-- api/              # API services + type files
|   |   |   +-- services/         # Base API, layout, loading
|   |   |   +-- nodes-services/   # Graph state management
|   |   |   +-- guards/           # Auth guards
|   |   |   +-- interceptors/     # HTTP interceptors
|   |   +-- views/                # Page components
|   |   |   +-- graph/playground/ # Node canvas editor
|   |   |   +-- triggers/         # Trigger management
|   |   |   +-- settings/         # Metadata CRUD pages
|   |   |   +-- jobs/             # Job browser
|   |   |   +-- authentication/   # Login/Register
|   |   +-- nodes/                # Node definitions + modals
|   |   +-- ui/                   # Reusable UI components
|   +-- angular.json
|   +-- package.json
|
+-- doc/                          # This documentation
+-- CLAUDE.md                     # AI assistant instructions
```

## Data Flow

### Creating a Pipeline
1. User opens Playground (canvas editor)
2. Drags nodes from sidebar onto canvas
3. Connects nodes via ports (data flow + execution flow)
4. Configures each node (SQL query, transform mapping, output table)
5. Saves as a Job (stored in PostgreSQL with node graph)

### Executing a Pipeline
1. User clicks Execute on a Job
2. Backend loads Job with Nodes from DB
3. Code generator traverses the node graph
4. Generates a standalone Go program (`main.go`)
5. Compiles and runs the generated program
6. Generated code publishes progress to NATS
7. WebSocket hub forwards progress to browser
8. Frontend updates node status in real-time

### Trigger-Driven Execution
1. User creates a Trigger (database/email type)
2. Links one or more Jobs to the Trigger
3. Trigger poller service runs in background
4. Every N seconds, polls the data source
5. If new data found, applies rule filters
6. Triggers linked jobs with event data

## Document Index

| Document | Contents |
|----------|----------|
| [architecture.md](architecture.md) | This file - overview |
| [backend.md](backend.md) | Models, repos, services, handlers, middleware, mappers, DTOs |
| [frontend.md](frontend.md) | Angular services, types, components, nodes, routing, UI |
| [codegen.md](codegen.md) | Code generation engine, generators, templates, runtime |
| [triggers.md](triggers.md) | Trigger system, poller, rules, execution tracking |
| [realtime.md](realtime.md) | WebSocket, NATS bridge, progress reporting |
| [infrastructure.md](infrastructure.md) | Docker, config, dependencies, environment |

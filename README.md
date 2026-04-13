# Data Open Studio

A self-hosted data engineering platform combining a **visual pipeline builder** and a **columnar visualization engine**. It lets users design ETL workflows as node graphs, execute them, automate them with triggers, and feed results into an in-memory analytics layer.

---

## Goals

- **Pipeline**: let analysts and engineers build data pipelines visually (drag-and-drop node graph → compiled Go program → executed on the server)
- **Visualization**: store query results as typed columnar frames, explore them with a planned SQL-like language (Resin), and build dashboards
- **Automation**: schedule or react to events (new database rows, incoming emails) through a trigger system
- **Self-contained**: one `docker compose up` starts everything — no external SaaS dependency

---

## Repository layout

```
data-open-studio/
├── api/          # Go pipeline backend (Gin, GORM, PostgreSQL, Redis, NATS)
├── viz/          # Rust visualization suite
│   ├── df-store/ #   columnar storage library
│   ├── api/      #   Axum HTTP API for frames/workspaces
│   └── resin/    #   query language (in progress)
├── gateway/      # Go reverse proxy — single entry-point for the frontend
├── front/        # Angular 21 SPA
└── doc/          # Detailed per-subsystem documentation
```

Detailed docs per subsystem live in `doc/`:

| File | Covers |
|---|---|
| `doc/architecture.md` | System overview, data-flow diagrams |
| `doc/backend.md` | Models, repos, services, all API routes |
| `doc/frontend.md` | Angular services, signal patterns, routing |
| `doc/codegen.md` | Code-generation engine, node generators, templates |
| `doc/triggers.md` | Trigger types, poller, rule evaluation |
| `doc/realtime.md` | WebSocket hub, NATS bridge |
| `doc/infrastructure.md` | Docker services, env vars, DB init |

---

## Architecture overview

### Runtime topology

```
Browser (Angular SPA)
        │  HTTP / WebSocket
        ▼
  Gateway :8000   (Go, stdlib reverse-proxy)
  ├── /api/*   ──▶  Pipeline API :8080  (Go / Gin)
  ├── /viz/*   ──▶  Viz API      :3030  (Rust / Axum)
  └── /ws*     ──▶  Realtime WS  :8081  (Go / gorilla/ws)

Pipeline API  ──▶  PostgreSQL :5432   (main DB)
              ──▶  Redis      :6379   (cache / sessions)
              ──▶  NATS       :4222   (job progress pub/sub)
              ──▶  Ollama     :11434  (LLM, optional)

Viz API       ──▶  PostgreSQL :5433   (workspace / source metadata)
                   (frames live in-memory, keyed by workspace UUID)
```

### Pipeline execution flow

```
1. User designs a job in the Playground (drag-drop nodes, configure each one)
2. Save → nodes + graph stored in PostgreSQL
3. Execute →
   a. Backend loads job + nodes
   b. Code-gen engine (gen/) traverses graph, renders Go templates per node type
   c. Compiles the generated program with `go build`
   d. Runs the binary; it reads sources, applies transforms, writes outputs
   e. Progress events published to NATS  →  WebSocket hub  →  browser
4. Node status updates appear in real-time in the playground
```

### Trigger / automation flow

```
TriggerPollerService (background, 10 workers)
  ├── Every N seconds per active trigger:
  │     ├── Database trigger  →  query watermark column, detect new rows
  │     └── Email trigger     →  IMAP IDLE / poll for unseen UIDs
  ├── Apply TriggerRules (dot-notation field filters on event data)
  └── If rules pass → fire linked jobs (follows pipeline flow above)
```

### Visualization flow

```
Dataset (SQL query + DB connection stored in PostgreSQL)
  └── "Load as Frame" button
        │  Pipeline API calls Viz API server-side (credentials never reach browser)
        ▼
  Viz API: POST /workspaces/{id}/sources   (create Postgres source)
           POST /workspaces/{id}/sources/{id}/load
        │  df-store postgres connector executes query → typed Frame in memory
        ▼
  WorkspaceDetail page: lists frames, expandable schema + 100-row preview
        │  (future: Resin query language, dashboards, reports)
```

---

## Key subsystems

### `api/` — Go pipeline backend

Layered architecture: `Handler → Service → Repository → Model`

```
cmd/main.go              entry point, route registration, auto-migrate (dev)
config.go                .env loading
pkg/                     JWT, gin helpers, Ollama client, Redis helpers
internal/api/
  models/                GORM entities  (User, Job, Node, Port, Trigger, Dataset, Metadata…)
  repo/                  thin GORM wrappers
  service/               business logic (JobService, TriggerPollerService, DatasetService…)
  handler/
    endpoints/           Gin route handlers
    middleware/          JWT auth, role checks
    request/ response/   DTOs
    mapper/              entity ↔ DTO (go-generate + dtomapper)
  gen/                   code-generation engine
    node_*.go            per-node-type generators
    templates/           Go text/template files (.go.tmpl)
    jobExecutor.go       compile + run generated code
  realtime/              WebSocket hub + NATS bridge
```

**Node types** (pipeline graph): DB Input, DB Output (insert/update/merge/delete/truncate), Map/Transform (inner join, left join, cross join, union, column transform), Email Output, Log.

**Auth**: JWT access token (60 min) + refresh token (30 days). Roles: `admin`, `user`. Per-job access roles: `owner`, `editor`, `viewer`.

### `viz/` — Rust visualization suite

**df-store** (library): columnar data store.
- `Frame` = named list of typed `Series`
- `Series` = optional dictionary-encoded chunked column
- Typed: Boolean, Int8…Int64, UInt8…UInt64, Float32/64, String, Binary, Date, Datetime, Duration, List, Struct, Null
- Connectors: CSV, PostgreSQL, Excel
- Cedrus binary format for on-disk persistence

**api** (Axum server, `:3030`):
- Workspaces (UUID-keyed containers for frames)
- Sources (CSV or Postgres configs stored in PostgreSQL)
- `POST /workspaces/{id}/sources/{id}/load` — runs connector, writes frame to in-memory store
- `GET /workspaces/{id}/frames/{name}/preview?offset&limit` — paginated column-oriented response
- `POST /workspaces/{id}/execute` — Resin script execution (WIP)

**resin** (query language, in progress): declarative language for cross-frame queries with `relate`, `from`, `select`, `where`, `group by`, `sort`, `limit`. Spec in `viz/resin/RESIN.md`.

### `gateway/` — Go reverse proxy

Stdlib `httputil.ReverseProxy`. Single entry-point on `:8000`. Strips `/viz` prefix before forwarding. Applies permissive CORS. No auth logic (auth lives in each backend).

### `front/` — Angular 21 SPA

Standalone components, Signals-based state (no RxJS in components), PrimeNG + Tailwind CSS.

```
core/api/              service layer — one service per domain, all extend BaseApiService
core/services/         base HTTP/WS helpers, interceptors, icon registry
nodes/                 node type definitions + config components (one folder per node type)
views/
  pipeline/
    graph/playground   main canvas editor (custom drag-drop, port connections)
    jobs/              job browser
    triggers/          trigger management UI
    settings/          database / SFTP / email credential management
  viz/
    datasets/          dataset list + SQL editor
    workspaces/        workspace list + detail (frame explorer)
  authentication/      login / register
```

**HTTP pattern**: `ApiResult<T>` (signal-based query with `.refresh()`) and `ApiMutation<T, Body>` (signal-based mutation with `.execute(body)`). Tokens auto-refreshed by `token-refresh.interceptor.ts`.

---

## Infrastructure

Started with:

```bash
cd api && docker compose up -d   # PostgreSQL, PostgreSQL-test, SQL Server, NATS, Redis, Mailpit
```

| Service | Port | Purpose |
|---|---|---|
| PostgreSQL 18 | 5432 | Pipeline API main DB |
| PostgreSQL 18 (test) | 5434 | Integration test DB |
| SQL Server 2022 | 1433 | Multi-DB testing |
| NATS 2.10 | 4222 / 8222 | Job progress pub/sub |
| Redis 8.4 | 6379 | Cache / session store |
| Mailpit | 1025 / 8025 | SMTP catch-all + web inbox |

See `api/.env.example` for required environment variables.

---

## Development

```bash
# Backend
cd api && go run cmd/main.go

# Frontend
cd front && npm install && ng serve        # http://localhost:4200

# Viz API
cd viz && cargo run -p api

# Gateway
cd gateway && go run main.go

# Database
cd api && docker compose up -d
```

Tests:
```bash
cd api && go test ./...                    # unit + integration (needs docker compose)
cd front && ng test                        # Vitest
```

---

## Roadmap

**Pipeline**
- Additional node types: SFTP, S3, CSV output
- Pipeline monitoring dashboard
- Polished map/transform nodes

**Viz**
- Complete Resin language (lexer → parser → executor → API endpoint)
- LSP for Resin (editor autocomplete)
- Frame persistence across restarts (Cedrus)
- Visualizations: charts, tables, dashboards
- Multi-source relationships and cross-frame joins

**General**
- E2E test suite
- Gateway-level auth / session management (consolidate JWT handling)
- Observability: Prometheus + Grafana
- Multi-tenant / team support with fine-grained RBAC

# viz — Columnar In-Memory Store

A from-scratch columnar data store inspired by Qlik and Power BI.
Built in Rust as a learning project.

```
viz/
├── df-store/   — core library (frames, series, dict encoding, connectors, query engine)
├── api/        — Axum HTTP server exposing the store to the frontend
└── RESIN.md    — Resin query language spec and implementation plan
```

---

## Architecture

```
Source data                  Connectors               In-memory store
─────────────────────────────────────────────────────────────────────
CSV files       ──────────►  csv.rs       ──────────►
Postgres query  ──────────►  postgres.rs  ──────────►  HashMap<String, Frame>
                                                               │
                                          ◄──────────  .cedr files (Cedrus)
                                                        persisted on disk

Model layer
──────────────────────────────────────────────────────────────────────
Relationships   ──────────►  relationship.rs  ──────►  Vec<Relationship>
                                                               │
                                             ◄──────  model.json (persisted)

Query / resolve
──────────────────────────────────────────────────────────────────────
resolver.rs     walks relationships ──────────────►  derived Frame
Resin language  parses + executes   ──────────────►  derived Frame
                                                               │
                                                        stored in-memory
                                                        + written to .cedr
```

### Key concepts

- **Frame** — named collection of `Series` (one per column)
- **Series** — typed column with optional dictionary encoding (low-cardinality values stored as integer indices + symbol table)
- **Cedrus** — binary file format (`.cedr`) for persisting frames to disk
- **Relationship** — declared link between a FK column and a PK column across two frames (Power BI style)
- **Resolve** — materialise a flat denormalized frame by following all relationships from a base frame
- **Resin** — query language for declaring relationships and querying across frames (see `RESIN.md`)

---

## Current state

### Done
- [x] `Frame` / `Series` / `Value` / `DataValue` / `DataType` types
- [x] Dictionary encoding — low-cardinality columns stored as `int` indices + symbol table
- [x] `InferDtype` trait — infer column type from raw data
- [x] Shared series-building helpers (`build_series`, `compute_min_max`, `build_dict`)
- [x] CSV connector (`csv_to_frame`, `load_csv`)
- [x] Postgres connector (`postgres_to_frame`, `load_postgres`) — sync client, full type mapping
- [x] Cedrus binary format — `write` (Frame → `.cedr`) and `read` (`.cedr` → Frame)
- [x] Relationship model — `Relationship`, `JoinType` (Left / Inner)
- [x] Frame resolver — joins dimension frames onto a base frame, decodes dict indices
- [x] Axum HTTP API — frame CRUD, relationship CRUD, resolve endpoint

### API surface today
```
GET    /frames
POST   /frames/csv          { path, delimiter, has_header, frame_name }
POST   /frames/postgres     { host, port, username, password, database, query, frame_name }
GET    /frames/:name
GET    /frames/:name/data   ?offset=0&limit=100
DELETE /frames/:name

GET    /relationships
POST   /relationships       { id?, from_frame, from_col, to_frame, to_col, join_type }
GET    /relationships/:id
DELETE /relationships/:id

POST   /resolve             { base, result_name }
```

---

## TODO

### 1 — Persistence (next session priority)

The store currently lives only in memory. Restart = everything gone.

- [ ] **On frame load**: `csv_to_frame` / `postgres_to_frame` already build the `Frame` —
  the API should immediately `Cedrus::write` it so it survives restarts
- [ ] **On startup**: scan the Cedrus store path (`CEDRUS_STORE_PATH`) for `.cedr` files
  and load them all back into `AppState.frames`
- [ ] **Model file**: persist the relationship registry to `{store_path}/model.json`
  (serialize `Vec<Relationship>`). Load it on startup after frames are restored
- [ ] **On resolve**: `POST /resolve` already produces a derived `Frame` — write it to
  `.cedr` under its `result_name` so it is also restored on restart
- [ ] **On frame delete**: remove the corresponding `.cedr` file; if it is a source frame,
  also invalidate (delete) any resolved frames that depend on it

```
{CEDRUS_STORE_PATH}/
  sales.cedr
  customers.cedr
  products.cedr
  sales_enriched.cedr   ← resolved frame, also persisted
  model.json            ← relationship registry
```

---

### 2 — Resin language (see RESIN.md for full spec)

See `RESIN.md` for the complete syntax reference and phase-by-phase plan.
Short version:

```resin
relate sales.customer_id -> customers.id
relate sales.product_id  -> products.id

from sales
select date, amount, customers.name, customers.country, products.category
where amount > 100
group by customers.country, products.category
  sum(amount) as total_revenue
  count(*)    as orders
sort total_revenue desc
limit 20
as sales_summary
```

- [ ] **Phase 1** — Lexer (`df-store/src/resin/lexer.rs`)
- [ ] **Phase 2** — AST (`df-store/src/resin/ast.rs`)
- [ ] **Phase 3** — Parser (`df-store/src/resin/parser.rs`) — recursive descent
- [ ] **Phase 4** — Executor (`df-store/src/resin/executor.rs`)
  - Filter (WHERE) with dict-aware predicate evaluation
  - Projection (SELECT)
  - Aggregation (GROUP BY + sum / count / avg / min / max)
  - Sort, Limit
- [ ] **Phase 5** — API endpoint `POST /query` accepting a Resin query string

**Resin ↔ frame store integration:**
- `relate` in a query → adds to the persistent relationship registry (same as `POST /relationships`)
- `from ... as <name>` → result frame is stored in-memory + written to `.cedr`
- If the named frame already exists, it is overwritten (re-resolved with latest data)

---

### 3 — Dict-aware operations

The point of building this instead of using Polars or DuckDB is that filters and
group-bys on dict-encoded columns should work at the integer level — no string decoding.

- [ ] **Dict-aware WHERE**: for `country = "France"`, find the dict index for `"France"`
  once, then scan chunks comparing integers — O(1) lookup + O(n) integer scan
- [ ] **Dict-aware GROUP BY**: group on chunk integer values directly; decode only for
  the final output

---

### 4 — Multi-hop relationships

`POST /resolve` currently only follows relationships one level deep
(`sales → customers`). Transitive resolution would allow:

```resin
relate sales.customer_id -> customers.id
relate customers.region_id -> regions.id

from sales
select amount, customers.name, regions.country   -- two hops
as enriched
```

- [ ] Detect the full reachable graph from the base frame
- [ ] Topological sort to resolve in dependency order
- [ ] Cycle detection (return an error, not an infinite loop)

---

### 5 — Frame refresh

Source frames (CSV / Postgres) can go stale. Add a way to re-run the connector
and update both the in-memory frame and the `.cedr` file.

- [ ] `POST /frames/:name/refresh` — re-runs the original connector config
  - Requires storing the connector config alongside the frame (add to `.cedr` footer
    or to a sidecar JSON)
- [ ] After refresh, automatically re-resolve any derived frames that depend on the
  refreshed frame

---

### 6 — Operations without Resin (stretch)

For callers that don't want to write Resin queries, expose the most common
operations as REST endpoints:

- [ ] `POST /frames/:name/filter`   `{ "col": "amount", "op": "gt", "value": 100 }`
- [ ] `POST /frames/:name/select`   `{ "cols": ["date", "amount"] }`
- [ ] `POST /frames/:name/groupby`  `{ "by": ["country"], "agg": [{ "fn": "sum", "col": "amount", "as": "total" }] }`
- [ ] `POST /frames/:name/sort`     `{ "by": "total", "desc": true }`

These map 1-to-1 to the Resin executor phases and can reuse the same code paths.

---

## Running locally

```bash
# Start the API
cd api
cargo run

# API is available at http://localhost:3030

# Load a frame
curl -X POST http://localhost:3030/frames/csv \
  -H "Content-Type: application/json" \
  -d '{ "path": "/data/sales.csv", "delimiter": 44, "has_header": true, "frame_name": "sales" }'

# Declare a relationship
curl -X POST http://localhost:3030/relationships \
  -H "Content-Type: application/json" \
  -d '{ "from_frame": "sales", "from_col": "customer_id", "to_frame": "customers", "to_col": "id" }'

# Resolve to a flat frame
curl -X POST http://localhost:3030/resolve \
  -H "Content-Type: application/json" \
  -d '{ "base": "sales", "result_name": "sales_enriched" }'

# Query the result
curl "http://localhost:3030/frames/sales_enriched/data?limit=10"
```

---

## Environment variables

| Variable | Default (Windows) | Default (Linux) | Purpose |
|---|---|---|---|
| `CEDRUS_STORE_PATH` | `C:/dev/cedrus` | `/var/lib/cedrus` | Where `.cedr` files are written |

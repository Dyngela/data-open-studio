# Resin

A query language for the Cedrus columnar store.

Resin separates **model declaration** (relationships between frames, defined once)
from **queries** (which reference related columns freely, without writing joins).
This mirrors the Power BI mental model: declare your data model first, then query
against it as if everything is a single flat table.

---

## Language overview

```resin
-- declare relationships (model layer)
relate sales.customer_id -> customers.id
relate sales.product_id  -> products.id

-- query against the model (no explicit joins needed)
from sales
select
  date,
  amount,
  customers.name,
  customers.country,
  products.category
where amount > 100
group by customers.country, products.category
  sum(amount) as total_revenue
  count(*)    as orders
sort total_revenue desc
limit 20
```

### Syntax reference

```
-- Comments
-- anything after -- on a line is a comment

-- Relate (model declaration)
relate <frame>.<col> -> <frame>.<col>

-- Query
from <frame>
[select <expr>, ...]
[where <predicate>]
[group by <col>, ...
  <agg_fn>(<col>) as <alias>
  ...]
[sort <col> [asc|desc]]
[limit <n>]
[as <result_name>]      -- stores result as a named frame in the store

-- Column reference
col                     -- column in the base frame
frame.col               -- column from a related frame (resolved via relate)

-- Predicates
amount > 100
status = "active"
country != "US"
amount > 100 and status = "active"
not archived

-- Aggregate functions
sum(<col>)
count(*)
count(<col>)
avg(<col>)
min(<col>)
max(<col>)

-- Types in literals
42          -- Int64
3.14        -- Float64
"hello"     -- String
true/false  -- Boolean
```

### Result naming

```resin
from sales
select date, amount, customers.name
as enriched_sales           -- stored in the frame store under this name
```

If `as` is omitted the result is returned directly but not stored.

---

## What is already built

| Component | Location | Status |
|---|---|---|
| Columnar store (`Frame` / `Series` / `Value`) | `df-store/src/store.rs` | Done |
| Dictionary encoding (int indices + symbol table) | `df-store/src/connectors/extractor.rs` | Done |
| CSV connector | `df-store/src/connectors/csv.rs` | Done |
| Postgres connector | `df-store/src/connectors/postgres.rs` | Done |
| Binary serialization (`.cedr` format) | `df-store/src/cedrus/` | Done |
| Relationship model (`Relationship`, `JoinType`) | `df-store/src/query/relationship.rs` | Done |
| Frame resolver (join + dict decode) | `df-store/src/query/resolver.rs` | Done |
| HTTP API (Axum) | `api/src/main.rs` | Done |

---

## Implementation plan

### Phase 1 — Lexer  `df-store/src/resin/lexer.rs`

Tokenise a raw `&str` into a flat `Vec<Token>`.

**Token types to handle:**
- Keywords: `relate`, `from`, `select`, `where`, `group`, `by`, `sort`, `limit`, `as`,
  `and`, `or`, `not`, `asc`, `desc`, `sum`, `count`, `avg`, `min`, `max`
- Symbols: `->`, `.`, `,`, `*`, `(`, `)`, `=`, `!=`, `>`, `<`, `>=`, `<=`
- Literals: integer (`42`), float (`3.14`), string (`"hello"`), boolean (`true`/`false`)
- Identifier: unquoted name (`sales`, `customer_id`, `total_revenue`)
- Comment: `--` to end of line → skip

**Output:** `Vec<Token>` where each token carries its kind and the original source span
(line + col) for error messages.

---

### Phase 2 — AST  `df-store/src/resin/ast.rs`

```rust
pub enum Statement {
    Relate(RelateStmt),
    Query(QueryStmt),
}

pub struct RelateStmt {
    pub from_frame: String,
    pub from_col:   String,
    pub to_frame:   String,
    pub to_col:     String,
}

pub struct QueryStmt {
    pub base:      String,
    pub select:    Vec<ColRef>,       // empty = select all
    pub filter:    Option<Expr>,
    pub group_by:  Option<GroupBy>,
    pub sort:      Option<Sort>,
    pub limit:     Option<usize>,
    pub result_as: Option<String>,
}

// customers.name  →  ColRef { frame: Some("customers"), col: "name" }
// amount          →  ColRef { frame: None, col: "amount" }
pub struct ColRef {
    pub frame: Option<String>,
    pub col:   String,
}

pub struct GroupBy {
    pub by:   Vec<ColRef>,
    pub aggs: Vec<Aggregation>,
}

pub struct Aggregation {
    pub func:  AggFn,
    pub col:   Option<ColRef>,   // None = COUNT(*)
    pub alias: String,
}

pub enum AggFn { Sum, Count, Avg, Min, Max }

pub struct Sort {
    pub col:  ColRef,
    pub desc: bool,
}

pub enum Expr {
    Col(ColRef),
    Lit(Literal),
    BinOp { op: BinOp, left: Box<Expr>, right: Box<Expr> },
    Not(Box<Expr>),
}

pub enum BinOp { Eq, Neq, Gt, Lt, Gte, Lte, And, Or }

pub enum Literal { Int(i64), Float(f64), Str(String), Bool(bool) }
```

---

### Phase 3 — Parser  `df-store/src/resin/parser.rs`

Hand-written **recursive descent** parser. Takes `&[Token]` and a cursor, returns
`Result<Statement, ParseError>`.

One function per grammar rule:

```rust
fn parse_statement(tokens: &[Token], pos: &mut usize) -> Result<Statement, ParseError>
fn parse_relate(tokens: &[Token], pos: &mut usize) -> Result<RelateStmt, ParseError>
fn parse_query(tokens: &[Token], pos: &mut usize) -> Result<QueryStmt, ParseError>
fn parse_select(tokens: &[Token], pos: &mut usize) -> Result<Vec<ColRef>, ParseError>
fn parse_where(tokens: &[Token], pos: &mut usize) -> Result<Expr, ParseError>
fn parse_expr(tokens: &[Token], pos: &mut usize) -> Result<Expr, ParseError>
fn parse_group_by(tokens: &[Token], pos: &mut usize) -> Result<GroupBy, ParseError>
fn parse_sort(tokens: &[Token], pos: &mut usize) -> Result<Sort, ParseError>
```

`ParseError` carries a message + source location (from token spans).

---

### Phase 4 — Executor  `df-store/src/resin/executor.rs`

Takes a `QueryStmt`, the relationship registry, and the frame store.
Returns a `Frame`.

```rust
pub fn execute(
    query:         &QueryStmt,
    relationships: &[Relationship],
    frames:        &HashMap<String, Frame>,
) -> Result<Frame, ExecError>
```

**Execution order (matches SQL semantics):**

```
1. FROM      — get the base frame
2. JOIN      — resolve all related frames needed by SELECT / WHERE / GROUP BY
               (reuse resolver.rs — only join what is actually referenced)
3. WHERE     — filter rows using the Expr tree
4. GROUP BY  — bucket rows, apply aggregation functions
5. SELECT    — project final columns (or pass-through if no select clause)
6. SORT      — order rows
7. LIMIT     — truncate
```

**Dict-aware WHERE filtering:**
For dict-encoded columns, evaluate the predicate against the dict first.
Build a `HashSet<i64>` of matching indices, then check `chunk.data` as an integer —
no string decoding needed for the hot path.

---

### Phase 5 — API endpoint  `api/src/main.rs`

```
POST /query
Body: {
  "query": "from sales select date, amount, customers.name where amount > 100"
}
```

The endpoint:
1. Parses the query string
2. Applies any `relate` statements to the relationship registry
3. Executes the query against the live frame store
4. If `as <name>` is present, stores the result frame and returns its schema
5. Otherwise returns the data directly (paginated, same format as `GET /frames/:name/data`)

---

## File layout after implementation

```
df-store/src/
  resin/
    mod.rs
    lexer.rs      ← Phase 1
    ast.rs        ← Phase 2
    parser.rs     ← Phase 3
    executor.rs   ← Phase 4
  query/
    mod.rs
    relationship.rs
    resolver.rs   ← already used by executor
```

---

## Design decisions to keep in mind

**Relationships are global state.**
`relate` statements accumulate in the API's relationship registry (same as
`POST /relationships`). A `relate` in a Resin query is equivalent to calling
that endpoint — it persists for the session.

**Dict-aware evaluation is the goal, not a nice-to-have.**
The point of building this instead of using SQL is that `WHERE country = "France"`
should never decode every string — it should find the dict index for `"France"` once,
then scan integer chunks. The executor must be written with this in mind from the start.

**No mutation.**
Resin is read-only plus `as` (materialise). There is no `INSERT`, `UPDATE`, or
`DELETE`. Frames enter the store via connectors only.

**Error messages matter.**
Every `ParseError` and `ExecError` should include the source location and a
human-readable message. Bad error messages make a DSL unusable.

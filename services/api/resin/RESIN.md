# Resin

A query language for the Cedrus columnar store.
Since we build our own query engine, we can design a custom DSL that fits our data model and execution engine perfectly.
Syntax is rxjs-inspired, with a focus on readability, chainability, and ease of use for data analysts.

---

## Core concepts

### Frames
A frame is an immutable columnar dataset. Frames enter the store via connectors only — Resin never mutates source data. Every operation produces a new frame.

### Dimensions (materialized joins)
A dimension is a frame built by **relating** other frames together. Dimensions are materialized and stored for reuse. This is the data-modeling layer.

### Queries (ephemeral pipelines)
A query is a chain of operations that reads from frames or dimensions and produces a result. Queries can `join` frames at query time without materializing anything.

### Null model
Cedrus tracks nullability at two levels:
- **Column tag** — a per-column boolean indicating whether _any_ value in the column is null. Allows the executor to skip null-checking entirely on non-nullable columns.
- **Value tag** — a per-value bitmask (one bit per row in a chunk) indicating which individual values are null.

**Null propagation rules:**
| Context | Behaviour |
|---|---|
| Arithmetic (`+`, `-`, `*`, `/`) | `null` in → `null` out. `42 + null = null`. |
| Comparison (`=`, `!=`, `>`, …) | `null` in → `null` out (three-valued logic). |
| Boolean (`and`, `or`, `not`) | SQL-style: `true or null = true`, `false and null = false`, otherwise `null`. |
| `filter` | Rows where the predicate is `null` are **excluded** (same as `false`). |
| `relate` / `join` | Null keys **never match**. `null = null` is `false` in join conditions. |
| Aggregations (`sum`, `avg`, …) | Nulls are **skipped**. `sum(1, null, 3) = 4`. `count` counts non-null values; use `count(*)` for total rows. |
| `coalesce(a, b, …)` | Returns the first non-null argument. |
| `is null` / `is not null` | Explicit null test — returns `true`/`false`, never `null`. |
| `map` | `null` propagates through expressions by default. Use `coalesce` to override. |

**Dict encoding and null:** `null` is stored as a **dedicated dict index** (index `0` by convention). This keeps the hot-path integer scan uniform — no special-case branch for missing values.

---

## Language overview

### 1 — Importing frames

```resin
use sales from "sales"
use customers from "customers_icar"
use products from "products"
```

`use <n> from <source>` imports a frame from a data source and binds it to a local name.

---

### 2 — Building dimensions (`frame` + `relate`)

`relate` is exclusively for **materialized dimension-building**. It lives inside a `frame` block.

```resin
frame (
  relate sales -> customers
    on sales.customer_id = customers.id
    or (sales.customer_name = customers.name and sales.customer_city = customers.city)
    as customer_info

  relate sales -> products
    on sales.product_id = products.id
    as product_info
    where products.category = "electronics"
) as sales_enriched
```

- `relate <left> -> <right> on <condition> as <alias>` — declares a relationship and materializes the joined columns under the given alias.
- `where <condition>` on a `relate` filters the **right** frame _before_ joining (predicate pushdown).
- `as <n>` on the `frame` block materializes and stores the resulting dimension.

The resulting dimension `sales_enriched` can be queried like any other frame.

---

### 3 — Querying (pipe chains)

Queries are **chainable pipelines** that read from a frame and flow through operators left to right using `.`:

```resin
sales_enriched
  .filter(product_info.category = "electronics" and customer_info.country is not null)
  .map(product_info.price * sales.quantity as total)
  .map(coalesce(customer_info.discount, 0) as safe_discount)
  .map(total * (1 - safe_discount) as net_revenue)
  .aggregate(sum(net_revenue) as revenue, count(*) as order_count by customer_info.country)
  .sort_by(revenue desc)
  .limit(20)
```

Every step produces a new intermediate frame. Nothing is mutated.

#### Chain termination

A pipe chain ends when:
- The chain simply stops — the result is returned to the caller.
- `as <n>` is appended — the result is materialized and stored as a named frame.

There is no explicit `return` operator. End of chain = return.

#### Materializing query results

```resin
sales_enriched
  .filter(sales.date >= "2024-01-01")
  .select(sales.revenue, sales.date, customer_info.name)
  as french_sales
```

---

### 4 — Ephemeral joins

Joins are for **query-time** combinations that are not materialized. They live inside pipe chains.
Each join type is a separate operator for clarity:

```resin
-- inner join (default)
sales
  .join(returns on sales.id = returns.sale_id)
  .filter(returns.reason = "defective")
  .select(sales.product_id, returns.date)

-- left join: keep all sales, nulls where no return exists
sales
  .left_join(returns on sales.id = returns.sale_id)
  .map(coalesce(returns.reason, "no return") as return_status)
  .select(sales.id, return_status)

-- right join: keep all returns, nulls where no sale matches
returns
  .right_join(sales on returns.sale_id = sales.id)
  .filter(sales.id is null)
  .select(returns.id, returns.reason)

-- cross join: cartesian product
sizes
  .cross_join(colors)
  .select(sizes.label, colors.name)
```

Unlike `relate`, joins do not create stored dimensions — they only exist for the duration of the query.

**Join types:**
| Operator | Behaviour |
|---|---|
| `.join(B on cond)` | Inner join — only matching rows from both sides. |
| `.left_join(B on cond)` | All rows from left, nulls on right where no match. |
| `.right_join(B on cond)` | All rows from right, nulls on left where no match. |
| `.cross_join(B)` | Cartesian product — no `on` clause. |

All join operators follow null semantics: null keys never match.

---

### 5 — Window functions

Window functions operate over a partition of the result set without collapsing rows.

```resin
sales_enriched
  .window(
    row_number() over (partition_by customer_info.id sort_by sales.date asc) as purchase_rank
  )
  .window(
    sum(sales.revenue) over (
      partition_by customer_info.country
      sort_by sales.date asc
      rows between unbounded_preceding and current_row
    ) as running_revenue
  )
  .filter(purchase_rank <= 3)
```

**Syntax:**
```
.window(
  <function>() over (
    [partition_by <columns>]
    [sort_by <columns> [asc|desc]]
    [rows between <start> and <end>]
  ) as <alias>
)
```

**Supported window functions:**
| Function | Description |
|---|---|
| `row_number()` | Sequential number within partition. |
| `rank()` | Rank with gaps on ties. |
| `dense_rank()` | Rank without gaps on ties. |
| `lag(<col>, <offset>, <default>)` | Value from a previous row. |
| `lead(<col>, <offset>, <default>)` | Value from a following row. |
| `sum(<col>)` | Running / partitioned sum. |
| `avg(<col>)` | Running / partitioned average. |
| `min(<col>)` | Running / partitioned minimum. |
| `max(<col>)` | Running / partitioned maximum. |
| `count(<col>)` / `count(*)` | Running / partitioned count. |

**Frame specifiers for `rows between`:**
- `unbounded_preceding` — start of partition
- `current_row`
- `<n> preceding` / `<n> following`
- `unbounded_following` — end of partition

---

## Operator reference

### Pipe operators

| Operator | Syntax | Description |
|---|---|---|
| `filter` | `.filter(<predicate>)` | Keep rows where predicate is `true` (null → excluded). |
| `select` | `.select(<cols>)` | Project specific columns. Projection only — no computation, no renaming. Use `map` to compute or rename. |
| `map` | `.map(<expr> as <alias>)` | Create a new column from an expression. `as` is **required**. |
| `aggregate` | `.aggregate(<agg> as <alias>, … by <cols>)` | Group and aggregate in one step. `by` clause is optional — omit for global aggregation. |
| `distinct` | `.distinct()` | Remove duplicate rows based on all columns in the current frame. |
| `sort_by` | `.sort_by(<col> [asc\|desc], …)` | Sort results. |
| `limit` | `.limit(<n>)` | Cap the number of returned rows. |
| `offset` | `.offset(<n>)` | Skip the first `<n>` rows. Typically used with `sort_by` + `limit` for pagination. |
| `join` | `.join(<frame> on <cond>)` | Inner join — ephemeral, query-time. |
| `left_join` | `.left_join(<frame> on <cond>)` | Left outer join. |
| `right_join` | `.right_join(<frame> on <cond>)` | Right outer join. |
| `cross_join` | `.cross_join(<frame>)` | Cartesian product — no `on` clause. |
| `window` | `.window(<func> over (…) as <alias>)` | Window function. |
| `union` | `.union(<frame>)` | Combine rows, deduplicate. |
| `intersect` | `.intersect(<frame>)` | Rows present in both frames. |
| `except` | `.except(<frame>)` | Rows in current frame but not in the other. |

### `select` vs `map`

These two operators have **strictly separate responsibilities**:

- **`select`** picks existing columns. It does not compute, rename, or transform. `.select(a, b, c)` is pure projection.
- **`map`** creates new columns from expressions. It always requires `as` to name the result. `.map(a * b as total)` adds `total` to the frame.

To project _and_ rename, use `map` then `select`:
```resin
  .map(customer_info.name as customer_name)
  .select(customer_name, revenue, date)
```

### `aggregate` syntax

The `by` clause comes **after** the aggregation expressions:

```resin
-- grouped aggregation
.aggregate(sum(revenue) as total, count(*) as n by country, city)

-- global aggregation (no by clause)
.aggregate(sum(revenue) as total, count(*) as n)

-- multiple aggregations with expressions
.aggregate(
  sum(price * qty) as gross,
  avg(coalesce(discount, 0)) as avg_discount,
  count(*) as order_count
  by region, year(order_date)
)
```

The `by` clause accepts any expression, not just bare column names — `year(order_date)` is valid.
After `aggregate`, only the `by` columns and the aliased aggregations are available downstream.

### Expression aliasing

`as` works on any computed expression, not just frames:

```resin
.map(price * qty as total)
.map(total * tax_rate as tax_amount)
.aggregate(sum(revenue) as total_rev, avg(revenue) as avg_rev by country)
.window(row_number() over (...) as rn)
```

Aliases created by `map` are available in subsequent steps of the chain.

### Built-in functions

| Category | Functions |
|---|---|
| Aggregation | `sum`, `count`, `count(*)`, `avg`, `min`, `max` |
| Null handling | `coalesce(<a>, <b>, …)`, `is null`, `is not null` |
| String | TBD — `len`, `lower`, `upper`, `trim`, `contains`, `starts_with`, `ends_with` |
| Date/Time | TBD — `year`, `month`, `day`, `date_diff`, `date_add` |
| Type conversion | `to_int(<expr>)`, `to_float(<expr>)`, `to_string(<expr>)` |

> **Note on `cast`:** Traditional `cast(x as type)` syntax conflicts with `as` aliasing.
> Resin uses named conversion functions instead: `to_int(x)`, `to_float(x)`, `to_string(x)`.

---

## Restrictions

### No subqueries (v1)

Pipe chains cannot be nested inline. This is **not** valid:

```resin
-- INVALID: subquery inside filter
sales
  .filter(customer_id in (
    customers.filter(country = "France").select(id)
  ))
```

**Workaround:** materialize the inner query first, then join:

```resin
customers
  .filter(country = "France")
  .select(id)
  as french_customer_ids

sales
  .join(french_customer_ids on sales.customer_id = french_customer_ids.id)
```

Subqueries may be introduced in a future version. The AST and planner are designed to accommodate them without breaking changes.

---

## Grammar summary (EBNF sketch)

```ebnf
program        = statement* ;
statement      = use_stmt | frame_stmt | query_stmt ;

use_stmt       = "use" IDENT "from" STRING ;

frame_stmt     = "frame" "(" relate_clause+ ")" "as" IDENT ;
relate_clause  = "relate" IDENT "->" IDENT "on" condition "as" IDENT ("where" condition)? ;

query_stmt     = IDENT pipe_chain ("as" IDENT)? ;
pipe_chain     = ("." pipe_op)* ;
pipe_op        = filter_op | select_op | map_op | aggregate_op
               | distinct_op | sort_by_op | limit_op | offset_op
               | join_op | left_join_op | right_join_op | cross_join_op
               | window_op
               | union_op | intersect_op | except_op ;

filter_op      = "filter" "(" condition ")" ;
select_op      = "select" "(" column_list ")" ;
map_op         = "map" "(" expr "as" IDENT ")" ;
aggregate_op   = "aggregate" "(" agg_list ("by" column_list)? ")" ;
distinct_op    = "distinct" "()" ;
sort_by_op     = "sort_by" "(" sort_col ("," sort_col)* ")" ;
limit_op       = "limit" "(" INT ")" ;
offset_op      = "offset" "(" INT ")" ;
join_op        = "join" "(" IDENT "on" condition ")" ;
left_join_op   = "left_join" "(" IDENT "on" condition ")" ;
right_join_op  = "right_join" "(" IDENT "on" condition ")" ;
cross_join_op  = "cross_join" "(" IDENT ")" ;
window_op      = "window" "(" window_expr ")" ;
union_op       = "union" "(" IDENT ")" ;
intersect_op   = "intersect" "(" IDENT ")" ;
except_op      = "except" "(" IDENT ")" ;

agg_list       = agg_expr ("," agg_expr)* ;
agg_expr       = agg_fn "as" IDENT ;
agg_fn         = IDENT "(" (expr | "*") ")" ;

window_expr    = window_fn "over" "(" window_spec ")" "as" IDENT ;
window_fn      = IDENT "(" (expr | "*")? ("," expr)* ")" ;
window_spec    = ("partition_by" column_list)? ("sort_by" sort_col ("," sort_col)*)? (frame_spec)? ;
frame_spec     = "rows" "between" frame_bound "and" frame_bound ;
frame_bound    = "unbounded_preceding" | "current_row" | "unbounded_following"
               | INT ("preceding" | "following") ;

sort_col       = expr ("asc" | "desc")? ;

condition      = condition "and" condition
               | condition "or" condition
               | "not" condition
               | "(" condition ")"
               | expr comparator expr
               | expr "is" "null"
               | expr "is" "not" "null" ;

comparator     = "=" | "!=" | ">" | "<" | ">=" | "<=" ;

expr           = expr ("+" | "-" | "*" | "/") expr
               | IDENT "(" expr_list? ")"        -- function call
               | IDENT "." IDENT                 -- qualified column
               | IDENT                           -- bare column
               | literal
               | "(" expr ")" ;

literal        = INT | FLOAT | STRING | "true" | "false" | "null" ;
column_list    = expr ("," expr)* ;
expr_list      = expr ("," expr)* ;
```

> **Parser note:** The `expr` and `condition` rules are left-recursive as written.
> The recursive descent parser must use **precedence climbing** (or Pratt parsing) to handle
> operator precedence and associativity without left recursion. Precedence from lowest to highest:
> `or` → `and` → `not` → comparisons → `+`/`-` → `*`/`/` → unary → atoms.

---

## Implementation plan

### Phase 1 — Lexer  `resin/src/lexer.rs`

Tokenise a raw `&str` into a flat `Vec<Token>`.

**Token types:**
- **Keywords:** `use`, `from`, `frame`, `relate`, `on`, `as`, `where`, `filter`, `select`, `map`, `aggregate`, `by`, `distinct`, `sort_by`, `limit`, `offset`, `join`, `left_join`, `right_join`, `cross_join`, `window`, `over`, `partition_by`, `rows`, `between`, `union`, `intersect`, `except`, `and`, `or`, `not`, `is`, `null`, `asc`, `desc`, `true`, `false`, `coalesce`, `preceding`, `following`, `unbounded_preceding`, `unbounded_following`, `current_row`
- **Aggregate/window functions (recognized as identifiers):** `sum`, `count`, `avg`, `min`, `max`, `row_number`, `rank`, `dense_rank`, `lag`, `lead`
- **Type conversion functions (recognized as identifiers):** `to_int`, `to_float`, `to_string`
- **Symbols:** `->`, `.`, `,`, `*`, `(`, `)`, `=`, `!=`, `>`, `<`, `>=`, `<=`, `+`, `-`, `/`
- **Literals:** integer (`42`), float (`3.14`), string (`"hello"`), boolean (`true`/`false`), `null`
- **Identifier:** unquoted name (`sales`, `customer_id`)
- **Comment:** `--` to end of line → skip

**Output:** `Vec<Token>` where each token carries its kind and source span (line + col).

---

### Phase 2 — AST  `resin/src/ast.rs`

Define the full AST types matching the grammar above. Key nodes:
- `Program` → `Vec<Statement>`
- `Statement` → `Use` | `Frame` | `Query`
- `Query` → `{ source: Ident, ops: Vec<PipeOp>, materialize: Option<Ident> }`
- `PipeOp` → `Filter` | `Select` | `Map { expr, alias }` | `Aggregate { aggs, group_by }` | `Distinct` | `SortBy` | `Limit` | `Offset` | `Join { kind, frame, condition }` | `Window` | `Union` | `Intersect` | `Except`
- `JoinKind` → `Inner` | `Left` | `Right` | `Cross`
- `WindowExpr` → `{ func, partition_by, sort_by, frame_spec, alias }`
- `Expr` → `BinaryOp` | `FunctionCall` | `Column` | `Literal` | `IsNull` | `IsNotNull`

Every node carries a `Span` for error reporting.

---

### Phase 3 — Parser  `resin/src/parser.rs`

Hand-written **recursive descent** parser with **Pratt parsing** for expressions.
Takes `&[Token]` and a cursor, returns `Result<Program, ParseError>`.

**Operator precedence (lowest to highest):**
1. `or`
2. `and`
3. `not` (prefix)
4. `=`, `!=`, `>`, `<`, `>=`, `<=`, `is null`, `is not null`
5. `+`, `-`
6. `*`, `/`
7. Unary `-`
8. Function calls, column references, literals, parenthesized expressions

`ParseError` carries a message + source location (from token spans).

**Error recovery:** on a parse error, the parser skips tokens until the next statement
boundary (next `use`, `frame`, or bare identifier at column 0) and continues parsing.
This allows reporting **multiple errors** in a single pass — critical for a DSL aimed
at analysts who are not professional programmers.

---

### Phase 3.5 — Semantic analysis  `resin/src/resolver.rs`

Runs over the parsed AST _before_ execution:

1. **Frame resolution** — verify that every referenced frame/dimension exists in the store.
2. **Column resolution** — verify that every `a.b` reference points to a real column. After `map(x as total)`, register `total` as available for subsequent steps. After `aggregate`, only `by` columns and aliases are available.
3. **Type checking** — verify type compatibility on join conditions, arithmetic, comparisons. Emit a clear error if joining an int column on a string column.
4. **Null-awareness tagging** — annotate each expression node with whether it can produce null, based on input column tags. `left_join` marks all right-side columns as nullable. This lets the executor skip null-checking branches for guaranteed-non-null paths.
5. **Output:** a validated IR (essentially the same tree, but with resolved types and column references) or a `Vec<ResolveError>` with source locations.

---

### Phase 4 — Query planner  `resin/src/planner.rs`

Transform the validated IR into an **operator plan tree**:

```
Scan → Filter → Map → Aggregate → SortBy → Offset → Limit → Emit
```

Optimizations applied at this stage:
- **Predicate pushdown** — move filters as close to the scan as possible.
- **Dict-aware filter rewriting** — for `filter(country = "France")` on a dict-encoded column, resolve `"France"` to its dict index at plan time and replace the predicate with an integer comparison.
- **Projection pruning** — only read columns that are actually referenced downstream.
- **Window function ordering** — determine execution order when multiple windows have different partition/sort keys.
- **Distinct pushdown** — if `distinct` precedes `sort_by`, consider sort-based deduplication.

---

### Phase 5 — Executor  `resin/src/executor.rs`

Walk the plan tree and operate on in-memory columnar data.

**Dict-aware WHERE filtering:**
For dict-encoded columns, evaluate the predicate against the dict first. Build a `HashSet<i64>` of matching indices, then check `chunk.data` as an integer — no string decoding on the hot path.

**Null handling in the executor:**
1. Check the column tag first — if `has_null = false`, skip all null checks for that column.
2. If `has_null = true`, consult the per-value bitmask before reading each value.
3. For aggregations, maintain a separate non-null count to compute `avg` correctly.
4. For `left_join` / `right_join`, emit null-filled rows for non-matching sides and set appropriate value tags.

---

### Phase 6 — API endpoint  `api/src/main.rs`

```
POST /query
Body: {
  "query": "use sales from \"sales\" ..."
}
```

The endpoint:
1. Lexes + parses the query string.
2. Runs semantic analysis against the live frame store.
3. Builds and optimizes the query plan.
4. Executes and returns results.
5. If `as <n>` is present, stores the result frame and returns its schema.
6. Otherwise, returns the data directly (paginated via `offset`/`limit`, same format as `GET /frames/:name/data`).

---

## File layout

```
resin/src/
    mod.rs
    lexer.rs        ← Phase 1
    ast.rs          ← Phase 2
    parser.rs       ← Phase 3
    resolver.rs     ← Phase 3.5
    planner.rs      ← Phase 4
    executor.rs     ← Phase 5
  query/
    mod.rs
    relationship.rs
    window.rs
```

---

## Design principles

1. **Dict-aware evaluation is the goal, not a nice-to-have.**
   `WHERE country = "France"` should never decode every string — find the dict index once, then scan integer chunks.

2. **No mutation.**
   Resin is read-only plus `as` (materialize). No `INSERT`, `UPDATE`, or `DELETE`. Frames enter via connectors only.

3. **Null is explicit, not magical.**
   Null has a dedicated dict index. The column tag lets the executor skip null checks entirely on clean columns. Null never silently matches in joins. `left_join` / `right_join` explicitly introduce nullable columns.

4. **Error messages matter.**
   Every `ParseError`, `ResolveError`, and `ExecError` includes source location and a human-readable message. The parser recovers from errors to report multiple issues per pass.

5. **`relate` builds, `join` queries.**
   `relate` is for materialized dimension-building inside `frame` blocks. `join` / `left_join` / `right_join` / `cross_join` are for ephemeral query-time joins inside pipe chains. They are distinct operations with different semantics and lifetimes.

6. **Aliases are first-class.**
   `as` works on frames, `map` expressions, `aggregate` results, and `window` functions. Aliases created in one step are available in all subsequent steps of the chain.

7. **`select` projects, `map` computes.**
   `select` is pure column picking — no expressions, no renaming. `map` is for derived columns and always requires an alias. No ambiguity.

8. **No subqueries (v1).**
   Pipe chains cannot be nested inline. Materialize intermediate results with `as`, then reference them. The AST is designed to support subqueries in a future version without grammar-breaking changes.
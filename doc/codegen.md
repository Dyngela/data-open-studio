# Code Generation System

The code generation system transforms a visual node graph into a standalone Go program. Located in `api/internal/gen/`.

## Overview

```
Job (DB) → Load Nodes → Topological Sort → Generate Structs → Generate Functions → Assemble main.go → Compile → Execute
```

When a user clicks "Execute" on a job:
1. `JobService.Execute(id)` loads the job with all nodes
2. The generator traverses the node graph
3. For each node, a `NodeGenerator` produces struct definitions and function bodies
4. All pieces are assembled into `main.go` using the main template
5. The program is compiled with `go build` and executed
6. Progress is reported via NATS

## Generator Interface (`generator.go`)

```go
type NodeGenerator interface {
    NodeType() models.NodeType
    GenerateStructData(node *models.Node) (*StructData, error)
    GenerateFuncData(node *models.Node, ctx *GeneratorContext) (*NodeFunctionData, error)
    GetLaunchArgs(node *models.Node, channels []ChannelInfo, dbConnections map[string]string) []string
}
```

### GeneratorContext
Maintains state during generation:
```go
type GeneratorContext struct {
    NodeStructNames map[int]string    // node ID -> "Node1Row"
    NodeFuncNames   map[int]string    // node ID -> "executeNode1"
    Imports         map[string]string // import path -> alias
}
```
Methods: `AddImport(path)`, `AddImportAlias(alias, path)`, `StructName(node)`, `FuncName(node)`.

### Registry
```go
// Registered in init():
var DefaultRegistry = NewRegistry()

func init() {
    RegisterGenerator(&DBInputGenerator{})
    RegisterGenerator(&DBOutputGenerator{})
    RegisterGenerator(&MapGenerator{})
    RegisterGenerator(&LogGenerator{})
    RegisterGenerator(&EmailOutputGenerator{})
}
```

## Template Data Structures (`template_data.go`)

### TemplateData (root structure for main.go.tmpl)
```go
type TemplateData struct {
    Imports       []ImportData         // Package imports
    Structs       []StructData         // Row type structs
    NodeFunctions []NodeFunctionData   // Function implementations
    DBConnections []DBConnectionData   // Database connections to open
    Channels      []ChannelData        // Inter-node channels
    NodeLaunches  []NodeLaunchData     // Goroutine launch config
    NodeCount     int
    UseFlags      bool                 // CLI flags for NATS config
    NatsURL       string
    TenantID      string
    JobID         uint
}
```

### StructData
```go
type StructData struct {
    Name   string       // "Node1Row"
    NodeID int
    Fields []FieldData
}

type FieldData struct {
    Name string          // "ID", "Amount" (PascalCase)
    Type string          // "int", "string", "float64", "time.Time"
    Tag  string          // `db:"column_name"` or `json:"column_name"`
}
```

### NodeFunctionData
```go
type NodeFunctionData struct {
    Name      string    // "executeNode1"
    NodeID    int
    NodeName  string    // Display name
    Signature string    // (sometimes empty)
    Body      string    // Complete function implementation
}
```

### DBConnectionData
```go
type DBConnectionData struct {
    ID         string   // "conn1"
    Driver     string   // "postgres", "mysql", "sqlserver"
    ConnString string   // Full DSN
}
```

### ChannelData
```go
type ChannelData struct {
    PortID     uint
    FromNodeID int
    ToNodeID   int
    RowType    string   // "Node1Row"
    BufferSize int
}
```

### NodeLaunchData
```go
type NodeLaunchData struct {
    NodeID           int
    NodeName         string
    FuncName         string    // "executeNode1"
    Args             []string  // ["db_conn1", "ch_1"]
    HasOutputChannel bool
    OutputChannel    string    // "ch_1"
}
```

## Node Generators

### DBInputGenerator (`node_db_input.go`)

**GenerateStructData**: Creates a struct from `DataModels` (column schema).
- Maps each DataModel to a FieldData with `GoFieldName()` (PascalCase) and `GoFieldType()`
- Tags: `db:"original_column_name"`

**GenerateFuncData**: Renders `node_db_input.go.tmpl`.
- Calls `config.EnforceSchema()` to add `SELECT ... FROM (query) AS sub LIMIT 0` for schema detection
- Adds imports: context, database/sql, fmt, lib, driver import

**GetLaunchArgs**: Returns `["db_<connectionID>", "ch_<outputPortID>"]`

### DBOutputGenerator (`node_db_output.go`)

**GenerateStructData**: Returns `nil` (sink node, no output struct).

**GenerateFuncData**: Renders `node_db_output_insert.go.tmpl`.
- Currently only supports `insert` mode
- Batch INSERT with parameterized queries
- Schema-qualified table names

**GetLaunchArgs**: Returns `["db_<connectionID>", "ch_<inputPortID>"]`

### MapGenerator (`node_map.go`) - 645 lines

The most complex generator. Handles both single-input transforms and multi-input joins.

**GenerateStructData**: Creates output struct from `config.Outputs[0].Columns`.
- Maps DataType -> Go type: `int`, `int64`, `float64`, `bool`, `string`, `time.Time`, `any`
- PascalCase field names, JSON tags

**GenerateFuncData**: Routes based on input count:

#### Single Input (Transform)
- Template: `node_map_transform.go.tmpl`
- Generates assignment code via `buildTransformCode()`:
  - **Direct** (`funcType: "direct"`): `out.Field = row.SourceField`
  - **Library** (`funcType: "library"`): `out.Field = lib.FuncName(args...)`
  - **Custom** (`funcType: "custom"`): `out.Field = expression` (with variable substitution)

#### Multiple Inputs (Join)
Routes by `config.Join.Type`:

| Join Type | Template | Logic |
|-----------|----------|-------|
| `inner` | `node_map_inner_join.go.tmpl` | Build right index, match left keys |
| `left` | `node_map_left_join.go.tmpl` | All left rows, optional right match |
| `right` | `node_map_right_join.go.tmpl` | All right rows, optional left match |
| `cross` | `node_map_cross_join.go.tmpl` | Cartesian product |
| `union` | `node_map_union.go.tmpl` | Sequential concatenation |

Join transforms use `buildJoinTransformCode()`:
- References like `left.field` -> `leftRow.PascalField`
- References like `right.field` -> `rightRow.PascalField`
- Nil-safe wrappers for outer joins (dereference pointers)

**Helper functions**:
- `toPascalCase(s)` - snake_case -> PascalCase
- `extractFieldName(ref)` - Parse "input.field" reference
- `parseInputRef(ref)` - Split into (inputName, fieldName)
- `buildColumnExpression()` - Generate Go expression for a column
- `buildFuncArgs()` - Build library function arguments
- `substituteExprVars()` / `substituteJoinExprVars()` - Replace input.field with Go field access
- `findInputRowTypes()` - Map input names to struct type names
- `getJoinKey()` - Extract join key field name
- `getZeroValue(goType)` - Default value for a type

### LogGenerator (`node_log.go`)

**GenerateStructData**: Returns `nil` (sink node).

**GenerateFuncData**: Renders `node_log.go.tmpl`.
- Finds input row type from connected upstream node
- Simple loop printing each row

**GetLaunchArgs**: Returns `["ch_<inputPortID>"]`

### EmailOutputGenerator (`node_email_output.go`)

**GenerateStructData**: Returns `nil` (sink node).

**GenerateFuncData**: Renders `node_email_output.go.tmpl`.
- Maps config to EmailOutputTemplateData
- Joins To/CC/BCC arrays with ", "
- Subject and Body are Go templates (e.g., `"Order {{.OrderID}}"`)
- Adds imports: context, fmt, bytes, text/template, lib, go-mail

**GetLaunchArgs**: Returns `["ch_<inputPortID>"]`

## Templates (`gen/templates/`)

### main.go.tmpl
The master template that assembles the complete program:
```go
package main

import (
    {{range .Imports}}"{{.Path}}"{{end}}
)

// Struct definitions
{{range .Structs}}
type {{.Name}} struct { ... }
{{end}}

// Node functions
{{range .NodeFunctions}}
{{.Body}}
{{end}}

// Orchestrator
func execute(ctx context.Context, progress lib.ProgressFunc) error {
    // Open DB connections
    // Create buffered channels
    // Launch goroutines (one per node)
    // Wait for all to complete
    // Return first error
}

func main() {
    // Signal handling (SIGINT, SIGTERM)
    // NATS progress reporter setup
    // Call execute()
}
```

### node_db_input.go.tmpl
```
func {{.FuncName}}(ctx, db, outChan, progress) error {
    rows, err := db.QueryContext(ctx, "{{.Query}}")
    for rows.Next() {
        row := &{{.StructName}}{}
        rows.Scan(&row.Field1, &row.Field2, ...)
        outChan <- row
        // Progress every 1000 rows
    }
    close(outChan)
}
```

### node_db_output_insert.go.tmpl
```
func {{.FuncName}}(ctx, db, inChan, progress) error {
    batch := make([]*{{.InputType}}, 0, {{.BatchSize}})
    for row := range inChan {
        batch = append(batch, row)
        if len(batch) >= {{.BatchSize}} {
            // Build INSERT INTO {{.TableName}} ({{.ColumnNames}}) VALUES ($1,$2,...),($N+1,...)
            // Execute with flattened args
            batch = batch[:0]
        }
    }
    // Flush remaining
}
```

### node_map_transform.go.tmpl
```
func {{.FuncName}}(ctx, inChan, outChan, progress) error {
    for row := range inChan {
        out := &{{.OutputType}}{}
        {{.Transforms}}   // out.Name = row.Name; out.Total = lib.Add(row.A, row.B)
        outChan <- out
    }
    close(outChan)
}
```

### node_map_inner_join.go.tmpl
```
func {{.FuncName}}(ctx, leftChan, rightChan, outChan, progress) error {
    // Build right index: map[key][]rightRow
    rightIndex := map[interface{}][]*{{.RightType}}{}
    for r := range rightChan { rightIndex[r.{{.RightKey}}] = append(...) }
    // Process left, emit matches
    for l := range leftChan {
        if matches, ok := rightIndex[l.{{.LeftKey}}]; ok {
            for _, r := range matches {
                out := &{{.OutputType}}{}
                {{.Transforms}}
                outChan <- out
            }
        }
    }
    close(outChan)
}
```

### node_map_left_join.go.tmpl
Same as inner but emits left rows even without right match (nil-safe right field access).

### node_map_right_join.go.tmpl
Builds left index, iterates right, emits all right rows with optional left match.

### node_map_cross_join.go.tmpl
Collects all right rows, then for each left row emits with each right row.

### node_map_union.go.tmpl
```
func {{.FuncName}}(ctx, leftChan, rightChan, outChan, progress) error {
    for row := range leftChan {
        out := &{{.OutputType}}{}
        {{.LeftTransforms}}
        outChan <- out
    }
    for row := range rightChan {
        out := &{{.OutputType}}{}
        {{.RightTransforms}}
        outChan <- out
    }
    close(outChan)
}
```

### node_log.go.tmpl
```
func {{.FuncName}}(ctx, inChan, progress) error {
    for row := range inChan { fmt.Printf("%+v\n", row) }
}
```

### node_email_output.go.tmpl
```
func {{.FuncName}}(ctx, inChan, progress) error {
    subjectTpl := template.Must(template.New("subject").Parse("{{.Subject}}"))
    bodyTpl := template.Must(template.New("body").Parse("{{.Body}}"))
    for row := range inChan {
        // Render templates with row data
        // Create go-mail message
        // Connect to SMTP (TLS policy based on .UseTLS)
        // Send
    }
}
```

## Runtime Library (`gen/lib/`)

### progress.go
```go
type Status string   // "running" | "completed" | "failed"

type Progress struct {
    NodeID   int    `json:"nodeId"`
    NodeName string `json:"nodeName"`
    Status   Status `json:"status"`
    RowCount int64  `json:"rowCount"`
    Message  string `json:"message"`
}

type ProgressFunc func(Progress)

type ProgressReporter struct { conn *nats.Conn; subject string; noop bool }
NewProgressReporter(natsURL, tenantID, jobID) *ProgressReporter
ReportFunc() ProgressFunc   // Returns callback that publishes to NATS
Close()                      // Drain NATS connection
```

NATS subject: `tenant.<tenantID>.job.<jobID>.progress`

## Adding a New Node Type

1. **Model config**: Create `models/node_<type>_config.go` with config struct
2. **Add to NodeType enum**: In `models/nodes.go`, add constant and `GetTyped<Type>Config()` method
3. **Generator**: Create `gen/node_<type>.go` implementing `NodeGenerator` interface:
   - `NodeType()` returns new constant
   - `GenerateStructData()` returns output struct (or nil for sinks)
   - `GenerateFuncData()` renders template into function body
   - `GetLaunchArgs()` returns channel/DB variable names
4. **Template**: Create `gen/templates/node_<type>.go.tmpl` with function body
5. **Register**: Add `RegisterGenerator(&<Type>Generator{})` in `gen/generator.go` `init()`
6. **Frontend**: Add node definition in `front/src/nodes/<type>/definition.ts`
7. **Frontend modal**: Create configuration modal component
8. **Register in NodeRegistry**: Add to `node-registry.service.ts`

package ir

// Node is the base interface for all IR nodes
type Node interface {
	irNode()
}

// Expr represents an expression
type Expr interface {
	Node
	irExpr()
}

// Stmt represents a statement
type Stmt interface {
	Node
	irStmt()
}

// File represents a Go source file
type File struct {
	Package string
	Imports []Import
	Decls   []Decl
}

func (File) irNode() {}

// Import represents an import declaration
type Import struct {
	Alias string // empty for no alias, "." for dot import
	Path  string
}

// Decl is a top-level declaration
type Decl interface {
	Node
	irDecl()
}

// StructDecl represents a struct type declaration
type StructDecl struct {
	Name   string
	Fields []FieldDef
}

func (StructDecl) irNode() {}
func (StructDecl) irDecl() {}

// FieldDef represents a struct field
type FieldDef struct {
	Name string
	Type string
	Tag  string // e.g., `json:"name"`
}

// FuncDecl represents a function declaration
type FuncDecl struct {
	Receiver   *Param // nil for non-method
	Name       string
	Params     []Param
	Results    []Param
	Body       []Stmt
	IsVariadic bool
}

func (FuncDecl) irNode() {}
func (FuncDecl) irDecl() {}

// Param represents a function parameter or return value
type Param struct {
	Name string // can be empty for unnamed returns
	Type string
}

// VarDecl represents a var declaration: var name Type = value
type VarDecl struct {
	Name  string
	Type  string // can be empty for type inference
	Value Expr   // can be nil
}

func (VarDecl) irNode() {}
func (VarDecl) irStmt() {}

// AssignStmt represents assignment: lhs = rhs or lhs := rhs
type AssignStmt struct {
	Left   []Expr
	Right  []Expr
	Define bool // true for :=, false for =
}

func (AssignStmt) irNode() {}
func (AssignStmt) irStmt() {}

// IfStmt represents an if statement
type IfStmt struct {
	Init Stmt // optional init statement
	Cond Expr
	Then []Stmt
	Else []Stmt // can be empty, or contain single IfStmt for else-if
}

func (IfStmt) irNode() {}
func (IfStmt) irStmt() {}

// ForStmt represents a for loop
type ForStmt struct {
	Init Stmt // optional
	Cond Expr // optional, nil = infinite loop
	Post Stmt // optional
	Body []Stmt
}

func (ForStmt) irNode() {}
func (ForStmt) irStmt() {}

// RangeStmt represents a for-range loop
type RangeStmt struct {
	Key    string // can be "_" or empty
	Value  string // can be "_" or empty
	X      Expr   // expression to range over
	Define bool   // true for :=, false for =
	Body   []Stmt
}

func (RangeStmt) irNode() {}
func (RangeStmt) irStmt() {}

// ReturnStmt represents a return statement
type ReturnStmt struct {
	Values []Expr
}

func (ReturnStmt) irNode() {}
func (ReturnStmt) irStmt() {}

// DeferStmt represents a defer statement
type DeferStmt struct {
	Call *CallExpr
}

func (DeferStmt) irNode() {}
func (DeferStmt) irStmt() {}

// ExprStmt wraps an expression as a statement
type ExprStmt struct {
	X Expr
}

func (ExprStmt) irNode() {}
func (ExprStmt) irStmt() {}

// SendStmt represents a channel send: ch <- value
type SendStmt struct {
	Chan  Expr
	Value Expr
}

func (SendStmt) irNode() {}
func (SendStmt) irStmt() {}

// GoStmt represents a go statement
type GoStmt struct {
	Call *CallExpr
}

func (GoStmt) irNode() {}
func (GoStmt) irStmt() {}

// BlockStmt represents a block of statements
type BlockStmt struct {
	Stmts []Stmt
}

func (BlockStmt) irNode() {}
func (BlockStmt) irStmt() {}

// Ident represents an identifier
type Ident struct {
	Name string
}

func (Ident) irNode() {}
func (Ident) irExpr() {}

// Literal represents a literal value
type Literal struct {
	Value any    // string, int, float64, bool, nil
	Kind  string // "string", "int", "float", "bool", "nil"
}

func (Literal) irNode() {}
func (Literal) irExpr() {}

// CallExpr represents a function call
type CallExpr struct {
	Func     Expr
	Args     []Expr
	Variadic bool // true if last arg is ...
}

func (CallExpr) irNode() {}
func (CallExpr) irExpr() {}

// SelectorExpr represents a.b
type SelectorExpr struct {
	X   Expr
	Sel string
}

func (SelectorExpr) irNode() {}
func (SelectorExpr) irExpr() {}

// IndexExpr represents a[i]
type IndexExpr struct {
	X     Expr
	Index Expr
}

func (IndexExpr) irNode() {}
func (IndexExpr) irExpr() {}

// UnaryExpr represents a unary expression: &x, *x, !x, -x
type UnaryExpr struct {
	Op string // "&", "*", "!", "-", "<-"
	X  Expr
}

func (UnaryExpr) irNode() {}
func (UnaryExpr) irExpr() {}

// BinaryExpr represents a binary expression: x + y, x == y, etc.
type BinaryExpr struct {
	X  Expr
	Op string // "+", "-", "*", "/", "==", "!=", "<", ">", "<=", ">=", "&&", "||"
	Y  Expr
}

func (BinaryExpr) irNode() {}
func (BinaryExpr) irExpr() {}

// CompositeLit represents a composite literal: Type{...}
type CompositeLit struct {
	Type     string // e.g., "[]string", "MyStruct", "map[string]int"
	Elements []Expr // can be KeyValueExpr for maps/structs
}

func (CompositeLit) irNode() {}
func (CompositeLit) irExpr() {}

// KeyValueExpr represents key: value in composite literals
type KeyValueExpr struct {
	Key   Expr
	Value Expr
}

func (KeyValueExpr) irNode() {}
func (KeyValueExpr) irExpr() {}

// SliceExpr represents a[low:high] or a[low:high:max]
type SliceExpr struct {
	X    Expr
	Low  Expr // can be nil
	High Expr // can be nil
	Max  Expr // can be nil
}

func (SliceExpr) irNode() {}
func (SliceExpr) irExpr() {}

// TypeAssertExpr represents x.(Type)
type TypeAssertExpr struct {
	X    Expr
	Type string
}

func (TypeAssertExpr) irNode() {}
func (TypeAssertExpr) irExpr() {}

// FuncLit represents a function literal (closure)
type FuncLit struct {
	Params  []Param
	Results []Param
	Body    []Stmt
}

func (FuncLit) irNode() {}
func (FuncLit) irExpr() {}

// RawExpr allows inserting raw Go code (escape hatch)
type RawExpr struct {
	Code string
}

func (RawExpr) irNode() {}
func (RawExpr) irExpr() {}

// RawStmt allows inserting raw Go code as a statement
type RawStmt struct {
	Code string
}

func (RawStmt) irNode() {}
func (RawStmt) irStmt() {}

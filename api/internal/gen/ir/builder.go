package ir

import "fmt"

// FileBuilder builds a File
type FileBuilder struct {
	file *File
}

// NewFile creates a new file builder
func NewFile(pkg string) *FileBuilder {
	return &FileBuilder{
		file: &File{
			Package: pkg,
			Imports: make([]Import, 0),
			Decls:   make([]Decl, 0),
		},
	}
}

// Import adds an import
func (b *FileBuilder) Import(path string) *FileBuilder {
	b.file.Imports = append(b.file.Imports, Import{Path: path})
	return b
}

// ImportAlias adds an aliased import
func (b *FileBuilder) ImportAlias(alias, path string) *FileBuilder {
	b.file.Imports = append(b.file.Imports, Import{Alias: alias, Path: path})
	return b
}

// AddDecl adds a declaration
func (b *FileBuilder) AddDecl(d Decl) *FileBuilder {
	b.file.Decls = append(b.file.Decls, d)
	return b
}

// AddStruct adds a struct declaration
func (b *FileBuilder) AddStruct(s *StructDecl) *FileBuilder {
	b.file.Decls = append(b.file.Decls, s)
	return b
}

// AddFunc adds a function declaration
func (b *FileBuilder) AddFunc(f *FuncDecl) *FileBuilder {
	b.file.Decls = append(b.file.Decls, f)
	return b
}

// Build returns the completed file
func (b *FileBuilder) Build() *File {
	return b.file
}

// StructBuilder builds a struct declaration
type StructBuilder struct {
	decl *StructDecl
}

// NewStruct creates a new struct builder
func NewStruct(name string) *StructBuilder {
	return &StructBuilder{
		decl: &StructDecl{
			Name:   name,
			Fields: make([]FieldDef, 0),
		},
	}
}

// Field adds a field
func (b *StructBuilder) Field(name, typ string) *StructBuilder {
	b.decl.Fields = append(b.decl.Fields, FieldDef{Name: name, Type: typ})
	return b
}

// FieldWithTag adds a field with a tag
func (b *StructBuilder) FieldWithTag(name, typ, tag string) *StructBuilder {
	b.decl.Fields = append(b.decl.Fields, FieldDef{Name: name, Type: typ, Tag: tag})
	return b
}

// Build returns the completed struct
func (b *StructBuilder) Build() *StructDecl {
	return b.decl
}

// FuncBuilder builds a function declaration
type FuncBuilder struct {
	decl *FuncDecl
}

// NewFunc creates a new function builder
func NewFunc(name string) *FuncBuilder {
	return &FuncBuilder{
		decl: &FuncDecl{
			Name:    name,
			Params:  make([]Param, 0),
			Results: make([]Param, 0),
			Body:    make([]Stmt, 0),
		},
	}
}

// Receiver sets the receiver for a method
func (b *FuncBuilder) Receiver(name, typ string) *FuncBuilder {
	b.decl.Receiver = &Param{Name: name, Type: typ}
	return b
}

// Param adds a parameter
func (b *FuncBuilder) Param(name, typ string) *FuncBuilder {
	b.decl.Params = append(b.decl.Params, Param{Name: name, Type: typ})
	return b
}

// Params adds multiple parameters
func (b *FuncBuilder) Params(params ...Param) *FuncBuilder {
	b.decl.Params = append(b.decl.Params, params...)
	return b
}

// Variadic marks the last parameter as variadic
func (b *FuncBuilder) Variadic() *FuncBuilder {
	b.decl.IsVariadic = true
	return b
}

// Returns adds return type(s)
func (b *FuncBuilder) Returns(types ...string) *FuncBuilder {
	for _, t := range types {
		b.decl.Results = append(b.decl.Results, Param{Type: t})
	}
	return b
}

// NamedReturns adds named return values
func (b *FuncBuilder) NamedReturns(params ...Param) *FuncBuilder {
	b.decl.Results = append(b.decl.Results, params...)
	return b
}

// Body sets the function body
func (b *FuncBuilder) Body(stmts ...Stmt) *FuncBuilder {
	b.decl.Body = stmts
	return b
}

// AddStmt adds a statement to the body
func (b *FuncBuilder) AddStmt(stmt Stmt) *FuncBuilder {
	b.decl.Body = append(b.decl.Body, stmt)
	return b
}

// Build returns the completed function
func (b *FuncBuilder) Build() *FuncDecl {
	return b.decl
}

// Expression builders

// Id creates an identifier
func Id(name string) *Ident {
	return &Ident{Name: name}
}

// Lit creates a literal
func Lit(value any) *Literal {
	switch v := value.(type) {
	case string:
		return &Literal{Value: v, Kind: "string"}
	case int:
		return &Literal{Value: v, Kind: "int"}
	case int64:
		return &Literal{Value: v, Kind: "int"}
	case float64:
		return &Literal{Value: v, Kind: "float"}
	case bool:
		return &Literal{Value: v, Kind: "bool"}
	case nil:
		return &Literal{Value: nil, Kind: "nil"}
	default:
		return &Literal{Value: v, Kind: "unknown"}
	}
}

// Nil creates a nil literal
func Nil() *Literal {
	return &Literal{Value: nil, Kind: "nil"}
}

// True creates a true literal
func True() *Literal {
	return &Literal{Value: true, Kind: "bool"}
}

// False creates a false literal
func False() *Literal {
	return &Literal{Value: false, Kind: "bool"}
}

// Call creates a function call
func Call(fn string, args ...Expr) *CallExpr {
	return &CallExpr{
		Func: parseExpr(fn),
		Args: args,
	}
}

// CallExprOn creates a method call on an expression
func CallOn(receiver Expr, method string, args ...Expr) *CallExpr {
	return &CallExpr{
		Func: &SelectorExpr{X: receiver, Sel: method},
		Args: args,
	}
}

// CallVariadic creates a variadic function call
func CallVariadic(fn string, args ...Expr) *CallExpr {
	return &CallExpr{
		Func:     parseExpr(fn),
		Args:     args,
		Variadic: true,
	}
}

// Sel creates a selector expression: x.name
func Sel(x Expr, name string) *SelectorExpr {
	return &SelectorExpr{X: x, Sel: name}
}

// Dot creates a chained selector from a string like "foo.bar.baz"
func Dot(path string) Expr {
	return parseExpr(path)
}

// Index creates an index expression: x[i]
func Index(x Expr, index Expr) *IndexExpr {
	return &IndexExpr{X: x, Index: index}
}

// Addr creates an address-of expression: &x
func Addr(x Expr) *UnaryExpr {
	return &UnaryExpr{Op: "&", X: x}
}

// Deref creates a dereference expression: *x
func Deref(x Expr) *UnaryExpr {
	return &UnaryExpr{Op: "*", X: x}
}

// Not creates a not expression: !x
func Not(x Expr) *UnaryExpr {
	return &UnaryExpr{Op: "!", X: x}
}

// Neg creates a negation expression: -x
func Neg(x Expr) *UnaryExpr {
	return &UnaryExpr{Op: "-", X: x}
}

// Recv creates a channel receive expression: <-ch
func Recv(ch Expr) *UnaryExpr {
	return &UnaryExpr{Op: "<-", X: ch}
}

// Binary operators

func Add(x, y Expr) *BinaryExpr      { return &BinaryExpr{X: x, Op: "+", Y: y} }
func Sub(x, y Expr) *BinaryExpr      { return &BinaryExpr{X: x, Op: "-", Y: y} }
func Mul(x, y Expr) *BinaryExpr      { return &BinaryExpr{X: x, Op: "*", Y: y} }
func Div(x, y Expr) *BinaryExpr      { return &BinaryExpr{X: x, Op: "/", Y: y} }
func Mod(x, y Expr) *BinaryExpr      { return &BinaryExpr{X: x, Op: "%", Y: y} }
func Eq(x, y Expr) *BinaryExpr       { return &BinaryExpr{X: x, Op: "==", Y: y} }
func Neq(x, y Expr) *BinaryExpr      { return &BinaryExpr{X: x, Op: "!=", Y: y} }
func Lt(x, y Expr) *BinaryExpr       { return &BinaryExpr{X: x, Op: "<", Y: y} }
func Gt(x, y Expr) *BinaryExpr       { return &BinaryExpr{X: x, Op: ">", Y: y} }
func Lte(x, y Expr) *BinaryExpr      { return &BinaryExpr{X: x, Op: "<=", Y: y} }
func Gte(x, y Expr) *BinaryExpr      { return &BinaryExpr{X: x, Op: ">=", Y: y} }
func And(x, y Expr) *BinaryExpr      { return &BinaryExpr{X: x, Op: "&&", Y: y} }
func Or(x, y Expr) *BinaryExpr       { return &BinaryExpr{X: x, Op: "||", Y: y} }
func BitAnd(x, y Expr) *BinaryExpr   { return &BinaryExpr{X: x, Op: "&", Y: y} }
func BitOr(x, y Expr) *BinaryExpr    { return &BinaryExpr{X: x, Op: "|", Y: y} }
func BitXor(x, y Expr) *BinaryExpr   { return &BinaryExpr{X: x, Op: "^", Y: y} }
func ShiftL(x, y Expr) *BinaryExpr   { return &BinaryExpr{X: x, Op: "<<", Y: y} }
func ShiftR(x, y Expr) *BinaryExpr   { return &BinaryExpr{X: x, Op: ">>", Y: y} }

// Composite creates a composite literal
func Composite(typ string, elems ...Expr) *CompositeLit {
	return &CompositeLit{Type: typ, Elements: elems}
}

// KV creates a key-value pair for composite literals
func KV(key, value Expr) *KeyValueExpr {
	return &KeyValueExpr{Key: key, Value: value}
}

// Slice creates a slice expression
func Slice(x Expr, low, high Expr) *SliceExpr {
	return &SliceExpr{X: x, Low: low, High: high}
}

// Assert creates a type assertion
func Assert(x Expr, typ string) *TypeAssertExpr {
	return &TypeAssertExpr{X: x, Type: typ}
}

// Closure creates a function literal
func Closure(params []Param, results []Param, body ...Stmt) *FuncLit {
	return &FuncLit{
		Params:  params,
		Results: results,
		Body:    body,
	}
}

// ClosureCall creates an immediately invoked function expression (IIFE)
// e.g., func() { ... }()
func ClosureCall(params []Param, results []Param, body ...Stmt) *CallExpr {
	return &CallExpr{
		Func: &FuncLit{
			Params:  params,
			Results: results,
			Body:    body,
		},
		Args: nil,
	}
}

// Raw creates a raw expression (escape hatch)
func Raw(code string) *RawExpr {
	return &RawExpr{Code: code}
}

// Rawf creates a formatted raw expression
func Rawf(format string, args ...any) *RawExpr {
	return &RawExpr{Code: fmt.Sprintf(format, args...)}
}

// Statement builders

// Var creates a variable declaration
func Var(name, typ string) *VarDecl {
	return &VarDecl{Name: name, Type: typ}
}

// VarInit creates a variable declaration with initialization
func VarInit(name, typ string, value Expr) *VarDecl {
	return &VarDecl{Name: name, Type: typ, Value: value}
}

// Assign creates an assignment statement
func Assign(left Expr, right Expr) *AssignStmt {
	return &AssignStmt{
		Left:   []Expr{left},
		Right:  []Expr{right},
		Define: false,
	}
}

// AssignMulti creates a multi-value assignment
func AssignMulti(left []Expr, right []Expr) *AssignStmt {
	return &AssignStmt{Left: left, Right: right, Define: false}
}

// Define creates a short variable declaration (:=)
func Define(left Expr, right Expr) *AssignStmt {
	return &AssignStmt{
		Left:   []Expr{left},
		Right:  []Expr{right},
		Define: true,
	}
}

// DefineMulti creates a multi-value short declaration
func DefineMulti(left []Expr, right []Expr) *AssignStmt {
	return &AssignStmt{Left: left, Right: right, Define: true}
}

// DefineN creates a short declaration with named identifiers
func DefineN(names []string, right ...Expr) *AssignStmt {
	left := make([]Expr, len(names))
	for i, name := range names {
		left[i] = Id(name)
	}
	return &AssignStmt{Left: left, Right: right, Define: true}
}

// If creates an if statement
func If(cond Expr, then ...Stmt) *IfStmt {
	return &IfStmt{Cond: cond, Then: then}
}

// IfElse creates an if-else statement
func IfElse(cond Expr, then []Stmt, els []Stmt) *IfStmt {
	return &IfStmt{Cond: cond, Then: then, Else: els}
}

// IfInit creates an if statement with init
func IfInit(init Stmt, cond Expr, then ...Stmt) *IfStmt {
	return &IfStmt{Init: init, Cond: cond, Then: then}
}

// For creates a for loop with condition only
func For(cond Expr, body ...Stmt) *ForStmt {
	return &ForStmt{Cond: cond, Body: body}
}

// ForClassic creates a classic for loop
func ForClassic(init Stmt, cond Expr, post Stmt, body ...Stmt) *ForStmt {
	return &ForStmt{Init: init, Cond: cond, Post: post, Body: body}
}

// ForInfinite creates an infinite for loop
func ForInfinite(body ...Stmt) *ForStmt {
	return &ForStmt{Body: body}
}

// Range creates a for-range loop
func Range(key, value string, x Expr, body ...Stmt) *RangeStmt {
	return &RangeStmt{
		Key:    key,
		Value:  value,
		X:      x,
		Define: true,
		Body:   body,
	}
}

// RangeValue creates a for-range loop with only value
func RangeValue(value string, x Expr, body ...Stmt) *RangeStmt {
	return &RangeStmt{
		Key:    "_",
		Value:  value,
		X:      x,
		Define: true,
		Body:   body,
	}
}

// RangeKey creates a for-range loop with only key
func RangeKey(key string, x Expr, body ...Stmt) *RangeStmt {
	return &RangeStmt{
		Key:    key,
		X:      x,
		Define: true,
		Body:   body,
	}
}

// Return creates a return statement
func Return(values ...Expr) *ReturnStmt {
	return &ReturnStmt{Values: values}
}

// Defer creates a defer statement
func Defer(call *CallExpr) *DeferStmt {
	return &DeferStmt{Call: call}
}

// ExprStatement wraps an expression as a statement
func ExprStatement(e Expr) *ExprStmt {
	return &ExprStmt{X: e}
}

// Send creates a channel send statement
func Send(ch Expr, value Expr) *SendStmt {
	return &SendStmt{Chan: ch, Value: value}
}

// Go creates a go statement
func Go(call *CallExpr) *GoStmt {
	return &GoStmt{Call: call}
}

// Block creates a block statement
func Block(stmts ...Stmt) *BlockStmt {
	return &BlockStmt{Stmts: stmts}
}

// RawStatement creates a raw statement (escape hatch)
func RawStatement(code string) *RawStmt {
	return &RawStmt{Code: code}
}

// RawStatementf creates a formatted raw statement
func RawStatementf(format string, args ...any) *RawStmt {
	return &RawStmt{Code: fmt.Sprintf(format, args...)}
}

// parseExpr parses a dot-separated path into an expression
// e.g., "foo.bar.baz" -> Sel(Sel(Id("foo"), "bar"), "baz")
func parseExpr(path string) Expr {
	var result Expr
	start := 0

	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '.' {
			part := path[start:i]
			if result == nil {
				result = Id(part)
			} else {
				result = &SelectorExpr{X: result, Sel: part}
			}
			start = i + 1
		}
	}

	return result
}

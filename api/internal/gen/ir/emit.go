package ir

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Emitter writes Go code from IR nodes
type Emitter struct {
	w      io.Writer
	indent int
	err    error
}

// NewEmitter creates a new emitter
func NewEmitter(w io.Writer) *Emitter {
	return &Emitter{w: w}
}

// Emit writes the IR node to the writer
func (e *Emitter) Emit(n Node) error {
	e.emit(n)
	return e.err
}

// EmitFile is a convenience method for emitting a file
func EmitFile(w io.Writer, f *File) error {
	return NewEmitter(w).Emit(f)
}

func (e *Emitter) write(s string) {
	if e.err != nil {
		return
	}
	_, e.err = io.WriteString(e.w, s)
}

func (e *Emitter) writef(format string, args ...any) {
	e.write(fmt.Sprintf(format, args...))
}

func (e *Emitter) writeIndent() {
	e.write(strings.Repeat("\t", e.indent))
}

func (e *Emitter) newline() {
	e.write("\n")
}

func (e *Emitter) emit(n Node) {
	if e.err != nil {
		return
	}

	switch v := n.(type) {
	case *File:
		e.emitFile(v)
	case *StructDecl:
		e.emitStruct(v)
	case *FuncDecl:
		e.emitFunc(v)
	case *VarDecl:
		e.emitVarDecl(v)
	case *AssignStmt:
		e.emitAssign(v)
	case *IfStmt:
		e.emitIf(v)
	case *ForStmt:
		e.emitFor(v)
	case *RangeStmt:
		e.emitRange(v)
	case *ReturnStmt:
		e.emitReturn(v)
	case *DeferStmt:
		e.emitDefer(v)
	case *ExprStmt:
		e.emitExpr(v.X)
	case *SendStmt:
		e.emitSend(v)
	case *GoStmt:
		e.emitGo(v)
	case *BlockStmt:
		e.emitBlock(v)
	case *RawStmt:
		e.write(v.Code)
	case Expr:
		e.emitExpr(v)
	default:
		e.err = fmt.Errorf("unknown node type: %T", n)
	}
}

func (e *Emitter) emitFile(f *File) {
	e.writef("package %s\n", f.Package)

	if len(f.Imports) > 0 {
		e.newline()
		if len(f.Imports) == 1 {
			e.write("import ")
			e.emitImport(f.Imports[0])
			e.newline()
		} else {
			e.write("import (\n")
			e.indent++
			for _, imp := range f.Imports {
				e.writeIndent()
				e.emitImport(imp)
				e.newline()
			}
			e.indent--
			e.write(")\n")
		}
	}

	for _, decl := range f.Decls {
		e.newline()
		e.emit(decl)
	}
}

func (e *Emitter) emitImport(imp Import) {
	if imp.Alias != "" {
		e.writef("%s %q", imp.Alias, imp.Path)
	} else {
		e.writef("%q", imp.Path)
	}
}

func (e *Emitter) emitStruct(s *StructDecl) {
	e.writef("type %s struct {\n", s.Name)
	e.indent++
	for _, f := range s.Fields {
		e.writeIndent()
		e.writef("%s %s", f.Name, f.Type)
		if f.Tag != "" {
			e.writef(" `%s`", f.Tag)
		}
		e.newline()
	}
	e.indent--
	e.write("}\n")
}

func (e *Emitter) emitFunc(f *FuncDecl) {
	e.write("func ")

	if f.Receiver != nil {
		e.writef("(%s %s) ", f.Receiver.Name, f.Receiver.Type)
	}

	e.write(f.Name)
	e.write("(")
	e.emitParams(f.Params, f.IsVariadic)
	e.write(")")

	if len(f.Results) > 0 {
		e.write(" ")
		if len(f.Results) == 1 && f.Results[0].Name == "" {
			e.write(f.Results[0].Type)
		} else {
			e.write("(")
			e.emitParams(f.Results, false)
			e.write(")")
		}
	}

	e.write(" {\n")
	e.indent++
	for _, stmt := range f.Body {
		e.writeIndent()
		e.emit(stmt)
		e.newline()
	}
	e.indent--
	e.write("}\n")
}

func (e *Emitter) emitParams(params []Param, variadic bool) {
	for i, p := range params {
		if i > 0 {
			e.write(", ")
		}
		if p.Name != "" {
			e.write(p.Name)
			e.write(" ")
		}
		if variadic && i == len(params)-1 {
			e.write("...")
		}
		e.write(p.Type)
	}
}

func (e *Emitter) emitVarDecl(v *VarDecl) {
	e.write("var ")
	e.write(v.Name)
	if v.Type != "" {
		e.write(" ")
		e.write(v.Type)
	}
	if v.Value != nil {
		e.write(" = ")
		e.emitExpr(v.Value)
	}
}

func (e *Emitter) emitAssign(a *AssignStmt) {
	for i, l := range a.Left {
		if i > 0 {
			e.write(", ")
		}
		e.emitExpr(l)
	}

	if a.Define {
		e.write(" := ")
	} else {
		e.write(" = ")
	}

	for i, r := range a.Right {
		if i > 0 {
			e.write(", ")
		}
		e.emitExpr(r)
	}
}

func (e *Emitter) emitIf(i *IfStmt) {
	e.write("if ")
	if i.Init != nil {
		e.emit(i.Init)
		e.write("; ")
	}
	e.emitExpr(i.Cond)
	e.write(" {\n")
	e.indent++
	for _, stmt := range i.Then {
		e.writeIndent()
		e.emit(stmt)
		e.newline()
	}
	e.indent--
	e.writeIndent()
	e.write("}")

	if len(i.Else) > 0 {
		e.write(" else ")
		// Check if it's an else-if
		if len(i.Else) == 1 {
			if elif, ok := i.Else[0].(*IfStmt); ok {
				e.emitIf(elif)
				return
			}
		}
		e.write("{\n")
		e.indent++
		for _, stmt := range i.Else {
			e.writeIndent()
			e.emit(stmt)
			e.newline()
		}
		e.indent--
		e.writeIndent()
		e.write("}")
	}
}

func (e *Emitter) emitFor(f *ForStmt) {
	e.write("for ")

	hasInit := f.Init != nil
	hasPost := f.Post != nil
	hasCond := f.Cond != nil

	if hasInit || hasPost {
		// Classic for loop style
		if hasInit {
			e.emit(f.Init)
		}
		e.write("; ")
		if hasCond {
			e.emitExpr(f.Cond)
		}
		e.write("; ")
		if hasPost {
			e.emit(f.Post)
		}
	} else if hasCond {
		// While-style
		e.emitExpr(f.Cond)
	}
	// else: infinite loop, no condition

	e.write(" {\n")
	e.indent++
	for _, stmt := range f.Body {
		e.writeIndent()
		e.emit(stmt)
		e.newline()
	}
	e.indent--
	e.writeIndent()
	e.write("}")
}

func (e *Emitter) emitRange(r *RangeStmt) {
	e.write("for ")

	hasKey := r.Key != "" && r.Key != "_"
	hasValue := r.Value != "" && r.Value != "_"

	if hasKey || hasValue {
		if hasKey {
			e.write(r.Key)
			if hasValue {
				e.write(", ")
				e.write(r.Value)
			}
		} else if hasValue {
			// Only value, no key - for channels or when ignoring index
			e.write(r.Value)
		}
		if r.Define {
			e.write(" := ")
		} else {
			e.write(" = ")
		}
	}

	e.write("range ")
	e.emitExpr(r.X)
	e.write(" {\n")
	e.indent++
	for _, stmt := range r.Body {
		e.writeIndent()
		e.emit(stmt)
		e.newline()
	}
	e.indent--
	e.writeIndent()
	e.write("}")
}

func (e *Emitter) emitReturn(r *ReturnStmt) {
	e.write("return")
	if len(r.Values) > 0 {
		e.write(" ")
		for i, v := range r.Values {
			if i > 0 {
				e.write(", ")
			}
			e.emitExpr(v)
		}
	}
}

func (e *Emitter) emitDefer(d *DeferStmt) {
	e.write("defer ")
	e.emitExpr(d.Call)
}

func (e *Emitter) emitSend(s *SendStmt) {
	e.emitExpr(s.Chan)
	e.write(" <- ")
	e.emitExpr(s.Value)
}

func (e *Emitter) emitGo(g *GoStmt) {
	e.write("go ")
	e.emitExpr(g.Call)
}

func (e *Emitter) emitBlock(b *BlockStmt) {
	e.write("{\n")
	e.indent++
	for _, stmt := range b.Stmts {
		e.writeIndent()
		e.emit(stmt)
		e.newline()
	}
	e.indent--
	e.writeIndent()
	e.write("}")
}

func (e *Emitter) emitExpr(expr Expr) {
	if e.err != nil {
		return
	}

	switch v := expr.(type) {
	case *Ident:
		e.write(v.Name)

	case *Literal:
		e.emitLiteral(v)

	case *CallExpr:
		e.emitExpr(v.Func)
		e.write("(")
		for i, arg := range v.Args {
			if i > 0 {
				e.write(", ")
			}
			if v.Variadic && i == len(v.Args)-1 {
				e.emitExpr(arg)
				e.write("...")
			} else {
				e.emitExpr(arg)
			}
		}
		e.write(")")

	case *SelectorExpr:
		e.emitExpr(v.X)
		e.write(".")
		e.write(v.Sel)

	case *IndexExpr:
		e.emitExpr(v.X)
		e.write("[")
		e.emitExpr(v.Index)
		e.write("]")

	case *UnaryExpr:
		e.write(v.Op)
		e.emitExpr(v.X)

	case *BinaryExpr:
		e.write("(")
		e.emitExpr(v.X)
		e.writef(" %s ", v.Op)
		e.emitExpr(v.Y)
		e.write(")")

	case *CompositeLit:
		e.write(v.Type)
		e.write("{")
		for i, elem := range v.Elements {
			if i > 0 {
				e.write(", ")
			}
			e.emitExpr(elem)
		}
		e.write("}")

	case *KeyValueExpr:
		e.emitExpr(v.Key)
		e.write(": ")
		e.emitExpr(v.Value)

	case *SliceExpr:
		e.emitExpr(v.X)
		e.write("[")
		if v.Low != nil {
			e.emitExpr(v.Low)
		}
		e.write(":")
		if v.High != nil {
			e.emitExpr(v.High)
		}
		if v.Max != nil {
			e.write(":")
			e.emitExpr(v.Max)
		}
		e.write("]")

	case *TypeAssertExpr:
		e.emitExpr(v.X)
		e.write(".(")
		e.write(v.Type)
		e.write(")")

	case *FuncLit:
		e.write("func(")
		e.emitParams(v.Params, false)
		e.write(")")
		if len(v.Results) > 0 {
			e.write(" ")
			if len(v.Results) == 1 && v.Results[0].Name == "" {
				e.write(v.Results[0].Type)
			} else {
				e.write("(")
				e.emitParams(v.Results, false)
				e.write(")")
			}
		}
		e.write(" {\n")
		e.indent++
		for _, stmt := range v.Body {
			e.writeIndent()
			e.emit(stmt)
			e.newline()
		}
		e.indent--
		e.writeIndent()
		e.write("}")

	case *RawExpr:
		e.write(v.Code)

	default:
		e.err = fmt.Errorf("unknown expression type: %T", expr)
	}
}

func (e *Emitter) emitLiteral(l *Literal) {
	switch l.Kind {
	case "string":
		e.write(strconv.Quote(l.Value.(string)))
	case "int":
		e.writef("%d", l.Value)
	case "float":
		e.writef("%v", l.Value)
	case "bool":
		e.writef("%t", l.Value)
	case "nil":
		e.write("nil")
	default:
		e.writef("%v", l.Value)
	}
}

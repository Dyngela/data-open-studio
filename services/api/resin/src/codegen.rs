//! Code generator: `Program` AST → Rust source code string.
//!
//! The generated code references a `resin_rt` module that the caller must
//! provide.  That module exposes the runtime helpers (`rt_filter`, `rt_map`,
//! `rt_col`, …) that wrap the raw `df_store` types.  This keeps the generated
//! code readable and decoupled from low-level columnar details.
//!
//! ## Generated code contract
//!
//! Every generated snippet is a free function with this signature:
//!
//! ```rust,ignore
//! fn execute(
//!     store: &std::collections::HashMap<String, df_store::frame::Frame>,
//! ) -> Result<std::collections::HashMap<String, df_store::frame::Frame>, String>
//! ```
//!
//! Inside the function:
//! - `use` statements bind `let <name> = store.get(...)`.
//! - `frame` statements build dimensions via `rt_relate` / `rt_build_dimension`.
//! - Query chains bind `let <alias> = { … };` blocks with one `let __f = rt_*(…);`
//!   line per pipe operation.
//! - Materialized results are inserted into `__out`.

use std::fmt::Write as FmtWrite;
use crate::ast::*;

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

#[derive(Debug)]
pub struct CodegenError {
    pub message: String,
}

impl std::fmt::Display for CodegenError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "codegen error: {}", self.message)
    }
}

/// Lex, parse, and generate Rust source from a Resin `src` string.
/// Returns the full source for an `execute` function.
pub fn generate_from_str(src: &str) -> Result<String, CodegenError> {
    let tokens = crate::lexer::Lexer::new(src)
        .tokenize()
        .map_err(|e| CodegenError { message: format!("lex errors: {e:#?}") })?;

    let (program, errors) = crate::parser::parse(tokens);
    if !errors.is_empty() {
        let msg = errors.iter()
            .map(|e| e.to_string())
            .collect::<Vec<_>>()
            .join("; ");
        return Err(CodegenError { message: format!("parse errors: {msg}") });
    }

    generate(&program)
}

/// Generate Rust source from an already-parsed `Program`.
pub fn generate(program: &Program) -> Result<String, CodegenError> {
    let mut out = String::new();
    let mut gen = Codegen { out: &mut out, indent: 1 };
    gen.emit_header();
    gen.emit_program(program)?;
    gen.emit_footer();
    Ok(out)
}

// ---------------------------------------------------------------------------
// Internal generator
// ---------------------------------------------------------------------------

struct Codegen<'a> {
    out: &'a mut String,
    indent: usize,
}

impl<'a> Codegen<'a> {
    // ── helpers ───────────────────────────────────────────────────────────

    fn line(&mut self, s: impl AsRef<str>) {
        let indent = "    ".repeat(self.indent);
        writeln!(self.out, "{indent}{}", s.as_ref()).unwrap();
    }

    fn blank(&mut self) {
        writeln!(self.out).unwrap();
    }

    // ── program skeleton ──────────────────────────────────────────────────

    fn emit_header(&mut self) {
        writeln!(self.out, "// === Resin generated code ===").unwrap();
        writeln!(self.out, "// Requires: `use resin_rt::*;` and `use df_store::data_type::DataValue;`").unwrap();
        writeln!(self.out, "//           `use df_store::frame::Frame;`").unwrap();
        writeln!(self.out, "fn execute(").unwrap();
        writeln!(self.out, "    store: &std::collections::HashMap<String, Frame>,").unwrap();
        writeln!(self.out, ") -> Result<std::collections::HashMap<String, Frame>, String> {{").unwrap();
        writeln!(self.out, "    let mut __out: std::collections::HashMap<String, Frame> = std::collections::HashMap::new();").unwrap();
    }

    fn emit_footer(&mut self) {
        writeln!(self.out, "    Ok(__out)").unwrap();
        writeln!(self.out, "}}").unwrap();
    }

    fn emit_program(&mut self, program: &Program) -> Result<(), CodegenError> {
        for stmt in &program.statements {
            self.blank();
            self.emit_statement(stmt)?;
        }
        Ok(())
    }

    // ── statements ────────────────────────────────────────────────────────

    fn emit_statement(&mut self, stmt: &Statement) -> Result<(), CodegenError> {
        match stmt {
            Statement::Use(u)   => self.emit_use(u),
            Statement::Frame(f) => self.emit_frame(f),
            Statement::Query(q) => self.emit_query(q),
        }
    }

    // `use sales from "sales"`
    fn emit_use(&mut self, u: &UseStmt) -> Result<(), CodegenError> {
        self.line(format!("// use {} from \"{}\"", u.name.name, u.source.value));
        self.line(format!(
            "let {name} = store.get(\"{src}\").ok_or_else(|| \"frame '{src}' not found\".to_string())?;",
            name = u.name.name,
            src  = u.source.value,
        ));
        Ok(())
    }

    // `frame ( relate … ) as dim`
    fn emit_frame(&mut self, f: &FrameStmt) -> Result<(), CodegenError> {
        self.line(format!("// frame … as {}", f.name.name));
        self.line(format!("let {} = {{", f.name.name));
        self.indent += 1;

        self.line("let mut __relates = Vec::new();");
        for r in &f.relates {
            // rt_relate(left, right, |__l, __r| { cond }, "alias", filter_opt)
            let cond = self.expr_to_string_join(&r.condition)?;
            let filter = match &r.filter {
                Some(fe) => {
                    let s = self.expr_to_string(fe)?;
                    format!("Some(|row: &resin_rt::Row| {{ {s} }})")
                }
                None => "None".to_string(),
            };
            self.line(format!(
                "__relates.push(rt_relate({left}, {right}, |__l, __r| {{ {cond} }}, \"{alias}\", {filter}));",
                left  = r.left.name,
                right = r.right.name,
                alias = r.alias.name,
            ));
        }
        self.line("rt_build_dimension(__relates)");

        self.indent -= 1;
        self.line("};");
        self.line(format!(
            "__out.insert(\"{name}\".to_string(), {name}.clone());",
            name = f.name.name
        ));
        Ok(())
    }

    // query chain
    fn emit_query(&mut self, q: &QueryStmt) -> Result<(), CodegenError> {
        let result_name = q.materialize.as_ref()
            .map(|m| m.name.as_str())
            .unwrap_or("__result");

        self.line(format!("// query: {} …", q.source.name));
        self.line(format!("let {result_name} = {{"));
        self.indent += 1;

        self.line(format!("let __f = {}.clone();", q.source.name));
        for op in &q.ops {
            self.emit_pipe_op(op)?;
        }
        self.line("__f");

        self.indent -= 1;
        self.line("};");

        if let Some(mat) = &q.materialize {
            self.line(format!(
                "__out.insert(\"{name}\".to_string(), {name}.clone());",
                name = mat.name
            ));
        }
        Ok(())
    }

    // ── pipe operators ─────────────────────────────────────────────────────

    fn emit_pipe_op(&mut self, op: &PipeOp) -> Result<(), CodegenError> {
        match op {
            PipeOp::Filter(f) => {
                let cond = self.expr_to_string(&f.condition)?;
                self.line(format!(
                    "let __f = rt_filter(&__f, |row| {{ {cond} }});"
                ));
            }

            PipeOp::Select(s) => {
                let cols = self.col_name_list(&s.columns)?;
                self.line(format!("let __f = rt_select(&__f, &[{cols}]);"));
            }

            PipeOp::Map(m) => {
                let expr = self.expr_to_string(&m.expr)?;
                self.line(format!(
                    "let __f = rt_map(&__f, \"{alias}\", |row| {{ {expr} }});",
                    alias = m.alias.name,
                ));
            }

            PipeOp::Aggregate(a) => {
                let aggs = a.aggregations.iter()
                    .map(|agg| {
                        let arg = agg.func.args.first()
                            .map(|a| match a {
                                FunctionArg::Star    => "*".to_owned(),
                                FunctionArg::Expr(e) => self.col_name_str(e),
                            })
                            .unwrap_or_default();
                        format!("(\"{fn}\", \"{arg}\", \"{alias}\")",
                            fn    = agg.func.name,
                            alias = agg.alias.name)
                    })
                    .collect::<Vec<_>>()
                    .join(", ");
                let by = a.group_by.iter()
                    .map(|e| format!("\"{}\"", self.col_name_str(e)))
                    .collect::<Vec<_>>()
                    .join(", ");
                self.line(format!(
                    "let __f = rt_aggregate(&__f, &[{aggs}], &[{by}]);"
                ));
            }

            PipeOp::Distinct(_) => {
                self.line("let __f = rt_distinct(&__f);");
            }

            PipeOp::SortBy(s) => {
                let cols = s.columns.iter()
                    .map(|c| format!("(\"{}\", {})",
                        self.col_name_str(&c.expr),
                        c.direction == SortDirection::Desc,
                    ))
                    .collect::<Vec<_>>()
                    .join(", ");
                self.line(format!("let __f = rt_sort_by(&__f, &[{cols}]);"));
            }

            PipeOp::Limit(l) => {
                self.line(format!("let __f = rt_limit(&__f, {});", l.n));
            }

            PipeOp::Offset(o) => {
                self.line(format!("let __f = rt_offset(&__f, {});", o.n));
            }

            PipeOp::Join(j) => {
                let fn_name = match j.kind {
                    JoinKind::Inner => "rt_join",
                    JoinKind::Left  => "rt_left_join",
                    JoinKind::Right => "rt_right_join",
                    JoinKind::Cross => "rt_cross_join",
                };
                match &j.condition {
                    Some(cond) => {
                        let cond_s = self.expr_to_string_join(cond)?;
                        self.line(format!(
                            "let __f = {fn_name}(&__f, &{frame}, |__l, __r| {{ {cond_s} }});",
                            frame = j.frame.name,
                        ));
                    }
                    None => {
                        self.line(format!(
                            "let __f = {fn_name}(&__f, &{});",
                            j.frame.name
                        ));
                    }
                }
            }

            PipeOp::Window(w) => {
                self.emit_window_op(w)?;
            }

            PipeOp::Union(s) => {
                self.line(format!("let __f = rt_union(&__f, &{});", s.frame.name));
            }
            PipeOp::Intersect(s) => {
                self.line(format!("let __f = rt_intersect(&__f, &{});", s.frame.name));
            }
            PipeOp::Except(s) => {
                self.line(format!("let __f = rt_except(&__f, &{});", s.frame.name));
            }
        }
        Ok(())
    }

    fn emit_window_op(&mut self, w: &WindowOp) -> Result<(), CodegenError> {
        let e = &w.expr;

        // function + first arg
        let fn_arg = e.func.args.first()
            .map(|a| match a {
                FunctionArg::Star    => "*".to_owned(),
                FunctionArg::Expr(ex) => self.col_name_str(ex),
            })
            .unwrap_or_default();
        let fn_call = if fn_arg.is_empty() {
            format!("\"{}\"", e.func.name)
        } else {
            format!("(\"{}\", \"{}\")", e.func.name, fn_arg)
        };

        let partition = e.partition_by.iter()
            .map(|ex| format!("\"{}\"", self.col_name_str(ex)))
            .collect::<Vec<_>>()
            .join(", ");

        let sort = e.sort_by.iter()
            .map(|sc| format!("(\"{}\", {})", self.col_name_str(&sc.expr), sc.direction == SortDirection::Desc))
            .collect::<Vec<_>>()
            .join(", ");

        let frame_spec = match &e.frame_spec {
            None      => "None".to_owned(),
            Some(fs)  => format!(
                "Some(rt_frame_spec({}, {}))",
                frame_bound_str(&fs.start),
                frame_bound_str(&fs.end)
            ),
        };

        self.line(format!(
            "let __f = rt_window(&__f, {fn_call}, &[{partition}], &[{sort}], {frame_spec}, \"{alias}\");",
            alias = e.alias.name,
        ));
        Ok(())
    }

    // ── expression stringification ────────────────────────────────────────

    /// Convert an expression to a Rust code string (row-closure context).
    /// Column references become `row.col("name")`.
    fn expr_to_string(&self, expr: &Expr) -> Result<String, CodegenError> {
        Ok(emit_expr(expr, ColCtx::Row))
    }

    /// Same but for join conditions: left-side columns are `__l.col(…)`,
    /// right-side are `__r.col(…)`.
    fn expr_to_string_join(&self, expr: &Expr) -> Result<String, CodegenError> {
        Ok(emit_expr(expr, ColCtx::Join))
    }

    /// Extract a plain column-name string from a column/function expression.
    fn col_name_str(&self, expr: &Expr) -> String {
        col_name_of(expr)
    }

    /// Produce a `"col1", "col2"` comma list for select / group-by.
    fn col_name_list(&self, exprs: &[Expr]) -> Result<String, CodegenError> {
        let v: Vec<String> = exprs.iter()
            .map(|e| format!("\"{}\"", col_name_of(e)))
            .collect();
        Ok(v.join(", "))
    }
}

// ---------------------------------------------------------------------------
// Pure expression emitter (stateless, returns String)
// ---------------------------------------------------------------------------

#[derive(Clone, Copy)]
enum ColCtx {
    /// Inside a row-closure: `row.col("name")`
    Row,
    /// Inside a join predicate: `__l.col(…)` / `__r.col(…)` based on table qualifier
    Join,
}

fn emit_expr(expr: &Expr, ctx: ColCtx) -> String {
    match expr {
        Expr::BinaryOp(b) => {
            let fn_name = match b.op {
                BinaryOp::Add   => "rt_add",
                BinaryOp::Sub   => "rt_sub",
                BinaryOp::Mul   => "rt_mul",
                BinaryOp::Div   => "rt_div",
                BinaryOp::Eq    => "rt_eq",
                BinaryOp::NotEq => "rt_neq",
                BinaryOp::Gt    => "rt_gt",
                BinaryOp::Lt    => "rt_lt",
                BinaryOp::Gte   => "rt_gte",
                BinaryOp::Lte   => "rt_lte",
                BinaryOp::And   => "rt_and",
                BinaryOp::Or    => "rt_or",
            };
            format!("{fn_name}({}, {})",
                emit_expr(&b.left, ctx),
                emit_expr(&b.right, ctx))
        }

        Expr::UnaryOp(u) => {
            let fn_name = match u.op {
                UnaryOp::Neg => "rt_neg",
                UnaryOp::Not => "rt_not",
            };
            format!("{fn_name}({})", emit_expr(&u.expr, ctx))
        }

        Expr::FunctionCall(f) => {
            let args: Vec<String> = f.args.iter().map(|a| match a {
                FunctionArg::Star    => "row".to_owned(),
                FunctionArg::Expr(e) => emit_expr(e, ctx),
            }).collect();
            format!("rt_fn_{}({})", f.name, args.join(", "))
        }

        Expr::Column(c) => {
            let col_key = match &c.table {
                Some(t) => format!("{}.{}", t, c.column),
                None    => c.column.clone(),
            };
            match ctx {
                ColCtx::Row => format!("row.col(\"{col_key}\")"),
                ColCtx::Join => {
                    // If the column is qualified, use the table name to pick the side.
                    // Unqualified columns fall back to left.
                    let side = match &c.table { Some(_) => "__l", None => "__l" };
                    format!("{side}.col(\"{col_key}\")")
                }
            }
        }

        Expr::Literal(l) => emit_literal(l),

        Expr::IsNull(e, _)    => format!("rt_is_null({})",     emit_expr(e, ctx)),
        Expr::IsNotNull(e, _) => format!("rt_is_not_null({})", emit_expr(e, ctx)),
        Expr::Grouped(e, _)   => emit_expr(e, ctx),

        Expr::If(if_expr) => {
            let mut out = format!(
                "rt_if({cond}, {then}",
                cond = emit_expr(&if_expr.condition, ctx),
                then = emit_expr(&if_expr.then_branch, ctx),
            );
            for branch in &if_expr.else_if_branches {
                out.push_str(&format!(
                    ", {}, {}",
                    emit_expr(&branch.condition, ctx),
                    emit_expr(&branch.body, ctx),
                ));
            }
            let else_part = if_expr.else_branch.as_ref()
                .map(|e| emit_expr(e, ctx))
                .unwrap_or_else(|| "DataValue::Null".to_owned());
            out.push_str(&format!(", {else_part})"));
            out
        }
    }
}

fn emit_literal(lit: &Literal) -> String {
    match lit {
        Literal::Int(n, _)    => format!("DataValue::Int64({n}_i64)"),
        Literal::Float(f, _)  => format!("DataValue::Float64({f}_f64)"),
        Literal::String(s, _) => format!("DataValue::String(\"{s}\".to_string())"),
        Literal::Bool(b, _)   => format!("DataValue::Boolean({b})"),
        Literal::Null(_)      => "DataValue::Null".to_owned(),
    }
}

// ---------------------------------------------------------------------------
// Column-name extraction (for select / group-by / sort-by)
// ---------------------------------------------------------------------------

fn col_name_of(expr: &Expr) -> String {
    match expr {
        Expr::Column(c) => match &c.table {
            Some(t) => format!("{}.{}", t, c.column),
            None    => c.column.clone(),
        },
        Expr::FunctionCall(f) => {
            let args: Vec<String> = f.args.iter().map(|a| match a {
                FunctionArg::Star    => "*".to_owned(),
                FunctionArg::Expr(e) => col_name_of(e),
            }).collect();
            format!("{}({})", f.name, args.join(","))
        }
        Expr::Grouped(e, _) => col_name_of(e),
        _ => "__expr".to_owned(),
    }
}

fn frame_bound_str(b: &FrameBound) -> String {
    match b {
        FrameBound::UnboundedPreceding => "FrameBound::UnboundedPreceding".to_owned(),
        FrameBound::UnboundedFollowing => "FrameBound::UnboundedFollowing".to_owned(),
        FrameBound::CurrentRow         => "FrameBound::CurrentRow".to_owned(),
        FrameBound::Preceding(n)       => format!("FrameBound::Preceding({n})"),
        FrameBound::Following(n)       => format!("FrameBound::Following({n})"),
    }
}

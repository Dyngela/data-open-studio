//! Integration tests for the full lexer → parser pipeline.
//!
//! Each `pass_*` test asserts that the input parses without errors and that
//! the resulting AST has the expected shape.  Each `fail_*` test asserts that
//! at least one parse / lex error is produced.

#![cfg(test)]

use crate::{
    ast::*,
    lexer::{LexError, Lexer, TokenKind},
    parser::{self, ParseError},
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Lex `src` and expect success; panic with lex errors on failure.
fn lex(src: &str) -> Vec<crate::lexer::Token> {
    Lexer::new(src)
        .tokenize()
        .unwrap_or_else(|e| panic!("lex errors: {e:#?}"))
}

/// Lex `src` and expect at least one lex error.
fn lex_expect_err(src: &str) -> Vec<LexError> {
    Lexer::new(src)
        .tokenize()
        .expect_err("expected lex error but got success")
}

/// Lex + parse `src`; assert no errors and return the program.
fn parse_ok(src: &str) -> Program {
    let tokens = lex(src);
    let (prog, errors) = parser::parse(tokens);
    assert!(
        errors.is_empty(),
        "expected no parse errors, got:\n{}",
        errors.iter().map(|e| e.to_string()).collect::<Vec<_>>().join("\n")
    );
    prog
}

/// Lex + parse `src`; assert at least one parse error.
fn parse_err(src: &str) -> Vec<ParseError> {
    let tokens = lex(src);
    let (_, errors) = parser::parse(tokens);
    assert!(!errors.is_empty(), "expected parse errors but got none");
    errors
}

/// Return the first statement from a single-statement program.
fn first_stmt(src: &str) -> Statement {
    let prog = parse_ok(src);
    assert_eq!(prog.statements.len(), 1);
    prog.statements.into_iter().next().unwrap()
}

// ---------------------------------------------------------------------------
// ── Lexer tests ──────────────────────────────────────────────────────────────
// ---------------------------------------------------------------------------

mod lexer_tests {
    use super::*;
    use crate::lexer::TokenKind::*;

    fn kinds(src: &str) -> Vec<TokenKind> {
        lex(src).into_iter().map(|t| t.kind).collect()
    }

    // ── pass ─────────────────────────────────────────────────────────────────

    #[test]
    fn pass_keywords() {
        let src = "use from frame relate on as where filter select map aggregate by";
        let k = kinds(src);
        assert_eq!(
            k,
            vec![Use, From, Frame, Relate, On, As, Where, Filter, Select, Map, Aggregate, By, Eof]
        );
    }

    #[test]
    fn pass_pipe_keywords() {
        let src = "distinct sort_by limit offset join left_join right_join cross_join window over partition_by";
        let k = kinds(src);
        assert_eq!(
            k,
            vec![Distinct, SortBy, Limit, Offset, Join, LeftJoin, RightJoin, CrossJoin, Window, Over, PartitionBy, Eof]
        );
    }

    #[test]
    fn pass_set_and_bool_keywords() {
        let src = "union intersect except and or not is null asc desc true false coalesce";
        let k = kinds(src);
        assert_eq!(
            k,
            vec![Union, Intersect, Except, And, Or, Not, Is, Null, Asc, Desc, True, False, Coalesce, Eof]
        );
    }

    #[test]
    fn pass_window_frame_keywords() {
        let src = "rows between preceding following unbounded_preceding unbounded_following current_row";
        let k = kinds(src);
        assert_eq!(
            k,
            vec![Rows, Between, Preceding, Following, UnboundedPreceding, UnboundedFollowing, CurrentRow, Eof]
        );
    }

    #[test]
    fn pass_symbols() {
        let src = "-> . , * ( ) = != > < >= <= + - /";
        let k = kinds(src);
        assert_eq!(
            k,
            vec![Arrow, Dot, Comma, Star, LParen, RParen, Eq, NotEq, Gt, Lt, Gte, Lte, Plus, Minus, Slash, Eof]
        );
    }

    #[test]
    fn pass_integer_literal() {
        let k = kinds("42");
        assert_eq!(k, vec![Int(42), Eof]);
    }

    #[test]
    fn pass_float_literal() {
        let k = kinds("3.14");
        assert_eq!(k, vec![Float(3.14), Eof]);
    }

    #[test]
    fn pass_string_literal() {
        let k = kinds(r#""hello world""#);
        assert_eq!(k, vec![String("hello world".to_owned()), Eof]);
    }

    #[test]
    fn pass_string_with_escape() {
        let k = kinds(r#""line1\nline2""#);
        assert_eq!(k, vec![String("line1\nline2".to_owned()), Eof]);
    }

    #[test]
    fn pass_identifier() {
        let k = kinds("my_table_42");
        assert_eq!(k, vec![Ident("my_table_42".to_owned()), Eof]);
    }

    #[test]
    fn pass_agg_functions_are_idents() {
        // sum, count, avg, min, max, row_number, rank, dense_rank, lag, lead
        // must remain as Ident per the spec.
        for name in &["sum", "count", "avg", "min", "max", "row_number", "rank", "dense_rank", "lag", "lead"] {
            let k = kinds(name);
            assert_eq!(k, vec![Ident(name.to_string()), Eof], "function '{name}' should be Ident");
        }
    }

    #[test]
    fn pass_comments_skipped() {
        let k = kinds("42 -- this is ignored\n99");
        assert_eq!(k, vec![Int(42), Int(99), Eof]);
    }

    #[test]
    fn pass_arrow_vs_minus() {
        let k = kinds("-> -");
        assert_eq!(k, vec![Arrow, Minus, Eof]);
    }

    #[test]
    fn pass_span_tracking() {
        let tokens = lex("use\nsales");
        assert_eq!(tokens[0].span.line, 1);
        assert_eq!(tokens[1].span.line, 2);
    }

    // ── fail ─────────────────────────────────────────────────────────────────

    #[test]
    fn fail_unterminated_string() {
        let errs = lex_expect_err(r#""unterminated"#);
        assert!(!errs.is_empty());
        assert!(matches!(errs[0], LexError::UnterminatedString { .. }));
    }

    #[test]
    fn fail_unexpected_char() {
        let errs = lex_expect_err("@ foo");
        assert!(!errs.is_empty());
        assert!(matches!(errs[0], LexError::UnexpectedChar { ch: '@', .. }));
    }

    #[test]
    fn fail_lone_exclamation() {
        // `!` without following `=` is unexpected
        let errs = lex_expect_err("! ");
        assert!(!errs.is_empty());
    }
}

// ---------------------------------------------------------------------------
// ── Parser: `use` statement ──────────────────────────────────────────────────
// ---------------------------------------------------------------------------

mod parse_use {
    use super::*;

    #[test]
    fn pass_basic() {
        let stmt = first_stmt(r#"use sales from "my_source""#);
        let Statement::Use(u) = stmt else { panic!("expected Use") };
        assert_eq!(u.name.name, "sales");
        assert_eq!(u.source.value, "my_source");
    }

    #[test]
    fn pass_multiple() {
        let prog = parse_ok(r#"use a from "x" use b from "y""#);
        assert_eq!(prog.statements.len(), 2);
    }

    #[test]
    fn fail_missing_from() {
        parse_err(r#"use sales "source""#);
    }

    #[test]
    fn fail_missing_source() {
        parse_err(r#"use sales from"#);
    }

    #[test]
    fn fail_source_not_string() {
        parse_err(r#"use sales from source_name"#);
    }
}

// ---------------------------------------------------------------------------
// ── Parser: `frame` statement ─────────────────────────────────────────────────
// ---------------------------------------------------------------------------

mod parse_frame {
    use super::*;

    #[test]
    fn pass_single_relate() {
        let stmt = first_stmt(
            r#"frame (
              relate sales -> customers
                on sales.customer_id = customers.id
                as customer_info
            ) as sales_dim"#,
        );
        let Statement::Frame(f) = stmt else { panic!("expected Frame") };
        assert_eq!(f.name.name, "sales_dim");
        assert_eq!(f.relates.len(), 1);
        let r = &f.relates[0];
        assert_eq!(r.left.name, "sales");
        assert_eq!(r.right.name, "customers");
        assert_eq!(r.alias.name, "customer_info");
        assert!(r.filter.is_none());
    }

    #[test]
    fn pass_relate_with_where() {
        let stmt = first_stmt(
            r#"frame (
              relate sales -> products
                on sales.product_id = products.id
                as product_info
                where products.active = true
            ) as dim"#,
        );
        let Statement::Frame(f) = stmt else { panic!() };
        assert!(f.relates[0].filter.is_some());
    }

    #[test]
    fn pass_multiple_relates() {
        let stmt = first_stmt(
            r#"frame (
              relate a -> b on a.id = b.a_id as b_info
              relate a -> c on a.id = c.a_id as c_info
            ) as enriched"#,
        );
        let Statement::Frame(f) = stmt else { panic!() };
        assert_eq!(f.relates.len(), 2);
    }

    #[test]
    fn fail_missing_as_on_frame() {
        parse_err(
            r#"frame (
              relate a -> b on a.id = b.id as info
            )"#,
        );
    }

    #[test]
    fn fail_missing_arrow() {
        parse_err(
            r#"frame (
              relate a b on a.id = b.id as info
            ) as d"#,
        );
    }
}

// ---------------------------------------------------------------------------
// ── Parser: query + pipe operators ───────────────────────────────────────────
// ---------------------------------------------------------------------------

mod parse_query {
    use super::*;

    #[test]
    fn pass_bare_query_no_ops() {
        let stmt = first_stmt("my_frame");
        let Statement::Query(q) = stmt else { panic!() };
        assert_eq!(q.source.name, "my_frame");
        assert!(q.ops.is_empty());
        assert!(q.materialize.is_none());
    }

    #[test]
    fn pass_materialize() {
        let stmt = first_stmt("my_frame as result");
        let Statement::Query(q) = stmt else { panic!() };
        assert_eq!(q.materialize.as_ref().unwrap().name, "result");
    }

    // ── filter ─────────────────────────────────────────────────────────────

    #[test]
    fn pass_filter_eq() {
        let stmt = first_stmt(r#"t.filter(country = "France")"#);
        let Statement::Query(q) = stmt else { panic!() };
        assert!(matches!(q.ops[0], PipeOp::Filter(_)));
    }

    #[test]
    fn pass_filter_is_not_null() {
        let stmt = first_stmt("t.filter(col is not null)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Filter(f) = &q.ops[0] else { panic!() };
        assert!(matches!(f.condition, Expr::IsNotNull(_, _)));
    }

    #[test]
    fn pass_filter_is_null() {
        let stmt = first_stmt("t.filter(col is null)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Filter(f) = &q.ops[0] else { panic!() };
        assert!(matches!(f.condition, Expr::IsNull(_, _)));
    }

    #[test]
    fn fail_filter_missing_rparen() {
        parse_err("t.filter(x = 1");
    }

    // ── select ──────────────────────────────────────────────────────────────

    #[test]
    fn pass_select_single() {
        let stmt = first_stmt("t.select(a)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Select(s) = &q.ops[0] else { panic!() };
        assert_eq!(s.columns.len(), 1);
    }

    #[test]
    fn pass_select_multiple() {
        let stmt = first_stmt("t.select(a, b, c)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Select(s) = &q.ops[0] else { panic!() };
        assert_eq!(s.columns.len(), 3);
    }

    #[test]
    fn pass_select_qualified_columns() {
        let stmt = first_stmt("t.select(a.x, b.y)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Select(s) = &q.ops[0] else { panic!() };
        let Expr::Column(c) = &s.columns[0] else { panic!() };
        assert_eq!(c.table.as_deref(), Some("a"));
        assert_eq!(c.column, "x");
    }

    #[test]
    fn fail_select_empty() {
        parse_err("t.select()");
    }

    // ── map ─────────────────────────────────────────────────────────────────

    #[test]
    fn pass_map_arithmetic() {
        let stmt = first_stmt("t.map(price * qty as total)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Map(m) = &q.ops[0] else { panic!() };
        assert_eq!(m.alias.name, "total");
        assert!(matches!(m.expr, Expr::BinaryOp(_)));
    }

    #[test]
    fn pass_map_coalesce() {
        let stmt = first_stmt("t.map(coalesce(discount, 0) as safe_discount)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Map(m) = &q.ops[0] else { panic!() };
        assert_eq!(m.alias.name, "safe_discount");
        assert!(matches!(m.expr, Expr::FunctionCall(_)));
    }

    #[test]
    fn fail_map_missing_as() {
        parse_err("t.map(x * y)");
    }

    // ── aggregate ───────────────────────────────────────────────────────────

    #[test]
    fn pass_aggregate_grouped() {
        let stmt = first_stmt("t.aggregate(sum(revenue) as total, count(*) as n by country)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Aggregate(a) = &q.ops[0] else { panic!() };
        assert_eq!(a.aggregations.len(), 2);
        assert_eq!(a.group_by.len(), 1);
        assert_eq!(a.aggregations[0].alias.name, "total");
        assert_eq!(a.aggregations[1].alias.name, "n");
        assert_eq!(a.aggregations[1].func.name, "count");
        assert!(matches!(a.aggregations[1].func.args[0], FunctionArg::Star));
    }

    #[test]
    fn pass_aggregate_global() {
        let stmt = first_stmt("t.aggregate(sum(revenue) as total)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Aggregate(a) = &q.ops[0] else { panic!() };
        assert_eq!(a.aggregations.len(), 1);
        assert!(a.group_by.is_empty());
    }

    #[test]
    fn pass_aggregate_multi_group_by() {
        let stmt = first_stmt("t.aggregate(sum(x) as s by region, year)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Aggregate(a) = &q.ops[0] else { panic!() };
        assert_eq!(a.group_by.len(), 2);
    }

    #[test]
    fn pass_aggregate_by_function_expr() {
        // `by year(order_date)` — group-by expression is a function call
        let stmt = first_stmt("t.aggregate(count(*) as n by year(order_date))");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Aggregate(a) = &q.ops[0] else { panic!() };
        assert!(matches!(a.group_by[0], Expr::FunctionCall(_)));
    }

    #[test]
    fn pass_aggregate_empty_is_parse_valid() {
        // Empty aggregate is a semantic error (no agg expressions), but the
        // parser itself does not enforce a minimum count — that belongs to the
        // resolver phase.  Confirm it parses cleanly with 0 aggregations.
        let stmt = first_stmt("t.aggregate()");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Aggregate(a) = &q.ops[0] else { panic!() };
        assert!(a.aggregations.is_empty());
        assert!(a.group_by.is_empty());
    }

    // ── distinct ─────────────────────────────────────────────────────────────

    #[test]
    fn pass_distinct() {
        let stmt = first_stmt("t.distinct()");
        let Statement::Query(q) = stmt else { panic!() };
        assert!(matches!(q.ops[0], PipeOp::Distinct(_)));
    }

    #[test]
    fn fail_distinct_with_args() {
        // distinct() takes no args; `distinct(x)` has a stray token before `)`
        parse_err("t.distinct(x)");
    }

    // ── sort_by ──────────────────────────────────────────────────────────────

    #[test]
    fn pass_sort_by_asc() {
        let stmt = first_stmt("t.sort_by(revenue asc)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::SortBy(s) = &q.ops[0] else { panic!() };
        assert_eq!(s.columns[0].direction, SortDirection::Asc);
    }

    #[test]
    fn pass_sort_by_desc() {
        let stmt = first_stmt("t.sort_by(revenue desc)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::SortBy(s) = &q.ops[0] else { panic!() };
        assert_eq!(s.columns[0].direction, SortDirection::Desc);
    }

    #[test]
    fn pass_sort_by_multiple() {
        let stmt = first_stmt("t.sort_by(a desc, b asc, c)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::SortBy(s) = &q.ops[0] else { panic!() };
        assert_eq!(s.columns.len(), 3);
        assert_eq!(s.columns[2].direction, SortDirection::Asc); // default
    }

    // ── limit / offset ───────────────────────────────────────────────────────

    #[test]
    fn pass_limit() {
        let stmt = first_stmt("t.limit(10)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Limit(l) = &q.ops[0] else { panic!() };
        assert_eq!(l.n, 10);
    }

    #[test]
    fn pass_offset() {
        let stmt = first_stmt("t.offset(20)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Offset(o) = &q.ops[0] else { panic!() };
        assert_eq!(o.n, 20);
    }

    #[test]
    fn fail_limit_non_integer() {
        parse_err("t.limit(1.5)");
    }

    // ── joins ────────────────────────────────────────────────────────────────

    #[test]
    fn pass_inner_join() {
        let stmt = first_stmt("t.join(other on t.id = other.t_id)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Join(j) = &q.ops[0] else { panic!() };
        assert_eq!(j.kind, JoinKind::Inner);
        assert!(j.condition.is_some());
    }

    #[test]
    fn pass_left_join() {
        let stmt = first_stmt("t.left_join(other on t.id = other.t_id)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Join(j) = &q.ops[0] else { panic!() };
        assert_eq!(j.kind, JoinKind::Left);
    }

    #[test]
    fn pass_right_join() {
        let stmt = first_stmt("t.right_join(other on t.id = other.t_id)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Join(j) = &q.ops[0] else { panic!() };
        assert_eq!(j.kind, JoinKind::Right);
    }

    #[test]
    fn pass_cross_join() {
        let stmt = first_stmt("t.cross_join(other)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Join(j) = &q.ops[0] else { panic!() };
        assert_eq!(j.kind, JoinKind::Cross);
        assert!(j.condition.is_none());
    }

    #[test]
    fn fail_join_missing_on() {
        parse_err("t.join(other)");
    }

    // ── window ───────────────────────────────────────────────────────────────

    #[test]
    fn pass_window_row_number() {
        let stmt = first_stmt(
            "t.window(row_number() over (partition_by customer_id sort_by date asc) as rn)",
        );
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Window(w) = &q.ops[0] else { panic!() };
        assert_eq!(w.expr.func.name, "row_number");
        assert_eq!(w.expr.alias.name, "rn");
        assert_eq!(w.expr.partition_by.len(), 1);
        assert_eq!(w.expr.sort_by.len(), 1);
        assert!(w.expr.frame_spec.is_none());
    }

    #[test]
    fn pass_window_with_frame_spec() {
        let stmt = first_stmt(
            "t.window(sum(revenue) over (partition_by country sort_by date asc rows between unbounded_preceding and current_row) as running_rev)",
        );
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Window(w) = &q.ops[0] else { panic!() };
        let spec = w.expr.frame_spec.as_ref().unwrap();
        assert!(matches!(spec.start, FrameBound::UnboundedPreceding));
        assert!(matches!(spec.end,   FrameBound::CurrentRow));
    }

    #[test]
    fn pass_window_n_preceding() {
        let stmt = first_stmt(
            "t.window(avg(x) over (rows between 3 preceding and 1 following) as ma)",
        );
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Window(w) = &q.ops[0] else { panic!() };
        let spec = w.expr.frame_spec.as_ref().unwrap();
        assert!(matches!(spec.start, FrameBound::Preceding(3)));
        assert!(matches!(spec.end,   FrameBound::Following(1)));
    }

    #[test]
    fn pass_window_no_partition_no_frame() {
        let stmt = first_stmt("t.window(rank() over (sort_by score desc) as rnk)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Window(w) = &q.ops[0] else { panic!() };
        assert!(w.expr.partition_by.is_empty());
        assert_eq!(w.expr.sort_by.len(), 1);
    }

    #[test]
    fn fail_window_missing_over() {
        parse_err("t.window(row_number() (partition_by x) as rn)");
    }

    #[test]
    fn fail_window_missing_alias() {
        parse_err("t.window(row_number() over ())");
    }

    // ── set operators ─────────────────────────────────────────────────────────

    #[test]
    fn pass_union() {
        let stmt = first_stmt("t.union(other)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Union(s) = &q.ops[0] else { panic!() };
        assert_eq!(s.frame.name, "other");
    }

    #[test]
    fn pass_intersect() {
        let stmt = first_stmt("t.intersect(other)");
        let Statement::Query(q) = stmt else { panic!() };
        assert!(matches!(q.ops[0], PipeOp::Intersect(_)));
    }

    #[test]
    fn pass_except() {
        let stmt = first_stmt("t.except(other)");
        let Statement::Query(q) = stmt else { panic!() };
        assert!(matches!(q.ops[0], PipeOp::Except(_)));
    }
}

// ---------------------------------------------------------------------------
// ── Parser: expression tests ─────────────────────────────────────────────────
// ---------------------------------------------------------------------------

mod parse_expr {
    use super::*;

    fn expr_from_filter(src: &str) -> Expr {
        let stmt = first_stmt(src);
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Filter(f) = q.ops.into_iter().next().unwrap() else { panic!() };
        f.condition
    }

    // ── literals ─────────────────────────────────────────────────────────────

    #[test]
    fn pass_integer_literal() {
        let e = expr_from_filter("t.filter(x = 42)");
        let Expr::BinaryOp(b) = e else { panic!() };
        assert!(matches!(b.right, Expr::Literal(Literal::Int(42, _))));
    }

    #[test]
    fn pass_float_literal() {
        let e = expr_from_filter("t.filter(x = 3.14)");
        let Expr::BinaryOp(b) = e else { panic!() };
        assert!(matches!(b.right, Expr::Literal(Literal::Float(_, _))));
    }

    #[test]
    fn pass_bool_true() {
        let e = expr_from_filter("t.filter(x = true)");
        let Expr::BinaryOp(b) = e else { panic!() };
        assert!(matches!(b.right, Expr::Literal(Literal::Bool(true, _))));
    }

    #[test]
    fn pass_bool_false() {
        let e = expr_from_filter("t.filter(flag = false)");
        let Expr::BinaryOp(b) = e else { panic!() };
        assert!(matches!(b.right, Expr::Literal(Literal::Bool(false, _))));
    }

    #[test]
    fn pass_null_literal() {
        let e = expr_from_filter("t.filter(x = null)");
        let Expr::BinaryOp(b) = e else { panic!() };
        assert!(matches!(b.right, Expr::Literal(Literal::Null(_))));
    }

    // ── operator precedence ───────────────────────────────────────────────────

    #[test]
    fn pass_mul_before_add() {
        // `a + b * c` should parse as `a + (b * c)`, i.e. the top node is Add
        let e = expr_from_filter("t.filter(a + b * c = 1)");
        let Expr::BinaryOp(top) = e else { panic!() };
        assert_eq!(top.op, BinaryOp::Eq);
        let Expr::BinaryOp(lhs) = &top.left else { panic!("left should be BinaryOp") };
        assert_eq!(lhs.op, BinaryOp::Add);
        let Expr::BinaryOp(rhs) = &lhs.right else { panic!("inner right should be BinaryOp") };
        assert_eq!(rhs.op, BinaryOp::Mul);
    }

    #[test]
    fn pass_and_before_or() {
        // `a or b and c` → `a or (b and c)`
        let e = expr_from_filter("t.filter(a or b and c)");
        let Expr::BinaryOp(top) = e else { panic!() };
        assert_eq!(top.op, BinaryOp::Or);
        assert!(matches!(&top.right, Expr::BinaryOp(r) if r.op == BinaryOp::And));
    }

    #[test]
    fn pass_not_binds_tighter_than_and() {
        // `not a and b` → `(not a) and b`
        let e = expr_from_filter("t.filter(not a and b)");
        let Expr::BinaryOp(top) = e else { panic!() };
        assert_eq!(top.op, BinaryOp::And);
        assert!(matches!(&top.left, Expr::UnaryOp(u) if u.op == UnaryOp::Not));
    }

    #[test]
    fn pass_parentheses_override_precedence() {
        // `(a + b) * c` — top node should be Mul
        let stmt = first_stmt("t.map((a + b) * c as result)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Map(m) = &q.ops[0] else { panic!() };
        let Expr::BinaryOp(top) = &m.expr else { panic!() };
        assert_eq!(top.op, BinaryOp::Mul);
        assert!(matches!(&top.left, Expr::Grouped(_,_)));
    }

    #[test]
    fn pass_unary_neg() {
        let stmt = first_stmt("t.map(-price as neg_price)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Map(m) = &q.ops[0] else { panic!() };
        assert!(matches!(m.expr, Expr::UnaryOp(ref u) if u.op == UnaryOp::Neg));
    }

    // ── comparisons ───────────────────────────────────────────────────────────

    #[test]
    fn pass_all_comparators() {
        for (op_str, expected_op) in [
            ("=",  BinaryOp::Eq),
            ("!=", BinaryOp::NotEq),
            (">",  BinaryOp::Gt),
            ("<",  BinaryOp::Lt),
            (">=", BinaryOp::Gte),
            ("<=", BinaryOp::Lte),
        ] {
            let e = expr_from_filter(&format!("t.filter(a {op_str} b)"));
            let Expr::BinaryOp(b) = e else { panic!("{op_str}") };
            assert_eq!(b.op, expected_op, "operator {op_str}");
        }
    }

    // ── qualified column ──────────────────────────────────────────────────────

    #[test]
    fn pass_qualified_column() {
        let e = expr_from_filter("t.filter(sales.revenue > 100)");
        let Expr::BinaryOp(b) = e else { panic!() };
        let Expr::Column(c) = &b.left else { panic!() };
        assert_eq!(c.table.as_deref(), Some("sales"));
        assert_eq!(c.column, "revenue");
    }

    #[test]
    fn pass_bare_column() {
        let e = expr_from_filter("t.filter(revenue > 100)");
        let Expr::BinaryOp(b) = e else { panic!() };
        let Expr::Column(c) = &b.left else { panic!() };
        assert!(c.table.is_none());
        assert_eq!(c.column, "revenue");
    }

    // ── function calls ────────────────────────────────────────────────────────

    #[test]
    fn pass_function_call_no_args() {
        let stmt = first_stmt("t.map(now() as ts)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Map(m) = &q.ops[0] else { panic!() };
        let Expr::FunctionCall(f) = &m.expr else { panic!() };
        assert_eq!(f.name, "now");
        assert!(f.args.is_empty());
    }

    #[test]
    fn pass_function_call_multiple_args() {
        let stmt = first_stmt("t.map(coalesce(a, b, 0) as val)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Map(m) = &q.ops[0] else { panic!() };
        let Expr::FunctionCall(f) = &m.expr else { panic!() };
        assert_eq!(f.name, "coalesce");
        assert_eq!(f.args.len(), 3);
    }

    #[test]
    fn pass_nested_function_call() {
        let stmt = first_stmt("t.map(to_int(year(date)) as yr_int)");
        let Statement::Query(q) = stmt else { panic!() };
        let PipeOp::Map(m) = &q.ops[0] else { panic!() };
        let Expr::FunctionCall(outer) = &m.expr else { panic!() };
        assert_eq!(outer.name, "to_int");
        assert!(matches!(&outer.args[0], FunctionArg::Expr(Expr::FunctionCall(_))));
    }

    // ── fail cases ────────────────────────────────────────────────────────────

    #[test]
    fn fail_bare_operator_no_left() {
        parse_err("t.filter(= 5)");
    }

    #[test]
    fn fail_operator_no_right() {
        parse_err("t.filter(x =)");
    }

    #[test]
    fn fail_unclosed_paren() {
        parse_err("t.filter((x = 1)");
    }
}

// ---------------------------------------------------------------------------
// ── Parser: full chain integration ───────────────────────────────────────────
// ---------------------------------------------------------------------------

mod parse_full_chain {
    use super::*;

    #[test]
    fn pass_all_ops_in_chain() {
        let prog = parse_ok(
            r#"
sales
  .filter(active = true)
  .map(price * qty as total)
  .aggregate(sum(total) as revenue by country)
  .sort_by(revenue desc)
  .limit(10)
  .offset(0)
  as top_countries
"#,
        );
        let Statement::Query(q) = &prog.statements[0] else { panic!() };
        assert_eq!(q.ops.len(), 6); // filter map aggregate sort_by limit offset
        assert_eq!(q.materialize.as_ref().unwrap().name, "top_countries");
    }

    #[test]
    fn pass_spec_example_query() {
        // Full example from RESIN.md §3
        parse_ok(
            r#"
sales_enriched
  .filter(product_info.category = "electronics" and customer_info.country is not null)
  .map(product_info.price * sales.quantity as total)
  .map(coalesce(customer_info.discount, 0) as safe_discount)
  .map(total * (1 - safe_discount) as net_revenue)
  .aggregate(sum(net_revenue) as revenue, count(*) as order_count by customer_info.country)
  .sort_by(revenue desc)
  .limit(20)
"#,
        );
    }

    #[test]
    fn pass_join_chain() {
        parse_ok(
            r#"
sales
  .join(returns on sales.id = returns.sale_id)
  .filter(returns.reason = "defective")
  .select(sales.product_id, returns.date)
"#,
        );
    }

    #[test]
    fn pass_error_recovery_multiple_statements() {
        // First statement is broken; parser should recover and parse the second.
        let tokens = lex("use sales from\nuse customers from \"c\"");
        let (prog, errors) = parser::parse(tokens);
        // At least one error (missing string after first `from`)
        assert!(!errors.is_empty());
        // Second `use` statement should still be parsed
        assert!(!prog.statements.is_empty());
    }
}

// ---------------------------------------------------------------------------
// Parser: if / else if / else expressions
// ---------------------------------------------------------------------------

mod parse_if_expr {
    use super::*;

    fn parse_if_in_map(src: &str) -> Expr {
        let prog = parse_ok(src);
        match prog.statements.first().unwrap() {
            Statement::Query(q) => match q.ops.first().unwrap() {
                PipeOp::Map(m) => m.expr.clone(),
                _ => panic!("expected map op"),
            },
            _ => panic!("expected query statement"),
        }
    }

    fn parse_if_in_filter(src: &str) -> Expr {
        let prog = parse_ok(src);
        match prog.statements.first().unwrap() {
            Statement::Query(q) => match q.ops.first().unwrap() {
                PipeOp::Filter(f) => f.condition.clone(),
                _ => panic!("expected filter op"),
            },
            _ => panic!("expected query statement"),
        }
    }

    // ── pass cases ─────────────────────────────────────────────────────────

    #[test]
    fn pass_if_else_in_map() {
        let expr = parse_if_in_map("t.map(if x > 0 { x } else { 0 } as result)");
        match expr {
            Expr::If(e) => {
                assert!(e.else_if_branches.is_empty());
                assert!(e.else_branch.is_some());
            }
            _ => panic!("expected IfExpr, got {expr:?}"),
        }
    }

    #[test]
    fn pass_if_no_else() {
        // `else` is optional — missing else returns null
        let expr = parse_if_in_map("t.map(if flag { 1 } as v)");
        match expr {
            Expr::If(e) => {
                assert!(e.else_if_branches.is_empty());
                assert!(e.else_branch.is_none());
            }
            _ => panic!("expected IfExpr"),
        }
    }

    #[test]
    fn pass_if_else_if_else() {
        let expr = parse_if_in_map(
            "t.map(if score >= 90 { 4 } else if score >= 70 { 3 } else if score >= 50 { 2 } else { 1 } as grade)",
        );
        match expr {
            Expr::If(e) => {
                assert_eq!(e.else_if_branches.len(), 2);
                assert!(e.else_branch.is_some());
            }
            _ => panic!("expected IfExpr"),
        }
    }

    #[test]
    fn pass_if_in_filter() {
        // if expression as the filter predicate itself
        let expr = parse_if_in_filter("t.filter(if x > 0 { true } else { false })");
        assert!(matches!(expr, Expr::If(_)));
    }

    #[test]
    fn pass_nested_if() {
        // if inside the body of another if
        let expr = parse_if_in_map(
            "t.map(if a > 0 { if b > 0 { 1 } else { 2 } } else { 3 } as v)",
        );
        match expr {
            Expr::If(outer) => {
                assert!(matches!(outer.then_branch, Expr::If(_)));
            }
            _ => panic!("expected nested IfExpr"),
        }
    }

    #[test]
    fn pass_if_condition_is_compound() {
        let expr = parse_if_in_map(
            r#"t.map(if country = "FR" and active = true { discount } else { 0 } as d)"#,
        );
        assert!(matches!(expr, Expr::If(_)));
    }

    #[test]
    fn pass_if_body_is_arithmetic() {
        let expr = parse_if_in_map("t.map(if flag { x * 2 } else { x + 1 } as v)");
        match expr {
            Expr::If(e) => {
                assert!(matches!(e.then_branch, Expr::BinaryOp(_)));
                assert!(matches!(e.else_branch.unwrap(), Expr::BinaryOp(_)));
            }
            _ => panic!("expected IfExpr"),
        }
    }

    // ── fail cases ──────────────────────────────────────────────────────────

    #[test]
    fn fail_if_missing_lbrace() {
        // Missing `{` after condition
        parse_err("t.map(if x > 0 x as v)");
    }

    #[test]
    fn fail_if_missing_rbrace() {
        // Missing closing `}` on then branch
        parse_err("t.map(if x > 0 { x as v)");
    }

    #[test]
    fn fail_if_else_missing_lbrace() {
        parse_err("t.map(if x > 0 { 1 } else 0 as v)");
    }

    #[test]
    fn fail_if_else_if_missing_condition() {
        parse_err("t.map(if x > 0 { 1 } else if { 2 } else { 3 } as v)");
    }
}

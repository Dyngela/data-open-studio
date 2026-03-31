//! Executor integration tests.
//!
//! Each test builds one or more `Frame` objects, loads them into `Executor`,
//! runs a Resin program, and asserts on the resulting frame's shape / values.

#![cfg(test)]

use std::collections::HashMap;

use df_store::chunk::Value;
use df_store::data_type::{DataType, DataValue};
use df_store::frame::Frame;
use df_store::store::{Column, Series};

use crate::executor::Executor;
use crate::lexer::Lexer;
use crate::parser::parse;

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

fn make_i64(v: i64) -> Value   { Value { data: DataValue::Int64(v),            validity: vec![1] } }
fn make_f64(v: f64) -> Value   { Value { data: DataValue::Float64(v),          validity: vec![1] } }
fn make_str(v: &str) -> Value  { Value { data: DataValue::String(v.to_owned()), validity: vec![1] } }
fn make_bool(v: bool) -> Value { Value { data: DataValue::Boolean(v),          validity: vec![1] } }
fn null_val() -> Value         { Value { data: DataValue::Null,                validity: vec![0] } }

fn int_series(name: &str, vals: &[i64]) -> Series {
    Series {
        field: Column { name: name.to_owned(), dtype: DataType::Int64 },
        chunks: vals.iter().map(|&v| make_i64(v)).collect(),
        len: vals.len(),
        null_count: 0,
        dict: HashMap::new(),
        min: None,
        max: None,
    }
}

fn str_series(name: &str, vals: &[&str]) -> Series {
    Series {
        field: Column { name: name.to_owned(), dtype: DataType::String },
        chunks: vals.iter().map(|v| make_str(v)).collect(),
        len: vals.len(),
        null_count: 0,
        dict: HashMap::new(),
        min: None,
        max: None,
    }
}

fn f64_series(name: &str, vals: &[f64]) -> Series {
    Series {
        field: Column { name: name.to_owned(), dtype: DataType::Float64 },
        chunks: vals.iter().map(|&v| make_f64(v)).collect(),
        len: vals.len(),
        null_count: 0,
        dict: HashMap::new(),
        min: None,
        max: None,
    }
}

fn bool_series(name: &str, vals: &[bool]) -> Series {
    Series {
        field: Column { name: name.to_owned(), dtype: DataType::Boolean },
        chunks: vals.iter().map(|&v| make_bool(v)).collect(),
        len: vals.len(),
        null_count: 0,
        dict: HashMap::new(),
        min: None,
        max: None,
    }
}

fn nullable_int_series(name: &str, vals: &[Option<i64>]) -> Series {
    let null_count = vals.iter().filter(|v| v.is_none()).count();
    Series {
        field: Column { name: name.to_owned(), dtype: DataType::Int64 },
        chunks: vals.iter().map(|v| match v {
            Some(n) => make_i64(*n),
            None    => null_val(),
        }).collect(),
        len: vals.len(),
        null_count,
        dict: HashMap::new(),
        min: None,
        max: None,
    }
}

fn mk_frame(name: &str, cols: Vec<Series>) -> Frame {
    Frame { name: name.to_owned(), columns: cols }
}

/// Lex + parse + run. Panics on any error.
fn run(ex: &mut Executor, src: &str) {
    let tokens = Lexer::new(src).tokenize()
        .unwrap_or_else(|e| panic!("lex errors: {e:#?}"));
    let (prog, errs) = parse(tokens);
    assert!(errs.is_empty(), "parse errors:\n{}", errs.iter().map(|e| e.to_string()).collect::<Vec<_>>().join("\n"));
    ex.run(&prog).expect("exec ok");
}

/// Collect i64 column values from a frame (i64::MIN as null sentinel).
fn col_i64(frame: &Frame, col: &str) -> Vec<i64> {
    let s = frame.columns.iter().find(|s| s.field.name == col)
        .unwrap_or_else(|| panic!("column '{col}' not found in {:?}", frame.columns.iter().map(|c| &c.field.name).collect::<Vec<_>>()));
    s.chunks.iter().map(|v| {
        let valid = v.validity.first().map(|b| b & 1 == 1).unwrap_or(false);
        if !valid { return i64::MIN; }
        match &v.data { DataValue::Int64(n) => *n, other => panic!("expected Int64, got {other:?}") }
    }).collect()
}

fn col_str(frame: &Frame, col: &str) -> Vec<String> {
    let s = frame.columns.iter().find(|s| s.field.name == col)
        .unwrap_or_else(|| panic!("column '{col}' not found"));
    s.chunks.iter().map(|v| {
        let valid = v.validity.first().map(|b| b & 1 == 1).unwrap_or(false);
        if !valid { return "__null__".to_owned(); }
        match &v.data { DataValue::String(s) => s.clone(), other => panic!("expected String, got {other:?}") }
    }).collect()
}

fn col_f64(frame: &Frame, col: &str) -> Vec<f64> {
    let s = frame.columns.iter().find(|s| s.field.name == col)
        .unwrap_or_else(|| panic!("column '{col}' not found"));
    s.chunks.iter().map(|v| {
        let valid = v.validity.first().map(|b| b & 1 == 1).unwrap_or(false);
        if !valid { return f64::NAN; }
        match &v.data {
            DataValue::Float64(f) => *f,
            DataValue::Int64(n)   => *n as f64,
            other => panic!("expected Float64/Int64, got {other:?}"),
        }
    }).collect()
}

fn is_null_at(frame: &Frame, col: &str, row: usize) -> bool {
    let s = frame.columns.iter().find(|s| s.field.name == col)
        .unwrap_or_else(|| panic!("column '{col}' not found"));
    let v = &s.chunks[row];
    v.validity.first().map(|b| b & 1 == 1).unwrap_or(false) == false
}

fn row_count(frame: &Frame) -> usize {
    frame.columns.first().map(|s| s.chunks.len()).unwrap_or(0)
}

fn col_names(frame: &Frame) -> Vec<&str> {
    frame.columns.iter().map(|s| s.field.name.as_str()).collect()
}

// ---------------------------------------------------------------------------
// use statement
// ---------------------------------------------------------------------------

#[test]
fn exec_use_aliases_frame() {
    let mut ex = Executor::new();
    ex.load("raw", mk_frame("raw", vec![int_series("id", &[1, 2, 3])]));

    run(&mut ex, r#"use orders from "raw""#);

    let f = ex.get("orders").expect("orders frame present");
    assert_eq!(row_count(f), 3);
    assert_eq!(col_i64(f, "id"), vec![1, 2, 3]);
}

#[test]
fn exec_use_same_name_ok() {
    let mut ex = Executor::new();
    ex.load("data", mk_frame("data", vec![int_series("x", &[10])]));

    run(&mut ex, r#"use data from "data""#);

    assert!(ex.get("data").is_some());
}

#[test]
fn exec_use_missing_frame_errors() {
    let mut ex = Executor::new();
    let tokens = Lexer::new(r#"use missing from "missing""#).tokenize().unwrap();
    let (prog, errs) = parse(tokens);
    assert!(errs.is_empty());
    let result = ex.run(&prog);
    assert!(result.is_err(), "should fail when frame not pre-loaded");
}

// ---------------------------------------------------------------------------
// filter
// ---------------------------------------------------------------------------

#[test]
fn exec_filter_gt() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("v", &[1, 5, 3, 8, 2])]));

    run(&mut ex, "t.filter(v > 3) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "v"), vec![5, 8]);
}

#[test]
fn exec_filter_eq_string() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        int_series("id",   &[1, 2, 3]),
        str_series("name", &["alice", "bob", "alice"]),
    ]));

    run(&mut ex, r#"t.filter(name = "alice") as result"#);

    let f = ex.get("result").unwrap();
    assert_eq!(row_count(f), 2);
    assert_eq!(col_i64(f, "id"), vec![1, 3]);
}

#[test]
fn exec_filter_and() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        int_series("a", &[1, 2, 3, 4]),
        int_series("b", &[10, 20, 10, 20]),
    ]));

    run(&mut ex, "t.filter(a > 1 and b = 10) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "a"), vec![3]);
}

#[test]
fn exec_filter_or() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("x", &[1, 2, 3, 4, 5])]));

    run(&mut ex, "t.filter(x = 1 or x = 5) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "x"), vec![1, 5]);
}

#[test]
fn exec_filter_not() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("x", &[1, 2, 3, 4])]));

    run(&mut ex, "t.filter(not x = 2) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "x"), vec![1, 3, 4]);
}

#[test]
fn exec_filter_is_null() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        nullable_int_series("v", &[Some(1), None, Some(3)]),
    ]));

    run(&mut ex, "t.filter(v is null) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(row_count(f), 1);
}

#[test]
fn exec_filter_is_not_null() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        nullable_int_series("v", &[Some(1), None, Some(3)]),
    ]));

    run(&mut ex, "t.filter(v is not null) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(row_count(f), 2);
    let vals = col_i64(f, "v");
    assert!(vals.contains(&1));
    assert!(vals.contains(&3));
}

#[test]
fn exec_filter_empty_result() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("x", &[1, 2, 3])]));

    run(&mut ex, "t.filter(x > 100) as result");

    assert_eq!(row_count(ex.get("result").unwrap()), 0);
}

#[test]
fn exec_filter_bool_column() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        int_series("id",    &[1, 2, 3, 4]),
        bool_series("active", &[true, false, true, false]),
    ]));

    run(&mut ex, "t.filter(active) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "id"), vec![1, 3]);
}

// ---------------------------------------------------------------------------
// select
// ---------------------------------------------------------------------------

#[test]
fn exec_select_drops_other_cols() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        int_series("a", &[1, 2]),
        int_series("b", &[3, 4]),
        int_series("c", &[5, 6]),
    ]));

    run(&mut ex, "t.select(a, c) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_names(f), vec!["a", "c"]);
    assert_eq!(col_i64(f, "a"), vec![1, 2]);
    assert_eq!(col_i64(f, "c"), vec![5, 6]);
}

// ---------------------------------------------------------------------------
// map
// ---------------------------------------------------------------------------

#[test]
fn exec_map_adds_column() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        int_series("x", &[1, 2, 3]),
        int_series("y", &[10, 20, 30]),
    ]));

    run(&mut ex, "t.map(x + y as total) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "total"), vec![11, 22, 33]);
}

#[test]
fn exec_map_multiply() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("x", &[2, 4, 6])]));

    run(&mut ex, "t.map(x * 3 as triple) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "triple"), vec![6, 12, 18]);
}

#[test]
fn exec_map_literal_column() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("id", &[1, 2, 3])]));

    run(&mut ex, "t.map(42 as flag) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "flag"), vec![42, 42, 42]);
}

// ---------------------------------------------------------------------------
// aggregate
// ---------------------------------------------------------------------------

#[test]
fn exec_aggregate_sum_global() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("v", &[1, 2, 3, 4])]));

    run(&mut ex, "t.aggregate(sum(v) as total) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(row_count(f), 1);
    assert_eq!(col_i64(f, "total"), vec![10]);
}

#[test]
fn exec_aggregate_count_global() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("x", &[5, 6, 7])]));

    run(&mut ex, "t.aggregate(count(*) as n) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "n"), vec![3]);
}

#[test]
fn exec_aggregate_group_by() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        str_series("cat", &["a", "b", "a", "b", "a"]),
        int_series("val", &[1,   2,   3,   4,   5]),
    ]));

    run(&mut ex, "t.aggregate(sum(val) as total by cat) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(row_count(f), 2);
    let cats   = col_str(f, "cat");
    let totals = col_i64(f, "total");
    let idx_a  = cats.iter().position(|c| c == "a").expect("group a");
    let idx_b  = cats.iter().position(|c| c == "b").expect("group b");
    assert_eq!(totals[idx_a], 9);  // 1+3+5
    assert_eq!(totals[idx_b], 6);  // 2+4
}

#[test]
fn exec_aggregate_min_max() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("v", &[3, 1, 4, 1, 5, 9])]));

    run(&mut ex, "t.aggregate(min(v) as lo, max(v) as hi) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "lo"), vec![1]);
    assert_eq!(col_i64(f, "hi"), vec![9]);
}

#[test]
fn exec_aggregate_avg() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("v", &[2, 4, 6])]));

    run(&mut ex, "t.aggregate(avg(v) as mean) as result");

    let f = ex.get("result").unwrap();
    let vals = col_f64(f, "mean");
    assert!((vals[0] - 4.0).abs() < 1e-9, "expected 4.0, got {}", vals[0]);
}

// ---------------------------------------------------------------------------
// distinct
// ---------------------------------------------------------------------------

#[test]
fn exec_distinct_removes_duplicates() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        int_series("x", &[1, 2, 1, 3, 2]),
    ]));

    run(&mut ex, "t.distinct() as result");

    let f = ex.get("result").unwrap();
    assert_eq!(row_count(f), 3);
}

#[test]
fn exec_distinct_multi_col() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        str_series("a", &["x", "x", "y", "x"]),
        int_series("b", &[1,   1,   1,   2]),
    ]));

    run(&mut ex, "t.distinct() as result");

    let f = ex.get("result").unwrap();
    // (x,1), (y,1), (x,2) = 3 distinct rows
    assert_eq!(row_count(f), 3);
}

// ---------------------------------------------------------------------------
// sort_by
// ---------------------------------------------------------------------------

#[test]
fn exec_sort_by_asc() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("v", &[3, 1, 4, 1, 5])]));

    run(&mut ex, "t.sort_by(v asc) as result");

    let f = ex.get("result").unwrap();
    let vals = col_i64(f, "v");
    for w in vals.windows(2) {
        assert!(w[0] <= w[1], "not sorted asc: {vals:?}");
    }
}

#[test]
fn exec_sort_by_desc() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("v", &[3, 1, 4, 1, 5])]));

    run(&mut ex, "t.sort_by(v desc) as result");

    let f = ex.get("result").unwrap();
    let vals = col_i64(f, "v");
    for w in vals.windows(2) {
        assert!(w[0] >= w[1], "not sorted desc: {vals:?}");
    }
}

#[test]
fn exec_sort_by_multi_key() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        str_series("cat", &["b", "a", "b", "a"]),
        int_series("val", &[2,   3,   1,   4]),
    ]));

    run(&mut ex, "t.sort_by(cat asc, val asc) as result");

    let f = ex.get("result").unwrap();
    let cats = col_str(f, "cat");
    let vals = col_i64(f, "val");
    assert_eq!(&cats[0..2], &["a", "a"]);
    assert_eq!(&cats[2..4], &["b", "b"]);
    assert!(vals[0] < vals[1], "a-group not sorted asc");
    assert!(vals[2] < vals[3], "b-group not sorted asc");
}

// ---------------------------------------------------------------------------
// limit / offset
// ---------------------------------------------------------------------------

#[test]
fn exec_limit() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("v", &[1, 2, 3, 4, 5])]));

    run(&mut ex, "t.limit(3) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "v"), vec![1, 2, 3]);
}

#[test]
fn exec_offset() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("v", &[1, 2, 3, 4, 5])]));

    run(&mut ex, "t.offset(2) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "v"), vec![3, 4, 5]);
}

#[test]
fn exec_limit_after_offset() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("v", &[1, 2, 3, 4, 5])]));

    run(&mut ex, "t.offset(1).limit(2) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_i64(f, "v"), vec![2, 3]);
}

#[test]
fn exec_limit_larger_than_frame() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("v", &[1, 2])]));

    run(&mut ex, "t.limit(100) as result");

    assert_eq!(row_count(ex.get("result").unwrap()), 2);
}

// ---------------------------------------------------------------------------
// join
// ---------------------------------------------------------------------------

#[test]
fn exec_inner_join() {
    let mut ex = Executor::new();
    ex.load("lft", mk_frame("lft", vec![
        int_series("id",  &[1, 2, 3]),
        str_series("l_v", &["a", "b", "c"]),
    ]));
    ex.load("rgt", mk_frame("rgt", vec![
        int_series("id",  &[2, 3, 4]),
        str_series("r_v", &["B", "C", "D"]),
    ]));

    run(&mut ex, "lft.join(rgt on lft.id = rgt.id) as result");

    let f = ex.get("result").unwrap();
    // Only rows where id matches: 2 and 3
    assert_eq!(row_count(f), 2);
}

#[test]
fn exec_left_join_keeps_all_left() {
    let mut ex = Executor::new();
    ex.load("lft", mk_frame("lft", vec![
        int_series("id", &[1, 2, 3]),
    ]));
    ex.load("rgt", mk_frame("rgt", vec![
        int_series("id",  &[2]),
        str_series("val", &["x"]),
    ]));

    run(&mut ex, "lft.left_join(rgt on lft.id = rgt.id) as result");

    let f = ex.get("result").unwrap();
    // All 3 left rows preserved
    assert_eq!(row_count(f), 3);
}

#[test]
fn exec_cross_join() {
    let mut ex = Executor::new();
    ex.load("a", mk_frame("a", vec![int_series("x", &[1, 2])]));
    ex.load("b", mk_frame("b", vec![int_series("y", &[10, 20, 30])]));

    run(&mut ex, "a.cross_join(b) as result");

    let f = ex.get("result").unwrap();
    // 2 * 3 = 6 rows
    assert_eq!(row_count(f), 6);
}

// ---------------------------------------------------------------------------
// union / intersect / except
// ---------------------------------------------------------------------------

#[test]
fn exec_union_deduplicates() {
    let mut ex = Executor::new();
    ex.load("a", mk_frame("a", vec![int_series("v", &[1, 2, 3])]));
    ex.load("b", mk_frame("b", vec![int_series("v", &[2, 3, 4])]));

    run(&mut ex, "a.union(b) as result");

    let f = ex.get("result").unwrap();
    // 1,2,3 + 4 (2 and 3 deduped) = 4 rows
    assert_eq!(row_count(f), 4);
}

#[test]
fn exec_intersect() {
    let mut ex = Executor::new();
    ex.load("a", mk_frame("a", vec![int_series("v", &[1, 2, 3])]));
    ex.load("b", mk_frame("b", vec![int_series("v", &[2, 3, 4])]));

    run(&mut ex, "a.intersect(b) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(row_count(f), 2);
    let vals = col_i64(f, "v");
    assert!(vals.contains(&2));
    assert!(vals.contains(&3));
}

#[test]
fn exec_except() {
    let mut ex = Executor::new();
    ex.load("a", mk_frame("a", vec![int_series("v", &[1, 2, 3])]));
    ex.load("b", mk_frame("b", vec![int_series("v", &[2, 3, 4])]));

    run(&mut ex, "a.except(b) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(row_count(f), 1);
    assert_eq!(col_i64(f, "v"), vec![1]);
}

// ---------------------------------------------------------------------------
// chained operations
// ---------------------------------------------------------------------------

#[test]
fn exec_filter_then_map_then_limit() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        int_series("x", &[1, 2, 3, 4, 5, 6]),
    ]));

    run(&mut ex, "t.filter(x > 2).map(x * 10 as y).limit(3) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(row_count(f), 3);
    assert_eq!(col_i64(f, "y"), vec![30, 40, 50]);
}

#[test]
fn exec_group_then_sort() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        str_series("cat", &["b", "a", "b", "a", "c"]),
        int_series("val", &[2,   1,   4,   3,   5]),
    ]));

    run(&mut ex, "t.aggregate(sum(val) as total by cat).sort_by(total asc) as result");

    let f = ex.get("result").unwrap();
    let vals = col_i64(f, "total");
    for w in vals.windows(2) {
        assert!(w[0] <= w[1], "not sorted asc: {vals:?}");
    }
}

#[test]
fn exec_select_then_aggregate() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        str_series("cat",  &["a", "b", "a"]),
        int_series("val",  &[10, 20, 30]),
        int_series("junk", &[99, 88, 77]),
    ]));

    run(&mut ex, "t.select(cat, val).aggregate(sum(val) as s by cat) as result");

    let f = ex.get("result").unwrap();
    assert_eq!(col_names(f).len(), 2); // cat + s
    assert_eq!(row_count(f), 2);
}

// ---------------------------------------------------------------------------
// materialization: default name vs explicit `as`
// ---------------------------------------------------------------------------

#[test]
fn exec_materialize_default_name() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("v", &[1])]));

    run(&mut ex, "t.limit(1)");

    // No `as` so result goes to "__result"
    assert!(ex.get("__result").is_some());
}

#[test]
fn exec_materialize_custom_name() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![int_series("v", &[1, 2])]));

    run(&mut ex, "t.filter(v > 0) as my_out");

    assert!(ex.get("my_out").is_some());
    assert_eq!(row_count(ex.get("my_out").unwrap()), 2);
}

// ---------------------------------------------------------------------------
// null propagation
// ---------------------------------------------------------------------------

#[test]
fn exec_arithmetic_with_null_propagates() {
    let mut ex = Executor::new();
    ex.load("t", mk_frame("t", vec![
        nullable_int_series("a", &[Some(5), None, Some(3)]),
        int_series("b", &[2, 2, 2]),
    ]));

    run(&mut ex, "t.map(a + b as c) as result");

    let f = ex.get("result").unwrap();
    // Row 0: 5+2=7 (valid)
    assert!(!is_null_at(f, "c", 0));
    assert_eq!(col_i64(f, "c")[0], 7);
    // Row 1: null+2=null
    assert!(is_null_at(f, "c", 1));
    // Row 2: 3+2=5 (valid)
    assert!(!is_null_at(f, "c", 2));
    assert_eq!(col_i64(f, "c")[2], 5);
}

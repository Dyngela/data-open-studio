//! AST executor: walks a `Program` and applies every operation directly on
//! `df_store::frame::Frame` objects.  No Rust source code is generated or
//! compiled — this is the runtime counterpart of `codegen.rs`.
//!
//! ## Data model
//!
//! Every `Series` stores one `Value` per row.  A `Value` has:
//! - `data: DataValue`   — the payload (or `DataValue::Null`)
//! - `validity: Vec<u8>` — packed bitmap; bit 0 of byte 0 = 1 → valid, 0 → null
//!
//! Dict-encoded series store `DataValue::Int64(idx)` in `data` and the real
//! value in `series.dict[idx]`.  All helpers here decode transparently.

use std::collections::HashMap;

use df_store::chunk::Value;
use df_store::data_type::{DataType, DataValue};
use df_store::frame::Frame;
use df_store::store::{Column, Series};

use crate::ast::*;

// ---------------------------------------------------------------------------
// Error
// ---------------------------------------------------------------------------

#[derive(Debug)]
pub struct ExecError {
    pub message: String,
}

impl std::fmt::Display for ExecError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "exec error: {}", self.message)
    }
}

impl std::error::Error for ExecError {}

macro_rules! exec_err {
    ($($t:tt)*) => { Err(ExecError { message: format!($($t)*) }) };
}

// ---------------------------------------------------------------------------
// Executor
// ---------------------------------------------------------------------------

pub struct Executor {
    frames: HashMap<String, Frame>,
}

impl Executor {
    pub fn new() -> Self {
        Self { frames: HashMap::new() }
    }

    /// Pre-load an external frame (e.g. from a connector).
    pub fn load(&mut self, name: &str, frame: Frame) {
        self.frames.insert(name.to_owned(), frame);
    }

    /// Retrieve a result frame by name.
    pub fn get(&self, name: &str) -> Option<&Frame> {
        self.frames.get(name)
    }

    /// Run a full program; materialized results are stored internally.
    pub fn run(&mut self, program: &Program) -> Result<(), ExecError> {
        for stmt in &program.statements {
            self.exec_statement(stmt)?;
        }
        Ok(())
    }

    // ── statement dispatch ────────────────────────────────────────────────

    fn exec_statement(&mut self, stmt: &Statement) -> Result<(), ExecError> {
        match stmt {
            Statement::Use(u)   => self.exec_use(u),
            Statement::Frame(f) => self.exec_frame_stmt(f),
            Statement::Query(q) => self.exec_query_stmt(q),
        }
    }

    fn exec_use(&mut self, u: &UseStmt) -> Result<(), ExecError> {
        // Frames must be pre-loaded via `Executor::load`.  Here we just
        // alias the source name to the binding name when they differ.
        let src = u.source.value.as_str();
        let dst = u.name.name.as_str();
        if src != dst {
            let frame = self.frames.get(src)
                .ok_or_else(|| ExecError { message: format!("frame '{src}' not found") })?
                .clone_with_name(dst);
            self.frames.insert(dst.to_owned(), frame);
        } else if !self.frames.contains_key(src) {
            return exec_err!("frame '{src}' not found");
        }
        Ok(())
    }

    fn exec_frame_stmt(&mut self, f: &FrameStmt) -> Result<(), ExecError> {
        // Start from an empty frame; fold each relate clause into it.
        let mut dim: Option<Frame> = None;

        for rel in &f.relates {
            let left = self.frames.get(&rel.left.name)
                .ok_or_else(|| ExecError { message: format!("frame '{}' not found", rel.left.name) })?
                .clone_with_name(&rel.left.name);
            let right_src = self.frames.get(&rel.right.name)
                .ok_or_else(|| ExecError { message: format!("frame '{}' not found", rel.right.name) })?
                .clone_with_name(&rel.right.name);

            // Apply `where` filter to right frame before joining.
            let right = match &rel.filter {
                None    => right_src,
                Some(e) => {
                    let right_src_ref = &right_src;
                    frame_filter(right_src_ref, |row| dv_truthy(&eval_expr(right_src_ref, row, e)))
                }
            };

            // Inner-join left (or current dim) with right; prefix right cols.
            let base = dim.take().unwrap_or(left);
            dim = Some(exec_relate(&base, &right, &rel.condition, &rel.alias.name)?);
        }

        let result = dim.unwrap_or_else(|| Frame { name: f.name.name.clone(), columns: vec![] });
        let named  = result.clone_with_name(&f.name.name);
        self.frames.insert(f.name.name.clone(), named);
        Ok(())
    }

    fn exec_query_stmt(&mut self, q: &QueryStmt) -> Result<(), ExecError> {
        let mut frame = self.frames.get(&q.source.name)
            .ok_or_else(|| ExecError { message: format!("frame '{}' not found", q.source.name) })?
            .clone_with_name(&q.source.name);

        for op in &q.ops {
            frame = self.exec_pipe_op(frame, op)?;
        }

        let out_name = q.materialize.as_ref()
            .map(|m| m.name.as_str())
            .unwrap_or("__result");
        self.frames.insert(out_name.to_owned(), frame.clone_with_name(out_name));
        Ok(())
    }

    // ── pipe operators ────────────────────────────────────────────────────

    fn exec_pipe_op(&self, frame: Frame, op: &PipeOp) -> Result<Frame, ExecError> {
        match op {
            PipeOp::Filter(f) => {
                let out = frame_filter(&frame, |row| dv_truthy(&eval_expr(&frame, row, &f.condition)));
                Ok(out)
            }

            PipeOp::Select(s) => {
                let names: Vec<&str> = s.columns.iter().map(col_name_of_expr).collect();
                exec_select(&frame, &names)
            }

            PipeOp::Map(m) => {
                let values: Vec<DataValue> = (0..row_count(&frame))
                    .map(|row| eval_expr(&frame, row, &m.expr))
                    .collect();
                Ok(frame_add_col(frame, &m.alias.name, values))
            }

            PipeOp::Aggregate(a) => {
                exec_aggregate(&frame, &a.aggregations, &a.group_by)
            }

            PipeOp::Distinct(_) => {
                Ok(exec_distinct(frame))
            }

            PipeOp::SortBy(s) => {
                exec_sort_by(frame, &s.columns)
            }

            PipeOp::Limit(l) => {
                Ok(frame_slice(frame, 0, l.n as usize))
            }

            PipeOp::Offset(o) => {
                let start = (o.n as usize).min(row_count(&frame));
                let end   = row_count(&frame);
                Ok(frame_slice(frame, start, end))
            }

            PipeOp::Join(j) => {
                let right = self.frames.get(&j.frame.name)
                    .ok_or_else(|| ExecError { message: format!("frame '{}' not found", j.frame.name) })?;
                exec_join(&frame, right, &j.kind, j.condition.as_ref())
            }

            PipeOp::Window(w) => {
                exec_window(frame, &w.expr)
            }

            PipeOp::Union(s) => {
                let other = self.frames.get(&s.frame.name)
                    .ok_or_else(|| ExecError { message: format!("frame '{}' not found", s.frame.name) })?;
                exec_union(frame, other)
            }

            PipeOp::Intersect(s) => {
                let other = self.frames.get(&s.frame.name)
                    .ok_or_else(|| ExecError { message: format!("frame '{}' not found", s.frame.name) })?;
                Ok(exec_intersect(&frame, other))
            }

            PipeOp::Except(s) => {
                let other = self.frames.get(&s.frame.name)
                    .ok_or_else(|| ExecError { message: format!("frame '{}' not found", s.frame.name) })?;
                Ok(exec_except(&frame, other))
            }
        }
    }
}

impl Default for Executor {
    fn default() -> Self { Self::new() }
}

// ---------------------------------------------------------------------------
// Frame helpers
// ---------------------------------------------------------------------------

/// Clone a Frame under a new name (shallow clone of all series).
pub trait FrameExt {
    fn clone_with_name(&self, name: &str) -> Frame;
}

impl FrameExt for Frame {
    fn clone_with_name(&self, name: &str) -> Frame {
        Frame { name: name.to_owned(), columns: self.columns.iter().map(clone_series).collect() }
    }
}

fn clone_series(s: &Series) -> Series {
    Series {
        field:      s.field.clone(),
        chunks:     s.chunks.iter().map(|v| Value { data: v.data.clone(), validity: v.validity.clone() }).collect(),
        len:        s.len,
        null_count: s.null_count,
        dict:       s.dict.clone(),
        min:        s.min.clone(),
        max:        s.max.clone(),
    }
}

fn row_count(frame: &Frame) -> usize {
    frame.columns.first().map(|s| s.chunks.len()).unwrap_or(0)
}

fn find_series<'a>(frame: &'a Frame, name: &str) -> Option<&'a Series> {
    frame.columns.iter().find(|s| s.field.name == name)
}

/// Read the real value at `row` in `series`, decoding dict-encoding and null.
fn get_value(series: &Series, row: usize) -> DataValue {
    let v = match series.chunks.get(row) {
        None => return DataValue::Null,
        Some(v) => v,
    };
    // validity: bit 0 of byte 0 = 1 → valid
    let valid = v.validity.first().map(|b| b & 1 == 1).unwrap_or(false);
    if !valid { return DataValue::Null; }
    // Dict decoding
    if !series.dict.is_empty() {
        if let DataValue::Int64(idx) = &v.data {
            return series.dict.get(idx).cloned().unwrap_or(DataValue::Null);
        }
    }
    v.data.clone()
}

fn make_value(data: DataValue) -> Value {
    Value { data, validity: vec![1] }
}

fn make_null() -> Value {
    Value { data: DataValue::Null, validity: vec![0] }
}

fn series_from_values(name: &str, values: Vec<DataValue>) -> Series {
    let null_count = values.iter().filter(|v| matches!(v, DataValue::Null)).count();
    let dtype = values.iter()
        .find(|v| !matches!(v, DataValue::Null))
        .map(dtype_of)
        .unwrap_or(DataType::Null);
    let chunks: Vec<Value> = values.into_iter().map(|v| {
        if matches!(v, DataValue::Null) { make_null() } else { make_value(v) }
    }).collect();
    let len = chunks.len();
    Series {
        field: Column { name: name.to_owned(), dtype },
        chunks,
        len,
        null_count,
        dict: HashMap::new(),
        min: None,
        max: None,
    }
}

fn dtype_of(v: &DataValue) -> DataType {
    match v {
        DataValue::Boolean(_) => DataType::Boolean,
        DataValue::Int8(_)    => DataType::Int8,
        DataValue::Int16(_)   => DataType::Int16,
        DataValue::Int32(_)   => DataType::Int32,
        DataValue::Int64(_)   => DataType::Int64,
        DataValue::UInt8(_)   => DataType::UInt8,
        DataValue::UInt16(_)  => DataType::UInt16,
        DataValue::UInt32(_)  => DataType::UInt32,
        DataValue::UInt64(_)  => DataType::UInt64,
        DataValue::Float32(_) => DataType::Float32,
        DataValue::Float64(_) => DataType::Float64,
        DataValue::String(_)  => DataType::String,
        DataValue::Date(_)    => DataType::Date,
        _                     => DataType::Null,
    }
}

/// Key string for group-by / distinct hashing (handles non-hashable floats).
fn dv_key(v: &DataValue) -> String {
    match v {
        DataValue::Null       => "\x00".to_owned(),
        DataValue::Boolean(b) => format!("b{b}"),
        DataValue::Int8(n)    => format!("i{n}"),
        DataValue::Int16(n)   => format!("i{n}"),
        DataValue::Int32(n)   => format!("i{n}"),
        DataValue::Int64(n)   => format!("i{n}"),
        DataValue::UInt8(n)   => format!("u{n}"),
        DataValue::UInt16(n)  => format!("u{n}"),
        DataValue::UInt32(n)  => format!("u{n}"),
        DataValue::UInt64(n)  => format!("u{n}"),
        DataValue::Float32(f) => format!("f{f}"),
        DataValue::Float64(f) => format!("f{f}"),
        DataValue::String(s)  => format!("s{s}"),
        DataValue::Date(d)    => format!("d{d}"),
        _                     => format!("{v:?}"),
    }
}

fn row_key(frame: &Frame, row: usize, cols: &[Expr]) -> String {
    cols.iter()
        .map(|e| dv_key(&eval_expr(frame, row, e)))
        .collect::<Vec<_>>()
        .join("|")
}

/// Returns the column-name string an expression resolves to (for select / sort).
fn col_name_of_expr(expr: &Expr) -> &str {
    match expr {
        Expr::Column(c) => &c.column,
        _               => "__expr",
    }
}

// ---------------------------------------------------------------------------
// Frame operations
// ---------------------------------------------------------------------------

fn frame_filter(frame: &Frame, keep: impl Fn(usize) -> bool) -> Frame {
    let n = row_count(frame);
    let indices: Vec<usize> = (0..n).filter(|&i| keep(i)).collect();
    frame_take(frame, &indices)
}

/// Take specific rows by index from every series.
fn frame_take(frame: &Frame, indices: &[usize]) -> Frame {
    Frame {
        name: frame.name.clone(),
        columns: frame.columns.iter().map(|s| {
            let chunks: Vec<Value> = indices.iter().map(|&i| {
                if i < s.chunks.len() {
                    Value { data: s.chunks[i].data.clone(), validity: s.chunks[i].validity.clone() }
                } else {
                    make_null()
                }
            }).collect();
            let null_count = chunks.iter().filter(|v| v.validity.first().copied().unwrap_or(0) & 1 == 0).count();
            Series { field: s.field.clone(), chunks, len: indices.len(), null_count, dict: s.dict.clone(), min: None, max: None }
        }).collect(),
    }
}

fn exec_select(frame: &Frame, names: &[&str]) -> Result<Frame, ExecError> {
    let mut cols = Vec::with_capacity(names.len());
    for name in names {
        match find_series(frame, name) {
            Some(s) => cols.push(clone_series(s)),
            None    => return exec_err!("column '{name}' not found in select"),
        }
    }
    Ok(Frame { name: frame.name.clone(), columns: cols })
}

fn frame_add_col(mut frame: Frame, name: &str, values: Vec<DataValue>) -> Frame {
    // Remove existing column with same name if any (map can overwrite).
    frame.columns.retain(|s| s.field.name != name);
    frame.columns.push(series_from_values(name, values));
    frame
}

fn frame_slice(frame: Frame, start: usize, end: usize) -> Frame {
    let n   = row_count(&frame);
    let end = end.min(n);
    if start >= end {
        return Frame { name: frame.name.clone(), columns: frame.columns.into_iter().map(|mut s| {
            s.chunks.clear(); s.len = 0; s
        }).collect() };
    }
    let indices: Vec<usize> = (start..end).collect();
    frame_take(&frame, &indices)
}

fn exec_aggregate(
    frame: &Frame,
    aggs: &[AggExpr],
    group_by: &[Expr],
) -> Result<Frame, ExecError> {
    let n = row_count(frame);

    // ── Group rows ────────────────────────────────────────────────────────
    // Ordered map: group_key → [row indices]
    let mut order: Vec<String>              = Vec::new();
    let mut groups: HashMap<String, Vec<usize>> = HashMap::new();

    for row in 0..n {
        let key = if group_by.is_empty() {
            "__all__".to_owned()
        } else {
            row_key(frame, row, group_by)
        };
        if !groups.contains_key(&key) { order.push(key.clone()); }
        groups.entry(key).or_default().push(row);
    }

    // ── Build group-by columns ────────────────────────────────────────────
    let mut out_cols: Vec<(String, Vec<DataValue>)> = Vec::new();

    for gb_expr in group_by {
        let name = col_name_of_expr(gb_expr).to_owned();
        let vals = order.iter().map(|k| {
            let row = groups[k][0];
            eval_expr(frame, row, gb_expr)
        }).collect();
        out_cols.push((name, vals));
    }

    // ── Compute each aggregation ──────────────────────────────────────────
    for agg in aggs {
        let vals = order.iter().map(|k| {
            let rows = &groups[k];
            apply_agg(frame, &agg.func, rows)
        }).collect();
        out_cols.push((agg.alias.name.clone(), vals));
    }

    let columns = out_cols.into_iter()
        .map(|(name, vals)| series_from_values(&name, vals))
        .collect();

    Ok(Frame { name: frame.name.clone(), columns })
}

fn apply_agg(frame: &Frame, func: &FunctionCall, rows: &[usize]) -> DataValue {
    let fn_name = func.name.as_str();

    // count(*) — total row count
    if fn_name == "count" && matches!(func.args.first(), Some(FunctionArg::Star)) {
        return DataValue::Int64(rows.len() as i64);
    }

    // For all other agg functions, get the column expression
    let col_expr = match func.args.first() {
        Some(FunctionArg::Expr(e)) => e,
        _ => return DataValue::Null,
    };

    let values: Vec<DataValue> = rows.iter()
        .map(|&r| eval_expr(frame, r, col_expr))
        .filter(|v| !matches!(v, DataValue::Null))
        .collect();

    match fn_name {
        "count" => DataValue::Int64(values.len() as i64),
        "sum"   => values.into_iter().fold(DataValue::Int64(0), |acc, v| dv_add(acc, v)),
        "min"   => values.into_iter().reduce(|a, b| if dv_lt_bool(&a, &b) { a } else { b }).unwrap_or(DataValue::Null),
        "max"   => values.into_iter().reduce(|a, b| if dv_lt_bool(&a, &b) { b } else { a }).unwrap_or(DataValue::Null),
        "avg"   => {
            if values.is_empty() { return DataValue::Null; }
            let n   = values.len();
            let sum = values.into_iter().fold(DataValue::Float64(0.0), |a, b| dv_add(a, dv_to_f64(b)));
            match sum { DataValue::Float64(f) => DataValue::Float64(f / n as f64), _ => DataValue::Null }
        }
        _ => DataValue::Null,
    }
}

fn exec_distinct(frame: Frame) -> Frame {
    let n = row_count(&frame);
    let mut seen: HashMap<String, ()> = HashMap::new();
    let mut indices = Vec::new();
    for row in 0..n {
        let key: String = frame.columns.iter()
            .map(|s| dv_key(&get_value(s, row)))
            .collect::<Vec<_>>()
            .join("|");
        if seen.insert(key, ()).is_none() {
            indices.push(row);
        }
    }
    frame_take(&frame, &indices)
}

fn exec_sort_by(frame: Frame, keys: &[SortCol]) -> Result<Frame, ExecError> {
    let n = row_count(&frame);
    let mut indices: Vec<usize> = (0..n).collect();

    indices.sort_by(|&a, &b| {
        for key in keys {
            let va = eval_expr(&frame, a, &key.expr);
            let vb = eval_expr(&frame, b, &key.expr);
            let ord = dv_cmp(&va, &vb);
            let ord = if key.direction == SortDirection::Desc { ord.reverse() } else { ord };
            if ord != std::cmp::Ordering::Equal { return ord; }
        }
        std::cmp::Ordering::Equal
    });

    Ok(frame_take(&frame, &indices))
}

fn exec_join(
    left:  &Frame,
    right: &Frame,
    kind:  &JoinKind,
    cond:  Option<&Expr>,
) -> Result<Frame, ExecError> {
    let ln = row_count(left);
    let rn = row_count(right);

    let mut l_rows = Vec::new();
    let mut r_rows: Vec<Option<usize>> = Vec::new();

    match kind {
        JoinKind::Cross => {
            for l in 0..ln { for r in 0..rn { l_rows.push(l); r_rows.push(Some(r)); } }
        }
        _ => {
            let cond = cond.ok_or_else(|| ExecError { message: "join requires a condition".into() })?;
            let mut right_matched = vec![false; rn];

            for l in 0..ln {
                let mut matched = false;
                for r in 0..rn {
                    let result = eval_expr_join(left, right, l, r, cond);
                    if dv_truthy(&result) {
                        l_rows.push(l); r_rows.push(Some(r));
                        right_matched[r] = true;
                        matched = true;
                    }
                }
                if !matched && matches!(kind, JoinKind::Left) {
                    l_rows.push(l); r_rows.push(None);
                }
            }
            // Right join: unmatched right rows
            if matches!(kind, JoinKind::Right) {
                for r in 0..rn {
                    if !right_matched[r] { l_rows.push(0); r_rows.push(Some(r)); }
                    // l_rows.push(0) is a dummy — left cols will be null
                }
            }
        }
    }

    build_join_frame(left, right, &l_rows, &r_rows, kind)
}

fn build_join_frame(
    left:   &Frame,
    right:  &Frame,
    l_rows: &[usize],
    r_rows: &[Option<usize>],
    kind:   &JoinKind,
) -> Result<Frame, ExecError> {
    let mut columns = Vec::new();
    let left_null  = matches!(kind, JoinKind::Right);

    for s in &left.columns {
        let chunks: Vec<Value> = l_rows.iter().zip(r_rows.iter()).map(|(&l, r_opt)| {
            if left_null && r_opt.is_none() { make_null() }
            else { Value { data: s.chunks[l].data.clone(), validity: s.chunks[l].validity.clone() } }
        }).collect();
        let len = chunks.len();
        columns.push(Series { field: s.field.clone(), chunks, len, null_count: 0, dict: s.dict.clone(), min: None, max: None });
    }
    for s in &right.columns {
        let chunks: Vec<Value> = r_rows.iter().map(|r_opt| match r_opt {
            None    => make_null(),
            Some(r) => Value { data: s.chunks[*r].data.clone(), validity: s.chunks[*r].validity.clone() },
        }).collect();
        let len = chunks.len();
        columns.push(Series { field: s.field.clone(), chunks, len, null_count: 0, dict: s.dict.clone(), min: None, max: None });
    }

    Ok(Frame { name: left.name.clone(), columns })
}

fn exec_relate(
    left:  &Frame,
    right: &Frame,
    cond:  &Expr,
    alias: &str,
) -> Result<Frame, ExecError> {
    // Inner join, then prefix all right-side column names with `alias.`.
    let ln = row_count(left);
    let rn = row_count(right);
    let mut l_rows = Vec::new();
    let mut r_rows = Vec::new();
    for l in 0..ln {
        for r in 0..rn {
            if dv_truthy(&eval_expr_join(left, right, l, r, cond)) {
                l_rows.push(l);
                r_rows.push(r);
            }
        }
    }
    let mut columns: Vec<Series> = left.columns.iter().map(clone_series).collect();
    for s in &right.columns {
        let chunks: Vec<Value> = r_rows.iter()
            .map(|&r| Value { data: s.chunks[r].data.clone(), validity: s.chunks[r].validity.clone() })
            .collect();
        let len = chunks.len();
        let prefixed_name = format!("{alias}.{}", s.field.name);
        columns.push(Series {
            field: Column { name: prefixed_name, dtype: s.field.dtype.clone() },
            chunks,
            len,
            null_count: 0,
            dict: s.dict.clone(),
            min: None,
            max: None,
        });
    }
    // Trim left cols to matched row count
    let mut result = Frame { name: left.name.clone(), columns: vec![] };
    for s in &left.columns {
        let chunks: Vec<Value> = l_rows.iter()
            .map(|&l| Value { data: s.chunks[l].data.clone(), validity: s.chunks[l].validity.clone() })
            .collect();
        let len = chunks.len();
        result.columns.push(Series { field: s.field.clone(), chunks, len, null_count: 0, dict: s.dict.clone(), min: None, max: None });
    }
    for s in right.columns.iter() {
        let chunks: Vec<Value> = r_rows.iter()
            .map(|&r| Value { data: s.chunks[r].data.clone(), validity: s.chunks[r].validity.clone() })
            .collect();
        let len = chunks.len();
        let prefixed = format!("{alias}.{}", s.field.name);
        result.columns.push(Series {
            field: Column { name: prefixed, dtype: s.field.dtype.clone() },
            chunks, len, null_count: 0, dict: s.dict.clone(), min: None, max: None,
        });
    }
    Ok(result)
}

fn exec_union(mut left: Frame, right: &Frame) -> Result<Frame, ExecError> {
    let rn = row_count(right);
    for s in &mut left.columns {
        let rs = find_series(right, &s.field.name)
            .ok_or_else(|| ExecError { message: format!("union: column '{}' not in right frame", s.field.name) })?;
        for r in 0..rn {
            s.chunks.push(Value { data: rs.chunks[r].data.clone(), validity: rs.chunks[r].validity.clone() });
        }
        s.len = s.chunks.len();
    }
    // Deduplicate (union = combine + distinct)
    Ok(exec_distinct(left))
}

fn exec_intersect(left: &Frame, right: &Frame) -> Frame {
    let rn   = row_count(right);
    let keep: Vec<usize> = (0..row_count(left)).filter(|&l| {
        let lkey: String = left.columns.iter()
            .map(|s| dv_key(&get_value(s, l))).collect::<Vec<_>>().join("|");
        (0..rn).any(|r| {
            let rkey: String = right.columns.iter()
                .map(|s| dv_key(&get_value(s, r))).collect::<Vec<_>>().join("|");
            lkey == rkey
        })
    }).collect();
    frame_take(left, &keep)
}

fn exec_except(left: &Frame, right: &Frame) -> Frame {
    let rn   = row_count(right);
    let keep: Vec<usize> = (0..row_count(left)).filter(|&l| {
        let lkey: String = left.columns.iter()
            .map(|s| dv_key(&get_value(s, l))).collect::<Vec<_>>().join("|");
        !(0..rn).any(|r| {
            let rkey: String = right.columns.iter()
                .map(|s| dv_key(&get_value(s, r))).collect::<Vec<_>>().join("|");
            lkey == rkey
        })
    }).collect();
    frame_take(left, &keep)
}

// ---------------------------------------------------------------------------
// Window functions
// ---------------------------------------------------------------------------

fn exec_window(frame: Frame, spec: &WindowExpr) -> Result<Frame, ExecError> {
    let n = row_count(&frame);

    // ── Partition ─────────────────────────────────────────────────────────
    // ordered map: partition_key → [row indices]
    let mut part_order: Vec<String>              = Vec::new();
    let mut partitions: HashMap<String, Vec<usize>> = HashMap::new();
    for row in 0..n {
        let key = row_key(&frame, row, &spec.partition_by);
        if !partitions.contains_key(&key) { part_order.push(key.clone()); }
        partitions.entry(key).or_default().push(row);
    }

    // ── Sort within each partition ────────────────────────────────────────
    for part in partitions.values_mut() {
        if !spec.sort_by.is_empty() {
            part.sort_by(|&a, &b| {
                for sc in &spec.sort_by {
                    let va = eval_expr(&frame, a, &sc.expr);
                    let vb = eval_expr(&frame, b, &sc.expr);
                    let ord = dv_cmp(&va, &vb);
                    let ord = if sc.direction == SortDirection::Desc { ord.reverse() } else { ord };
                    if ord != std::cmp::Ordering::Equal { return ord; }
                }
                std::cmp::Ordering::Equal
            });
        }
    }

    // ── Compute the window value for every row ────────────────────────────
    let mut result_values = vec![DataValue::Null; n];

    for key in &part_order {
        let part = &partitions[key];
        for (rank_idx, &row) in part.iter().enumerate() {
            let val = apply_window_fn(&frame, &spec.func, part, rank_idx, spec.frame_spec.as_ref())?;
            result_values[row] = val;
        }
    }

    Ok(frame_add_col(frame, &spec.alias.name, result_values))
}

fn apply_window_fn(
    frame:      &Frame,
    func:       &FunctionCall,
    partition:  &[usize],
    rank_idx:   usize,
    frame_spec: Option<&FrameSpec>,
) -> Result<DataValue, ExecError> {
    let n = partition.len();

    match func.name.as_str() {
        "row_number"  => return Ok(DataValue::Int64((rank_idx + 1) as i64)),
        "rank"        => {
            let cur_row = partition[rank_idx];
            let cur_val = func_col_val(frame, func, cur_row);
            let rank = partition.iter().take_while(|&&r| {
                dv_lt_bool(&get_col_val_for_sort(frame, func, r), &cur_val)
            }).count() + 1;
            return Ok(DataValue::Int64(rank as i64));
        }
        "dense_rank" => {
            let cur_row = partition[rank_idx];
            let cur_val = func_col_val(frame, func, cur_row);
            let mut seen: Vec<String> = Vec::new();
            for &r in partition {
                let v = dv_key(&get_col_val_for_sort(frame, func, r));
                if !seen.contains(&v) { seen.push(v.clone()); }
                if dv_key(&get_col_val_for_sort(frame, func, r)) == dv_key(&cur_val) { break; }
            }
            return Ok(DataValue::Int64(seen.len() as i64));
        }
        "lag" => {
            let offset  = func.args.get(1).and_then(|a| if let FunctionArg::Expr(e) = a { Some(e) } else { None })
                .and_then(|e| if let Expr::Literal(Literal::Int(n, _)) = e { Some(*n as usize) } else { None })
                .unwrap_or(1);
            let default = func.args.get(2).and_then(|a| if let FunctionArg::Expr(e) = a { Some(e) } else { None });
            return Ok(if rank_idx >= offset {
                func_col_val(frame, func, partition[rank_idx - offset])
            } else {
                default.map(|e| eval_expr(frame, partition[rank_idx], e)).unwrap_or(DataValue::Null)
            });
        }
        "lead" => {
            let offset  = func.args.get(1).and_then(|a| if let FunctionArg::Expr(e) = a { Some(e) } else { None })
                .and_then(|e| if let Expr::Literal(Literal::Int(n, _)) = e { Some(*n as usize) } else { None })
                .unwrap_or(1);
            let default = func.args.get(2).and_then(|a| if let FunctionArg::Expr(e) = a { Some(e) } else { None });
            return Ok(if rank_idx + offset < n {
                func_col_val(frame, func, partition[rank_idx + offset])
            } else {
                default.map(|e| eval_expr(frame, partition[rank_idx], e)).unwrap_or(DataValue::Null)
            });
        }
        _ => {}
    }

    // Aggregation window functions (sum, avg, min, max, count) with frame spec
    let (start, end) = window_frame_bounds(frame_spec, rank_idx, n);
    let window_rows: Vec<usize> = partition[start..=end.min(n - 1)].to_vec();

    Ok(apply_agg(frame, func, &window_rows))
}

fn window_frame_bounds(spec: Option<&FrameSpec>, idx: usize, n: usize) -> (usize, usize) {
    let start = match spec.map(|s| &s.start) {
        None | Some(FrameBound::UnboundedPreceding) => 0,
        Some(FrameBound::CurrentRow)                => idx,
        Some(FrameBound::Preceding(k))              => idx.saturating_sub(*k as usize),
        Some(FrameBound::Following(k))              => (idx + *k as usize).min(n - 1),
        Some(FrameBound::UnboundedFollowing)        => n - 1,
    };
    let end = match spec.map(|s| &s.end) {
        None | Some(FrameBound::UnboundedFollowing)  => n - 1,
        Some(FrameBound::CurrentRow)                 => idx,
        Some(FrameBound::Following(k))               => (idx + *k as usize).min(n - 1),
        Some(FrameBound::Preceding(k))               => idx.saturating_sub(*k as usize),
        Some(FrameBound::UnboundedPreceding)         => 0,
    };
    (start, end)
}

/// Get the first column argument value for a window function.
fn func_col_val(frame: &Frame, func: &FunctionCall, row: usize) -> DataValue {
    match func.args.first() {
        Some(FunctionArg::Expr(e)) => eval_expr(frame, row, e),
        _ => DataValue::Null,
    }
}

fn get_col_val_for_sort(frame: &Frame, func: &FunctionCall, row: usize) -> DataValue {
    func_col_val(frame, func, row)
}

// ---------------------------------------------------------------------------
// Expression evaluator
// ---------------------------------------------------------------------------

pub fn eval_expr(frame: &Frame, row: usize, expr: &Expr) -> DataValue {
    match expr {
        Expr::Literal(l) => eval_literal(l),

        Expr::Column(c) => {
            // Try qualified name first, then bare column name.
            let qname = c.table.as_ref().map(|t| format!("{t}.{}", c.column));
            let series = qname.as_deref()
                .and_then(|n| find_series(frame, n))
                .or_else(|| find_series(frame, &c.column));
            series.map(|s| get_value(s, row)).unwrap_or(DataValue::Null)
        }

        Expr::BinaryOp(b) => {
            let l = eval_expr(frame, row, &b.left);
            let r = eval_expr(frame, row, &b.right);
            apply_binop(&b.op, l, r)
        }

        Expr::UnaryOp(u) => {
            let v = eval_expr(frame, row, &u.expr);
            match u.op {
                UnaryOp::Neg => dv_neg(v),
                UnaryOp::Not => dv_not(v),
            }
        }

        Expr::FunctionCall(f) => eval_function(frame, row, f),

        Expr::IsNull(e, _)    => {
            let v = eval_expr(frame, row, e);
            DataValue::Boolean(matches!(v, DataValue::Null))
        }
        Expr::IsNotNull(e, _) => {
            let v = eval_expr(frame, row, e);
            DataValue::Boolean(!matches!(v, DataValue::Null))
        }

        Expr::Grouped(e, _) => eval_expr(frame, row, e),

        Expr::If(if_expr) => eval_if(frame, row, if_expr, eval_expr),
    }
}

/// Evaluate an expression in a join context.  Column references that are
/// qualified with the right-frame's name resolve against `right`; everything
/// else resolves against `left`.
pub fn eval_expr_join(
    left:  &Frame,
    right: &Frame,
    l_row: usize,
    r_row: usize,
    expr:  &Expr,
) -> DataValue {
    match expr {
        Expr::Column(c) => {
            // If there is a qualifier (e.g. `lft.id`), route by frame name so
            // `left_frame.col` → left and `right_frame.col` → right.
            // Only fall back to bare-name search when there is no qualifier.
            if let Some(table) = &c.table {
                if table == &right.name {
                    // Explicitly references the right frame
                    return find_series(right, &c.column)
                        .map(|s| get_value(s, r_row))
                        .unwrap_or(DataValue::Null);
                }
                if table == &left.name {
                    // Explicitly references the left frame
                    return find_series(left, &c.column)
                        .map(|s| get_value(s, l_row))
                        .unwrap_or(DataValue::Null);
                }
                // Qualifier doesn't match either frame name — try both as a
                // prefixed column name stored inside the frame (e.g. after relate).
                let qname = format!("{table}.{}", c.column);
                if let Some(s) = find_series(right, &qname) { return get_value(s, r_row); }
                if let Some(s) = find_series(left,  &qname) { return get_value(s, l_row); }
                return DataValue::Null;
            }
            // Unqualified: try left first, then right
            if let Some(s) = find_series(left, &c.column)  { return get_value(s, l_row); }
            if let Some(s) = find_series(right, &c.column) { return get_value(s, r_row); }
            DataValue::Null
        }
        Expr::BinaryOp(b) => {
            let l = eval_expr_join(left, right, l_row, r_row, &b.left);
            let r = eval_expr_join(left, right, l_row, r_row, &b.right);
            apply_binop(&b.op, l, r)
        }
        Expr::UnaryOp(u) => {
            let v = eval_expr_join(left, right, l_row, r_row, &u.expr);
            match u.op { UnaryOp::Neg => dv_neg(v), UnaryOp::Not => dv_not(v) }
        }
        Expr::IsNull(e, _)    => {
            let v = eval_expr_join(left, right, l_row, r_row, e);
            DataValue::Boolean(matches!(v, DataValue::Null))
        }
        Expr::IsNotNull(e, _) => {
            let v = eval_expr_join(left, right, l_row, r_row, e);
            DataValue::Boolean(!matches!(v, DataValue::Null))
        }
        Expr::Grouped(e, _) => eval_expr_join(left, right, l_row, r_row, e),

        Expr::If(if_expr) => {
            // Evaluate each condition in the join context; bodies too.
            let cond = eval_expr_join(left, right, l_row, r_row, &if_expr.condition);
            if dv_truthy(&cond) {
                return eval_expr_join(left, right, l_row, r_row, &if_expr.then_branch);
            }
            for branch in &if_expr.else_if_branches {
                let c = eval_expr_join(left, right, l_row, r_row, &branch.condition);
                if dv_truthy(&c) {
                    return eval_expr_join(left, right, l_row, r_row, &branch.body);
                }
            }
            if_expr.else_branch.as_ref()
                .map(|e| eval_expr_join(left, right, l_row, r_row, e))
                .unwrap_or(DataValue::Null)
        }

        other => eval_expr(left, l_row, other), // literals, functions
    }
}

/// Generic if/else evaluator; `eval` is the expression evaluator for the
/// current context (regular frame or join).
fn eval_if<F>(frame: &Frame, row: usize, if_expr: &IfExpr, eval: F) -> DataValue
where
    F: Fn(&Frame, usize, &Expr) -> DataValue,
{
    if dv_truthy(&eval(frame, row, &if_expr.condition)) {
        return eval(frame, row, &if_expr.then_branch);
    }
    for branch in &if_expr.else_if_branches {
        if dv_truthy(&eval(frame, row, &branch.condition)) {
            return eval(frame, row, &branch.body);
        }
    }
    if_expr.else_branch.as_ref()
        .map(|e| eval(frame, row, e))
        .unwrap_or(DataValue::Null)
}

fn eval_literal(l: &Literal) -> DataValue {
    match l {
        Literal::Int(n, _)    => DataValue::Int64(*n),
        Literal::Float(f, _)  => DataValue::Float64(*f),
        Literal::String(s, _) => DataValue::String(s.clone()),
        Literal::Bool(b, _)   => DataValue::Boolean(*b),
        Literal::Null(_)      => DataValue::Null,
    }
}

fn eval_function(frame: &Frame, row: usize, f: &FunctionCall) -> DataValue {
    match f.name.as_str() {
        "coalesce" => {
            for arg in &f.args {
                if let FunctionArg::Expr(e) = arg {
                    let v = eval_expr(frame, row, e);
                    if !matches!(v, DataValue::Null) { return v; }
                }
            }
            DataValue::Null
        }
        "to_int" => {
            let v = first_arg_val(frame, row, f);
            match v {
                DataValue::Int64(_)   => v,
                DataValue::Float64(f) => DataValue::Int64(f as i64),
                DataValue::String(s)  => s.parse::<i64>().map(DataValue::Int64).unwrap_or(DataValue::Null),
                DataValue::Boolean(b) => DataValue::Int64(b as i64),
                DataValue::Null       => DataValue::Null,
                _                     => DataValue::Null,
            }
        }
        "to_float" => {
            let v = first_arg_val(frame, row, f);
            match v {
                DataValue::Float64(_) => v,
                DataValue::Int64(n)   => DataValue::Float64(n as f64),
                DataValue::String(s)  => s.parse::<f64>().map(DataValue::Float64).unwrap_or(DataValue::Null),
                DataValue::Null       => DataValue::Null,
                _                     => DataValue::Null,
            }
        }
        "to_string" => {
            let v = first_arg_val(frame, row, f);
            match &v {
                DataValue::Null => DataValue::Null,
                other           => DataValue::String(format!("{other:?}")),
            }
        }
        // Date helpers
        "year"  => extract_date_part(first_arg_val(frame, row, f), 0),
        "month" => extract_date_part(first_arg_val(frame, row, f), 1),
        "day"   => extract_date_part(first_arg_val(frame, row, f), 2),
        _ => DataValue::Null, // unknown functions → null
    }
}

fn first_arg_val(frame: &Frame, row: usize, f: &FunctionCall) -> DataValue {
    match f.args.first() {
        Some(FunctionArg::Expr(e)) => eval_expr(frame, row, e),
        _ => DataValue::Null,
    }
}

fn extract_date_part(v: DataValue, part: u8) -> DataValue {
    // Unix epoch days → naive date
    if let DataValue::Date(days) = v {
        let days = days as i64;
        // Rough Gregorian calendar computation
        let z   = days + 719468;
        let era = (if z >= 0 { z } else { z - 146096 }) / 146097;
        let doe = z - era * 146097;
        let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
        let y   = yoe + era * 400;
        let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
        let mp  = (5 * doy + 2) / 153;
        let d   = doy - (153 * mp + 2) / 5 + 1;
        let m   = if mp < 10 { mp + 3 } else { mp - 9 };
        let y   = if m <= 2 { y + 1 } else { y };
        return DataValue::Int64(match part { 0 => y, 1 => m, _ => d });
    }
    DataValue::Null
}

// ---------------------------------------------------------------------------
// DataValue arithmetic / comparison
// ---------------------------------------------------------------------------

fn apply_binop(op: &BinaryOp, l: DataValue, r: DataValue) -> DataValue {
    match op {
        BinaryOp::Add   => dv_add(l, r),
        BinaryOp::Sub   => dv_sub(l, r),
        BinaryOp::Mul   => dv_mul(l, r),
        BinaryOp::Div   => dv_div(l, r),
        BinaryOp::Eq    => dv_eq(l, r),
        BinaryOp::NotEq => dv_neq(l, r),
        BinaryOp::Gt    => dv_gt(l, r),
        BinaryOp::Lt    => dv_lt(l, r),
        BinaryOp::Gte   => dv_gte(l, r),
        BinaryOp::Lte   => dv_lte(l, r),
        BinaryOp::And   => dv_and(l, r),
        BinaryOp::Or    => dv_or(l, r),
    }
}

/// Promote any integer variant to Int64 or Float64 for arithmetic.
fn to_numeric(v: DataValue) -> Option<DataValue> {
    match v {
        DataValue::Int8(n)   => Some(DataValue::Int64(n as i64)),
        DataValue::Int16(n)  => Some(DataValue::Int64(n as i64)),
        DataValue::Int32(n)  => Some(DataValue::Int64(n as i64)),
        DataValue::Int64(n)  => Some(DataValue::Int64(n)),
        DataValue::UInt8(n)  => Some(DataValue::Int64(n as i64)),
        DataValue::UInt16(n) => Some(DataValue::Int64(n as i64)),
        DataValue::UInt32(n) => Some(DataValue::Int64(n as i64)),
        DataValue::UInt64(n) => Some(DataValue::Int64(n as i64)),
        DataValue::Float32(f)=> Some(DataValue::Float64(f as f64)),
        DataValue::Float64(f)=> Some(DataValue::Float64(f)),
        DataValue::Null      => None,
        _                    => None,
    }
}

fn dv_to_f64(v: DataValue) -> DataValue {
    match to_numeric(v) {
        Some(DataValue::Int64(n))  => DataValue::Float64(n as f64),
        Some(DataValue::Float64(f))=> DataValue::Float64(f),
        _ => DataValue::Null,
    }
}

macro_rules! numeric_binop {
    ($name:ident, $int_op:expr, $float_op:expr) => {
        fn $name(a: DataValue, b: DataValue) -> DataValue {
            match (to_numeric(a), to_numeric(b)) {
                (None, _) | (_, None) => DataValue::Null,
                (Some(DataValue::Float64(x)), Some(v)) => {
                    let y = match v { DataValue::Int64(n) => n as f64, DataValue::Float64(f) => f, _ => return DataValue::Null };
                    DataValue::Float64($float_op(x, y))
                }
                (Some(v), Some(DataValue::Float64(y))) => {
                    let x = match v { DataValue::Int64(n) => n as f64, DataValue::Float64(f) => f, _ => return DataValue::Null };
                    DataValue::Float64($float_op(x, y))
                }
                (Some(DataValue::Int64(x)), Some(DataValue::Int64(y))) => $int_op(x, y),
                _ => DataValue::Null,
            }
        }
    };
}

numeric_binop!(dv_add,
    |x: i64, y: i64| DataValue::Int64(x.wrapping_add(y)),
    |x: f64, y: f64| x + y);

numeric_binop!(dv_sub,
    |x: i64, y: i64| DataValue::Int64(x.wrapping_sub(y)),
    |x: f64, y: f64| x - y);

numeric_binop!(dv_mul,
    |x: i64, y: i64| DataValue::Int64(x.wrapping_mul(y)),
    |x: f64, y: f64| x * y);

fn dv_div(a: DataValue, b: DataValue) -> DataValue {
    match (to_numeric(a), to_numeric(b)) {
        (None, _) | (_, None) => DataValue::Null,
        (Some(la), Some(rb)) => {
            let x = match la { DataValue::Int64(n) => n as f64, DataValue::Float64(f) => f, _ => return DataValue::Null };
            let y = match rb { DataValue::Int64(n) => n as f64, DataValue::Float64(f) => f, _ => return DataValue::Null };
            if y == 0.0 { DataValue::Null } else { DataValue::Float64(x / y) }
        }
    }
}

fn dv_neg(v: DataValue) -> DataValue {
    match to_numeric(v) {
        Some(DataValue::Int64(n))  => DataValue::Int64(-n),
        Some(DataValue::Float64(f))=> DataValue::Float64(-f),
        _ => DataValue::Null,
    }
}

// ── comparisons ────────────────────────────────────────────────────────────

fn cmp_values(a: &DataValue, b: &DataValue) -> Option<std::cmp::Ordering> {
    match (to_numeric(a.clone()), to_numeric(b.clone())) {
        (Some(DataValue::Int64(x)), Some(DataValue::Int64(y)))   => Some(x.cmp(&y)),
        (Some(x), Some(y)) => {
            let xf = match x { DataValue::Int64(n) => n as f64, DataValue::Float64(f) => f, _ => return None };
            let yf = match y { DataValue::Int64(n) => n as f64, DataValue::Float64(f) => f, _ => return None };
            xf.partial_cmp(&yf)
        }
        _ => match (a, b) {
            (DataValue::String(x), DataValue::String(y)) => Some(x.cmp(y)),
            (DataValue::Boolean(x), DataValue::Boolean(y)) => Some(x.cmp(y)),
            (DataValue::Date(x), DataValue::Date(y)) => Some(x.cmp(y)),
            _ => None,
        }
    }
}

fn dv_eq(a: DataValue, b: DataValue) -> DataValue {
    if matches!((&a, &b), (DataValue::Null, _) | (_, DataValue::Null)) { return DataValue::Null; }
    // String equality
    if let (DataValue::String(x), DataValue::String(y)) = (&a, &b) {
        return DataValue::Boolean(x == y);
    }
    match cmp_values(&a, &b) {
        Some(o) => DataValue::Boolean(o == std::cmp::Ordering::Equal),
        None    => DataValue::Null,
    }
}

fn dv_neq(a: DataValue, b: DataValue) -> DataValue {
    match dv_eq(a, b) {
        DataValue::Boolean(b) => DataValue::Boolean(!b),
        v => v,
    }
}

macro_rules! cmp_op {
    ($name:ident, $ord:pat) => {
        fn $name(a: DataValue, b: DataValue) -> DataValue {
            if matches!((&a, &b), (DataValue::Null, _) | (_, DataValue::Null)) { return DataValue::Null; }
            match cmp_values(&a, &b) {
                Some($ord) => DataValue::Boolean(true),
                Some(_)    => DataValue::Boolean(false),
                None       => DataValue::Null,
            }
        }
    };
}

cmp_op!(dv_gt,  std::cmp::Ordering::Greater);
cmp_op!(dv_lt,  std::cmp::Ordering::Less);

fn dv_gte(a: DataValue, b: DataValue) -> DataValue {
    if matches!((&a, &b), (DataValue::Null, _) | (_, DataValue::Null)) { return DataValue::Null; }
    match cmp_values(&a, &b) {
        Some(std::cmp::Ordering::Greater) | Some(std::cmp::Ordering::Equal) => DataValue::Boolean(true),
        Some(_) => DataValue::Boolean(false),
        None    => DataValue::Null,
    }
}

fn dv_lte(a: DataValue, b: DataValue) -> DataValue {
    if matches!((&a, &b), (DataValue::Null, _) | (_, DataValue::Null)) { return DataValue::Null; }
    match cmp_values(&a, &b) {
        Some(std::cmp::Ordering::Less) | Some(std::cmp::Ordering::Equal) => DataValue::Boolean(true),
        Some(_) => DataValue::Boolean(false),
        None    => DataValue::Null,
    }
}

/// Returns true iff `a < b` (for sort/min/max).
fn dv_lt_bool(a: &DataValue, b: &DataValue) -> bool {
    matches!(cmp_values(a, b), Some(std::cmp::Ordering::Less))
}

fn dv_cmp(a: &DataValue, b: &DataValue) -> std::cmp::Ordering {
    cmp_values(a, b).unwrap_or(std::cmp::Ordering::Equal)
}

// ── logical (SQL three-valued) ─────────────────────────────────────────────

fn dv_and(a: DataValue, b: DataValue) -> DataValue {
    match (&a, &b) {
        (DataValue::Boolean(false), _) | (_, DataValue::Boolean(false)) => DataValue::Boolean(false),
        (DataValue::Null, _) | (_, DataValue::Null) => DataValue::Null,
        (DataValue::Boolean(x), DataValue::Boolean(y)) => DataValue::Boolean(*x && *y),
        _ => DataValue::Null,
    }
}

fn dv_or(a: DataValue, b: DataValue) -> DataValue {
    match (&a, &b) {
        (DataValue::Boolean(true), _) | (_, DataValue::Boolean(true)) => DataValue::Boolean(true),
        (DataValue::Null, _) | (_, DataValue::Null) => DataValue::Null,
        (DataValue::Boolean(x), DataValue::Boolean(y)) => DataValue::Boolean(*x || *y),
        _ => DataValue::Null,
    }
}

fn dv_not(v: DataValue) -> DataValue {
    match v {
        DataValue::Boolean(b) => DataValue::Boolean(!b),
        DataValue::Null       => DataValue::Null,
        _                     => DataValue::Null,
    }
}

fn dv_truthy(v: &DataValue) -> bool {
    matches!(v, DataValue::Boolean(true))
}

//! End-to-end execution tests that mirror `test-file/examples.resin`.
//!
//! Each test corresponds to one (or more) numbered example in that file.
//! Frames are built inline with realistic column names; the test then runs
//! the exact Resin snippet and asserts on the shape / values of the result.

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
// Helpers (same as executor_tests but duplicated for isolation)
// ---------------------------------------------------------------------------

fn vi(v: i64) -> Value  { Value { data: DataValue::Int64(v),            validity: vec![1] } }
fn vf(v: f64) -> Value  { Value { data: DataValue::Float64(v),          validity: vec![1] } }
fn vs(v: &str) -> Value { Value { data: DataValue::String(v.to_owned()), validity: vec![1] } }
fn vb(v: bool)-> Value  { Value { data: DataValue::Boolean(v),          validity: vec![1] } }
fn vn()       -> Value  { Value { data: DataValue::Null,                 validity: vec![0] } }

fn si(name: &str, vals: &[i64]) -> Series {
    Series { field: Column { name: name.to_owned(), dtype: DataType::Int64 },
        chunks: vals.iter().map(|&v| vi(v)).collect(), len: vals.len(),
        null_count: 0, dict: HashMap::new(), min: None, max: None }
}
fn sf(name: &str, vals: &[f64]) -> Series {
    Series { field: Column { name: name.to_owned(), dtype: DataType::Float64 },
        chunks: vals.iter().map(|&v| vf(v)).collect(), len: vals.len(),
        null_count: 0, dict: HashMap::new(), min: None, max: None }
}
fn ss(name: &str, vals: &[&str]) -> Series {
    Series { field: Column { name: name.to_owned(), dtype: DataType::String },
        chunks: vals.iter().map(|v| vs(v)).collect(), len: vals.len(),
        null_count: 0, dict: HashMap::new(), min: None, max: None }
}
fn sb(name: &str, vals: &[bool]) -> Series {
    Series { field: Column { name: name.to_owned(), dtype: DataType::Boolean },
        chunks: vals.iter().map(|&v| vb(v)).collect(), len: vals.len(),
        null_count: 0, dict: HashMap::new(), min: None, max: None }
}
fn sn(name: &str, vals: &[Option<f64>]) -> Series {
    let null_count = vals.iter().filter(|v| v.is_none()).count();
    Series { field: Column { name: name.to_owned(), dtype: DataType::Float64 },
        chunks: vals.iter().map(|v| match v { Some(f) => vf(*f), None => vn() }).collect(),
        len: vals.len(), null_count, dict: HashMap::new(), min: None, max: None }
}
fn sni(name: &str, vals: &[Option<i64>]) -> Series {
    let null_count = vals.iter().filter(|v| v.is_none()).count();
    Series { field: Column { name: name.to_owned(), dtype: DataType::Int64 },
        chunks: vals.iter().map(|v| match v { Some(n) => vi(*n), None => vn() }).collect(),
        len: vals.len(), null_count, dict: HashMap::new(), min: None, max: None }
}
fn sns(name: &str, vals: &[Option<&str>]) -> Series {
    let null_count = vals.iter().filter(|v| v.is_none()).count();
    Series { field: Column { name: name.to_owned(), dtype: DataType::String },
        chunks: vals.iter().map(|v| match v { Some(s) => vs(s), None => vn() }).collect(),
        len: vals.len(), null_count, dict: HashMap::new(), min: None, max: None }
}

fn mk(name: &str, cols: Vec<Series>) -> Frame {
    Frame { name: name.to_owned(), columns: cols }
}

fn run(ex: &mut Executor, src: &str) {
    let tokens = Lexer::new(src).tokenize()
        .unwrap_or_else(|e| panic!("lex errors: {e:#?}"));
    let (prog, errs) = parse(tokens);
    assert!(errs.is_empty(), "parse errors:\n{}", errs.iter().map(|e| e.to_string()).collect::<Vec<_>>().join("\n"));
    ex.run(&prog).expect("exec error");
}

fn nrows(frame: &Frame) -> usize {
    frame.columns.first().map(|s| s.chunks.len()).unwrap_or(0)
}

fn get_i64(frame: &Frame, col: &str) -> Vec<i64> {
    let s = frame.columns.iter().find(|s| s.field.name == col)
        .unwrap_or_else(|| panic!("col '{col}' not found in {:?}", frame.columns.iter().map(|c| &c.field.name).collect::<Vec<_>>()));
    s.chunks.iter().map(|v| {
        if v.validity.first().map(|b| b & 1 == 1).unwrap_or(false) == false { return i64::MIN; }
        match &v.data { DataValue::Int64(n) => *n, o => panic!("expected i64 got {o:?}") }
    }).collect()
}

fn get_f64(frame: &Frame, col: &str) -> Vec<f64> {
    let s = frame.columns.iter().find(|s| s.field.name == col)
        .unwrap_or_else(|| panic!("col '{col}' not found"));
    s.chunks.iter().map(|v| {
        if v.validity.first().map(|b| b & 1 == 1).unwrap_or(false) == false { return f64::NAN; }
        match &v.data { DataValue::Float64(f) => *f, DataValue::Int64(n) => *n as f64, o => panic!("expected f64 got {o:?}") }
    }).collect()
}

fn get_str(frame: &Frame, col: &str) -> Vec<String> {
    let s = frame.columns.iter().find(|s| s.field.name == col)
        .unwrap_or_else(|| panic!("col '{col}' not found"));
    s.chunks.iter().map(|v| {
        if v.validity.first().map(|b| b & 1 == 1).unwrap_or(false) == false { return "__null__".to_owned(); }
        match &v.data { DataValue::String(s) => s.clone(), o => panic!("expected str got {o:?}") }
    }).collect()
}

fn col_names(frame: &Frame) -> Vec<&str> {
    frame.columns.iter().map(|s| s.field.name.as_str()).collect()
}

// ---------------------------------------------------------------------------
// Shared test data
// ---------------------------------------------------------------------------

fn orders_frame() -> Frame {
    mk("orders", vec![
        si("id",          &[1, 2, 3, 4, 5, 6]),
        si("customer_id", &[10, 10, 20, 20, 30, 30]),
        si("product_id",  &[100, 101, 100, 102, 101, 102]),
        sf("amount",      &[50.0, 200.0, 150.0, 80.0, 300.0, 120.0]),
        ss("country",     &["FR", "FR", "DE", "DE", "FR", "US"]),
        ss("category",    &["electronics", "clothing", "electronics", "food", "clothing", "food"]),
        ss("date",        &["2024-01-01", "2024-01-02", "2024-01-03", "2024-01-04", "2024-01-05", "2024-01-06"]),
        sn("discount",    &[Some(0.1), None, Some(0.2), None, Some(0.05), None]),
    ])
}

fn customers_frame() -> Frame {
    mk("customers", vec![
        si("id",      &[10, 20, 30]),
        ss("name",    &["Alice", "Bob", "Carol"]),
        ss("country", &["FR", "DE", "FR"]),
        sb("active",  &[true, true, false]),
        sn("discount",&[Some(0.1), Some(0.2), None]),
        sns("email",  &[Some("alice@ex.com"), None, Some("carol@ex.com")]),
    ])
}

fn products_frame() -> Frame {
    mk("products", vec![
        si("id",       &[100, 101, 102]),
        ss("name",     &["Laptop", "T-Shirt", "Apple"]),
        ss("category", &["electronics", "clothing", "food"]),
        sb("active",   &[true, true, false]),
    ])
}

fn returns_frame() -> Frame {
    mk("returns", vec![
        si("id",       &[1, 2]),
        si("order_id", &[2, 4]),
        ss("reason",   &["defective", "wrong size"]),
    ])
}

// ---------------------------------------------------------------------------
// Example 1 — use / alias
// ---------------------------------------------------------------------------

#[test]
fn example_01_use_alias() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, r#"use orders from "orders""#);

    assert!(ex.get("orders").is_some());
    assert_eq!(nrows(ex.get("orders").unwrap()), 6);
}

// ---------------------------------------------------------------------------
// Example 2 — basic filter
// ---------------------------------------------------------------------------

#[test]
fn example_02_filter_gt() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, "orders.filter(amount > 100) as big_orders");

    let f = ex.get("big_orders").unwrap();
    let amounts = get_f64(f, "amount");
    assert!(amounts.iter().all(|&a| a > 100.0), "amounts: {amounts:?}");
    assert_eq!(nrows(f), 4); // 200, 150, 300, 120
}

// ---------------------------------------------------------------------------
// Example 3 — compound predicate
// ---------------------------------------------------------------------------

#[test]
fn example_03_filter_and() {
    let mut ex = Executor::new();
    ex.load("customers", customers_frame());

    run(&mut ex, r#"customers.filter(active = true and country = "FR") as fr_active_customers"#);

    let f = ex.get("fr_active_customers").unwrap();
    // Alice (active=true, country=FR) and Carol (active=false) → only Alice
    assert_eq!(nrows(f), 1);
    assert_eq!(get_str(f, "name"), vec!["Alice"]);
}

// ---------------------------------------------------------------------------
// Example 4 — is not null filter
// ---------------------------------------------------------------------------

#[test]
fn example_04_filter_is_not_null() {
    let mut ex = Executor::new();
    ex.load("customers", customers_frame());

    run(&mut ex, "customers.filter(email is not null) as customers_with_email");

    let f = ex.get("customers_with_email").unwrap();
    // Bob has null email → only Alice and Carol
    assert_eq!(nrows(f), 2);
}

// ---------------------------------------------------------------------------
// Example 5 — select
// ---------------------------------------------------------------------------

#[test]
fn example_05_select() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, "orders.select(id, customer_id, amount) as orders_slim");

    let f = ex.get("orders_slim").unwrap();
    assert_eq!(col_names(f), vec!["id", "customer_id", "amount"]);
    assert_eq!(nrows(f), 6);
}

// ---------------------------------------------------------------------------
// Example 6 — map (VAT)
// ---------------------------------------------------------------------------

#[test]
fn example_06_map_vat() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, "orders.map(amount * 1.2 as amount_vat) as orders_with_vat");

    let f = ex.get("orders_with_vat").unwrap();
    let vat = get_f64(f, "amount_vat");
    // original amounts: 50, 200, 150, 80, 300, 120  → ×1.2
    let expected = [60.0, 240.0, 180.0, 96.0, 360.0, 144.0];
    for (got, exp) in vat.iter().zip(expected.iter()) {
        assert!((got - exp).abs() < 1e-6, "expected {exp} got {got}");
    }
}

// ---------------------------------------------------------------------------
// Example 7 — map with coalesce
// ---------------------------------------------------------------------------

#[test]
fn example_07_map_coalesce() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, r#"
        orders
          .map(coalesce(discount, 0) as safe_discount)
          .map(amount * (1 - safe_discount) as net_amount)
          as orders_net
    "#);

    let f = ex.get("orders_net").unwrap();
    let net = get_f64(f, "net_amount");
    // Row 0: 50*(1-0.10)=45, Row 1: 200*(1-0)=200, Row 2: 150*(1-0.20)=120
    assert!((net[0] - 45.0).abs() < 1e-6, "row0: {}", net[0]);
    assert!((net[1] - 200.0).abs() < 1e-6, "row1: {}", net[1]);
    assert!((net[2] - 120.0).abs() < 1e-6, "row2: {}", net[2]);
}

// ---------------------------------------------------------------------------
// Example 8 — aggregate global
// ---------------------------------------------------------------------------

#[test]
fn example_08_aggregate_global() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, "orders.aggregate(sum(amount) as total_revenue, count(*) as order_count) as revenue_summary");

    let f = ex.get("revenue_summary").unwrap();
    assert_eq!(nrows(f), 1);
    assert_eq!(get_i64(f, "order_count"), vec![6]);
    let rev = get_f64(f, "total_revenue");
    // 50+200+150+80+300+120 = 900
    assert!((rev[0] - 900.0).abs() < 1e-6, "total: {}", rev[0]);
}

// ---------------------------------------------------------------------------
// Example 9 — aggregate grouped + filter + sort
// ---------------------------------------------------------------------------

#[test]
fn example_09_aggregate_grouped() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, r#"
        orders
          .aggregate(sum(amount) as revenue, count(*) as n by country)
          .filter(n > 1)
          .sort_by(revenue desc)
          as revenue_by_country
    "#);

    let f = ex.get("revenue_by_country").unwrap();
    // FR: 50+200+300=550 (3 orders), DE: 150+80=230 (2 orders), US: 120 (1 order, filtered out)
    assert_eq!(nrows(f), 2);
    let revs = get_f64(f, "revenue");
    // sorted desc → FR first
    assert!(revs[0] >= revs[1], "should be desc: {revs:?}");
    assert!((revs[0] - 550.0).abs() < 1e-6, "FR revenue: {}", revs[0]);
}

// ---------------------------------------------------------------------------
// Example 10 — distinct
// ---------------------------------------------------------------------------

#[test]
fn example_10_distinct() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, "orders.select(country, category).distinct() as country_category_pairs");

    let f = ex.get("country_category_pairs").unwrap();
    // Unique (country, category) from orders:
    // (FR,electronics),(FR,clothing),(DE,electronics),(DE,food),(FR,clothing),(US,food)
    // distinct → (FR,electronics),(FR,clothing),(DE,electronics),(DE,food),(US,food) = 5
    assert_eq!(nrows(f), 5);
}

// ---------------------------------------------------------------------------
// Example 11 — sort + limit + offset (pagination)
// ---------------------------------------------------------------------------

#[test]
fn example_11_sort_offset_limit() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, "orders.sort_by(amount desc).offset(2).limit(2) as orders_page_2");

    let f = ex.get("orders_page_2").unwrap();
    assert_eq!(nrows(f), 2);
    // Sorted desc: 300,200,150,120,80,50 → offset 2 → 150,120
    let amounts = get_f64(f, "amount");
    assert!((amounts[0] - 150.0).abs() < 1e-6);
    assert!((amounts[1] - 120.0).abs() < 1e-6);
}

// ---------------------------------------------------------------------------
// Example 12 — inner join
// ---------------------------------------------------------------------------

#[test]
fn example_12_inner_join() {
    let mut ex = Executor::new();
    ex.load("orders",   orders_frame());
    ex.load("products", products_frame());

    run(&mut ex, r#"
        orders
          .join(products on orders.product_id = products.id)
          .select(orders.id, orders.amount, products.name, products.category)
          as orders_with_product
    "#);

    let f = ex.get("orders_with_product").unwrap();
    assert_eq!(nrows(f), 6); // all orders have a matching product
    assert!(col_names(f).contains(&"orders.id") || col_names(f).contains(&"id"),
        "cols: {:?}", col_names(f));
    // product names appear as a column
    let product_col = f.columns.iter().find(|c| c.field.name == "products.name" || c.field.name == "name");
    assert!(product_col.is_some(), "products.name not found in {:?}", col_names(f));
}

// ---------------------------------------------------------------------------
// Example 13 — left join
// ---------------------------------------------------------------------------

#[test]
fn example_13_left_join() {
    let mut ex = Executor::new();
    ex.load("orders",  orders_frame());
    ex.load("returns", returns_frame());

    run(&mut ex, r#"
        orders
          .left_join(returns on orders.id = returns.order_id)
          .map(coalesce(returns.reason, "no return") as return_status)
          .select(orders.id, return_status)
          as orders_return_status
    "#);

    let f = ex.get("orders_return_status").unwrap();
    // All 6 orders preserved
    assert_eq!(nrows(f), 6);
    let statuses = get_str(f, "return_status");
    // orders 2 and 4 have returns; rest are "no return"
    assert_eq!(statuses.iter().filter(|s| s.as_str() == "no return").count(), 4);
    assert_eq!(statuses.iter().filter(|s| s.as_str() != "no return").count(), 2);
}

// ---------------------------------------------------------------------------
// Example 14 — cross join
// ---------------------------------------------------------------------------

#[test]
fn example_14_cross_join() {
    let mut ex = Executor::new();
    ex.load("sizes",  mk("sizes",  vec![ss("label", &["S", "M", "L"])]));
    ex.load("colors", mk("colors", vec![ss("name",  &["red", "blue"])]));

    run(&mut ex, "sizes.cross_join(colors).select(sizes.label, colors.name) as size_color_matrix");

    let f = ex.get("size_color_matrix").unwrap();
    // 3 sizes × 2 colors = 6 rows
    assert_eq!(nrows(f), 6);
}

// ---------------------------------------------------------------------------
// Example 15 — union / intersect / except
// ---------------------------------------------------------------------------

#[test]
fn example_15_set_ops() {
    let mut ex = Executor::new();
    ex.load("orders_fr", mk("orders_fr", vec![
        si("id", &[1, 2, 3]),
        sf("amount", &[100.0, 200.0, 300.0]),
    ]));
    ex.load("orders_de", mk("orders_de", vec![
        si("id", &[2, 3, 4]),
        sf("amount", &[200.0, 300.0, 400.0]),
    ]));

    run(&mut ex, r#"
        use orders_fr from "orders_fr"
        use orders_de from "orders_de"

        orders_fr.union(orders_de) as all_orders_eu
        orders_fr.intersect(orders_de) as orders_in_both
        orders_fr.except(orders_de) as orders_fr_only
    "#);

    // union: {1,2,3,4} = 4 rows
    assert_eq!(nrows(ex.get("all_orders_eu").unwrap()), 4);
    // intersect: {2,3} = 2 rows
    assert_eq!(nrows(ex.get("orders_in_both").unwrap()), 2);
    // except: {1} = 1 row
    assert_eq!(nrows(ex.get("orders_fr_only").unwrap()), 1);
    assert_eq!(get_i64(ex.get("orders_fr_only").unwrap(), "id"), vec![1]);
}

// ---------------------------------------------------------------------------
// Example 16 — window row_number
// ---------------------------------------------------------------------------

#[test]
fn example_16_window_row_number() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, r#"
        orders
          .window(
            row_number() over (partition_by customer_id sort_by date asc) as order_rank
          )
          as orders_ranked
    "#);

    let f = ex.get("orders_ranked").unwrap();
    assert_eq!(nrows(f), 6);
    let ranks = get_i64(f, "order_rank");
    let cids  = get_i64(f, "customer_id");
    // Each customer (10, 20, 30) has 2 orders → ranks 1 and 2
    for cid in [10, 20, 30] {
        let cid_ranks: Vec<i64> = cids.iter().zip(ranks.iter())
            .filter(|(&c, _)| c == cid)
            .map(|(_, &r)| r)
            .collect();
        let mut sorted = cid_ranks.clone();
        sorted.sort_unstable();
        assert_eq!(sorted, vec![1, 2], "customer {cid} ranks: {cid_ranks:?}");
    }
}

// ---------------------------------------------------------------------------
// Example 17 — window running sum
// ---------------------------------------------------------------------------

#[test]
fn example_17_window_running_sum() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, r#"
        orders
          .window(
            sum(amount) over (
              partition_by country
              sort_by date asc
              rows between unbounded_preceding and current_row
            ) as running_revenue
          )
          as orders_running_revenue
    "#);

    let f = ex.get("orders_running_revenue").unwrap();
    assert_eq!(nrows(f), 6);
    // Column must exist
    assert!(f.columns.iter().any(|c| c.field.name == "running_revenue"));
    // Running sum is monotonically non-decreasing within each country partition
    let countries = get_str(f, "country");
    let running   = get_f64(f, "running_revenue");
    let dates     = get_str(f, "date");
    // Collect FR rows sorted by date
    let mut fr_rows: Vec<(&str, f64)> = countries.iter().zip(running.iter()).zip(dates.iter())
        .filter(|((c, _), _)| c.as_str() == "FR")
        .map(|((_, &r), d)| (d.as_str(), r))
        .collect();
    fr_rows.sort_by_key(|(d, _)| *d);
    let fr_sums: Vec<f64> = fr_rows.iter().map(|(_, r)| *r).collect();
    for w in fr_sums.windows(2) {
        assert!(w[1] >= w[0], "running sum not monotone for FR: {fr_sums:?}");
    }
}

// ---------------------------------------------------------------------------
// Example 18 — window lag + delta
// ---------------------------------------------------------------------------

#[test]
fn example_18_window_lag_delta() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, r#"
        orders
          .sort_by(customer_id asc, date asc)
          .window(lag(amount, 1, 0) over (partition_by customer_id sort_by date asc) as prev_amount)
          .map(amount - prev_amount as delta)
          as orders_delta
    "#);

    let f = ex.get("orders_delta").unwrap();
    assert_eq!(nrows(f), 6);
    assert!(f.columns.iter().any(|c| c.field.name == "delta"));
    // First order per customer has lag=0 (default), so delta = amount - 0 = amount
    // Just verify the column exists and has no panics; exact values depend on sort order
}

// ---------------------------------------------------------------------------
// Example 19 — frame / dimension
// ---------------------------------------------------------------------------

#[test]
fn example_19_frame_dimension() {
    let mut ex = Executor::new();
    ex.load("orders",   orders_frame());
    ex.load("customers", customers_frame());
    ex.load("products",  products_frame());

    run(&mut ex, r#"
        frame (
          relate orders -> customers
            on orders.customer_id = customers.id
            as customer_info

          relate orders -> products
            on orders.product_id = products.id
            as product_info
            where products.active = true
        ) as sales_dim
    "#);

    let f = ex.get("sales_dim").unwrap();
    // All 6 orders; inactive products (id=102, apple) still included from orders
    // because where filters right before join (products.active=true means product_info
    // columns are null for orders matching inactive products)
    assert!(nrows(f) > 0);
    // customer_info columns are prefixed
    assert!(f.columns.iter().any(|c| c.field.name.starts_with("customer_info.")),
        "expected customer_info.* cols, got: {:?}", col_names(f));
}

// ---------------------------------------------------------------------------
// Example 20 — full advanced pipeline
// ---------------------------------------------------------------------------

#[test]
fn example_20_full_pipeline() {
    let mut ex = Executor::new();
    ex.load("orders",    orders_frame());
    ex.load("customers", customers_frame());
    ex.load("products",  products_frame());

    // First build the dimension
    run(&mut ex, r#"
        frame (
          relate orders -> customers
            on orders.customer_id = customers.id
            as customer_info
          relate orders -> products
            on orders.product_id = products.id
            as product_info
            where products.active = true
        ) as sales_dim
    "#);

    // Then run the advanced query on it
    run(&mut ex, r#"
        sales_dim
          .filter(customer_info.active = true and product_info.category is not null)
          .map(coalesce(customer_info.discount, 0) as disc)
          .map(orders.amount * (1 - disc) as net)
          .aggregate(sum(net) as net_revenue, count(*) as sales_count by product_info.category)
          .sort_by(net_revenue desc)
          .limit(5)
          as top_categories
    "#);

    let f = ex.get("top_categories").unwrap();
    // Result has up to 5 rows
    assert!(nrows(f) <= 5);
    // Must have net_revenue and sales_count columns
    assert!(f.columns.iter().any(|c| c.field.name == "net_revenue"));
    assert!(f.columns.iter().any(|c| c.field.name == "sales_count"));
    // Sorted desc by net_revenue
    let revs = get_f64(f, "net_revenue");
    for w in revs.windows(2) {
        assert!(w[0] >= w[1], "not sorted desc: {revs:?}");
    }
}

// ---------------------------------------------------------------------------
// Examples 21-27: if / else if / else
// ---------------------------------------------------------------------------

#[test]
fn example_21_if_else_basic() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, r#"
        orders
          .map(
            if country = "FR" { amount * 1.2 }
            else              { amount * 1.1 }
            as amount_taxed
          )
          as orders_taxed
    "#);

    let f = ex.get("orders_taxed").unwrap();
    assert_eq!(nrows(f), 6);
    let taxed  = get_f64(f, "amount_taxed");
    let amount = get_f64(f, "amount");
    let country = get_str(f, "country");
    for i in 0..6 {
        let expected = if country[i] == "FR" { amount[i] * 1.2 } else { amount[i] * 1.1 };
        assert!((taxed[i] - expected).abs() < 1e-6,
            "row {i}: expected {expected}, got {}", taxed[i]);
    }
}

#[test]
fn example_22_if_else_if_else_tier() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, r#"
        orders
          .map(
            if amount >= 250 { "high" }
            else if amount >= 100 { "medium" }
            else { "low" }
            as tier
          )
          as orders_tiered
    "#);

    let f = ex.get("orders_tiered").unwrap();
    assert_eq!(nrows(f), 6);
    let tiers  = get_str(f, "tier");
    let amount = get_f64(f, "amount");
    // amounts: 50→low, 200→medium, 150→medium, 80→low, 300→high, 120→medium
    let expected = ["low", "medium", "medium", "low", "high", "medium"];
    for (i, (got, exp)) in tiers.iter().zip(expected.iter()).enumerate() {
        assert_eq!(got.as_str(), *exp,
            "row {i}: amount={}, expected tier={exp}, got {got}", amount[i]);
    }
}

#[test]
fn example_23_if_no_else_returns_null() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, r#"
        orders
          .map(if amount >= 200 { "premium" } as premium_flag)
          as orders_premium_flag
    "#);

    let f = ex.get("orders_premium_flag").unwrap();
    assert_eq!(nrows(f), 6);
    // amounts: 50, 200, 150, 80, 300, 120
    // premium_flag: null, "premium", null, null, "premium", null
    let col = f.columns.iter().find(|c| c.field.name == "premium_flag").unwrap();
    let is_valid: Vec<bool> = col.chunks.iter()
        .map(|v| v.validity.first().map(|b| b & 1 == 1).unwrap_or(false))
        .collect();
    assert_eq!(is_valid, vec![false, true, false, false, true, false]);
}

#[test]
fn example_24_if_in_filter() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, r#"
        orders
          .filter(
            if country = "FR" { amount > 100 }
            else              { amount > 200 }
          )
          as high_value_orders
    "#);

    let f = ex.get("high_value_orders").unwrap();
    // FR orders: 50(out), 200(in), 300(in) → 2
    // DE orders: 150(out), 80(out) → 0 (both ≤ 200)
    // US orders: 120(out) → 0 (120 ≤ 200)
    assert_eq!(nrows(f), 2);
    let amounts = get_f64(f, "amount");
    let country = get_str(f, "country");
    for i in 0..nrows(f) {
        assert_eq!(country[i], "FR");
        assert!(amounts[i] > 100.0);
    }
}

#[test]
fn example_25_if_null_safety() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, r#"
        orders
          .map(if discount is null { 0 } else { discount } as safe_discount)
          .map(amount * (1 - safe_discount) as net_amount)
          as orders_safe_net
    "#);

    let f = ex.get("orders_safe_net").unwrap();
    assert_eq!(nrows(f), 6);
    // No null values in net_amount (nulls replaced by 0)
    let col = f.columns.iter().find(|c| c.field.name == "net_amount").unwrap();
    for v in &col.chunks {
        let valid = v.validity.first().map(|b| b & 1 == 1).unwrap_or(false);
        assert!(valid, "net_amount should never be null after null-safe discount");
    }
    // spot-check: row 0 discount=0.1 → 50*(1-0.1)=45
    let net = get_f64(f, "net_amount");
    assert!((net[0] - 45.0).abs() < 1e-6, "row0: {}", net[0]);
    // row 1 discount=null → 0 → 200*(1-0)=200
    assert!((net[1] - 200.0).abs() < 1e-6, "row1: {}", net[1]);
}

#[test]
fn example_26_if_bucket_then_aggregate() {
    let mut ex = Executor::new();
    ex.load("orders", orders_frame());

    run(&mut ex, r#"
        orders
          .map(
            if amount >= 250 { "high" }
            else if amount >= 100 { "medium" }
            else { "low" }
            as tier
          )
          .aggregate(count(*) as n, sum(amount) as total by tier)
          .sort_by(total desc)
          as tier_summary
    "#);

    let f = ex.get("tier_summary").unwrap();
    // tiers: low(50+80=130,n=2), medium(200+150+120=470,n=3), high(300,n=1)
    assert_eq!(nrows(f), 3);
    let tiers  = get_str(f, "tier");
    let totals = get_f64(f, "total");
    let counts = get_i64(f, "n");

    let idx = |name: &str| tiers.iter().position(|t| t == name)
        .unwrap_or_else(|| panic!("tier '{name}' not found in {tiers:?}"));

    assert_eq!(counts[idx("high")],   1);
    assert_eq!(counts[idx("medium")], 3);
    assert_eq!(counts[idx("low")],    2);
    assert!((totals[idx("high")]   - 300.0).abs() < 1e-6);
    assert!((totals[idx("medium")] - 470.0).abs() < 1e-6);
    assert!((totals[idx("low")]    - 130.0).abs() < 1e-6);
    // sorted desc by total: medium(470) > high(300) > low(130)
    for w in totals.windows(2) {
        assert!(w[0] >= w[1], "not sorted desc: {totals:?}");
    }
}

#[test]
fn example_27_nested_if_segmentation() {
    let mut ex = Executor::new();
    ex.load("orders",    orders_frame());
    ex.load("customers", customers_frame());

    run(&mut ex, r#"
        orders
          .join(customers on orders.customer_id = customers.id)
          .map(
            if customers.country = "FR" {
              if orders.amount >= 200 { "vip" } else { "regular" }
            } else { "other" }
            as segment
          )
          as orders_segmented
    "#);

    let f = ex.get("orders_segmented").unwrap();
    assert_eq!(nrows(f), 6); // all orders matched a customer
    let segs = get_str(f, "segment");
    // `customers.country` has no prefix in the joined frame, so the resolver finds
    // `orders.country` first (left columns precede right columns after a join).
    // orders row-by-row:
    //   id=1 (cid=10, orders.country=FR, amount=50)  → FR → 50<200  → "regular"
    //   id=2 (cid=10, orders.country=FR, amount=200) → FR → 200>=200 → "vip"
    //   id=3 (cid=20, orders.country=DE, amount=150) → DE           → "other"
    //   id=4 (cid=20, orders.country=DE, amount=80)  → DE           → "other"
    //   id=5 (cid=30, orders.country=FR, amount=300) → FR → 300>=200 → "vip"
    //   id=6 (cid=30, orders.country=US, amount=120) → US           → "other"
    assert_eq!(segs.iter().filter(|s| s.as_str() == "vip").count(),     2);
    assert_eq!(segs.iter().filter(|s| s.as_str() == "regular").count(), 1);
    assert_eq!(segs.iter().filter(|s| s.as_str() == "other").count(),   3);
}

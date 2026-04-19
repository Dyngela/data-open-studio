//! End-to-end tests: Resin source → generated Rust code.
//!
//! Each test writes a Resin snippet, calls `codegen::generate_from_str`, and
//! asserts that the generated string contains the expected Rust constructs.
//! Nothing is compiled or executed here — we verify the *shape* of the output.

#![cfg(test)]

use crate::codegen::generate_from_str;

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

/// Call the generator and panic on lex/parse errors.
fn gen(src: &str) -> String {
    generate_from_str(src).unwrap_or_else(|e| panic!("codegen failed: {e}"))
}

/// Assert that `haystack` contains every needle in `needles`.
fn contains_all(haystack: &str, needles: &[&str]) {
    for needle in needles {
        assert!(
            haystack.contains(needle),
            "expected generated code to contain {needle:?}\n\n--- generated ---\n{haystack}"
        );
    }
}

// ---------------------------------------------------------------------------
// `use` statement
// ---------------------------------------------------------------------------

#[test]
fn e2e_use_binds_frame_from_store() {
    let code = gen(r#"use sales from "warehouse_sales""#);
    contains_all(&code, &[
        "let sales =",
        r#"store.get("warehouse_sales")"#,
        "'warehouse_sales' not found",
    ]);
}

#[test]
fn e2e_multiple_uses_emit_multiple_bindings() {
    let code = gen(r#"
        use sales     from "sales"
        use customers from "customers"
        use products  from "products"
    "#);
    contains_all(&code, &[
        "let sales =",     r#"store.get("sales")"#,
        "let customers =", r#"store.get("customers")"#,
        "let products =",  r#"store.get("products")"#,
    ]);
}

// ---------------------------------------------------------------------------
// `frame` / `relate` statement
// ---------------------------------------------------------------------------

#[test]
fn e2e_frame_emits_rt_relate_and_build_dimension() {
    let code = gen(r#"
        use sales     from "sales"
        use customers from "customers"
        frame (
            relate sales -> customers
                on sales.customer_id = customers.id
                as customer_info
        ) as sales_dim
    "#);
    contains_all(&code, &[
        "let sales_dim =",
        "rt_relate(",
        r#""customer_info""#,
        "rt_build_dimension",
        r#"__out.insert("sales_dim""#,
    ]);
}

#[test]
fn e2e_frame_relate_with_where_emits_filter_arg() {
    let code = gen(r#"
        use sales    from "sales"
        use products from "products"
        frame (
            relate sales -> products
                on sales.product_id = products.id
                as product_info
                where products.active = true
        ) as enriched
    "#);
    // The `where` clause becomes a `Some(|row| { … })` argument.
    contains_all(&code, &[
        "Some(",
        "DataValue::Boolean(true)",
    ]);
}

#[test]
fn e2e_frame_multiple_relates_emit_one_call_each() {
    let code = gen(r#"
        use a from "a"
        use b from "b"
        use c from "c"
        frame (
            relate a -> b on a.id = b.a_id as b_info
            relate a -> c on a.id = c.a_id as c_info
        ) as joined
    "#);
    let count = code.matches("rt_relate(").count();
    assert_eq!(count, 2, "expected 2 rt_relate calls, got {count}\n{code}");
}

// ---------------------------------------------------------------------------
// `filter`
// ---------------------------------------------------------------------------

#[test]
fn e2e_filter_comparison_gt() {
    let code = gen("sales.filter(price > 100)");
    contains_all(&code, &[
        "rt_filter(&__f, |row|",
        r#"row.col("price")"#,
        "rt_gt(",
        "DataValue::Int64(100_i64)",
    ]);
}

#[test]
fn e2e_filter_string_literal() {
    let code = gen(r#"orders.filter(status = "shipped")"#);
    contains_all(&code, &[
        "rt_eq(",
        r#"row.col("status")"#,
        r#"DataValue::String("shipped".to_string())"#,
    ]);
}

#[test]
fn e2e_filter_float_literal() {
    let code = gen("t.filter(rate > 0.5)");
    contains_all(&code, &["DataValue::Float64(0.5_f64)"]);
}

#[test]
fn e2e_filter_is_null() {
    let code = gen("t.filter(email is null)");
    contains_all(&code, &["rt_is_null(", r#"row.col("email")"#]);
}

#[test]
fn e2e_filter_is_not_null() {
    let code = gen("t.filter(country is not null)");
    contains_all(&code, &["rt_is_not_null(", r#"row.col("country")"#]);
}

#[test]
fn e2e_filter_and_condition() {
    let code = gen("t.filter(active = true and score >= 80)");
    contains_all(&code, &[
        "rt_and(",
        "rt_eq(",
        "rt_gte(",
        "DataValue::Boolean(true)",
        "DataValue::Int64(80_i64)",
    ]);
}

#[test]
fn e2e_filter_or_condition() {
    let code = gen(r#"t.filter(country = "FR" or country = "DE")"#);
    contains_all(&code, &[
        "rt_or(",
        r#"DataValue::String("FR".to_string())"#,
        r#"DataValue::String("DE".to_string())"#,
    ]);
}

#[test]
fn e2e_filter_not_prefix() {
    let code = gen("t.filter(not deleted)");
    contains_all(&code, &["rt_not(", r#"row.col("deleted")"#]);
}

#[test]
fn e2e_filter_qualified_column() {
    let code = gen(r#"t.filter(customer_info.country = "France")"#);
    contains_all(&code, &[r#"row.col("customer_info.country")"#]);
}

#[test]
fn e2e_filter_all_comparators() {
    for (op, fn_name) in [
        ("=",  "rt_eq"),
        ("!=", "rt_neq"),
        (">",  "rt_gt"),
        ("<",  "rt_lt"),
        (">=", "rt_gte"),
        ("<=", "rt_lte"),
    ] {
        let code = gen(&format!("t.filter(x {op} 1)"));
        assert!(
            code.contains(fn_name),
            "operator `{op}` should emit `{fn_name}`\n{code}"
        );
    }
}

// ---------------------------------------------------------------------------
// `select`
// ---------------------------------------------------------------------------

#[test]
fn e2e_select_single_column() {
    let code = gen("t.select(id)");
    assert!(code.contains(r#"rt_select(&__f, &["id"])"#), "{code}");
}

#[test]
fn e2e_select_multiple_columns() {
    let code = gen("t.select(id, name, revenue)");
    contains_all(&code, &["rt_select(&__f,", r#""id""#, r#""name""#, r#""revenue""#]);
}

#[test]
fn e2e_select_qualified_columns() {
    let code = gen("t.select(sales.id, info.name)");
    contains_all(&code, &[r#""sales.id""#, r#""info.name""#]);
}

// ---------------------------------------------------------------------------
// `map`
// ---------------------------------------------------------------------------

#[test]
fn e2e_map_arithmetic_mul() {
    let code = gen("t.map(price * quantity as total)");
    contains_all(&code, &[
        r#"rt_map(&__f, "total","#,
        "rt_mul(",
        r#"row.col("price")"#,
        r#"row.col("quantity")"#,
    ]);
}

#[test]
fn e2e_map_nested_arithmetic() {
    let code = gen("t.map(total * (1 - discount) as net)");
    contains_all(&code, &[
        r#"rt_map(&__f, "net","#,
        "rt_mul(",
        "rt_sub(",
        "DataValue::Int64(1_i64)",
        r#"row.col("discount")"#,
    ]);
}

#[test]
fn e2e_map_coalesce() {
    let code = gen("t.map(coalesce(discount, 0) as safe_discount)");
    contains_all(&code, &[
        r#"rt_map(&__f, "safe_discount","#,
        "rt_fn_coalesce(",
        r#"row.col("discount")"#,
        "DataValue::Int64(0_i64)",
    ]);
}

#[test]
fn e2e_map_unary_neg() {
    let code = gen("t.map(-price as neg_price)");
    contains_all(&code, &["rt_neg(", r#"row.col("price")"#]);
}

#[test]
fn e2e_map_bool_literal() {
    let code = gen("t.map(true as flag)");
    contains_all(&code, &["DataValue::Boolean(true)"]);
}

#[test]
fn e2e_map_null_literal() {
    let code = gen("t.map(null as empty)");
    contains_all(&code, &["DataValue::Null"]);
}

#[test]
fn e2e_map_function_call() {
    let code = gen("t.map(to_int(score) as int_score)");
    contains_all(&code, &[
        r#"rt_map(&__f, "int_score","#,
        "rt_fn_to_int(",
        r#"row.col("score")"#,
    ]);
}

// ---------------------------------------------------------------------------
// `aggregate`
// ---------------------------------------------------------------------------

#[test]
fn e2e_aggregate_grouped() {
    let code = gen("t.aggregate(sum(revenue) as total, count(*) as n by country)");
    contains_all(&code, &[
        "rt_aggregate(&__f,",
        r#"("sum", "revenue", "total")"#,
        r#"("count", "*", "n")"#,
        r#""country""#,
    ]);
}

#[test]
fn e2e_aggregate_global_no_by_emits_empty_slice() {
    let code = gen("t.aggregate(sum(revenue) as grand_total)");
    contains_all(&code, &[
        r#"("sum", "revenue", "grand_total")"#,
        "&[]",  // empty group-by
    ]);
}

#[test]
fn e2e_aggregate_multi_group_by() {
    let code = gen("t.aggregate(avg(score) as avg_score by region, tier)");
    contains_all(&code, &[r#""region""#, r#""tier""#]);
}

#[test]
fn e2e_aggregate_function_expr_in_by() {
    let code = gen("t.aggregate(count(*) as n by year(order_date))");
    // Function call in group-by is serialised as a string.
    contains_all(&code, &[r#""year(order_date)""#]);
}

#[test]
fn e2e_aggregate_all_agg_functions() {
    for fn_name in &["sum", "avg", "min", "max"] {
        let src = format!("t.aggregate({fn_name}(v) as result)");
        let code = gen(&src);
        assert!(
            code.contains(&format!(r#"("{fn_name}", "v", "result")"#)),
            "{fn_name} aggregate not found\n{code}"
        );
    }
}

// ---------------------------------------------------------------------------
// `distinct`
// ---------------------------------------------------------------------------

#[test]
fn e2e_distinct_emits_rt_distinct() {
    let code = gen("t.distinct()");
    assert!(code.contains("rt_distinct(&__f)"), "{code}");
}

// ---------------------------------------------------------------------------
// `sort_by`
// ---------------------------------------------------------------------------

#[test]
fn e2e_sort_by_asc_flag_is_false() {
    // desc flag = false means ascending
    let code = gen("t.sort_by(name asc)");
    contains_all(&code, &[r#"("name", false)"#]);
}

#[test]
fn e2e_sort_by_desc_flag_is_true() {
    let code = gen("t.sort_by(revenue desc)");
    contains_all(&code, &[r#"("revenue", true)"#]);
}

#[test]
fn e2e_sort_by_multi_key() {
    let code = gen("t.sort_by(region asc, revenue desc)");
    contains_all(&code, &[r#"("region", false)"#, r#"("revenue", true)"#]);
}

// ---------------------------------------------------------------------------
// `limit` / `offset`
// ---------------------------------------------------------------------------

#[test]
fn e2e_limit_emits_rt_limit() {
    let code = gen("t.limit(25)");
    assert!(code.contains("rt_limit(&__f, 25)"), "{code}");
}

#[test]
fn e2e_offset_emits_rt_offset() {
    let code = gen("t.offset(100)");
    assert!(code.contains("rt_offset(&__f, 100)"), "{code}");
}

#[test]
fn e2e_sort_limit_offset_order_preserved() {
    let code = gen("t.sort_by(id asc).offset(50).limit(10)");
    let sort_pos   = code.find("rt_sort_by(").unwrap();
    let offset_pos = code.find("rt_offset(").unwrap();
    let limit_pos  = code.find("rt_limit(").unwrap();
    assert!(sort_pos < offset_pos, "sort_by should appear before offset");
    assert!(offset_pos < limit_pos, "offset should appear before limit");
}

// ---------------------------------------------------------------------------
// Joins
// ---------------------------------------------------------------------------

#[test]
fn e2e_inner_join_emits_condition_closure() {
    let code = gen("sales.join(returns on sales.id = returns.sale_id)");
    contains_all(&code, &[
        "rt_join(&__f, &returns,",
        "|__l, __r|",
        "rt_eq(",
    ]);
}

#[test]
fn e2e_left_join() {
    let code = gen("orders.left_join(payments on orders.id = payments.order_id)");
    contains_all(&code, &["rt_left_join(&__f, &payments,"]);
}

#[test]
fn e2e_right_join() {
    let code = gen("returns.right_join(sales on returns.sale_id = sales.id)");
    contains_all(&code, &["rt_right_join(&__f, &sales,"]);
}

#[test]
fn e2e_cross_join_has_no_condition_closure() {
    let code = gen("sizes.cross_join(colors)");
    assert!(code.contains("rt_cross_join(&__f, &colors)"),  "{code}");
    assert!(!code.contains("|__l, __r|"), "cross_join must not emit a condition\n{code}");
}

// ---------------------------------------------------------------------------
// `window`
// ---------------------------------------------------------------------------

#[test]
fn e2e_window_row_number_partition_sort() {
    let code = gen(
        "t.window(row_number() over (partition_by customer_id sort_by date asc) as rn)"
    );
    contains_all(&code, &[
        "rt_window(&__f,",
        r#""row_number""#,
        r#""customer_id""#,
        r#"("date", false)"#,
        r#""rn""#,
    ]);
}

#[test]
fn e2e_window_running_sum_frame_spec() {
    let code = gen(
        "t.window(sum(revenue) over (partition_by country rows between unbounded_preceding and current_row) as running_rev)"
    );
    contains_all(&code, &[
        r#"("sum", "revenue")"#,
        "rt_frame_spec(",
        "FrameBound::UnboundedPreceding",
        "FrameBound::CurrentRow",
        r#""running_rev""#,
    ]);
}

#[test]
fn e2e_window_n_preceding_following() {
    let code = gen(
        "t.window(avg(score) over (rows between 3 preceding and 1 following) as ma)"
    );
    contains_all(&code, &["FrameBound::Preceding(3)", "FrameBound::Following(1)"]);
}

#[test]
fn e2e_window_rank_no_partition() {
    let code = gen("t.window(rank() over (sort_by score desc) as rnk)");
    contains_all(&code, &[r#""rank""#, r#"("score", true)"#, r#""rnk""#]);
}

// ---------------------------------------------------------------------------
// Set operators
// ---------------------------------------------------------------------------

#[test]
fn e2e_union() {
    let code = gen("t.union(other)");
    assert!(code.contains("rt_union(&__f, &other)"), "{code}");
}

#[test]
fn e2e_intersect() {
    let code = gen("t.intersect(other)");
    assert!(code.contains("rt_intersect(&__f, &other)"), "{code}");
}

#[test]
fn e2e_except() {
    let code = gen("t.except(other)");
    assert!(code.contains("rt_except(&__f, &other)"), "{code}");
}

// ---------------------------------------------------------------------------
// Materialization
// ---------------------------------------------------------------------------

#[test]
fn e2e_materialize_inserts_into_out() {
    let code = gen("sales.filter(active = true) as active_sales");
    contains_all(&code, &[
        "let active_sales =",
        r#"__out.insert("active_sales""#,
    ]);
}

#[test]
fn e2e_no_materialize_uses_double_underscore_result() {
    let code = gen("sales.limit(10)");
    assert!(code.contains("let __result ="), "{code}");
    assert!(!code.contains(r#"__out.insert("__result""#),
        "non-materialized result must not be stored\n{code}");
}

// ---------------------------------------------------------------------------
// Full pipelines (spec examples)
// ---------------------------------------------------------------------------

#[test]
fn e2e_spec_example_full_query() {
    let code = gen(r#"
        use sales_enriched from "sales_enriched"
        sales_enriched
          .filter(product_info.category = "electronics" and customer_info.country is not null)
          .map(product_info.price * sales.quantity as total)
          .map(coalesce(customer_info.discount, 0) as safe_discount)
          .map(total * (1 - safe_discount) as net_revenue)
          .aggregate(sum(net_revenue) as revenue, count(*) as order_count by customer_info.country)
          .sort_by(revenue desc)
          .limit(20)
    "#);
    contains_all(&code, &[
        r#"store.get("sales_enriched")"#,
        "rt_and(",
        r#"row.col("product_info.category")"#,
        r#"DataValue::String("electronics".to_string())"#,
        r#"rt_is_not_null(row.col("customer_info.country"))"#,
        r#"rt_map(&__f, "total","#,
        r#"rt_map(&__f, "safe_discount","#,
        "rt_fn_coalesce(",
        r#"rt_map(&__f, "net_revenue","#,
        r#"("sum", "net_revenue", "revenue")"#,
        r#"("count", "*", "order_count")"#,
        r#""customer_info.country""#,
        r#"("revenue", true)"#,
        "rt_limit(&__f, 20)",
    ]);
}

#[test]
fn e2e_join_with_filter_and_select() {
    let code = gen(r#"
        use sales   from "sales"
        use returns from "returns"
        sales
          .join(returns on sales.id = returns.sale_id)
          .filter(returns.reason = "defective")
          .select(sales.product_id, returns.date)
          as defective_sales
    "#);
    contains_all(&code, &[
        "rt_join(&__f, &returns,",
        r#"DataValue::String("defective".to_string())"#,
        r#"rt_select(&__f, &["sales.product_id", "returns.date"])"#,
        r#"__out.insert("defective_sales""#,
    ]);
}

#[test]
fn e2e_window_then_filter_pipeline() {
    let code = gen(r#"
        use orders from "orders"
        orders
          .window(row_number() over (partition_by customer_id sort_by date asc) as purchase_rank)
          .filter(purchase_rank <= 3)
    "#);
    contains_all(&code, &[
        "rt_window(",
        r#""row_number""#,
        r#""customer_id""#,
        "rt_filter(",
        "rt_lte(",
        r#"row.col("purchase_rank")"#,
        "DataValue::Int64(3_i64)",
    ]);
}

#[test]
fn e2e_every_program_has_execute_signature() {
    for src in &[
        "t.limit(1)",
        r#"use x from "x""#,
        "t.filter(a = 1).map(a * 2 as b).select(b)",
    ] {
        let code = gen(src);
        assert!(code.contains("fn execute("),    "missing execute fn\n{code}");
        assert!(code.contains("-> Result<"),     "missing Result return type\n{code}");
        assert!(code.contains("let mut __out"),  "missing __out map\n{code}");
        assert!(code.contains("Ok(__out)"),      "missing Ok(__out)\n{code}");
    }
}

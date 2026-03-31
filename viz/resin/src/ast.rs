//! Abstract Syntax Tree for the Resin query language.
//!
//! Every node carries a `Span` for error reporting.  The tree mirrors the
//! EBNF grammar in RESIN.md almost 1-to-1, extended with the semantic
//! distinctions the spec calls out (e.g. `JoinKind`, `SortDirection`).

use crate::lexer::Span;

// ---------------------------------------------------------------------------
// Top-level
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct Program {
    pub statements: Vec<Statement>,
}

#[derive(Debug, Clone)]
pub enum Statement {
    Use(UseStmt),
    Frame(FrameStmt),
    Query(QueryStmt),
}

// ---------------------------------------------------------------------------
// `use <name> from <source>`
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct UseStmt {
    pub name: Ident,
    pub source: StringLit,
    pub span: Span,
}

// ---------------------------------------------------------------------------
// `frame ( relate_clause+ ) as <name>`
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct FrameStmt {
    pub relates: Vec<RelateClause>,
    pub name: Ident,
    pub span: Span,
}

/// `relate <left> -> <right> on <cond> as <alias> [where <filter>]`
#[derive(Debug, Clone)]
pub struct RelateClause {
    pub left: Ident,
    pub right: Ident,
    /// Join condition for the relation.
    pub condition: Expr,
    /// Alias under which the joined columns are stored in the dimension.
    pub alias: Ident,
    /// Optional predicate that filters the *right* frame before joining.
    pub filter: Option<Expr>,
    pub span: Span,
}

// ---------------------------------------------------------------------------
// Query statement  (`<source> pipe_chain [as <name>]`)
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct QueryStmt {
    pub source: Ident,
    pub ops: Vec<PipeOp>,
    /// If present, the result is materialized and stored under this name.
    pub materialize: Option<Ident>,
    pub span: Span,
}

// ---------------------------------------------------------------------------
// Pipe operators
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub enum PipeOp {
    Filter(FilterOp),
    Select(SelectOp),
    Map(MapOp),
    Aggregate(AggregateOp),
    Distinct(Span),
    SortBy(SortByOp),
    Limit(LimitOp),
    Offset(OffsetOp),
    Join(JoinOp),
    Window(WindowOp),
    Union(SetOp),
    Intersect(SetOp),
    Except(SetOp),
}

/// `.filter(<condition>)`
#[derive(Debug, Clone)]
pub struct FilterOp {
    pub condition: Expr,
    pub span: Span,
}

/// `.select(<col>, …)`  — pure projection, no computation or renaming.
#[derive(Debug, Clone)]
pub struct SelectOp {
    pub columns: Vec<Expr>,
    pub span: Span,
}

/// `.map(<expr> as <alias>)`  — always requires an alias.
#[derive(Debug, Clone)]
pub struct MapOp {
    pub expr: Expr,
    pub alias: Ident,
    pub span: Span,
}

/// `.aggregate(<agg_expr>, … [by <expr>, …])`
#[derive(Debug, Clone)]
pub struct AggregateOp {
    pub aggregations: Vec<AggExpr>,
    pub group_by: Vec<Expr>,
    pub span: Span,
}

/// A single aggregation expression with its alias: `sum(revenue) as total`.
#[derive(Debug, Clone)]
pub struct AggExpr {
    pub func: FunctionCall,
    pub alias: Ident,
    pub span: Span,
}

/// `.sort_by(<col> [asc|desc], …)`
#[derive(Debug, Clone)]
pub struct SortByOp {
    pub columns: Vec<SortCol>,
    pub span: Span,
}

#[derive(Debug, Clone)]
pub struct SortCol {
    pub expr: Expr,
    pub direction: SortDirection,
    pub span: Span,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SortDirection {
    Asc,
    Desc,
}

/// `.limit(<n>)`
#[derive(Debug, Clone)]
pub struct LimitOp {
    pub n: i64,
    pub span: Span,
}

/// `.offset(<n>)`
#[derive(Debug, Clone)]
pub struct OffsetOp {
    pub n: i64,
    pub span: Span,
}

/// `.join(…)` / `.left_join(…)` / `.right_join(…)` / `.cross_join(…)`
#[derive(Debug, Clone)]
pub struct JoinOp {
    pub kind: JoinKind,
    pub frame: Ident,
    /// `None` only for `cross_join` (cartesian product, no condition).
    pub condition: Option<Expr>,
    pub span: Span,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum JoinKind {
    Inner,
    Left,
    Right,
    Cross,
}

/// `.window(<window_expr>)`
#[derive(Debug, Clone)]
pub struct WindowOp {
    pub expr: WindowExpr,
    pub span: Span,
}

#[derive(Debug, Clone)]
pub struct WindowExpr {
    pub func: FunctionCall,
    pub partition_by: Vec<Expr>,
    pub sort_by: Vec<SortCol>,
    pub frame_spec: Option<FrameSpec>,
    pub alias: Ident,
    pub span: Span,
}

/// `rows between <start> and <end>`
#[derive(Debug, Clone)]
pub struct FrameSpec {
    pub start: FrameBound,
    pub end: FrameBound,
    pub span: Span,
}

#[derive(Debug, Clone)]
pub enum FrameBound {
    UnboundedPreceding,
    CurrentRow,
    UnboundedFollowing,
    /// `<n> preceding`
    Preceding(i64),
    /// `<n> following`
    Following(i64),
}

/// `.union(…)` / `.intersect(…)` / `.except(…)`
#[derive(Debug, Clone)]
pub struct SetOp {
    pub frame: Ident,
    pub span: Span,
}

// ---------------------------------------------------------------------------
// Expressions
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub enum Expr {
    BinaryOp(Box<BinaryOpExpr>),
    UnaryOp(Box<UnaryOpExpr>),
    FunctionCall(FunctionCall),
    Column(ColumnRef),
    Literal(Literal),
    /// `<expr> is null`
    IsNull(Box<Expr>, Span),
    /// `<expr> is not null`
    IsNotNull(Box<Expr>, Span),
    /// Parenthesised expression — kept in the tree for source-fidelity.
    Grouped(Box<Expr>, Span),
}

impl Expr {
    pub fn span(&self) -> &Span {
        match self {
            Expr::BinaryOp(e)    => &e.span,
            Expr::UnaryOp(e)     => &e.span,
            Expr::FunctionCall(f)=> &f.span,
            Expr::Column(c)      => &c.span,
            Expr::Literal(l)     => l.span(),
            Expr::IsNull(_, s)   => s,
            Expr::IsNotNull(_, s)=> s,
            Expr::Grouped(_, s)  => s,
        }
    }
}

#[derive(Debug, Clone)]
pub struct BinaryOpExpr {
    pub left: Expr,
    pub op: BinaryOp,
    pub right: Expr,
    pub span: Span,
}

/// Binary operators in precedence order (lowest → highest at the bottom of
/// the list).  Precedence is enforced by the Pratt parser, not this enum.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum BinaryOp {
    // Logical (lowest)
    Or,
    And,
    // Comparison
    Eq,
    NotEq,
    Gt,
    Lt,
    Gte,
    Lte,
    // Arithmetic (highest binary)
    Add,
    Sub,
    Mul,
    Div,
}

#[derive(Debug, Clone)]
pub struct UnaryOpExpr {
    pub op: UnaryOp,
    pub expr: Expr,
    pub span: Span,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum UnaryOp {
    /// Unary minus: `-expr`
    Neg,
    /// Logical NOT: `not expr`
    Not,
}

// ---------------------------------------------------------------------------
// Function call
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct FunctionCall {
    pub name: String,
    pub args: Vec<FunctionArg>,
    pub span: Span,
}

#[derive(Debug, Clone)]
pub enum FunctionArg {
    Expr(Expr),
    /// The bare `*` in `count(*)`
    Star,
}

// ---------------------------------------------------------------------------
// Column reference
// ---------------------------------------------------------------------------

/// Either `column` (bare) or `table.column` (qualified).
#[derive(Debug, Clone)]
pub struct ColumnRef {
    /// `Some("sales")` for `sales.revenue`, `None` for bare `revenue`.
    pub table: Option<String>,
    pub column: String,
    pub span: Span,
}

// ---------------------------------------------------------------------------
// Literals
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub enum Literal {
    Int(i64, Span),
    Float(f64, Span),
    String(String, Span),
    Bool(bool, Span),
    Null(Span),
}

impl Literal {
    pub fn span(&self) -> &Span {
        match self {
            Literal::Int(_, s)    => s,
            Literal::Float(_, s)  => s,
            Literal::String(_, s) => s,
            Literal::Bool(_, s)   => s,
            Literal::Null(s)      => s,
        }
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct Ident {
    pub name: String,
    pub span: Span,
}

#[derive(Debug, Clone)]
pub struct StringLit {
    pub value: String,
    pub span: Span,
}

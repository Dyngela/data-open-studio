//! Hand-written recursive descent parser with Pratt parsing for expressions.
//!
//! Takes a `Vec<Token>` produced by the lexer and returns a `(Program,
//! Vec<ParseError>)`.  Errors are collected rather than aborting: on a parse
//! failure the parser skips to the next statement boundary and continues,
//! so analysts see all errors in a single pass.

use crate::ast::*;
use crate::lexer::{Span, Token, TokenKind};
use std::fmt;

// ---------------------------------------------------------------------------
// Parse error
// ---------------------------------------------------------------------------

#[derive(Debug)]
pub struct ParseError {
    pub message: String,
    pub span: Span,
}

impl fmt::Display for ParseError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "parse error at {}: {}", self.span, self.message)
    }
}

impl std::error::Error for ParseError {}

// ---------------------------------------------------------------------------
// Parser state
// ---------------------------------------------------------------------------

struct Parser {
    tokens: Vec<Token>,
    /// Current position in `tokens`.  Always points to a valid index; the
    /// last token is always `Eof` so we never go out of bounds.
    pos: usize,
    errors: Vec<ParseError>,
}

impl Parser {
    fn new(tokens: Vec<Token>) -> Self {
        // Guarantee at least one token (Eof) so `current()` is always safe.
        assert!(!tokens.is_empty(), "token list must contain at least Eof");
        Self { tokens, pos: 0, errors: Vec::new() }
    }

    // ── token access ──────────────────────────────────────────────────────

    fn current(&self) -> &Token {
        &self.tokens[self.pos]
    }

    fn advance(&mut self) {
        if self.pos + 1 < self.tokens.len() {
            self.pos += 1;
        }
    }

    fn span(&self) -> Span {
        self.current().span.clone()
    }

    fn at_eof(&self) -> bool {
        self.current().kind == TokenKind::Eof
    }

    // ── expectations ──────────────────────────────────────────────────────

    /// Consume the current token if it matches `kind`; otherwise return an
    /// error.  Returns the span of the consumed token on success.
    fn expect(&mut self, kind: &TokenKind) -> Result<Span, ParseError> {
        if &self.current().kind == kind {
            let span = self.span();
            self.advance();
            Ok(span)
        } else {
            Err(self.err(format!(
                "expected {}, got {}",
                token_kind_name(kind),
                token_kind_name(&self.current().kind)
            )))
        }
    }

    /// Consume and return the current token as an `Ident`, or error.
    fn expect_ident(&mut self) -> Result<Ident, ParseError> {
        let span = self.span();
        match self.current().kind.clone() {
            TokenKind::Ident(name) => {
                self.advance();
                Ok(Ident { name, span })
            }
            _ => Err(self.err(format!(
                "expected identifier, got {}",
                token_kind_name(&self.current().kind)
            ))),
        }
    }

    fn err(&self, message: impl Into<String>) -> ParseError {
        ParseError { message: message.into(), span: self.span() }
    }

    // ── error recovery ─────────────────────────────────────────────────────
    /// Skip tokens until we reach something that looks like the start of a new
    /// statement (`use`, `frame`, a bare `Ident`).
    fn recover_to_statement(&mut self) {
        while !self.at_eof() {
            match &self.current().kind {
                TokenKind::Use | TokenKind::Frame | TokenKind::Ident(_) => return,
                _ => { self.advance(); }
            }
        }
    }

    // ── top-level ──────────────────────────────────────────────────────────

    fn parse_program(mut self) -> (Program, Vec<ParseError>) {
        let mut statements = Vec::new();
        while !self.at_eof() {
            match self.parse_statement() {
                Ok(s)  => statements.push(s),
                Err(e) => {
                    self.errors.push(e);
                    self.recover_to_statement();
                }
            }
        }
        let errors = self.errors;
        (Program { statements }, errors)
    }

    fn parse_statement(&mut self) -> Result<Statement, ParseError> {
        match &self.current().kind {
            TokenKind::Use   => self.parse_use().map(Statement::Use),
            TokenKind::Frame => self.parse_frame().map(Statement::Frame),
            TokenKind::Ident(_) => self.parse_query().map(Statement::Query),
            _ => Err(self.err(format!(
                "expected statement (use / frame / query), got {}",
                token_kind_name(&self.current().kind)
            ))),
        }
    }

    // ── `use` statement ────────────────────────────────────────────────────

    fn parse_use(&mut self) -> Result<UseStmt, ParseError> {
        let span = self.span();
        self.expect(&TokenKind::Use)?;
        let name   = self.expect_ident()?;
        self.expect(&TokenKind::From)?;
        let source = self.parse_string_lit()?;
        Ok(UseStmt { name, source, span })
    }

    fn parse_string_lit(&mut self) -> Result<StringLit, ParseError> {
        let span = self.span();
        match self.current().kind.clone() {
            TokenKind::String(s) => { self.advance(); Ok(StringLit { value: s, span }) }
            _ => Err(self.err("expected string literal")),
        }
    }

    // ── `frame` statement ──────────────────────────────────────────────────

    fn parse_frame(&mut self) -> Result<FrameStmt, ParseError> {
        let span = self.span();
        self.expect(&TokenKind::Frame)?;
        self.expect(&TokenKind::LParen)?;
        let mut relates = Vec::new();
        while self.current().kind != TokenKind::RParen && !self.at_eof() {
            relates.push(self.parse_relate_clause()?);
        }
        self.expect(&TokenKind::RParen)?;
        self.expect(&TokenKind::As)?;
        let name = self.expect_ident()?;
        Ok(FrameStmt { relates, name, span })
    }

    fn parse_relate_clause(&mut self) -> Result<RelateClause, ParseError> {
        let span = self.span();
        self.expect(&TokenKind::Relate)?;
        let left  = self.expect_ident()?;
        self.expect(&TokenKind::Arrow)?;
        let right = self.expect_ident()?;
        self.expect(&TokenKind::On)?;
        let condition = self.parse_expr(0)?;
        self.expect(&TokenKind::As)?;
        let alias = self.expect_ident()?;
        let filter = if self.current().kind == TokenKind::Where {
            self.advance();
            Some(self.parse_expr(0)?)
        } else {
            None
        };
        Ok(RelateClause { left, right, condition, alias, filter, span })
    }

    // ── query statement ────────────────────────────────────────────────────

    fn parse_query(&mut self) -> Result<QueryStmt, ParseError> {
        let span   = self.span();
        let source = self.expect_ident()?;
        let mut ops = Vec::new();
        while self.current().kind == TokenKind::Dot && !self.at_eof() {
            self.advance(); // consume '.'
            ops.push(self.parse_pipe_op()?);
        }
        let materialize = if self.current().kind == TokenKind::As {
            self.advance();
            Some(self.expect_ident()?)
        } else {
            None
        };
        Ok(QueryStmt { source, ops, materialize, span })
    }

    // ── pipe operators ─────────────────────────────────────────────────────

    fn parse_pipe_op(&mut self) -> Result<PipeOp, ParseError> {
        let span = self.span();
        match self.current().kind.clone() {
            TokenKind::Filter => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let condition = self.parse_expr(0)?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Filter(FilterOp { condition, span }))
            }

            TokenKind::Select => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let columns = self.parse_comma_list(|p| p.parse_expr(0))?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Select(SelectOp { columns, span }))
            }

            TokenKind::Map => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let expr  = self.parse_expr(0)?;
                self.expect(&TokenKind::As)?;
                let alias = self.expect_ident()?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Map(MapOp { expr, alias, span }))
            }

            TokenKind::Aggregate => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let (aggregations, group_by) = self.parse_aggregate_body()?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Aggregate(AggregateOp { aggregations, group_by, span }))
            }

            TokenKind::Distinct => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Distinct(span))
            }

            TokenKind::SortBy => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let columns = self.parse_comma_list(|p| p.parse_sort_col())?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::SortBy(SortByOp { columns, span }))
            }

            TokenKind::Limit => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let n = self.parse_int()?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Limit(LimitOp { n, span }))
            }

            TokenKind::Offset => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let n = self.parse_int()?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Offset(OffsetOp { n, span }))
            }

            TokenKind::Join => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let frame = self.expect_ident()?;
                self.expect(&TokenKind::On)?;
                let cond = self.parse_expr(0)?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Join(JoinOp { kind: JoinKind::Inner, frame, condition: Some(cond), span }))
            }

            TokenKind::LeftJoin => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let frame = self.expect_ident()?;
                self.expect(&TokenKind::On)?;
                let cond = self.parse_expr(0)?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Join(JoinOp { kind: JoinKind::Left, frame, condition: Some(cond), span }))
            }

            TokenKind::RightJoin => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let frame = self.expect_ident()?;
                self.expect(&TokenKind::On)?;
                let cond = self.parse_expr(0)?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Join(JoinOp { kind: JoinKind::Right, frame, condition: Some(cond), span }))
            }

            TokenKind::CrossJoin => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let frame = self.expect_ident()?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Join(JoinOp { kind: JoinKind::Cross, frame, condition: None, span }))
            }

            TokenKind::Window => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let expr = self.parse_window_expr()?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Window(WindowOp { expr, span }))
            }

            TokenKind::Union => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let frame = self.expect_ident()?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Union(SetOp { frame, span }))
            }

            TokenKind::Intersect => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let frame = self.expect_ident()?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Intersect(SetOp { frame, span }))
            }

            TokenKind::Except => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let frame = self.expect_ident()?;
                self.expect(&TokenKind::RParen)?;
                Ok(PipeOp::Except(SetOp { frame, span }))
            }

            _ => Err(self.err(format!(
                "expected pipe operator (filter / select / map / …), got {}",
                token_kind_name(&self.current().kind)
            ))),
        }
    }

    // ── aggregate body ─────────────────────────────────────────────────────

    /// Parses `agg_expr (, agg_expr)* [by expr (, expr)*]` up to (but not
    /// including) the closing `)`.
    fn parse_aggregate_body(&mut self) -> Result<(Vec<AggExpr>, Vec<Expr>), ParseError> {
        let mut aggs = Vec::new();
        loop {
            if matches!(self.current().kind, TokenKind::By | TokenKind::RParen) {
                break;
            }
            aggs.push(self.parse_agg_expr()?);
            if self.current().kind == TokenKind::Comma {
                self.advance();
            } else {
                break;
            }
        }
        let group_by = if self.current().kind == TokenKind::By {
            self.advance();
            self.parse_comma_list(|p| p.parse_expr(0))?
        } else {
            Vec::new()
        };
        Ok((aggs, group_by))
    }

    fn parse_agg_expr(&mut self) -> Result<AggExpr, ParseError> {
        let span = self.span();
        let func  = self.parse_function_call()?;
        self.expect(&TokenKind::As)?;
        let alias = self.expect_ident()?;
        Ok(AggExpr { func, alias, span })
    }

    // ── sort column ────────────────────────────────────────────────────────

    fn parse_sort_col(&mut self) -> Result<SortCol, ParseError> {
        let span      = self.span();
        let expr      = self.parse_expr(0)?;
        let direction = match self.current().kind {
            TokenKind::Asc  => { self.advance(); SortDirection::Asc }
            TokenKind::Desc => { self.advance(); SortDirection::Desc }
            _               => SortDirection::Asc,
        };
        Ok(SortCol { expr, direction, span })
    }

    // ── window expression ──────────────────────────────────────────────────

    fn parse_window_expr(&mut self) -> Result<WindowExpr, ParseError> {
        let span = self.span();
        let func = self.parse_function_call()?;
        self.expect(&TokenKind::Over)?;
        self.expect(&TokenKind::LParen)?;

        let partition_by = if self.current().kind == TokenKind::PartitionBy {
            self.advance();
            self.parse_comma_until(
                |p| p.parse_expr(0),
                |t| matches!(t, TokenKind::SortBy | TokenKind::Rows | TokenKind::RParen),
            )?
        } else {
            Vec::new()
        };

        let sort_by = if self.current().kind == TokenKind::SortBy {
            self.advance();
            self.parse_comma_until(
                |p| p.parse_sort_col(),
                |t| matches!(t, TokenKind::Rows | TokenKind::RParen),
            )?
        } else {
            Vec::new()
        };

        let frame_spec = if self.current().kind == TokenKind::Rows {
            Some(self.parse_frame_spec()?)
        } else {
            None
        };

        self.expect(&TokenKind::RParen)?;
        self.expect(&TokenKind::As)?;
        let alias = self.expect_ident()?;

        Ok(WindowExpr { func, partition_by, sort_by, frame_spec, alias, span })
    }

    // ── frame spec ─────────────────────────────────────────────────────────

    fn parse_frame_spec(&mut self) -> Result<FrameSpec, ParseError> {
        let span = self.span();
        self.expect(&TokenKind::Rows)?;
        self.expect(&TokenKind::Between)?;
        let start = self.parse_frame_bound()?;
        self.expect(&TokenKind::And)?;
        let end   = self.parse_frame_bound()?;
        Ok(FrameSpec { start, end, span })
    }

    fn parse_frame_bound(&mut self) -> Result<FrameBound, ParseError> {
        match self.current().kind.clone() {
            TokenKind::UnboundedPreceding => { self.advance(); Ok(FrameBound::UnboundedPreceding) }
            TokenKind::UnboundedFollowing => { self.advance(); Ok(FrameBound::UnboundedFollowing) }
            TokenKind::CurrentRow         => { self.advance(); Ok(FrameBound::CurrentRow) }
            TokenKind::Int(n) => {
                self.advance();
                match self.current().kind {
                    TokenKind::Preceding => { self.advance(); Ok(FrameBound::Preceding(n)) }
                    TokenKind::Following => { self.advance(); Ok(FrameBound::Following(n)) }
                    _ => Err(self.err("expected 'preceding' or 'following' after integer in frame bound")),
                }
            }
            _ => Err(self.err(format!(
                "expected frame bound (unbounded_preceding / current_row / n preceding / …), got {}",
                token_kind_name(&self.current().kind)
            ))),
        }
    }

    // ── function call ──────────────────────────────────────────────────────

    fn parse_function_call(&mut self) -> Result<FunctionCall, ParseError> {
        let span = self.span();
        let name = match self.current().kind.clone() {
            TokenKind::Ident(n)  => { self.advance(); n }
            TokenKind::Coalesce  => { self.advance(); "coalesce".to_owned() }
            _ => return Err(self.err(format!(
                "expected function name, got {}",
                token_kind_name(&self.current().kind)
            ))),
        };
        self.expect(&TokenKind::LParen)?;
        let args = self.parse_function_args()?;
        self.expect(&TokenKind::RParen)?;
        Ok(FunctionCall { name, args, span })
    }

    fn parse_function_args(&mut self) -> Result<Vec<FunctionArg>, ParseError> {
        // No arguments
        if self.current().kind == TokenKind::RParen {
            return Ok(Vec::new());
        }
        // Sole `*` argument: count(*)
        if self.current().kind == TokenKind::Star {
            self.advance();
            return Ok(vec![FunctionArg::Star]);
        }
        // General argument list
        let mut args = Vec::new();
        loop {
            if self.current().kind == TokenKind::Star {
                self.advance();
                args.push(FunctionArg::Star);
            } else {
                args.push(FunctionArg::Expr(self.parse_expr(0)?));
            }
            if self.current().kind == TokenKind::Comma {
                self.advance();
            } else {
                break;
            }
        }
        Ok(args)
    }

    // ── helpers ────────────────────────────────────────────────────────────

    fn parse_int(&mut self) -> Result<i64, ParseError> {
        match self.current().kind {
            TokenKind::Int(n) => { let v = n; self.advance(); Ok(v) }
            _ => Err(self.err("expected integer literal")),
        }
    }

    /// Parse a non-empty comma-separated list using parser `f`.
    fn parse_comma_list<T, F>(&mut self, mut f: F) -> Result<Vec<T>, ParseError>
    where
        F: FnMut(&mut Self) -> Result<T, ParseError>,
    {
        let mut items = vec![f(self)?];
        while self.current().kind == TokenKind::Comma {
            self.advance();
            items.push(f(self)?);
        }
        Ok(items)
    }

    /// Parse a (possibly empty) comma-separated list, stopping when `stop`
    /// returns true for the current token kind.
    fn parse_comma_until<T, F, S>(&mut self, mut f: F, stop: S) -> Result<Vec<T>, ParseError>
    where
        F: FnMut(&mut Self) -> Result<T, ParseError>,
        S: Fn(&TokenKind) -> bool,
    {
        let mut items = Vec::new();
        while !stop(&self.current().kind) && !self.at_eof() {
            items.push(f(self)?);
            if self.current().kind == TokenKind::Comma {
                self.advance();
            } else {
                break;
            }
        }
        Ok(items)
    }

    // ── Pratt expression parser ────────────────────────────────────────────

    /// Returns the infix binding power (precedence) for the current token.
    /// Returns 0 if the token is not an infix operator.
    fn infix_bp(kind: &TokenKind) -> u8 {
        match kind {
            TokenKind::Or                  => 1,
            TokenKind::And                 => 2,
            // level 3 is reserved for prefix `not`
            TokenKind::Eq  | TokenKind::NotEq
            | TokenKind::Gt | TokenKind::Lt
            | TokenKind::Gte | TokenKind::Lte => 4,
            TokenKind::Plus  | TokenKind::Minus => 5,
            TokenKind::Star  | TokenKind::Slash => 6,
            _                              => 0,
        }
    }

    /// Parse an expression with minimum binding power `min_bp`.
    ///
    /// Operator precedence (lowest → highest):
    ///   or(1) → and(2) → not(3, prefix) → comparisons(4) → +/-(5) → *///(6)
    ///   → unary -(7, prefix) → atoms
    fn parse_expr(&mut self, min_bp: u8) -> Result<Expr, ParseError> {
        // ── prefix `not` ──────────────────────────────────────────────────
        let mut left = if self.current().kind == TokenKind::Not {
            let span = self.span();
            self.advance();
            // `not` has binding power 3; we parse the operand at bp=3 so
            // that `not a and b` parses as `(not a) and b`.
            let expr = self.parse_expr(3)?;
            Expr::UnaryOp(Box::new(UnaryOpExpr { op: UnaryOp::Not, expr, span }))
        } else {
            self.parse_unary()?
        };

        loop {
            // ── postfix `is null` / `is not null` ─────────────────────────
            if self.current().kind == TokenKind::Is {
                let span = self.span();
                self.advance();
                if self.current().kind == TokenKind::Not {
                    self.advance();
                    self.expect(&TokenKind::Null)?;
                    left = Expr::IsNotNull(Box::new(left), span);
                } else {
                    self.expect(&TokenKind::Null)?;
                    left = Expr::IsNull(Box::new(left), span);
                }
                continue;
            }

            let bp = Self::infix_bp(&self.current().kind);
            if bp <= min_bp {
                break;
            }

            let op_span = self.span();
            let op = match &self.current().kind {
                TokenKind::Or    => BinaryOp::Or,
                TokenKind::And   => BinaryOp::And,
                TokenKind::Eq    => BinaryOp::Eq,
                TokenKind::NotEq => BinaryOp::NotEq,
                TokenKind::Gt    => BinaryOp::Gt,
                TokenKind::Lt    => BinaryOp::Lt,
                TokenKind::Gte   => BinaryOp::Gte,
                TokenKind::Lte   => BinaryOp::Lte,
                TokenKind::Plus  => BinaryOp::Add,
                TokenKind::Minus => BinaryOp::Sub,
                TokenKind::Star  => BinaryOp::Mul,
                TokenKind::Slash => BinaryOp::Div,
                _ => break,
            };
            self.advance();

            // All operators are left-associative: right side uses the same bp.
            let right = self.parse_expr(bp)?;
            left = Expr::BinaryOp(Box::new(BinaryOpExpr { left, op, right, span: op_span }));
        }

        Ok(left)
    }

    /// Handles prefix unary minus; falls through to atom parsing.
    fn parse_unary(&mut self) -> Result<Expr, ParseError> {
        if self.current().kind == TokenKind::Minus {
            let span = self.span();
            self.advance();
            let expr = self.parse_unary()?; // right-associative
            return Ok(Expr::UnaryOp(Box::new(UnaryOpExpr { op: UnaryOp::Neg, expr, span })));
        }
        self.parse_atom()
    }

    /// Parse the smallest expression unit: literals, identifiers, function
    /// calls, qualified column references, and parenthesised expressions.
    fn parse_atom(&mut self) -> Result<Expr, ParseError> {
        let span = self.span();
        match self.current().kind.clone() {
            // ── parenthesised expression ───────────────────────────────────
            TokenKind::LParen => {
                self.advance();
                let inner = self.parse_expr(0)?;
                self.expect(&TokenKind::RParen)?;
                Ok(Expr::Grouped(Box::new(inner), span))
            }

            // ── literals ───────────────────────────────────────────────────
            TokenKind::Int(n)    => { self.advance(); Ok(Expr::Literal(Literal::Int(n, span))) }
            TokenKind::Float(f)  => { self.advance(); Ok(Expr::Literal(Literal::Float(f, span))) }
            TokenKind::String(s) => { self.advance(); Ok(Expr::Literal(Literal::String(s, span))) }
            TokenKind::True      => { self.advance(); Ok(Expr::Literal(Literal::Bool(true, span))) }
            TokenKind::False     => { self.advance(); Ok(Expr::Literal(Literal::Bool(false, span))) }
            TokenKind::Null      => { self.advance(); Ok(Expr::Literal(Literal::Null(span))) }

            // ── coalesce(<args>) — keyword function ────────────────────────
            TokenKind::Coalesce => {
                self.advance();
                self.expect(&TokenKind::LParen)?;
                let args = self.parse_function_args()?;
                self.expect(&TokenKind::RParen)?;
                Ok(Expr::FunctionCall(FunctionCall { name: "coalesce".to_owned(), args, span }))
            }

            // ── identifier: function call, qualified column, or bare column
            TokenKind::Ident(name) => {
                self.advance();
                if self.current().kind == TokenKind::LParen {
                    // function call: name(args)
                    self.advance();
                    let args = self.parse_function_args()?;
                    self.expect(&TokenKind::RParen)?;
                    Ok(Expr::FunctionCall(FunctionCall { name, args, span }))
                } else if self.current().kind == TokenKind::Dot {
                    // qualified column: table.column
                    self.advance();
                    match self.current().kind.clone() {
                        TokenKind::Ident(col) => {
                            self.advance();
                            Ok(Expr::Column(ColumnRef { table: Some(name), column: col, span }))
                        }
                        _ => Err(self.err(format!(
                            "expected column name after '.', got {}",
                            token_kind_name(&self.current().kind)
                        ))),
                    }
                } else {
                    // bare column reference
                    Ok(Expr::Column(ColumnRef { table: None, column: name, span }))
                }
            }

            _ => Err(self.err(format!(
                "expected expression, got {}",
                token_kind_name(&self.current().kind)
            ))),
        }
    }
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/// Lex-and-parse a Resin source string.
///
/// Returns `(program, errors)`.  When `errors` is non-empty the program may
/// be incomplete — call sites can choose to abort or display the errors and
/// continue.
pub fn parse(tokens: Vec<Token>) -> (Program, Vec<ParseError>) {
    Parser::new(tokens).parse_program()
}

// ---------------------------------------------------------------------------
// Formatting helper
// ---------------------------------------------------------------------------

fn token_kind_name(kind: &TokenKind) -> &'static str {
    match kind {
        TokenKind::Use              => "use",
        TokenKind::From             => "from",
        TokenKind::Frame            => "frame",
        TokenKind::Relate           => "relate",
        TokenKind::On               => "on",
        TokenKind::As               => "as",
        TokenKind::Where            => "where",
        TokenKind::Filter           => "filter",
        TokenKind::Select           => "select",
        TokenKind::Map              => "map",
        TokenKind::Aggregate        => "aggregate",
        TokenKind::By               => "by",
        TokenKind::Distinct         => "distinct",
        TokenKind::SortBy           => "sort_by",
        TokenKind::Limit            => "limit",
        TokenKind::Offset           => "offset",
        TokenKind::Join             => "join",
        TokenKind::LeftJoin         => "left_join",
        TokenKind::RightJoin        => "right_join",
        TokenKind::CrossJoin        => "cross_join",
        TokenKind::Window           => "window",
        TokenKind::Over             => "over",
        TokenKind::PartitionBy      => "partition_by",
        TokenKind::Rows             => "rows",
        TokenKind::Between          => "between",
        TokenKind::Union            => "union",
        TokenKind::Intersect        => "intersect",
        TokenKind::Except           => "except",
        TokenKind::And              => "and",
        TokenKind::Or               => "or",
        TokenKind::Not              => "not",
        TokenKind::Is               => "is",
        TokenKind::Null             => "null",
        TokenKind::Asc              => "asc",
        TokenKind::Desc             => "desc",
        TokenKind::True             => "true",
        TokenKind::False            => "false",
        TokenKind::Coalesce         => "coalesce",
        TokenKind::Preceding        => "preceding",
        TokenKind::Following        => "following",
        TokenKind::UnboundedPreceding => "unbounded_preceding",
        TokenKind::UnboundedFollowing => "unbounded_following",
        TokenKind::CurrentRow       => "current_row",
        TokenKind::Arrow            => "'->'",
        TokenKind::Dot              => "'.'",
        TokenKind::Comma            => "','",
        TokenKind::Star             => "'*'",
        TokenKind::LParen           => "'('",
        TokenKind::RParen           => "')'",
        TokenKind::Eq               => "'='",
        TokenKind::NotEq            => "'!='",
        TokenKind::Gt               => "'>'",
        TokenKind::Lt               => "'<'",
        TokenKind::Gte              => "'>='",
        TokenKind::Lte              => "'<='",
        TokenKind::Plus             => "'+'",
        TokenKind::Minus            => "'-'",
        TokenKind::Slash            => "'/'",
        TokenKind::Int(_)           => "integer",
        TokenKind::Float(_)         => "float",
        TokenKind::String(_)        => "string",
        TokenKind::Ident(_)         => "identifier",
        TokenKind::Eof              => "end of input",
    }
}

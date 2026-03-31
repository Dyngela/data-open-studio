use std::fmt;
use thiserror::Error;

// ---------------------------------------------------------------------------
// Span
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, PartialEq)]
pub struct Span {
    pub line: usize,
    pub col: usize,
}

impl Span {
    pub fn new(line: usize, col: usize) -> Self {
        Self { line, col }
    }
}

impl fmt::Display for Span {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}:{}", self.line, self.col)
    }
}

// ---------------------------------------------------------------------------
// Token kinds
// ---------------------------------------------------------------------------

/// All token variants.  Aggregate/window function names (`sum`, `count`, …)
/// remain as `Ident` – they are listed in the spec as "recognized as
/// identifiers".  Only the tokens that need special parser treatment are
/// given their own variant.
#[derive(Debug, Clone, PartialEq)]
pub enum TokenKind {
    // ── statement-level keywords ──────────────────────────────────────────
    Use,
    From,
    Frame,
    Relate,
    On,
    As,
    Where,

    // ── pipe-operator keywords ────────────────────────────────────────────
    Filter,
    Select,
    Map,
    Aggregate,
    By,
    Distinct,
    SortBy,
    Limit,
    Offset,
    Join,
    LeftJoin,
    RightJoin,
    CrossJoin,
    Window,
    Over,
    PartitionBy,
    Rows,
    Between,
    Union,
    Intersect,
    Except,

    // ── expression keywords ───────────────────────────────────────────────
    And,
    Or,
    Not,
    Is,
    Null,
    Asc,
    Desc,
    True,
    False,
    Coalesce,

    // ── window-frame keywords ─────────────────────────────────────────────
    Preceding,
    Following,
    UnboundedPreceding,
    UnboundedFollowing,
    CurrentRow,

    // ── symbols ───────────────────────────────────────────────────────────
    Arrow,  // ->
    Dot,    // .
    Comma,  // ,
    Star,   // *
    LParen, // (
    RParen, // )
    Eq,     // =
    NotEq,  // !=
    Gt,     // >
    Lt,     // <
    Gte,    // >=
    Lte,    // <=
    Plus,   // +
    Minus,  // -
    Slash,  // /

    // ── literals ──────────────────────────────────────────────────────────
    Int(i64),
    Float(f64),
    String(std::string::String),

    // ── identifier (catch-all for names not matched above) ────────────────
    Ident(std::string::String),

    // ── end of input ──────────────────────────────────────────────────────
    Eof,
}

// ---------------------------------------------------------------------------
// Token
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct Token {
    pub kind: TokenKind,
    pub span: Span,
}

impl Token {
    pub fn new(kind: TokenKind, span: Span) -> Self {
        Self { kind, span }
    }
}

// ---------------------------------------------------------------------------
// Lex error
// ---------------------------------------------------------------------------

#[derive(Debug, Error)]
pub enum LexError {
    #[error("unexpected character '{ch}' at {span}")]
    UnexpectedChar { ch: char, span: Span },

    #[error("unterminated string literal starting at {span}")]
    UnterminatedString { span: Span },

    #[error("invalid number '{text}' at {span}")]
    InvalidNumber { text: std::string::String, span: Span },
}

// ---------------------------------------------------------------------------
// Lexer
// ---------------------------------------------------------------------------

pub struct Lexer<'a> {
    _input: &'a str,
    chars: Vec<char>,
    pos: usize,
    line: usize,
    col: usize,
}

impl<'a> Lexer<'a> {
    pub fn new(input: &'a str) -> Self {
        Self {
            _input: input,
            chars: input.chars().collect(),
            pos: 0,
            line: 1,
            col: 1,
        }
    }

    // ── character helpers ──────────────────────────────────────────────────

    fn current(&self) -> Option<char> {
        self.chars.get(self.pos).copied()
    }

    fn peek(&self) -> Option<char> {
        self.chars.get(self.pos + 1).copied()
    }

    /// Advance one character and update line/col tracking.
    fn advance(&mut self) {
        if let Some(c) = self.chars.get(self.pos) {
            self.pos += 1;
            if *c == '\n' {
                self.line += 1;
                self.col = 1;
            } else {
                self.col += 1;
            }
        }
    }

    fn span(&self) -> Span {
        Span::new(self.line, self.col)
    }

    // ── skip whitespace and `--` comments ─────────────────────────────────

    fn skip_trivia(&mut self) {
        loop {
            // whitespace
            while matches!(self.current(), Some(c) if c.is_whitespace()) {
                self.advance();
            }
            // line comment
            if self.current() == Some('-') && self.peek() == Some('-') {
                while !matches!(self.current(), None | Some('\n')) {
                    self.advance();
                }
            } else {
                break;
            }
        }
    }

    // ── sub-scanners ───────────────────────────────────────────────────────

    fn scan_string(&mut self, start: Span) -> Result<Token, LexError> {
        self.advance(); // consume opening `"`
        let mut s = std::string::String::new();
        loop {
            match self.current() {
                None => return Err(LexError::UnterminatedString { span: start }),
                Some('"') => {
                    self.advance();
                    break;
                }
                Some('\\') => {
                    self.advance();
                    match self.current() {
                        Some('n') => { s.push('\n'); self.advance(); }
                        Some('t') => { s.push('\t'); self.advance(); }
                        Some('"') => { s.push('"'); self.advance(); }
                        Some('\\') => { s.push('\\'); self.advance(); }
                        Some(c) => { s.push('\\'); s.push(c); self.advance(); }
                        None => return Err(LexError::UnterminatedString { span: start }),
                    }
                }
                Some(c) => { s.push(c); self.advance(); }
            }
        }
        Ok(Token::new(TokenKind::String(s), start))
    }

    fn scan_number(&mut self, start: Span) -> Result<Token, LexError> {
        let begin = self.pos;
        let mut is_float = false;

        while matches!(self.current(), Some(c) if c.is_ascii_digit()) {
            self.advance();
        }
        // optional fractional part
        if self.current() == Some('.') && matches!(self.peek(), Some(c) if c.is_ascii_digit()) {
            is_float = true;
            self.advance();
            while matches!(self.current(), Some(c) if c.is_ascii_digit()) {
                self.advance();
            }
        }

        let text: std::string::String = self.chars[begin..self.pos].iter().collect();
        if is_float {
            text.parse::<f64>()
                .map(|f| Token::new(TokenKind::Float(f), start.clone()))
                .map_err(|_| LexError::InvalidNumber { text, span: start })
        } else {
            text.parse::<i64>()
                .map(|i| Token::new(TokenKind::Int(i), start.clone()))
                .map_err(|_| LexError::InvalidNumber { text, span: start })
        }
    }

    fn scan_word(&mut self, start: Span) -> Token {
        let begin = self.pos;
        while matches!(self.current(), Some(c) if c.is_alphanumeric() || c == '_') {
            self.advance();
        }
        let word: std::string::String = self.chars[begin..self.pos].iter().collect();
        let kind = keyword_or_ident(&word);
        Token::new(kind, start)
    }

    // ── public entry point ─────────────────────────────────────────────────

    /// Tokenise the full input.  Returns all tokens including a trailing `Eof`.
    /// On error(s) returns the collected errors; successfully scanned tokens
    /// up to the first error are not included (callers should treat the error
    /// list as fatal for now).
    pub fn tokenize(&mut self) -> Result<Vec<Token>, Vec<LexError>> {
        let mut tokens = Vec::new();
        let mut errors = Vec::new();

        loop {
            self.skip_trivia();
            let span = self.span();

            match self.current() {
                None => {
                    tokens.push(Token::new(TokenKind::Eof, span));
                    break;
                }

                Some('"') => match self.scan_string(span) {
                    Ok(t) => tokens.push(t),
                    Err(e) => { errors.push(e); break; }
                },

                Some(c) if c.is_ascii_digit() => match self.scan_number(span) {
                    Ok(t) => tokens.push(t),
                    Err(e) => errors.push(e),
                },

                Some(c) if c.is_alphabetic() || c == '_' => {
                    tokens.push(self.scan_word(span));
                }

                Some('-') => {
                    self.advance();
                    if self.current() == Some('>') {
                        self.advance();
                        tokens.push(Token::new(TokenKind::Arrow, span));
                    } else {
                        tokens.push(Token::new(TokenKind::Minus, span));
                    }
                }

                Some('.') => { self.advance(); tokens.push(Token::new(TokenKind::Dot, span)); }
                Some(',') => { self.advance(); tokens.push(Token::new(TokenKind::Comma, span)); }
                Some('*') => { self.advance(); tokens.push(Token::new(TokenKind::Star, span)); }
                Some('(') => { self.advance(); tokens.push(Token::new(TokenKind::LParen, span)); }
                Some(')') => { self.advance(); tokens.push(Token::new(TokenKind::RParen, span)); }
                Some('+') => { self.advance(); tokens.push(Token::new(TokenKind::Plus, span)); }
                Some('/') => { self.advance(); tokens.push(Token::new(TokenKind::Slash, span)); }

                Some('=') => { self.advance(); tokens.push(Token::new(TokenKind::Eq, span)); }

                Some('!') => {
                    self.advance();
                    if self.current() == Some('=') {
                        self.advance();
                        tokens.push(Token::new(TokenKind::NotEq, span));
                    } else {
                        errors.push(LexError::UnexpectedChar { ch: '!', span });
                    }
                }

                Some('>') => {
                    self.advance();
                    if self.current() == Some('=') {
                        self.advance();
                        tokens.push(Token::new(TokenKind::Gte, span));
                    } else {
                        tokens.push(Token::new(TokenKind::Gt, span));
                    }
                }

                Some('<') => {
                    self.advance();
                    if self.current() == Some('=') {
                        self.advance();
                        tokens.push(Token::new(TokenKind::Lte, span));
                    } else {
                        tokens.push(Token::new(TokenKind::Lt, span));
                    }
                }

                Some(c) => {
                    errors.push(LexError::UnexpectedChar { ch: c, span });
                    self.advance();
                }
            }
        }

        if errors.is_empty() { Ok(tokens) } else { Err(errors) }
    }
}

// ---------------------------------------------------------------------------
// Keyword table
// ---------------------------------------------------------------------------

fn keyword_or_ident(word: &str) -> TokenKind {
    match word {
        "use"                  => TokenKind::Use,
        "from"                 => TokenKind::From,
        "frame"                => TokenKind::Frame,
        "relate"               => TokenKind::Relate,
        "on"                   => TokenKind::On,
        "as"                   => TokenKind::As,
        "where"                => TokenKind::Where,
        "filter"               => TokenKind::Filter,
        "select"               => TokenKind::Select,
        "map"                  => TokenKind::Map,
        "aggregate"            => TokenKind::Aggregate,
        "by"                   => TokenKind::By,
        "distinct"             => TokenKind::Distinct,
        "sort_by"              => TokenKind::SortBy,
        "limit"                => TokenKind::Limit,
        "offset"               => TokenKind::Offset,
        "join"                 => TokenKind::Join,
        "left_join"            => TokenKind::LeftJoin,
        "right_join"           => TokenKind::RightJoin,
        "cross_join"           => TokenKind::CrossJoin,
        "window"               => TokenKind::Window,
        "over"                 => TokenKind::Over,
        "partition_by"         => TokenKind::PartitionBy,
        "rows"                 => TokenKind::Rows,
        "between"              => TokenKind::Between,
        "union"                => TokenKind::Union,
        "intersect"            => TokenKind::Intersect,
        "except"               => TokenKind::Except,
        "and"                  => TokenKind::And,
        "or"                   => TokenKind::Or,
        "not"                  => TokenKind::Not,
        "is"                   => TokenKind::Is,
        "null"                 => TokenKind::Null,
        "asc"                  => TokenKind::Asc,
        "desc"                 => TokenKind::Desc,
        "true"                 => TokenKind::True,
        "false"                => TokenKind::False,
        "coalesce"             => TokenKind::Coalesce,
        "preceding"            => TokenKind::Preceding,
        "following"            => TokenKind::Following,
        "unbounded_preceding"  => TokenKind::UnboundedPreceding,
        "unbounded_following"  => TokenKind::UnboundedFollowing,
        "current_row"          => TokenKind::CurrentRow,
        _                      => TokenKind::Ident(word.to_owned()),
    }
}

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Config ──────────────────────────────────────────────────────────────────

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ─── DB Schema ───────────────────────────────────────────────────────────────

type TableMetadata struct {
	TableName              string         `json:"table_name"`
	TableDescription       sql.NullString `json:"table_description"`
	ColumnName             string         `json:"column_name"`
	DataType               string         `json:"data_type"`
	CharacterMaximumLength sql.NullInt64  `json:"character_maximum_length"`
	NumericPrecision       sql.NullInt64  `json:"numeric_precision"`
	NumericScale           sql.NullInt64  `json:"numeric_scale"`
	IsNullable             string         `json:"is_nullable"`
	ColumnDefault          sql.NullString `json:"column_default"`
	ColumnDescription      sql.NullString `json:"column_description"`
	ConstraintType         sql.NullString `json:"constraint_type"`
	ConstraintName         sql.NullString `json:"constraint_name"`
	ForeignTableName       sql.NullString `json:"foreign_table_name"`
	ForeignColumnName      sql.NullString `json:"foreign_column_name"`
	UpdateRule             sql.NullString `json:"update_rule"`
	DeleteRule             sql.NullString `json:"delete_rule"`
}

func fetchSchema(ctx context.Context, pool *pgxpool.Pool) ([]TableMetadata, error) {
	query := `
        SELECT
            isc.table_name,
            obj_description((isc.table_schema || '.' || isc.table_name)::regclass, 'pg_class') as table_description,
            isc.column_name,
            isc.data_type,
            isc.character_maximum_length,
            isc.numeric_precision,
            isc.numeric_scale,
            isc.is_nullable,
            isc.column_default,
            pg_catalog.col_description((isc.table_schema || '.' || isc.table_name)::regclass, isc.ordinal_position) as column_description,
            tc.constraint_type,
            kcu.constraint_name,
            ccu.table_name AS foreign_table_name,
            ccu.column_name AS foreign_column_name,
            rc.update_rule,
            rc.delete_rule
        FROM information_schema.columns isc
        LEFT JOIN information_schema.key_column_usage kcu
            ON isc.table_schema = kcu.table_schema
            AND isc.table_name = kcu.table_name
            AND isc.column_name = kcu.column_name
        LEFT JOIN information_schema.table_constraints tc
            ON kcu.constraint_name = tc.constraint_name
            AND kcu.table_schema = tc.table_schema
        LEFT JOIN information_schema.constraint_column_usage ccu
            ON tc.constraint_name = ccu.constraint_name
            AND tc.table_schema = ccu.table_schema
            AND tc.constraint_type = 'FOREIGN KEY'
        LEFT JOIN information_schema.referential_constraints rc
            ON tc.constraint_name = rc.constraint_name
            AND tc.table_schema = rc.constraint_schema
        WHERE isc.table_schema = 'public' and isc.table_name in ('client', 'vehicule', 'dossier_commercial')
        ORDER BY isc.table_name, isc.ordinal_position
    `

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var result []TableMetadata
	for rows.Next() {
		var tm TableMetadata
		if err := rows.Scan(
			&tm.TableName, &tm.TableDescription, &tm.ColumnName, &tm.DataType,
			&tm.CharacterMaximumLength, &tm.NumericPrecision, &tm.NumericScale,
			&tm.IsNullable, &tm.ColumnDefault, &tm.ColumnDescription,
			&tm.ConstraintType, &tm.ConstraintName,
			&tm.ForeignTableName, &tm.ForeignColumnName,
			&tm.UpdateRule, &tm.DeleteRule,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		result = append(result, tm)
	}
	return result, rows.Err()
}

func schemaToLLMFormat(schema []TableMetadata) string {
	b, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Sprintf("marshal error: %v", err)
	}
	return string(b)
}

// ─── Ollama ───────────────────────────────────────────────────────────────────

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   map[string]any  `json:"format"`
	Options  map[string]any  `json:"options"`
}

type ollamaRawResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

type guessQueryResponse struct {
	Query       string `json:"query"`
	Explanation string `json:"explanation"`
}

func callOllama(host, model, prompt string) (guessQueryResponse, error) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":       map[string]any{"type": "string"},
			"explanation": map[string]any{"type": "string"},
		},
		"required": []string{"query", "explanation"},
	}

	payload := ollamaRequest{
		Model:    model,
		Messages: []ollamaMessage{{Role: "user", Content: prompt}},
		Stream:   false,
		Format:   schema,
		Options: map[string]any{
			"temperature": 0,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return guessQueryResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, host+"/api/chat", bytes.NewBuffer(data))
	if err != nil {
		return guessQueryResponse{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return guessQueryResponse{}, fmt.Errorf("ollama call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return guessQueryResponse{}, fmt.Errorf("read body: %w", err)
	}

	var raw ollamaRawResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return guessQueryResponse{}, fmt.Errorf("unmarshal raw: %w\nbody: %s", err, body)
	}
	if !raw.Done {
		return guessQueryResponse{}, fmt.Errorf("ollama response not done")
	}

	var result guessQueryResponse
	if err := json.Unmarshal([]byte(raw.Message.Content), &result); err != nil {
		return guessQueryResponse{}, fmt.Errorf("unmarshal result: %w\ncontent: %s", err, raw.Message.Content)
	}
	return result, nil
}

func isSafeSelect(query string) bool {
	q := strings.TrimSpace(strings.ToUpper(query))
	return strings.HasPrefix(q, "SELECT") || strings.HasPrefix(q, "WITH")
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	timer := time.Now()
	defer func() {
		fmt.Printf("\nTotal execution time: %s\n", time.Since(timer))
	}()
	// DB connection to "centric"
	dbHost := getEnv("DB_HOST", "dc-centric-01")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "centric")
	dbSSL := getEnv("DB_SSLMODE", "disable")

	// Ollama
	ollamaHost := getEnv("OLLAMA_HOST", "http://localhost:11434")
	ollamaModel := getEnv("OLLAMA_MODEL", "qwen3-coder:30b")

	// The natural language prompt to test
	userPrompt := getEnv("PROMPT", "Donne-moi les 10 premiers clients selon leur date de dernier OR avec leur historique véhicule respectif. La date de dernier OR se trouve dans le dossier commercial, c'est la date de vente. On peut lié un client et un dossier commercial via l'id client et l'id client cond du dossier commercial")
	fmt.Println("User prompt: " + userPrompt)
	// ── 1. Connect to DB ──
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPass, dbName, dbSSL)

	fmt.Printf("Connecting to %s/%s ...\n", dbHost, dbName)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: db connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: db ping: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("DB connected.")

	// ── 2. Fetch schema ──
	fmt.Println("Fetching schema...")
	schemaCtx, schemaCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer schemaCancel()

	schema, err := fetchSchema(schemaCtx, pool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: fetch schema: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Schema fetched: %d columns across tables.\n", len(schema))

	// ── 3. Build Ollama prompt ──
	llmPrompt := fmt.Sprintf(`Tu es un expert PostgreSQL. Génère une requête SQL SELECT performante basée sur le schéma fourni.

### SCHÉMA DES TABLES (postgres)
%s

### QUESTION UTILISATEUR:
%s

`, schemaToLLMFormat(schema), userPrompt)

	fmt.Println(llmPrompt)

	// ── 4. Call Ollama ──
	fmt.Printf("\nCalling Ollama (%s, model=%s)...\n", ollamaHost, ollamaModel)
	result, err := callOllama(ollamaHost, ollamaModel, llmPrompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: ollama: %v\n", err)
		os.Exit(1)
	}

	// ── 5. Validate & print ──
	if !isSafeSelect(result.Query) {
		fmt.Fprintf(os.Stderr, "WARN: generated query is not a SELECT:\n%s\n", result.Query)
	}

	fmt.Println("\n══════════════════════════════════════")
	fmt.Println("GENERATED QUERY:")
	fmt.Println("══════════════════════════════════════")
	fmt.Println(result.Query)
	fmt.Println("\n══════════════════════════════════════")
	fmt.Println("EXPLANATION:")
	fmt.Println("══════════════════════════════════════")
	fmt.Println(result.Explanation)
}

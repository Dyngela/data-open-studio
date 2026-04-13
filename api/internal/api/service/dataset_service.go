package service

import (
	"api"
	"api/internal/api/handler/response"
	"api/internal/api/models"
	"api/internal/api/repo"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type DatasetService struct {
	datasetRepo *repo.DatasetRepository
	logger      zerolog.Logger
}

func NewDatasetService() *DatasetService {
	return &DatasetService{
		datasetRepo: repo.NewDatasetRepository(),
		logger:      api.Logger,
	}
}

// FindAllForUser retrieves all datasets for a given user
func (s *DatasetService) FindAllForUser(userID uint) ([]models.Dataset, error) {
	datasets, err := s.datasetRepo.FindAllByCreator(userID)
	if err != nil {
		s.logger.Error().Err(err).Uint("userID", userID).Msg("Error getting datasets for user")
		return nil, err
	}
	return datasets, nil
}

// FindByID retrieves a single dataset by ID
func (s *DatasetService) FindByID(id uint) (*models.Dataset, error) {
	dataset, err := s.datasetRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("dataset not found")
		}
		s.logger.Error().Err(err).Uint("datasetId", id).Msg("Error getting dataset")
		return nil, err
	}
	return &dataset, nil
}

// CanUserAccess checks if a user owns the given dataset
func (s *DatasetService) CanUserAccess(datasetID, userID uint) (bool, error) {
	dataset, err := s.datasetRepo.FindByID(datasetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("dataset not found")
		}
		return false, err
	}
	return dataset.CreatorID == userID, nil
}

// Create creates a new dataset and performs an initial schema detection
func (s *DatasetService) Create(dataset models.Dataset) (*models.Dataset, error) {
	if dataset.Query == "" {
		return nil, errors.New("query is required")
	}
	if dataset.MetadataDatabaseID == 0 {
		return nil, errors.New("metadataDatabaseId is required")
	}

	// Attempt schema detection; if it fails, save with draft status
	cfg, err := s.resolveConnection(dataset.MetadataDatabaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database connection: %w", err)
	}

	schema, detectErr := s.detectSchema(dataset.Query, cfg)
	if detectErr != nil {
		s.logger.Warn().Err(detectErr).Msg("Schema detection failed during create; saving as draft")
		dataset.Status = models.DatasetStatusError
		dataset.LastError = detectErr.Error()
	} else {
		dataset.Schema = schema
		dataset.Status = models.DatasetStatusReady
		now := time.Now()
		dataset.LastRefreshedAt = &now
	}

	if err := s.datasetRepo.Create(&dataset); err != nil {
		s.logger.Error().Err(err).Msg("Error creating dataset")
		return nil, err
	}
	return &dataset, nil
}

// Update updates a dataset's name, description, query, or connection
func (s *DatasetService) Update(id uint, patch models.Dataset) (*models.Dataset, error) {
	existing, err := s.datasetRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("dataset not found")
		}
		return nil, err
	}

	if patch.Name != "" {
		existing.Name = patch.Name
	}
	if patch.Description != "" {
		existing.Description = patch.Description
	}

	// If query or connection changed, re-detect schema
	queryChanged := patch.Query != "" && patch.Query != existing.Query
	connChanged := patch.MetadataDatabaseID != 0 && patch.MetadataDatabaseID != existing.MetadataDatabaseID

	if patch.Query != "" {
		existing.Query = patch.Query
	}
	if patch.MetadataDatabaseID != 0 {
		existing.MetadataDatabaseID = patch.MetadataDatabaseID
	}

	if queryChanged || connChanged {
		cfg, err := s.resolveConnection(existing.MetadataDatabaseID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve database connection: %w", err)
		}
		schema, detectErr := s.detectSchema(existing.Query, cfg)
		if detectErr != nil {
			s.logger.Warn().Err(detectErr).Msg("Schema detection failed during update")
			existing.Status = models.DatasetStatusError
			existing.LastError = detectErr.Error()
		} else {
			existing.Schema = schema
			existing.Status = models.DatasetStatusReady
			now := time.Now()
			existing.LastRefreshedAt = &now
			existing.LastError = ""
		}
	}

	if err := s.datasetRepo.Update(&existing); err != nil {
		s.logger.Error().Err(err).Uint("datasetId", id).Msg("Error updating dataset")
		return nil, err
	}
	return &existing, nil
}

// Delete soft-deletes a dataset
func (s *DatasetService) Delete(id uint) error {
	_, err := s.datasetRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("dataset not found")
		}
		return err
	}
	if err := s.datasetRepo.Delete(id); err != nil {
		s.logger.Error().Err(err).Uint("datasetId", id).Msg("Error deleting dataset")
		return err
	}
	return nil
}

// Refresh re-executes the query to detect the latest schema
func (s *DatasetService) Refresh(id uint) (*models.Dataset, error) {
	dataset, err := s.datasetRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("dataset not found")
		}
		return nil, err
	}

	cfg, err := s.resolveConnection(dataset.MetadataDatabaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database connection: %w", err)
	}

	schema, detectErr := s.detectSchema(dataset.Query, cfg)
	if detectErr != nil {
		dataset.Status = models.DatasetStatusError
		dataset.LastError = detectErr.Error()
	} else {
		dataset.Schema = schema
		dataset.Status = models.DatasetStatusReady
		now := time.Now()
		dataset.LastRefreshedAt = &now
		dataset.LastError = ""
	}

	if err := s.datasetRepo.Update(&dataset); err != nil {
		return nil, err
	}
	return &dataset, detectErr
}

// Preview executes the query with a row limit and returns sample data
func (s *DatasetService) Preview(id uint, limit int) (*response.DatasetPreviewResult, error) {
	dataset, err := s.datasetRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("dataset not found")
		}
		return nil, err
	}

	cfg, err := s.resolveConnection(dataset.MetadataDatabaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database connection: %w", err)
	}

	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	r, err := s.executePreview(dataset.Query, cfg, limit)
	if err != nil {
		return nil, err
	}
	rows := r.Rows
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	return &response.DatasetPreviewResult{Columns: r.Columns, Rows: rows, RowCount: r.RowCount}, nil
}

// Query executes the dataset query with optional filters
func (s *DatasetService) Query(id uint, filters []models.QueryFilter, limit int) (*response.DatasetQueryResult, error) {
	dataset, err := s.datasetRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("dataset not found")
		}
		return nil, err
	}

	cfg, err := s.resolveConnection(dataset.MetadataDatabaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database connection: %w", err)
	}

	if limit <= 0 || limit > 10000 {
		limit = 1000
	}

	r, err := s.executeQuery(dataset.Query, cfg, filters, limit)
	if err != nil {
		return nil, err
	}
	rows := r.Rows
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	return &response.DatasetQueryResult{Columns: r.Columns, Rows: rows, RowCount: r.RowCount}, nil
}

// LoadAsFrame loads the dataset into a viz workspace as an in-memory frame.
// It calls the viz API server-side so database credentials never reach the frontend.
// Returns the "loaded" schema object returned by the viz API.
func (s *DatasetService) LoadAsFrame(datasetID uint, workspaceID string, vizAPIURL string) (map[string]interface{}, error) {
	dataset, err := s.datasetRepo.FindByID(datasetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("dataset not found")
		}
		return nil, err
	}

	var meta models.MetadataDatabase
	if err := s.datasetRepo.Db.First(&meta, dataset.MetadataDatabaseID).Error; err != nil {
		return nil, fmt.Errorf("failed to load database metadata: %w", err)
	}

	frameName := toSnakeCase(dataset.Name)

	// Build the source config matching viz PostgresConfig.
	sourcePayload := map[string]interface{}{
		"name":        dataset.Name,
		"source_type": "postgres",
		"config": map[string]interface{}{
			"host":       meta.Host,
			"port":       meta.Port,
			"username":   meta.User,
			"password":   meta.Password,
			"database":   meta.DatabaseName,
			"query":      dataset.Query,
			"frame_name": frameName,
		},
	}

	body, err := json.Marshal(sourcePayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal source payload: %w", err)
	}

	// Step 1: create source in the viz workspace.
	createURL := fmt.Sprintf("%s/workspaces/%s/sources", vizAPIURL, workspaceID)
	resp, err := http.Post(createURL, "application/json", bytes.NewReader(body)) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to create viz source: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("viz API returned %d when creating source: %v", resp.StatusCode, errBody)
	}

	var sourceResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&sourceResp); err != nil {
		return nil, fmt.Errorf("failed to decode source response: %w", err)
	}

	sourceID, ok := sourceResp["id"].(string)
	if !ok || sourceID == "" {
		return nil, errors.New("viz API did not return a source id")
	}

	// Step 2: load the source into an in-memory frame.
	loadURL := fmt.Sprintf("%s/workspaces/%s/sources/%s/load", vizAPIURL, workspaceID, sourceID)
	loadResp, err := http.Post(loadURL, "application/json", nil) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to load viz source: %w", err)
	}
	defer loadResp.Body.Close()

	if loadResp.StatusCode >= 300 {
		var errBody map[string]interface{}
		_ = json.NewDecoder(loadResp.Body).Decode(&errBody)
		return nil, fmt.Errorf("viz API returned %d when loading source: %v", loadResp.StatusCode, errBody)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(loadResp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode load response: %w", err)
	}
	result["frame_name"] = frameName
	return result, nil
}

// ---- internal helpers ----

func (s *DatasetService) resolveConnection(metadataDatabaseID uint) (models.DBConnectionConfig, error) {
	var meta models.MetadataDatabase
	if err := s.datasetRepo.Db.First(&meta, metadataDatabaseID).Error; err != nil {
		return models.DBConnectionConfig{}, fmt.Errorf("failed to load database metadata: %w", err)
	}
	return models.DBConnectionConfig{
		Type:     meta.DbType,
		Host:     meta.Host,
		Port:     meta.Port,
		Database: meta.DatabaseName,
		Username: meta.User,
		Password: meta.Password,
		SSLMode:  meta.SSLMode,
	}, nil
}

func (s *DatasetService) openDB(cfg models.DBConnectionConfig) (*sql.DB, error) {
	db, err := sql.Open(cfg.GetDriverName(), cfg.BuildConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	db.SetConnMaxLifetime(30 * time.Second)
	db.SetMaxOpenConns(2)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return db, nil
}

// detectSchema runs the query with LIMIT 0 and reads column metadata
func (s *DatasetService) detectSchema(query string, cfg models.DBConnectionConfig) (models.DatasetSchema, error) {
	db, err := s.openDB(cfg)
	if err != nil {
		return models.DatasetSchema{}, err
	}
	defer db.Close()

	wrappedQuery := fmt.Sprintf("SELECT * FROM (%s) AS _ds_schema_detect LIMIT 0", query)
	if cfg.Type == models.DBTypeSQLServer {
		wrappedQuery = fmt.Sprintf("SELECT TOP 0 * FROM (%s) AS _ds_schema_detect", query)
	}

	rows, err := db.Query(wrappedQuery)
	if err != nil {
		return models.DatasetSchema{}, fmt.Errorf("query validation failed: %w", err)
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return models.DatasetSchema{}, fmt.Errorf("failed to read column types: %w", err)
	}

	rawNames := make([]string, len(colTypes))
	for i, ct := range colTypes {
		rawNames[i] = ct.Name()
	}
	prefixes := computeColumnPrefixes(rawNames, query, db, cfg.Type)
	dedupedNames := applyColumnPrefixes(rawNames, prefixes)

	var columns []models.DatasetColumn
	for i, ct := range colTypes {
		nullable, _ := ct.Nullable()
		columns = append(columns, models.DatasetColumn{
			Name:     dedupedNames[i],
			DataType: mapSQLTypeToDataType(ct.DatabaseTypeName()),
			Nullable: nullable,
		})
	}
	return models.DatasetSchema{Columns: columns}, nil
}

type rawScanResult struct {
	Columns  []string
	Rows     []map[string]interface{}
	RowCount int
}

func (s *DatasetService) executePreview(query string, cfg models.DBConnectionConfig, limit int) (*rawScanResult, error) {
	db, err := s.openDB(cfg)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	wrappedQuery := fmt.Sprintf("SELECT * FROM (%s) AS _ds_preview LIMIT %d", query, limit)
	if cfg.Type == models.DBTypeSQLServer {
		wrappedQuery = fmt.Sprintf("SELECT TOP %d * FROM (%s) AS _ds_preview", limit, query)
	}

	rows, err := db.Query(wrappedQuery)
	if err != nil {
		return nil, fmt.Errorf("preview query failed: %w", err)
	}
	defer rows.Close()

	rawNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}
	prefixes := computeColumnPrefixes(rawNames, query, db, cfg.Type)
	return scanRows(rows, rawNames, prefixes)
}

func (s *DatasetService) executeQuery(query string, cfg models.DBConnectionConfig, filters []models.QueryFilter, limit int) (*rawScanResult, error) {
	db, err := s.openDB(cfg)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Build WHERE clause from filters
	whereClauses, args := buildWhereClause(filters, cfg.Type)

	var finalQuery string
	if whereClauses != "" {
		finalQuery = fmt.Sprintf("SELECT * FROM (%s) AS _ds_query WHERE %s LIMIT %d", query, whereClauses, limit)
		if cfg.Type == models.DBTypeSQLServer {
			finalQuery = fmt.Sprintf("SELECT TOP %d * FROM (%s) AS _ds_query WHERE %s", limit, query, whereClauses)
		}
	} else {
		finalQuery = fmt.Sprintf("SELECT * FROM (%s) AS _ds_query LIMIT %d", query, limit)
		if cfg.Type == models.DBTypeSQLServer {
			finalQuery = fmt.Sprintf("SELECT TOP %d * FROM (%s) AS _ds_query", limit, query)
		}
	}

	rows, err := db.Query(finalQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	rawNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}
	prefixes := computeColumnPrefixes(rawNames, query, db, cfg.Type)
	return scanRows(rows, rawNames, prefixes)
}

// scanRows reads all rows from a *sql.Rows into a rawScanResult.
// rawNames and prefixes are pre-computed by the caller (who already holds the db).
func scanRows(rows *sql.Rows, rawNames []string, prefixes []string) (*rawScanResult, error) {
	colNames := applyColumnPrefixes(rawNames, prefixes)

	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(colNames))
		valuePtrs := make([]interface{}, len(colNames))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		row := make(map[string]interface{}, len(colNames))
		for i, col := range colNames {
			val := values[i]
			// Convert []byte to string for JSON serialization
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			row[col] = val
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return &rawScanResult{
		Columns:  colNames,
		Rows:     result,
		RowCount: len(result),
	}, nil
}

// buildWhereClause converts filters to a SQL WHERE clause with positional args
func buildWhereClause(filters []models.QueryFilter, dbType models.DBType) (string, []interface{}) {
	if len(filters) == 0 {
		return "", nil
	}

	var clauses []string
	var args []interface{}
	argIndex := 1

	for _, f := range filters {
		colQuoted := quoteIdentifier(f.Column, dbType)
		var placeholder string
		if dbType == models.DBTypeSQLServer {
			placeholder = fmt.Sprintf("@p%d", argIndex)
		} else {
			placeholder = fmt.Sprintf("$%d", argIndex)
		}
		if dbType == models.DBTypeMySQL {
			placeholder = "?"
		}

		switch f.Operator {
		case "eq":
			clauses = append(clauses, fmt.Sprintf("%s = %s", colQuoted, placeholder))
			args = append(args, f.Value)
			argIndex++
		case "neq":
			clauses = append(clauses, fmt.Sprintf("%s != %s", colQuoted, placeholder))
			args = append(args, f.Value)
			argIndex++
		case "gt":
			clauses = append(clauses, fmt.Sprintf("%s > %s", colQuoted, placeholder))
			args = append(args, f.Value)
			argIndex++
		case "lt":
			clauses = append(clauses, fmt.Sprintf("%s < %s", colQuoted, placeholder))
			args = append(args, f.Value)
			argIndex++
		case "gte":
			clauses = append(clauses, fmt.Sprintf("%s >= %s", colQuoted, placeholder))
			args = append(args, f.Value)
			argIndex++
		case "lte":
			clauses = append(clauses, fmt.Sprintf("%s <= %s", colQuoted, placeholder))
			args = append(args, f.Value)
			argIndex++
		case "like":
			clauses = append(clauses, fmt.Sprintf("%s LIKE %s", colQuoted, placeholder))
			args = append(args, f.Value)
			argIndex++
		}
	}

	return strings.Join(clauses, " AND "), args
}

func quoteIdentifier(name string, dbType models.DBType) string {
	switch dbType {
	case models.DBTypeSQLServer:
		return fmt.Sprintf("[%s]", name)
	case models.DBTypeMySQL:
		return fmt.Sprintf("`%s`", name)
	default:
		return fmt.Sprintf(`"%s"`, name)
	}
}

// computeColumnPrefixes returns the table/alias prefix for each column position.
// First tries to parse explicit "alias.col" references from the SELECT list.
// If any duplicate column still lacks a prefix, falls back to resolving table
// column positions via information_schema (handles SELECT * … JOIN queries).
func computeColumnPrefixes(names []string, query string, db *sql.DB, dbType models.DBType) []string {
	prefixes := selectColumnPrefixes(query)

	// Check whether every duplicate already has a prefix from the SELECT list.
	count := make(map[string]int, len(names))
	for _, n := range names {
		count[n]++
	}
	needsResolve := false
	for i, n := range names {
		if count[n] > 1 && (i >= len(prefixes) || prefixes[i] == "") {
			needsResolve = true
			break
		}
	}
	if !needsResolve {
		return prefixes
	}

	// Resolve via information_schema: query each table's columns in order.
	if star := resolveStarPrefixes(query, db, dbType); len(star) > 0 {
		return star
	}
	return prefixes
}

// applyColumnPrefixes renames only duplicate column names using the supplied
// per-position prefixes. Unique columns are never renamed.
func applyColumnPrefixes(names []string, prefixes []string) []string {
	count := make(map[string]int, len(names))
	for _, n := range names {
		count[n]++
	}
	hasDups := false
	for _, c := range count {
		if c > 1 {
			hasDups = true
			break
		}
	}
	if !hasDups {
		return names
	}

	seen := make(map[string]int, len(names))
	result := make([]string, len(names))
	for i, n := range names {
		seen[n]++
		if count[n] <= 1 {
			result[i] = n
			continue
		}
		if i < len(prefixes) && prefixes[i] != "" {
			result[i] = prefixes[i] + "." + n
		} else {
			result[i] = fmt.Sprintf("%s_%d", n, seen[n])
		}
	}
	return result
}

// ---- information_schema resolution for SELECT * queries ----

// tableRef holds a parsed table reference from a FROM/JOIN clause.
type tableRef struct {
	schema  string // e.g. "public"
	name    string // e.g. "order_items"
	display string // prefix to use in output, e.g. "order_items" or "myschema.order_items"
}

// resolveStarPrefixes queries information_schema to determine which table each
// result column belongs to, returning one prefix string per column position.
// Returns nil on any parse or DB error (caller falls back to _N suffix).
func resolveStarPrefixes(query string, db *sql.DB, dbType models.DBType) []string {
	tables := parseFromTables(query)
	if len(tables) == 0 {
		return nil
	}
	var prefixes []string
	for _, tbl := range tables {
		cols, err := getTableColumns(db, tbl, dbType)
		if err != nil || len(cols) == 0 {
			return nil
		}
		for range cols {
			prefixes = append(prefixes, tbl.display)
		}
	}
	return prefixes
}

// parseFromTables extracts ordered table references from the FROM/JOIN clause.
func parseFromTables(query string) []tableRef {
	upper := strings.ToUpper(query)

	// Find the first top-level FROM keyword.
	fromIdx := -1
	depth := 0
	for i := 0; i+4 <= len(upper); i++ {
		switch upper[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 && upper[i:i+4] == "FROM" {
			prev := i == 0 || !isSQLIdentRune(rune(upper[i-1]))
			next := i+4 >= len(upper) || !isSQLIdentRune(rune(upper[i+4]))
			if prev && next {
				fromIdx = i + 4
				break
			}
		}
		if fromIdx >= 0 {
			break
		}
	}
	if fromIdx < 0 {
		return nil
	}

	// Find end of FROM clause at depth 0.
	endKeywords := []string{"WHERE", "GROUP", "HAVING", "ORDER", "LIMIT", "UNION", "INTERSECT", "EXCEPT"}
	endIdx := len(query)
	depth = 0
	for i := fromIdx; i < len(upper); i++ {
		switch upper[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth != 0 {
			continue
		}
		for _, kw := range endKeywords {
			if i+len(kw) <= len(upper) && upper[i:i+len(kw)] == kw {
				prev := i == 0 || !isSQLIdentRune(rune(upper[i-1]))
				next := i+len(kw) >= len(upper) || !isSQLIdentRune(rune(upper[i+len(kw)]))
				if prev && next {
					endIdx = i
					goto endFound
				}
			}
		}
	}
endFound:

	fromClause := query[fromIdx:endIdx]
	upperFrom := upper[fromIdx:endIdx]

	// Split by JOIN keywords (longest patterns first to avoid partial matches).
	joinKeywords := []string{
		"LEFT OUTER JOIN", "RIGHT OUTER JOIN", "FULL OUTER JOIN",
		"INNER JOIN", "LEFT JOIN", "RIGHT JOIN", "FULL JOIN",
		"CROSS JOIN", "JOIN",
	}

	segments := []string{fromClause}
	for _, jk := range joinKeywords {
		var next []string
		jkUpper := strings.ToUpper(jk)
		for _, seg := range segments {
			segUpper := strings.ToUpper(seg)
			for {
				idx := strings.Index(segUpper, jkUpper)
				if idx < 0 {
					next = append(next, seg)
					break
				}
				prev := idx == 0 || !isSQLIdentRune(rune(segUpper[idx-1]))
				after := idx + len(jkUpper)
				nextC := after >= len(segUpper) || !isSQLIdentRune(rune(segUpper[after]))
				if !prev || !nextC {
					next = append(next, seg)
					break
				}
				next = append(next, seg[:idx])
				seg = seg[after:]
				segUpper = segUpper[after:]
			}
		}
		segments = next
		upperFrom = strings.ToUpper(strings.Join(segments, " "))
		_ = upperFrom
	}

	var tables []tableRef
	for _, seg := range segments {
		segUpper := strings.ToUpper(seg)

		// Strip ON / USING clause.
		for _, stopper := range []string{" ON ", " USING "} {
			if idx := strings.Index(segUpper, stopper); idx >= 0 {
				seg = seg[:idx]
				break
			}
		}

		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		// First token is the table spec ("schema.table" or "table").
		fields := strings.Fields(seg)
		if len(fields) == 0 {
			continue
		}
		tableSpec := stripSQLQuotes(fields[0])

		var schema, tableName string
		if dot := strings.LastIndex(tableSpec, "."); dot >= 0 {
			schema = tableSpec[:dot]
			tableName = tableSpec[dot+1:]
		} else {
			tableName = tableSpec
		}

		// Use schema-qualified display only when schema is non-default.
		display := tableName
		if schema != "" && strings.ToLower(schema) != "public" {
			display = schema + "." + tableName
		}

		tables = append(tables, tableRef{schema: schema, name: tableName, display: display})
	}
	return tables
}

// getTableColumns returns the ordered column names of a table via information_schema.
func getTableColumns(db *sql.DB, ref tableRef, dbType models.DBType) ([]string, error) {
	schema := ref.schema
	if schema == "" {
		switch dbType {
		case models.DBTypeSQLServer:
			schema = "dbo"
		case models.DBTypeMySQL:
			// leave empty; use database-level filter below
		default: // PostgreSQL
			schema = "public"
		}
	}

	var query string
	var args []interface{}
	switch dbType {
	case models.DBTypeSQLServer:
		query = `SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = @p1 AND TABLE_NAME = @p2 ORDER BY ORDINAL_POSITION`
		args = []interface{}{schema, ref.name}
	case models.DBTypeMySQL:
		if schema != "" {
			query = `SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION`
			args = []interface{}{schema, ref.name}
		} else {
			query = `SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? ORDER BY ORDINAL_POSITION`
			args = []interface{}{ref.name}
		}
	default: // PostgreSQL
		query = `SELECT column_name FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`
		args = []interface{}{schema, ref.name}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, err
		}
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

// selectColumnPrefixes parses the SELECT list of a SQL query and returns, for
// each positional column, the table/alias prefix extracted from "alias.column"
// expressions. Returns "" for columns with no explicit prefix (expressions,
// literals, SELECT *).
func selectColumnPrefixes(query string) []string {
	cols := splitSelectList(query)
	if len(cols) == 0 {
		return nil
	}
	prefixes := make([]string, len(cols))
	for i, col := range cols {
		col = strings.TrimSpace(col)
		// Strip trailing AS alias
		if j := indexWordCI(col, "AS"); j >= 0 {
			col = strings.TrimSpace(col[:j])
		}
		// Look for table.column: take everything before the last dot
		if dot := strings.LastIndex(col, "."); dot >= 0 {
			prefix := strings.TrimSpace(col[:dot])
			prefix = stripSQLQuotes(prefix)
			if isSQLIdentifier(prefix) {
				prefixes[i] = prefix
			}
		}
	}
	return prefixes
}

// splitSelectList extracts the comma-separated column list between SELECT and
// the top-level FROM keyword. Returns nil when the query cannot be parsed.
func splitSelectList(query string) []string {
	upper := strings.ToUpper(query)

	// Find SELECT keyword
	selectEnd := -1
	for i := 0; i+6 <= len(upper); i++ {
		if upper[i:i+6] == "SELECT" && (i == 0 || !isSQLIdentRune(rune(upper[i-1]))) &&
			(i+6 >= len(upper) || !isSQLIdentRune(rune(upper[i+6]))) {
			selectEnd = i + 6
			break
		}
	}
	if selectEnd < 0 {
		return nil
	}

	// Skip optional DISTINCT / ALL modifier
	trimmed := strings.TrimSpace(upper[selectEnd:])
	for _, mod := range []string{"DISTINCT ", "ALL "} {
		if strings.HasPrefix(trimmed, mod) {
			selectEnd += strings.Index(upper[selectEnd:], upper[selectEnd:selectEnd+strings.Index(trimmed, " ")+1]) + len(mod)
			break
		}
	}

	// Find FROM at depth 0
	fromStart := -1
	depth := 0
	for i := selectEnd; i < len(upper); i++ {
		switch upper[i] {
		case '(':
			depth++
		case ')':
			depth--
		default:
			if depth == 0 && i+4 <= len(upper) && upper[i:i+4] == "FROM" {
				prev := i == 0 || !isSQLIdentRune(rune(upper[i-1]))
				next := i+4 >= len(upper) || !isSQLIdentRune(rune(upper[i+4]))
				if prev && next {
					fromStart = i
				}
			}
		}
		if fromStart >= 0 {
			break
		}
	}

	var list string
	if fromStart >= 0 {
		list = query[selectEnd:fromStart]
	} else {
		list = query[selectEnd:]
	}

	// Split by top-level commas
	var cols []string
	depth = 0
	start := 0
	for i, ch := range list {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				cols = append(cols, list[start:i])
				start = i + 1
			}
		}
	}
	cols = append(cols, list[start:])
	return cols
}

// indexWordCI finds the byte index of word (whole-word, case-insensitive) in s,
// preceded and followed by a non-identifier character. Returns -1 if not found.
func indexWordCI(s, word string) int {
	upper := strings.ToUpper(s)
	wUpper := strings.ToUpper(word)
	for i := 0; i+len(word) <= len(upper); i++ {
		if upper[i:i+len(word)] == wUpper {
			prev := i == 0 || !isSQLIdentRune(rune(upper[i-1]))
			next := i+len(word) >= len(upper) || !isSQLIdentRune(rune(upper[i+len(word)]))
			if prev && next {
				return i
			}
		}
	}
	return -1
}

func isSQLIdentRune(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_'
}

func isSQLIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, ch := range s {
		if !isSQLIdentRune(ch) {
			return false
		}
	}
	return true
}

func stripSQLQuotes(s string) string {
	if len(s) >= 2 {
		f, l := s[0], s[len(s)-1]
		if (f == '"' && l == '"') || (f == '`' && l == '`') || (f == '[' && l == ']') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// toSnakeCase converts a human-readable name to snake_case.
// e.g. "My Orders 2024" → "my_orders_2024"
func toSnakeCase(s string) string {
	var result []rune
	prev := '_'
	for _, ch := range strings.ToLower(s) {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			result = append(result, ch)
			prev = ch
		} else if prev != '_' {
			result = append(result, '_')
			prev = '_'
		}
	}
	name := strings.Trim(string(result), "_")
	if name == "" {
		return "frame"
	}
	return name
}

// mapSQLTypeToDataType converts a database type name to a simplified type string
func mapSQLTypeToDataType(dbTypeName string) string {
	upper := strings.ToUpper(dbTypeName)
	switch {
	case strings.Contains(upper, "INT") || upper == "SERIAL" || upper == "BIGSERIAL" || upper == "SMALLSERIAL":
		return "integer"
	case strings.Contains(upper, "FLOAT") || strings.Contains(upper, "DOUBLE") ||
		strings.Contains(upper, "DECIMAL") || strings.Contains(upper, "NUMERIC") ||
		strings.Contains(upper, "REAL") || strings.Contains(upper, "MONEY"):
		return "float"
	case upper == "DATE":
		return "date"
	case strings.Contains(upper, "TIME") || strings.Contains(upper, "DATETIME"):
		return "datetime"
	case upper == "BOOL" || upper == "BOOLEAN" || upper == "BIT":
		return "boolean"
	default:
		return "string"
	}
}

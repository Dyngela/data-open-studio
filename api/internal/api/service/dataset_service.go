package service

import (
	"api"
	"api/internal/api/handler/response"
	"api/internal/api/models"
	"api/internal/api/repo"
	"database/sql"
	"errors"
	"fmt"
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

	var columns []models.DatasetColumn
	for _, ct := range colTypes {
		nullable, _ := ct.Nullable()
		columns = append(columns, models.DatasetColumn{
			Name:     ct.Name(),
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

	return scanRows(rows)
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

	return scanRows(rows)
}

// scanRows reads all rows from a *sql.Rows into a rawScanResult
func scanRows(rows *sql.Rows) (*rawScanResult, error) {
	colNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}

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

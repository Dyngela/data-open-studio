package service

import (
	"api"
	"api/internal/api/handler/response"
	"api/internal/api/models"
	"api/pkg"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// TestDatabaseConnection tests if a database connection can be established
func TestDatabaseConnection(cfg models.DBConnectionConfig) response.TestConnectionResult {
	db, err := sql.Open(cfg.GetDriverName(), cfg.BuildConnectionString())
	if err != nil {
		return response.TestConnectionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to open connection: %v", err),
		}
	}
	defer db.Close()

	db.SetConnMaxLifetime(10 * time.Second)
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return response.TestConnectionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to ping database: %v", err),
		}
	}

	// Get database version
	var version string
	versionQuery := getVersionQuery(cfg.Type)
	if versionQuery != "" {
		if err := db.QueryRow(versionQuery).Scan(&version); err != nil {
			version = "Unknown"
		}
	}

	return response.TestConnectionResult{
		Success: true,
		Message: "Connection successful",
		Version: version,
	}
}

// IntrospectTables returns a list of tables from the database
func IntrospectTables(metadataID *uint, connection *models.DBConnectionConfig) ([]response.DatabaseTable, error) {
	cfg, err := getConnectionConfig(metadataID, connection)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(cfg.GetDriverName(), cfg.BuildConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(30 * time.Second)
	db.SetMaxOpenConns(1)

	query := getTablesQuery(cfg.Type)
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []response.DatabaseTable
	for rows.Next() {
		var schema, name string
		if err := rows.Scan(&schema, &name); err != nil {
			continue
		}
		tables = append(tables, response.DatabaseTable{
			Schema: schema,
			Name:   name,
		})
	}

	return tables, nil
}

// IntrospectColumns returns columns for a specific table
func IntrospectColumns(metadataID *uint, connection *models.DBConnectionConfig, tableName string) ([]response.DatabaseColumn, error) {
	cfg, err := getConnectionConfig(metadataID, connection)
	if err != nil {
		return nil, err
	}

	// Validate table name
	if !isValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	db, err := sql.Open(cfg.GetDriverName(), cfg.BuildConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(30 * time.Second)
	db.SetMaxOpenConns(1)

	query := getColumnsQuery(cfg.Type, tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []response.DatabaseColumn
	for rows.Next() {
		var name, dataType string
		var isNullable, isPrimary bool
		if err := rows.Scan(&name, &dataType, &isNullable, &isPrimary); err != nil {
			continue
		}
		columns = append(columns, response.DatabaseColumn{
			Name:       name,
			DataType:   dataType,
			IsNullable: isNullable,
			IsPrimary:  isPrimary,
		})
	}

	pkg.PrettyPrint(columns)

	return columns, nil
}

// getConnectionConfig resolves the connection configuration
func getConnectionConfig(metadataID *uint, connection *models.DBConnectionConfig) (models.DBConnectionConfig, error) {
	if connection != nil {
		return *connection, nil
	}

	if metadataID == nil {
		return models.DBConnectionConfig{}, fmt.Errorf("no connection configuration provided")
	}

	// Load from metadata
	var meta models.MetadataDatabase
	if err := api.DB.First(&meta, *metadataID).Error; err != nil {
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

// getVersionQuery returns the version query for a database type
func getVersionQuery(dbType models.DBType) string {
	switch dbType {
	case models.DBTypePostgres:
		return "SELECT version()"
	case models.DBTypeMySQL:
		return "SELECT version()"
	case models.DBTypeSQLServer:
		return "SELECT @@VERSION"
	default:
		return ""
	}
}

// getTablesQuery returns the query to list tables for a database type
func getTablesQuery(dbType models.DBType) string {
	switch dbType {
	case models.DBTypePostgres:
		return `
			SELECT table_schema, table_name
			FROM information_schema.tables
			WHERE table_type = 'BASE TABLE'
			  AND table_schema NOT IN ('pg_catalog', 'information_schema')
			ORDER BY table_schema, table_name`
	case models.DBTypeMySQL:
		return `
			SELECT table_schema, table_name
			FROM information_schema.tables
			WHERE table_type = 'BASE TABLE'
			  AND table_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
			ORDER BY table_schema, table_name`
	case models.DBTypeSQLServer:
		return `
			SELECT SCHEMA_NAME(schema_id) AS table_schema, name AS table_name
			FROM sys.tables
			WHERE is_ms_shipped = 0
			ORDER BY table_schema, table_name`
	default:
		return ""
	}
}

// getColumnsQuery returns the query to list columns for a table
func getColumnsQuery(dbType models.DBType, tableName string) string {
	switch dbType {
	case models.DBTypePostgres:
		return fmt.Sprintf(`
			SELECT
				c.column_name,
				c.data_type,
				c.is_nullable = 'YES' as is_nullable,
				COALESCE(tc.constraint_type = 'PRIMARY KEY', false) as is_primary
			FROM information_schema.columns c
			LEFT JOIN information_schema.key_column_usage kcu
				ON c.table_name = kcu.table_name
				AND c.column_name = kcu.column_name
			LEFT JOIN information_schema.table_constraints tc
				ON kcu.constraint_name = tc.constraint_name
				AND tc.constraint_type = 'PRIMARY KEY'
			WHERE c.table_name = '%s'
			ORDER BY c.ordinal_position`, tableName)
	case models.DBTypeMySQL:
		return fmt.Sprintf(`
			SELECT
				column_name,
				data_type,
				is_nullable = 'YES' as is_nullable,
				column_key = 'PRI' as is_primary
			FROM information_schema.columns
			WHERE table_name = '%s'
			ORDER BY ordinal_position`, tableName)
	case models.DBTypeSQLServer:
		return fmt.Sprintf(`
			SELECT
				c.name AS column_name,
				t.name AS data_type,
				c.is_nullable,
				ISNULL(pk.is_primary_key, 0) AS is_primary
			FROM sys.columns c
			JOIN sys.types t ON c.user_type_id = t.user_type_id
			JOIN sys.tables tbl ON c.object_id = tbl.object_id
			LEFT JOIN (
				SELECT ic.object_id, ic.column_id, 1 as is_primary_key
				FROM sys.index_columns ic
				JOIN sys.indexes i ON ic.object_id = i.object_id AND ic.index_id = i.index_id
				WHERE i.is_primary_key = 1
			) pk ON c.object_id = pk.object_id AND c.column_id = pk.column_id
			WHERE tbl.name = '%s'
			ORDER BY c.column_id`, tableName)
	default:
		return ""
	}
}

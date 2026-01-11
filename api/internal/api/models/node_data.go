package models

import "fmt"

// DBType represents the type of database
type DBType string

const (
	DBTypePostgres  DBType = "postgres"
	DBTypeSQLServer DBType = "sqlserver"
	DBTypeMySQL     DBType = "mysql"
)

// DBConnectionConfig holds database connection details
type DBConnectionConfig struct {
	Type     DBType `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"sslMode"`
	// Extra parameters for specific database types
	Extra map[string]string `json:"extra,omitempty"`
}

// BuildConnectionString creates a connection string based on the database type
func (c DBConnectionConfig) BuildConnectionString() string {
	switch c.Type {
	case DBTypePostgres:
		return c.buildPostgresConnectionString()
	case DBTypeSQLServer:
		return c.buildSQLServerConnectionString()
	case DBTypeMySQL:
		return c.buildMySQLConnectionString()

	default:
		// Default to PostgreSQL
		return c.buildPostgresConnectionString()
	}
}

// buildPostgresConnectionString creates a PostgreSQL DSN
func (c DBConnectionConfig) buildPostgresConnectionString() string {
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.Database, sslMode)
}

// buildSQLServerConnectionString creates a SQL Server DSN
func (c DBConnectionConfig) buildSQLServerConnectionString() string {
	// Format: sqlserver://username:password@host:port?database=dbname
	// Or: server=host;user id=username;password=pwd;port=port;database=dbname
	if c.Port == 0 {
		c.Port = 1433 // Default SQL Server port
	}

	encrypt := "disable"
	if c.SSLMode != "" && c.SSLMode != "disable" {
		encrypt = "true"
	}

	return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s&encrypt=%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, encrypt)
}

// buildMySQLConnectionString creates a MySQL DSN
func (c DBConnectionConfig) buildMySQLConnectionString() string {
	// Format: username:password@tcp(host:port)/database
	if c.Port == 0 {
		c.Port = 3306 // Default MySQL port
	}

	tls := "false"
	if c.SSLMode != "" && c.SSLMode != "disable" {
		tls = "true"
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=%s&parseTime=true",
		c.Username, c.Password, c.Host, c.Port, c.Database, tls)
}

// GetDriverName returns the Go SQL driver name for this database type
func (c DBConnectionConfig) GetDriverName() string {
	switch c.Type {
	case DBTypePostgres:
		return "postgres"
	case DBTypeSQLServer:
		return "sqlserver"
	case DBTypeMySQL:
		return "mysql"
	default:
		return "postgres"
	}
}

// GetImportPath returns the import path for the database driver
func (c DBConnectionConfig) GetImportPath() string {
	switch c.Type {
	case DBTypePostgres:
		return "github.com/lib/pq"
	case DBTypeSQLServer:
		return "github.com/denisenkom/go-mssqldb"
	case DBTypeMySQL:
		return "github.com/go-sql-driver/mysql"
	default:
		return "github.com/lib/pq"
	}
}

// GetConnectionID returns a unique identifier for this connection configuration
// Used to share connection pools across nodes with identical configs
func (c DBConnectionConfig) GetConnectionID() string {
	// Create a unique identifier based on connection parameters
	// Sanitize for use as Go variable name
	id := fmt.Sprintf("%s_%s_%d_%s_%s", c.Type, c.Host, c.Port, c.Database, c.Username)

	// Replace invalid characters with underscores
	sanitized := ""
	for _, ch := range id {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			sanitized += string(ch)
		} else {
			sanitized += "_"
		}
	}
	return sanitized
}

type DBInputConfig struct {
	Query      string             `json:"query"`
	Schema     string             `json:"schema"`
	Table      string             `json:"table"`
	Connection DBConnectionConfig `json:"connection"`
}

type DBOutputConfig struct {
	Table      string             `json:"table"`
	Mode       string             `json:"mode"`
	BatchSize  int                `json:"batchSize"`
	Connection DBConnectionConfig `json:"connection"`
}

type MapConfig struct {
}

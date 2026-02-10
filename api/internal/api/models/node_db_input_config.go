package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type DBInputConfig struct {
	// Query SQL query to execute raw
	Query string `json:"query"`
	// DbSchema Schema name according to dbtype ref: FindDefaultSchema()
	DbSchema string `json:"dbschema"`
	// QueryWithSchema Query with schema prefix according to dbtype
	QueryWithSchema string `json:"queryWithSchema"`

	BatchSize int `json:"batchSize"`

	Connection DBConnectionConfig `json:"connection"`
	// DataModels Give the query result data model with type and col name
	DataModels []DataModel `json:"dataModels"`
}

func (slf *DBInputConfig) Validate() error {
	if slf.Query == "" {
		return errors.New("query is empty")
	}

	if len(slf.DataModels) <= 0 {
		return errors.New("data model is empty")
	}

	return nil
}

func (slf *DBInputConfig) EnforceSchema() {
	if slf.DbSchema == "" {
		slf.DbSchema = slf.findDefaultSchema()
	}

	var gotoSchema string
	switch slf.Connection.Type {
	case DBTypePostgres:
		gotoSchema = fmt.Sprintf("SET search_path TO %s;", slf.DbSchema)
	case DBTypeSQLServer:
		gotoSchema = fmt.Sprintf("/* %s */", slf.DbSchema)
	case DBTypeMySQL:
		gotoSchema = fmt.Sprintf("/* %s */", slf.DbSchema)
	}

	slf.QueryWithSchema = fmt.Sprintf(`%s %s`, gotoSchema, slf.Query)
}

func (slf *DBInputConfig) FillDataModels() error {
	if slf.Query == "" {
		return fmt.Errorf("query is empty, can fill data models")
	}
	if slf.Query[len(slf.Query)-1] == ';' {
		slf.Query = slf.Query[:len(slf.Query)-1]
	}
	slf.EnforceSchema()
	switch slf.Connection.Type {
	case DBTypePostgres:
		conn, err := sql.Open("postgres", slf.Connection.BuildConnectionString())
		if err != nil {
			return err
		}
		defer conn.Close()
		return slf.findPostgresDataModels(conn)
	case DBTypeSQLServer:
		conn, err := sql.Open("sqlserver", slf.Connection.BuildConnectionString())
		if err != nil {
			return err
		}
		defer conn.Close()
		return slf.findSqlServerDataModels(conn)
	default:
		return errors.New("unsupported database type for filling data models")
	}
}

func (slf *DBInputConfig) findDefaultSchema() string {
	if slf.DbSchema != "" {
		return slf.DbSchema
	}
	switch slf.Connection.Type {
	case DBTypePostgres:
		return "public"
	case DBTypeSQLServer:
		return "dbo"
	case DBTypeMySQL:
		return slf.Connection.Database
	default:
		panic("Unsupported DB type")
	}
}

func (slf *DBInputConfig) findPostgresDataModels(conn *sql.DB) error {
	// Execute query with LIMIT 0 to get only metadata
	query := fmt.Sprintf("SELECT * FROM (%s) AS subquery LIMIT 0", slf.Query)

	rows, err := conn.Query(query)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return fmt.Errorf("failed to get column types: %w", err)
	}

	slf.DataModels = make([]DataModel, 0, len(columnTypes))

	for _, col := range columnTypes {
		model := DataModel{
			Name:   col.Name(),
			Type:   col.DatabaseTypeName(),
			GoType: col.ScanType().String(),
		}

		slf.DataModels = append(slf.DataModels, model)
	}

	return nil
}

func (slf *DBInputConfig) findSqlServerDataModels(conn *sql.DB) error {
	query := fmt.Sprintf("SELECT TOP 0 * FROM (%s) AS subquery", slf.Query)

	rows, err := conn.Query(query)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return fmt.Errorf("failed to get column types: %w", err)
	}

	slf.DataModels = make([]DataModel, 0, len(columnTypes))

	for _, col := range columnTypes {
		model := DataModel{
			Name:   col.Name(),
			Type:   col.DatabaseTypeName(),
			GoType: col.ScanType().String(),
		}

		slf.DataModels = append(slf.DataModels, model)
	}

	return nil
}

package models

import (
	"database/sql"
	"errors"

	_ "github.com/lib/pq"
)

type DbOutputMode string

const (
	DbOutputModeInsert   DbOutputMode = "insert"
	DbOutputModeUpdate   DbOutputMode = "update"
	DbOutputModeMerge    DbOutputMode = "merge"
	DbOutputModeDelete   DbOutputMode = "delete"
	DbOutputModeTruncate DbOutputMode = "truncate"
)

type DBOutputConfig struct {
	Table      string             `json:"table"`
	Mode       DbOutputMode       `json:"mode"`
	BatchSize  int                `json:"batchSize"`
	DbSchema   string             `json:"dbschema"`
	Connection DBConnectionConfig `json:"connection"`
	DataModels []DataModel        `json:"dataModel"`
	KeyColumns []string           `json:"keyColumns"`
}

func (slf *DBOutputConfig) FillDataModels() error {
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

func (slf *DBOutputConfig) findPostgresDataModels(conn *sql.DB) error {
	query := `
		SELECT
			column_name,
			data_type,
			is_nullable,
			COALESCE(character_maximum_length, 0),
			COALESCE(numeric_precision, 0),
			COALESCE(numeric_scale, 0)
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position;
	`

	rows, err := conn.Query(query, slf.Table)
	if err != nil {
		return err
	}
	defer rows.Close()

	var models []DataModel

	for rows.Next() {
		var (
			name       string
			dataType   string
			isNullable string
			length     int64
			precision  int64
			scale      int64
		)

		if err := rows.Scan(
			&name,
			&dataType,
			&isNullable,
			&length,
			&precision,
			&scale,
		); err != nil {
			return err
		}

		nullable := isNullable == "YES"

		models = append(models, DataModel{
			Name:      name,
			Type:      dataType,
			GoType:    "",
			Nullable:  nullable,
			Length:    length,
			Precision: precision,
			Scale:     scale,
		})
	}

	slf.DataModels = models
	return rows.Err()
}

func (slf *DBOutputConfig) findSqlServerDataModels(conn *sql.DB) error {
	query := `
		SELECT
			COLUMN_NAME,
			DATA_TYPE,
			IS_NULLABLE,
			COALESCE(CHARACTER_MAXIMUM_LENGTH, 0),
			COALESCE(NUMERIC_PRECISION, 0),
			COALESCE(NUMERIC_SCALE, 0)
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_NAME = @p1
		ORDER BY ORDINAL_POSITION;
	`

	rows, err := conn.Query(query, slf.Table)
	if err != nil {
		return err
	}
	defer rows.Close()

	var models []DataModel

	for rows.Next() {
		var (
			name       string
			dataType   string
			isNullable string
			length     int64
			precision  int64
			scale      int64
		)

		if err := rows.Scan(
			&name,
			&dataType,
			&isNullable,
			&length,
			&precision,
			&scale,
		); err != nil {
			return err
		}

		nullable := isNullable == "YES"

		models = append(models, DataModel{
			Name:      name,
			Type:      dataType,
			GoType:    "",
			Nullable:  nullable,
			Length:    length,
			Precision: precision,
			Scale:     scale,
		})
	}

	slf.DataModels = models
	return rows.Err()
}

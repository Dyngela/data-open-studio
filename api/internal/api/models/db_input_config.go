package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type DBInputConfig struct {
	Query    string `json:"query"`
	DbSchema string `json:"dbschema"`

	Connection DBConnectionConfig `json:"connection"`
	// DataModels Give the query result data model with type and col name
	DataModels []DataModel `json:"dataModel"`
}

func (slf *DBInputConfig) FillDataModels() error {
	if slf.Query == "" {
		return errors.New("Query is empty, can fill data models")
	}

	// Logic to fill DataModels
	// - connect to DB using slf.Connection
	// - analyze fields and table involved in slf.Query
	// - populate slf.DataModels with column names and types

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

	return nil
}

func (slf *DBInputConfig) findPostgresDataModels(conn *sql.DB) error {
	// Exécute la requête avec LIMIT 0 pour ne récupérer que les métadonnées
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

		// Récupère nullable si disponible
		if nullable, ok := col.Nullable(); ok {
			model.Nullable = nullable
		}

		// Récupère la longueur pour les types varchar, text, etc.
		if length, ok := col.Length(); ok {
			model.Length = length
		}

		// Récupère précision et scale pour les types numériques
		if precision, scale, ok := col.DecimalSize(); ok {
			model.Precision = precision
			model.Scale = scale
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

		if nullable, ok := col.Nullable(); ok {
			model.Nullable = nullable
		}

		if length, ok := col.Length(); ok {
			model.Length = length
		}

		if precision, scale, ok := col.DecimalSize(); ok {
			model.Precision = precision
			model.Scale = scale
		}

		slf.DataModels = append(slf.DataModels, model)
	}

	return nil
}

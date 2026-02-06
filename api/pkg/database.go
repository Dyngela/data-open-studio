package pkg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TableMetadata représente les métadonnées d'une colonne de table avec ses relations
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

// FindPostgresSchemaDatabaseSchema récupère le schéma enrichi de la base de données POSTGRES et le retourne en JSON
func FindPostgresSchemaDatabaseSchema(ctx context.Context, pool *pgxpool.Pool) ([]TableMetadata, error) {
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
        WHERE isc.table_schema = 'public'
        ORDER BY isc.table_name, isc.ordinal_position
    `

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var metadata []TableMetadata

	for rows.Next() {
		var tm TableMetadata
		err := rows.Scan(
			&tm.TableName,
			&tm.TableDescription,
			&tm.ColumnName,
			&tm.DataType,
			&tm.CharacterMaximumLength,
			&tm.NumericPrecision,
			&tm.NumericScale,
			&tm.IsNullable,
			&tm.ColumnDefault,
			&tm.ColumnDescription,
			&tm.ConstraintType,
			&tm.ConstraintName,
			&tm.ForeignTableName,
			&tm.ForeignColumnName,
			&tm.UpdateRule,
			&tm.DeleteRule,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		metadata = append(metadata, tm)
	}

	return metadata, nil
}

// FindSQLServerSchemaDatabaseSchema récupère le schéma enrichi de la base de données SQL SERVER et le retourne en JSON
func FindSQLServerSchemaDatabaseSchema(ctx context.Context, db *sql.DB) ([]TableMetadata, error) {
	query := `
  SELECT
    t.name AS table_name,
    CAST(ep_table.value AS NVARCHAR(MAX)) AS table_description,
    c.name AS column_name,
    ty.name AS data_type,
    c.max_length AS character_maximum_length,
    c.precision AS numeric_precision,
    c.scale AS numeric_scale,
    CASE WHEN c.is_nullable = 1 THEN 'YES' ELSE 'NO' END AS is_nullable,
    dc.definition AS column_default,
    CAST(ep_column.value AS NVARCHAR(MAX)) AS column_description,
    CASE
        WHEN pk.constraint_type IS NOT NULL THEN pk.constraint_type
        WHEN fk.constraint_type IS NOT NULL THEN fk.constraint_type
        WHEN uq.constraint_type IS NOT NULL THEN uq.constraint_type
        ELSE NULL
        END AS constraint_type,
    COALESCE(pk.constraint_name, fk.constraint_name, uq.constraint_name) AS constraint_name,
    fk.referenced_table_name AS foreign_table_name,
    fk.referenced_column_name AS foreign_column_name,
    fk.update_rule,
    fk.delete_rule
FROM sys.tables t
         INNER JOIN sys.columns c ON t.object_id = c.object_id
         INNER JOIN sys.types ty ON c.user_type_id = ty.user_type_id
         LEFT JOIN sys.extended_properties ep_table
                   ON ep_table.major_id = t.object_id
                       AND ep_table.minor_id = 0
                       AND ep_table.name = 'MS_Description'
         LEFT JOIN sys.extended_properties ep_column
                   ON ep_column.major_id = c.object_id
                       AND ep_column.minor_id = c.column_id
                       AND ep_column.name = 'MS_Description'
         LEFT JOIN sys.default_constraints dc
                   ON dc.parent_object_id = c.object_id
                       AND dc.parent_column_id = c.column_id
-- Primary Keys
         LEFT JOIN (
    SELECT
        ic.object_id,
        ic.column_id,
        'PRIMARY KEY' AS constraint_type,
        kc.name AS constraint_name
    FROM sys.key_constraints kc
             INNER JOIN sys.index_columns ic
                        ON kc.parent_object_id = ic.object_id
                            AND kc.unique_index_id = ic.index_id
    WHERE kc.type = 'PK'
) pk ON pk.object_id = c.object_id AND pk.column_id = c.column_id
-- Foreign Keys
         LEFT JOIN (
    SELECT
        fkc.parent_object_id,
        fkc.parent_column_id,
        'FOREIGN KEY' AS constraint_type,
        fk.name AS constraint_name,
        OBJECT_NAME(fkc.referenced_object_id) AS referenced_table_name,
        COL_NAME(fkc.referenced_object_id, fkc.referenced_column_id) AS referenced_column_name,
        CASE fk.update_referential_action
            WHEN 0 THEN 'NO ACTION'
            WHEN 1 THEN 'CASCADE'
            WHEN 2 THEN 'SET NULL'
            WHEN 3 THEN 'SET DEFAULT'
            END AS update_rule,
        CASE fk.delete_referential_action
            WHEN 0 THEN 'NO ACTION'
            WHEN 1 THEN 'CASCADE'
            WHEN 2 THEN 'SET NULL'
            WHEN 3 THEN 'SET DEFAULT'
            END AS delete_rule
    FROM sys.foreign_keys fk
             INNER JOIN sys.foreign_key_columns fkc
                        ON fk.object_id = fkc.constraint_object_id
) fk ON fk.parent_object_id = c.object_id AND fk.parent_column_id = c.column_id
-- Unique Constraints
         LEFT JOIN (
    SELECT
        ic.object_id,
        ic.column_id,
        'UNIQUE' AS constraint_type,
        kc.name AS constraint_name
    FROM sys.key_constraints kc
             INNER JOIN sys.index_columns ic
                        ON kc.parent_object_id = ic.object_id
                            AND kc.unique_index_id = ic.index_id
    WHERE kc.type = 'UQ'
) uq ON uq.object_id = c.object_id AND uq.column_id = c.column_id
WHERE t.is_ms_shipped = 0
ORDER BY t.name, c.column_id;
    `

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var metadata []TableMetadata

	for rows.Next() {
		var tm TableMetadata
		err := rows.Scan(
			&tm.TableName,
			&tm.TableDescription,
			&tm.ColumnName,
			&tm.DataType,
			&tm.CharacterMaximumLength,
			&tm.NumericPrecision,
			&tm.NumericScale,
			&tm.IsNullable,
			&tm.ColumnDefault,
			&tm.ColumnDescription,
			&tm.ConstraintType,
			&tm.ConstraintName,
			&tm.ForeignTableName,
			&tm.ForeignColumnName,
			&tm.UpdateRule,
			&tm.DeleteRule,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		metadata = append(metadata, tm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return metadata, nil
}

// TableMetadataToLLMFormat converts a slice of TableMetadata into a JSON string formatted for use with LLMs.
func TableMetadataToLLMFormat(metadata []TableMetadata) string {
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Sprintf("error marshaling: %v", err)
	}
	return string(jsonData)
}

func IsSafeSelect(sql string) bool {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return false
	}
	switch stmt.(type) {
	case *sqlparser.Select:
		return true
	default:
		return false
	}
}

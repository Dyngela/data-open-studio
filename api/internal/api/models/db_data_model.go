package models

import (
	"strings"
	"unicode"
)

// DataModel represents a generic data model structure mainly for database interactions
type DataModel struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	GoType   string `json:"goType"`
	Nullable bool   `json:"nullable"`
	// Length est la longueur maximale pour les types de chaîne.
	Length int64 `json:"length,omitempty"`
	// Precision est le nombre total de chiffres. Applicable pour les types numériques.
	Precision int64 `json:"precision,omitempty"`
	// Scale est le nombre de chiffres après la virgule décimale. Applicable pour les types numériques.
	Scale int64 `json:"scale,omitempty"`
}

// GoFieldName returns a valid Go exported field name from the column name
func (d *DataModel) GoFieldName() string {
	// Convert to PascalCase
	words := strings.FieldsFunc(d.Name, func(r rune) bool {
		return r == '_' || r == ' ' || r == '-'
	})

	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			result.WriteString(string(runes))
		}
	}

	name := result.String()
	if name == "" {
		return "Field"
	}

	// Ensure first character is uppercase
	runes := []rune(name)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// GoFieldType returns the Go type for struct field generation.
// All columns use sql.Null* types to handle NULL safely during Scan/Exec.
func (d *DataModel) GoFieldType() string {
	baseType := d.normalizeGoType()
	return d.sqlNullType(baseType)
}

// sqlNullType returns the sql.Null* wrapper for a base Go type
func (d *DataModel) sqlNullType(baseType string) string {
	switch baseType {
	case "string":
		return "sql.NullString"
	case "int64":
		return "sql.NullInt64"
	case "int32":
		return "sql.NullInt32"
	case "int16":
		return "sql.NullInt16"
	case "int":
		return "sql.NullInt64"
	case "float64":
		return "sql.NullFloat64"
	case "float32":
		return "sql.NullFloat64"
	case "bool":
		return "sql.NullBool"
	case "time.Time":
		return "sql.NullTime"
	case "[]byte":
		return "[]byte" // already nil-able
	default:
		return "*" + baseType
	}
}

// normalizeGoType converts the raw GoType from database driver to a clean Go type
func (d *DataModel) normalizeGoType() string {
	goType := d.GoType
	// Handle sql.Null* types
	switch goType {
	case "sql.NullString":
		return "string"
	case "sql.NullInt64", "sql.NullInt32":
		return "int64"
	case "sql.NullFloat64":
		return "float64"
	case "sql.NullBool":
		return "bool"
	case "sql.NullTime":
		return "time.Time"
	}

	// Handle common type variations
	switch {
	case goType == "interface {}" || goType == "interface{}":
		return "interface{}"

	// 2. Handle JSON types explicitly
	case strings.Contains(strings.ToLower(goType), "json"):
		return "[]byte"

	// 3. Use more specific integer checks
	case goType == "int64", strings.HasPrefix(goType, "int64"):
		return "int64"
	case goType == "int32", strings.HasPrefix(goType, "int32"):
		return "int32"
	// Use exact match or ensure it's not 'interface'
	case goType == "int", (strings.Contains(goType, "int") && !strings.Contains(goType, "interface")):
		return "int"
	case strings.Contains(goType, "float64"):
		return "float64"
	case strings.Contains(goType, "float32"):
		return "float32"
	case strings.Contains(goType, "bool"):
		return "bool"
	case strings.Contains(goType, "time.Time"), strings.Contains(goType, "Time"):
		return "time.Time"
	case strings.Contains(goType, "[]uint8"), goType == "[]byte":
		return "[]byte"
	case strings.Contains(goType, "string"):
		return "string"
	}

	// Default to interface{} for unknown types
	if goType == "" {
		return "interface{}"
	}

	return goType
}

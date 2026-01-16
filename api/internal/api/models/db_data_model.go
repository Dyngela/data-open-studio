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

// GoFieldType returns the Go type for struct field generation
// Returns pointer types for nullable fields
func (d *DataModel) GoFieldType() string {
	baseType := d.normalizeGoType()
	if d.Nullable {
		return "*" + baseType
	}
	return baseType
}

// GoScanType returns the type to use when scanning from database
// Always uses pointer for nullable fields to handle NULL values
func (d *DataModel) GoScanType() string {
	baseType := d.normalizeGoType()
	if d.Nullable {
		return "*" + baseType
	}
	return baseType
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
	case strings.Contains(goType, "int64"):
		return "int64"
	case strings.Contains(goType, "int32"):
		return "int32"
	case strings.Contains(goType, "int"):
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
	case goType == "interface {}":
		return "interface{}"
	}

	// Default to interface{} for unknown types
	if goType == "" {
		return "interface{}"
	}

	return goType
}

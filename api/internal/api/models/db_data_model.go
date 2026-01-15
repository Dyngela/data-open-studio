package models

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

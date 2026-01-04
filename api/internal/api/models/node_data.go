package models

type DBInputConfig struct {
	Query  string `json:"query"`
	Schema string `json:"schema"`
	Table  string `json:"table"`
}

type DBOutputConfig struct {
	Table     string `json:"table"`
	Mode      string `json:"mode"`
	BatchSize int    `json:"batchSize"`
}

type MapConfig struct {
}

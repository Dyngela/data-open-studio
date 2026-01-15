package models

type DBOutputConfig struct {
	Table      string             `json:"table"`
	Mode       string             `json:"mode"`
	BatchSize  int                `json:"batchSize"`
	Connection DBConnectionConfig `json:"connection"`
}

type MapConfig struct {
}

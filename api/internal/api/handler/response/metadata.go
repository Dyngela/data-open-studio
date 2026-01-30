package response

import "api/internal/api/models"

type Metadata struct {
	ID           uint          `json:"id"`
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	User         string        `json:"user"`
	Password     string        `json:"password"`
	DatabaseName string        `json:"databaseName"`
	DbType       models.DBType `json:"databaseType"`
	SSLMode      string        `json:"sslMode"`
	Extra        string        `json:"extra"`
}

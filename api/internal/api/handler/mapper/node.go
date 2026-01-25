package mapper

import (
	"api/internal/api/handler/request"
	"api/internal/api/models"
)

//go:generate go run ../../../../tools/dtomapper -type=NodeMapper
type NodeMapper interface {
	GuessSchemaRequestToDBInputConfig(req request.GuessSchemaRequest) models.DBInputConfig
}


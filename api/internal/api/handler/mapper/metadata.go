package mapper

import (
	"api/internal/api/handler/response"
	"api/internal/api/models"
)

//go:generate go run ../../../../tools/dtomapper -type=MetadataMapper
type MetadataMapper interface {
	ToMetadataResponses(entities []models.MetadataDatabase) []response.Metadata
	ToMetadataResponse(m models.MetadataDatabase) response.Metadata
}

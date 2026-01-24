package mapper

import (
	"api/internal/api/handler/request"
	"api/internal/api/handler/response"
	"api/internal/api/models"
)

//go:generate go run ../../../../tools/dtomapper -type=MetadataMapper
type MetadataMapper interface {
	// Database metadata
	ToMetadataResponses(entities []models.MetadataDatabase) []response.Metadata
	ToMetadataResponse(m models.MetadataDatabase) response.Metadata
	CreateDbMetadata(req request.CreateMetadata) models.MetadataDatabase
	// update
	UpdateDbMetadata(req request.UpdateMetadata, m *models.MetadataDatabase)
	// patch
	PatchDbMetadata(req request.UpdateMetadata) map[string]any

	// SFTP metadata
	ToSftpMetadataResponses(entities []models.MetadataSftp) []response.SftpMetadata
	ToSftpMetadataResponse(m models.MetadataSftp) response.SftpMetadata
	CreateSftpMetadata(req request.CreateSftpMetadata) models.MetadataSftp
	// patch
	PatchSftpMetadata(req request.UpdateSftpMetadata) map[string]any
}

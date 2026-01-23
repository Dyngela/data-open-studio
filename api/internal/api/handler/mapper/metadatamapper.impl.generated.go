package mapper

import (
	"api/internal/api/handler/response"
	"api/internal/api/models"
)

// MetadataMapperImpl implements MetadataMapper
type MetadataMapperImpl struct{}

func (m MetadataMapperImpl) ToMetadataResponses(entities []models.MetadataDatabase) []response.Metadata {
	responses := make([]response.Metadata, len(entities))
	for i, e := range entities {
		responses[i] = m.ToMetadataResponse(e)
	}
	return responses
}

func (m MetadataMapperImpl) ToMetadataResponse(mo models.MetadataDatabase) response.Metadata {
	return response.Metadata{
		ID:           mo.ID,
		Host:         mo.Host,
		Port:         mo.Port,
		User:         mo.User,
		Password:     mo.Password,
		DatabaseName: mo.DatabaseName,
		SSLMode:      mo.SSLMode,
		Extra:        mo.Extra,
	}
}

// NewMetadataMapper creates a new instance of MetadataMapperImpl
func NewMetadataMapper() MetadataMapper {
	return &MetadataMapperImpl{}
}


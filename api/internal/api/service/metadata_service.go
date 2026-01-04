package service

import (
	"api"
	"api/internal/api/handler/mapper"
	"api/internal/api/handler/request"
	"api/internal/api/handler/response"
	"api/internal/api/models"
	"api/internal/api/repo"

	"github.com/rs/zerolog"
)

type MetadataService struct {
	logger         zerolog.Logger
	metadataMapper mapper.MetadataMapper
	metadataRepo   repo.MetadataRepository
}

func NewMetadataService() *MetadataService {
	return &MetadataService{
		logger:         api.Logger,
		metadataMapper: mapper.NewMetadataMapper(),
		metadataRepo:   *repo.NewMetadataRepository(),
	}
}
func (s *MetadataService) FindAll() ([]response.Metadata, error) {
	var entities []models.Metadata
	err := s.metadataRepo.Db.Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return s.metadataMapper.ToMetadataResponses(entities), nil
}

func (s *MetadataService) FindByID(id uint) (*response.Metadata, error) {
	var entity models.Metadata
	err := s.metadataRepo.Db.First(&entity, id).Error
	if err != nil {
		return nil, err
	}
	mapped := s.metadataMapper.ToMetadataResponse(entity)
	return &mapped, nil
}

func (s *MetadataService) Update(update request.UpdateMetadata) error {
	return s.metadataRepo.Db.Model(&models.Metadata{}).Updates(update).Error
}

func (s *MetadataService) Create(metadata models.Metadata) error {
	return s.metadataRepo.Db.Create(&metadata).Error
}

func (s *MetadataService) Delete(id uint) error {
	return s.metadataRepo.Db.Delete(&models.Metadata{}, id).Error
}

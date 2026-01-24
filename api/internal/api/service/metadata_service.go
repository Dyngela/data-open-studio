package service

import (
	"api"
	"api/internal/api/models"
	"api/internal/api/repo"

	"github.com/rs/zerolog"
)

type MetadataService struct {
	logger       zerolog.Logger
	metadataRepo repo.MetadataRepository
}

func NewMetadataService() *MetadataService {
	return &MetadataService{
		logger:       api.Logger,
		metadataRepo: *repo.NewMetadataRepository(),
	}
}

func (s *MetadataService) FindAll() ([]models.MetadataDatabase, error) {
	var entities []models.MetadataDatabase
	err := s.metadataRepo.Db.Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (s *MetadataService) FindByID(id uint) (*models.MetadataDatabase, error) {
	var entity models.MetadataDatabase
	err := s.metadataRepo.Db.First(&entity, id).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (s *MetadataService) Update(id uint, patch map[string]any) (*models.MetadataDatabase, error) {
	if err := s.metadataRepo.Db.Model(&models.MetadataDatabase{}).Where("id = ?", id).Updates(patch).Error; err != nil {
		return nil, err
	}
	return s.FindByID(id)
}

func (s *MetadataService) Create(metadata models.MetadataDatabase) (*models.MetadataDatabase, error) {
	if err := s.metadataRepo.Db.Create(&metadata).Error; err != nil {
		return nil, err
	}
	return &metadata, nil
}

func (s *MetadataService) Delete(id uint) error {
	return s.metadataRepo.Db.Delete(&models.MetadataDatabase{}, id).Error
}

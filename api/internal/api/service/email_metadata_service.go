package service

import (
	"api"
	"api/internal/api/models"
	"api/internal/api/repo"

	"github.com/rs/zerolog"
)

type EmailMetadataService struct {
	logger    zerolog.Logger
	emailRepo repo.EmailMetadataRepository
}

func NewEmailMetadataService() *EmailMetadataService {
	return &EmailMetadataService{
		logger:    api.Logger,
		emailRepo: *repo.NewEmailMetadataRepository(),
	}
}

func (s *EmailMetadataService) FindAll() ([]models.MetadataEmail, error) {
	var entities []models.MetadataEmail
	err := s.emailRepo.Db.Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (s *EmailMetadataService) FindByID(id uint) (*models.MetadataEmail, error) {
	var entity models.MetadataEmail
	err := s.emailRepo.Db.First(&entity, id).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (s *EmailMetadataService) Update(id uint, patch map[string]any) (*models.MetadataEmail, error) {
	if err := s.emailRepo.Db.Model(&models.MetadataEmail{}).Where("id = ?", id).Updates(patch).Error; err != nil {
		return nil, err
	}
	return s.FindByID(id)
}

func (s *EmailMetadataService) Create(metadata models.MetadataEmail) (*models.MetadataEmail, error) {
	if err := s.emailRepo.Db.Create(&metadata).Error; err != nil {
		return nil, err
	}
	return &metadata, nil
}

func (s *EmailMetadataService) Delete(id uint) error {
	return s.emailRepo.Db.Delete(&models.MetadataEmail{}, id).Error
}

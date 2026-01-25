package service

import (
	"api"
	"api/internal/api/models"
	"api/internal/api/repo"

	"github.com/rs/zerolog"
)

type SftpMetadataService struct {
	logger   zerolog.Logger
	sftpRepo repo.SftpMetadataRepository
}

func NewSftpMetadataService() *SftpMetadataService {
	return &SftpMetadataService{
		logger:   api.Logger,
		sftpRepo: *repo.NewSftpMetadataRepository(),
	}
}

func (s *SftpMetadataService) FindAll() ([]models.MetadataSftp, error) {
	var entities []models.MetadataSftp
	err := s.sftpRepo.Db.Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (s *SftpMetadataService) FindByID(id uint) (*models.MetadataSftp, error) {
	var entity models.MetadataSftp
	err := s.sftpRepo.Db.First(&entity, id).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (s *SftpMetadataService) Update(id uint, patch map[string]any) (*models.MetadataSftp, error) {
	if err := s.sftpRepo.Db.Model(&models.MetadataSftp{}).Where("id = ?", id).Updates(patch).Error; err != nil {
		return nil, err
	}
	return s.FindByID(id)
}

func (s *SftpMetadataService) Create(metadata models.MetadataSftp) (*models.MetadataSftp, error) {
	if err := s.sftpRepo.Db.Create(&metadata).Error; err != nil {
		return nil, err
	}
	return &metadata, nil
}

func (s *SftpMetadataService) Delete(id uint) error {
	return s.sftpRepo.Db.Delete(&models.MetadataSftp{}, id).Error
}

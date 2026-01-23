package service

import (
	"api"
	"api/internal/api/handler/response"
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

func (s *SftpMetadataService) FindAll() ([]response.SftpMetadata, error) {
	var entities []models.MetadataSftp
	err := s.sftpRepo.Db.Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return s.toResponses(entities), nil
}

func (s *SftpMetadataService) FindByID(id uint) (*response.SftpMetadata, error) {
	var entity models.MetadataSftp
	err := s.sftpRepo.Db.First(&entity, id).Error
	if err != nil {
		return nil, err
	}
	mapped := s.toResponse(entity)
	return &mapped, nil
}

func (s *SftpMetadataService) Update(id uint, patch map[string]any) (*response.SftpMetadata, error) {
	if err := s.sftpRepo.Db.Model(&models.MetadataSftp{}).Where("id = ?", id).Updates(patch).Error; err != nil {
		return nil, err
	}
	return s.FindByID(id)
}

func (s *SftpMetadataService) Create(metadata models.MetadataSftp) (*response.SftpMetadata, error) {
	if err := s.sftpRepo.Db.Create(&metadata).Error; err != nil {
		return nil, err
	}
	mapped := s.toResponse(metadata)
	return &mapped, nil
}

func (s *SftpMetadataService) Delete(id uint) error {
	return s.sftpRepo.Db.Delete(&models.MetadataSftp{}, id).Error
}

func (s *SftpMetadataService) toResponse(m models.MetadataSftp) response.SftpMetadata {
	return response.SftpMetadata{
		ID:         m.ID,
		Host:       m.Host,
		Port:       m.Port,
		User:       m.User,
		Password:   m.Password,
		PrivateKey: m.PrivateKey,
		BasePath:   m.BasePath,
		Extra:      m.Extra,
	}
}

func (s *SftpMetadataService) toResponses(entities []models.MetadataSftp) []response.SftpMetadata {
	responses := make([]response.SftpMetadata, len(entities))
	for i, e := range entities {
		responses[i] = s.toResponse(e)
	}
	return responses
}

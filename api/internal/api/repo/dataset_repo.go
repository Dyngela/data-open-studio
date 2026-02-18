package repo

import (
	"api"
	"api/internal/api/models"

	"gorm.io/gorm"
)

type DatasetRepository struct {
	Db *gorm.DB
}

func NewDatasetRepository() *DatasetRepository {
	return &DatasetRepository{Db: api.DB}
}

func (r *DatasetRepository) FindByID(id uint) (models.Dataset, error) {
	var dataset models.Dataset
	err := r.Db.First(&dataset, id).Error
	return dataset, err
}

func (r *DatasetRepository) FindAllByCreator(creatorID uint) ([]models.Dataset, error) {
	var datasets []models.Dataset
	err := r.Db.
		Where("creator_id = ?", creatorID).
		Order("created_at DESC").
		Find(&datasets).Error
	return datasets, err
}

func (r *DatasetRepository) Create(dataset *models.Dataset) error {
	return r.Db.Create(dataset).Error
}

func (r *DatasetRepository) Update(dataset *models.Dataset) error {
	return r.Db.Save(dataset).Error
}

func (r *DatasetRepository) Delete(id uint) error {
	return r.Db.Delete(&models.Dataset{}, id).Error
}

package repo

import (
	"api"

	"gorm.io/gorm"
)

type MetadataRepository struct {
	Db *gorm.DB
}

func NewMetadataRepository() *MetadataRepository {
	return &MetadataRepository{
		Db: api.DB,
	}
}

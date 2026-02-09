package repo

import (
	"api"

	"gorm.io/gorm"
)

type EmailMetadataRepository struct {
	Db *gorm.DB
}

func NewEmailMetadataRepository() *EmailMetadataRepository {
	return &EmailMetadataRepository{
		Db: api.DB,
	}
}

package repo

import (
	"api"

	"gorm.io/gorm"
)

type SftpMetadataRepository struct {
	Db *gorm.DB
}

func NewSftpMetadataRepository() *SftpMetadataRepository {
	return &SftpMetadataRepository{
		Db: api.DB,
	}
}

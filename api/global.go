package api

import (
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

var (
	DB     *gorm.DB
	Logger zerolog.Logger
)

package api

import (
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

var (
	DB     *gorm.DB
	Logger zerolog.Logger
	Redis  *redis.Client
)

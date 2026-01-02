package models

import (
	"time"

	"gorm.io/gorm"
)

type AppRole string

const (
	RoleUser  AppRole = "user"
	RoleAdmin AppRole = "admin"
)

type User struct {
	ID           uint           `gorm:"primaryKey"`
	Email        string         `gorm:"uniqueIndex;not null"`
	Password     string         `gorm:"not null;column:password"`
	Prenom       string         `gorm:"not null;column:prenom"`
	Nom          string         `gorm:"not null;column:nom"`
	Role         AppRole        `gorm:"type:varchar(50);default:'user';column:role"`
	Actif        bool           `gorm:"default:true;column:actif"`
	RefreshToken string         `gorm:"type:text;column:refresh_token"`
	CreatedAt    time.Time      `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime;column:updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index;column:deleted_at"`
}

func (User) TableName() string {
	return "users"
}

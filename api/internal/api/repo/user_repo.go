package repo

import (
	"api"
	"api/internal/api/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	Db *gorm.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{Db: api.DB}
}

func (slf *UserRepository) FindByEmail(email string) (models.User, error) {
	var user models.User
	err := slf.Db.Where("email = ?", email).First(&user).Error
	return user, err
}

func (slf *UserRepository) FindByID(id uint) (models.User, error) {
	var user models.User
	err := slf.Db.First(&user, id).Error
	return user, err
}

func (slf *UserRepository) Create(user *models.User) error {
	return slf.Db.Create(user).Error
}

func (slf *UserRepository) Update(user *models.User) error {
	return slf.Db.Save(user).Error
}

func (slf *UserRepository) Delete(id uint) error {
	return slf.Db.Delete(&models.User{}, id).Error
}

func (slf *UserRepository) ExistsByEmail(email string) (bool, error) {
	var count int64
	err := slf.Db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (slf *UserRepository) GetAll() ([]models.User, error) {
	var users []models.User
	err := slf.Db.Find(&users).Error
	return users, err
}

func (slf *UserRepository) SearchByQuery(query string) ([]models.User, error) {
	var users []models.User
	pattern := "%" + query + "%"
	err := slf.Db.Where("email ILIKE ? OR prenom ILIKE ? OR nom ILIKE ?", pattern, pattern, pattern).
		Limit(20).
		Find(&users).Error
	return users, err
}

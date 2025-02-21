package repository

import (
	"diabetify/database"
	"diabetify/internal/models"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (ur *UserRepository) CreateUser(user *models.User) error {
	return database.DB.Create(user).Error
}

func (ur *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := database.DB.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (ur *UserRepository) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	err := database.DB.First(&user, id).Error
	return &user, err
}

func (ur *UserRepository) UpdateUser(user *models.User) error {
	return database.DB.Save(user).Error
}

func (ur *UserRepository) DeleteUser(id uint) error {
	return database.DB.Delete(&models.User{}, id).Error
}

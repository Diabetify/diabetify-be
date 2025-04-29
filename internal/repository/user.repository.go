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
	user.Verified = false
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

func (ur *UserRepository) PatchUser(id uint, data map[string]interface{}) error {
	var user models.User

	if err := database.DB.First(&user, id).Error; err != nil {
		return err
	}

	if err := database.DB.Model(&user).Updates(data).Error; err != nil {
		return err
	}

	return nil
}
func (ur *UserRepository) UpdateUser(user *models.User) error {
	return database.DB.Save(user).Error
}

func (ur *UserRepository) DeleteUser(id uint) error {
	return database.DB.Delete(&models.User{}, id).Error
}

func (ur *UserRepository) SetUserVerified(email string) error {
	return database.DB.Model(&models.User{}).Where("email = ?", email).Update("verified", true).Error
}

func (ur *UserRepository) IsUserVerified(email string) (bool, error) {
	var user models.User
	err := database.DB.Select("verified").Where("email = ?", email).First(&user).Error
	if err != nil {
		return false, err
	}
	return user.Verified, nil
}

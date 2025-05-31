package repository

import (
	"diabetify/internal/models"
	"time"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (ur *UserRepository) CreateUser(user *models.User) error {
	user.Verified = false
	return ur.db.Create(user).Error
}

func (ur *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := ur.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (ur *UserRepository) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	err := ur.db.First(&user, id).Error
	return &user, err
}

func (ur *UserRepository) PatchUser(id uint, data map[string]interface{}) error {
	var user models.User

	if err := ur.db.First(&user, id).Error; err != nil {
		return err
	}

	if err := ur.db.Model(&user).Updates(data).Error; err != nil {
		return err
	}

	return nil
}

func (ur *UserRepository) UpdateUser(user *models.User) error {
	return ur.db.Save(user).Error
}

func (ur *UserRepository) DeleteUser(id uint) error {
	return ur.db.Delete(&models.User{}, id).Error
}

func (ur *UserRepository) SetUserVerified(email string) error {
	return ur.db.Model(&models.User{}).Where("email = ?", email).Update("verified", true).Error
}

func (ur *UserRepository) IsUserVerified(email string) (bool, error) {
	var user models.User
	err := ur.db.Select("verified").Where("email = ?", email).First(&user).Error
	if err != nil {
		return false, err
	}
	return user.Verified, nil
}

func (ur *UserRepository) UpdateLastPredictionTime(userID uint, lastPredictionTime *time.Time) error {
	return ur.db.Model(&models.User{}).Where("id = ?", userID).Update("last_prediction_at", lastPredictionTime).Error
}

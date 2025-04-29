package repository

import (
	"diabetify/internal/models"

	"gorm.io/gorm"
)

type UserProfileRepository interface {
	Create(profile *models.UserProfile) error
	FindByID(id uint) (*models.UserProfile, error)
	FindByUserID(userID uint) (*models.UserProfile, error)
	Update(profile *models.UserProfile) error
	Delete(id uint) error
	DeleteByUserID(userID uint) error
	Patch(userID uint, data map[string]interface{}) error
}

type userProfileRepository struct {
	db *gorm.DB
}

func NewUserProfileRepository(db *gorm.DB) UserProfileRepository {
	return &userProfileRepository{db}
}

func (r *userProfileRepository) Create(profile *models.UserProfile) error {
	return r.db.Create(profile).Error
}

func (r *userProfileRepository) FindByID(id uint) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := r.db.First(&profile, id).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userProfileRepository) FindByUserID(userID uint) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userProfileRepository) Update(profile *models.UserProfile) error {
	return r.db.Save(profile).Error
}

func (r *userProfileRepository) Delete(id uint) error {
	return r.db.Unscoped().Delete(&models.UserProfile{}, id).Error
}

func (r *userProfileRepository) DeleteByUserID(userID uint) error {
	return r.db.Unscoped().Where("user_id = ?", userID).Delete(&models.UserProfile{}).Error
}

func (r *userProfileRepository) Patch(userID uint, data map[string]interface{}) error {
	var profile models.UserProfile
	if err := r.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return err
	}
	return r.db.Model(&profile).Updates(data).Error
}

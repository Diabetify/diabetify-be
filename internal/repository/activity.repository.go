package repository

import (
	"diabetify/internal/models"

	"gorm.io/gorm"
)

type ActivityRepository interface {
	Create(activity *models.Activity) error
	FindAllByUserID(userID uint) ([]models.Activity, error)
	FindByID(id uint) (*models.Activity, error)
	Update(activity *models.Activity) error
	Delete(id uint) error
}

type activityRepository struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) ActivityRepository {
	return &activityRepository{db}
}

func (r *activityRepository) Create(activity *models.Activity) error {
	return r.db.Create(activity).Error
}

func (r *activityRepository) FindAllByUserID(userID uint) ([]models.Activity, error) {
	var activities []models.Activity
	err := r.db.Where("user_id = ?", userID).Find(&activities).Error
	return activities, err
}

func (r *activityRepository) FindByID(id uint) (*models.Activity, error) {
	var activity models.Activity
	err := r.db.First(&activity, id).Error
	if err != nil {
		return nil, err
	}
	return &activity, nil
}

func (r *activityRepository) Update(activity *models.Activity) error {
	return r.db.Save(activity).Error
}

func (r *activityRepository) Delete(id uint) error {
	return r.db.Delete(&models.Activity{}, id).Error
}

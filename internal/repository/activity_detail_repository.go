package repository

import (
	"diabetify/internal/models"

	"gorm.io/gorm"
)

type ActivityDetailRepository interface {
	Create(detail *models.ActivityDetail) error
	FindByActivityID(activityID uint) ([]models.ActivityDetail, error)
	FindByID(id uint) (*models.ActivityDetail, error)
	Update(detail *models.ActivityDetail) error
	Delete(id uint) error
	DeleteByActivityID(activityID uint) error
}

type activityDetailRepository struct {
	db *gorm.DB
}

func NewActivityDetailRepository(db *gorm.DB) ActivityDetailRepository {
	return &activityDetailRepository{db}
}

func (r *activityDetailRepository) Create(detail *models.ActivityDetail) error {
	return r.db.Create(detail).Error
}

func (r *activityDetailRepository) FindByActivityID(activityID uint) ([]models.ActivityDetail, error) {
	var details []models.ActivityDetail
	err := r.db.Where("activity_id = ?", activityID).Find(&details).Error
	return details, err
}

func (r *activityDetailRepository) FindByID(id uint) (*models.ActivityDetail, error) {
	var detail models.ActivityDetail
	err := r.db.First(&detail, id).Error
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

func (r *activityDetailRepository) Update(detail *models.ActivityDetail) error {
	return r.db.Save(detail).Error
}

func (r *activityDetailRepository) Delete(id uint) error {
	return r.db.Delete(&models.ActivityDetail{}, id).Error
}

func (r *activityDetailRepository) DeleteByActivityID(activityID uint) error {
	return r.db.Where("activity_id = ?", activityID).Delete(&models.ActivityDetail{}).Error
}

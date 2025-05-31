package repository

import (
	"diabetify/internal/models"
	"log"
	"time"

	"gorm.io/gorm"
)

type ActivityRepository interface {
	Create(activity *models.Activity) error
	FindAllByUserID(userID uint) ([]models.Activity, error)
	FindByID(id uint) (*models.Activity, error)
	Update(activity *models.Activity) error
	Delete(id uint) error
	FindByUserIDAndActivityDateRange(userID uint, startDate, endDate time.Time) ([]models.Activity, error)
	GetActivitiesByUserIDAndTypeAndDateRange(userID uint, activityType string, startDate, endDate time.Time) ([]models.Activity, error)
	GetActivitiesByUserIDAndType(userID uint, activityType string) ([]models.Activity, error)
	CountUserActivities(userID uint) (int64, error)
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

func (r *activityRepository) FindByUserIDAndActivityDateRange(userID uint, startDate, endDate time.Time) ([]models.Activity, error) {
	var activities []models.Activity

	// Debug the query
	log.Printf("Query: SELECT * FROM activities WHERE user_id = %d AND activity_date BETWEEN '%v' AND '%v'",
		userID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	err := r.db.Where("user_id = ? AND activity_date BETWEEN ? AND ?", userID, startDate, endDate).
		Order("activity_date DESC").
		Find(&activities).Error

	if err != nil {
		log.Printf("Error querying activities: %v", err)
	} else {
		log.Printf("Found %d activities", len(activities))
		for i, a := range activities {
			log.Printf("Activity %d: ID=%d, Type=%s, Date=%v", i+1, a.ID, a.ActivityType, a.ActivityDate)
		}
	}

	return activities, err
}
func (r *activityRepository) GetActivitiesByUserIDAndTypeAndDateRange(userID uint, activityType string, startDate, endDate time.Time) ([]models.Activity, error) {
	var activities []models.Activity

	err := r.db.Where("user_id = ? AND activity_type = ? AND activity_date BETWEEN ? AND ?",
		userID, activityType, startDate, endDate).
		Order("activity_date DESC").
		Find(&activities).Error

	return activities, err
}

func (r *activityRepository) GetActivitiesByUserIDAndType(userID uint, activityType string) ([]models.Activity, error) {
	var activities []models.Activity

	err := r.db.Where("user_id = ? AND activity_type = ?", userID, activityType).
		Order("activity_date DESC").
		Find(&activities).Error
	return activities, err
}

func (r *activityRepository) CountUserActivities(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Activity{}).Where("user_id = ?", userID).Count(&count).Error
	if err != nil {
		log.Printf("Error counting activities for user %d: %v", userID, err)
		return 0, err
	}
	log.Printf("User %d has %d activities", userID, count)
	return count, nil
}

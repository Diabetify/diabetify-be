package repository

import (
	"diabetify/database"
	"diabetify/internal/models"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

type ActivityRepository interface {
	Create(activity *models.Activity) error
	FindAllByUserID(userID uint, limit int) ([]models.Activity, error)
	FindByID(id uint) (*models.Activity, error)
	Update(activity *models.Activity) error
	Delete(id uint) error
	FindByUserIDAndActivityDateRange(userID uint, startDate, endDate time.Time) ([]models.Activity, error)
	GetActivitiesByUserIDAndTypeAndDateRange(userID uint, activityType string, startDate, endDate time.Time) ([]models.Activity, error)
	GetActivitiesByUserIDAndType(userID uint, activityType string) ([]models.Activity, error)
	CountUserActivities(userID uint) (int64, error)
}

type activityRepository struct {
	db        *gorm.DB // Keep for backward compatibility
	useShards bool     // Flag to enable/disable sharding
}

func NewActivityRepository(db *gorm.DB) ActivityRepository {
	return &activityRepository{
		db:        db,
		useShards: db == nil, // If no db provided, use sharding
	}
}

// NewShardedActivityRepository creates an activity repository that uses sharding
func NewShardedActivityRepository() ActivityRepository {
	return &activityRepository{
		db:        nil,
		useShards: true,
	}
}

func (r *activityRepository) Create(activity *models.Activity) error {
	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(activity.UserID), func(db *gorm.DB) error {
			return db.Create(activity).Error
		})
	}

	return r.db.Create(activity).Error
}

func (r *activityRepository) FindAllByUserID(userID uint, limit int) ([]models.Activity, error) {
	if r.useShards {
		var activities []models.Activity
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Raw(`
				SELECT *
				FROM activities 
				WHERE user_id = ? 
				ORDER BY activity_date DESC 
				LIMIT ?
			`, userID, limit).Scan(&activities).Error
		})

		if err != nil {
			log.Printf("Error querying activities for user %d: %v", userID, err)
			return nil, err
		}
		return activities, err
	}

	var activities []models.Activity
	err := r.db.Raw(`
        SELECT *
        FROM activities 
        WHERE user_id = ? 
        ORDER BY activity_date DESC 
        LIMIT ?
    `, userID, limit).Scan(&activities).Error

	if err != nil {
		log.Printf("Error querying activities for user %d: %v", userID, err)
		return nil, err
	}
	return activities, err
}

func (r *activityRepository) FindByID(id uint) (*models.Activity, error) {
	if r.useShards {
		// Since we don't know the user_id from just the activity ID, we need to search all shards
		var foundActivity *models.Activity

		shards := database.Manager.GetAllShards()
		for shardName, db := range shards {
			var activity models.Activity
			err := db.First(&activity, id).Error
			if err == nil {
				foundActivity = &activity
				break
			} else if err != gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("error searching shard %s: %v", shardName, err)
			}
		}

		if foundActivity == nil {
			return nil, gorm.ErrRecordNotFound
		}

		return foundActivity, nil
	}

	var activity models.Activity
	err := r.db.First(&activity, id).Error
	if err != nil {
		return nil, err
	}
	return &activity, nil
}

func (r *activityRepository) Update(activity *models.Activity) error {
	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(activity.UserID), func(db *gorm.DB) error {
			return db.Save(activity).Error
		})
	}

	return r.db.Save(activity).Error
}

func (r *activityRepository) Delete(id uint) error {
	if r.useShards {
		// Since we don't know the user_id, we need to find it first
		activity, err := r.FindByID(id)
		if err != nil {
			return err
		}

		return database.Manager.ExecuteOnUserShard(int(activity.UserID), func(db *gorm.DB) error {
			return db.Delete(&models.Activity{}, id).Error
		})
	}

	return r.db.Delete(&models.Activity{}, id).Error
}

func (r *activityRepository) FindByUserIDAndActivityDateRange(userID uint, startDate, endDate time.Time) ([]models.Activity, error) {
	if r.useShards {
		var activities []models.Activity
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Where("user_id = ? AND activity_date BETWEEN ? AND ?", userID, startDate, endDate).
				Order("activity_date DESC").
				Find(&activities).Error
		})

		if err != nil {
			log.Printf("Error querying activities: %v", err)
		}

		return activities, err
	}

	var activities []models.Activity
	err := r.db.Where("user_id = ? AND activity_date BETWEEN ? AND ?", userID, startDate, endDate).
		Order("activity_date DESC").
		Find(&activities).Error

	if err != nil {
		log.Printf("Error querying activities: %v", err)
	}

	return activities, err
}

func (r *activityRepository) GetActivitiesByUserIDAndTypeAndDateRange(userID uint, activityType string, startDate, endDate time.Time) ([]models.Activity, error) {
	if r.useShards {
		var activities []models.Activity
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Where("user_id = ? AND activity_type = ? AND activity_date BETWEEN ? AND ?",
				userID, activityType, startDate, endDate).
				Order("activity_date DESC").
				Find(&activities).Error
		})
		return activities, err
	}

	var activities []models.Activity
	err := r.db.Where("user_id = ? AND activity_type = ? AND activity_date BETWEEN ? AND ?",
		userID, activityType, startDate, endDate).
		Order("activity_date DESC").
		Find(&activities).Error
	return activities, err
}

func (r *activityRepository) GetActivitiesByUserIDAndType(userID uint, activityType string) ([]models.Activity, error) {
	if r.useShards {
		var activities []models.Activity
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Where("user_id = ? AND activity_type = ?", userID, activityType).
				Order("activity_date DESC").
				Find(&activities).Error
		})
		return activities, err
	}

	var activities []models.Activity
	err := r.db.Where("user_id = ? AND activity_type = ?", userID, activityType).
		Order("activity_date DESC").
		Find(&activities).Error
	return activities, err
}

func (r *activityRepository) CountUserActivities(userID uint) (int64, error) {
	if r.useShards {
		var count int64
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Raw(`
				SELECT COUNT(*) FROM activities
				WHERE user_id = $1`,
				userID).Scan(&count).Error
		})

		if err != nil {
			log.Printf("Error counting activities for user %d: %v", userID, err)
			return 0, err
		}
		return count, nil
	}

	var count int64
	err := r.db.Raw(`
		SELECT COUNT(*) FROM activities
		WHERE user_id = $1`,
		userID).Scan(&count).Error
	if err != nil {
		log.Printf("Error counting activities for user %d: %v", userID, err)
		return 0, err
	}
	return count, nil
}

package repository

import (
	"diabetify/database"
	"diabetify/internal/models"
	"fmt"

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
	db        *gorm.DB // Keep for backward compatibility
	useShards bool     // Flag to enable/disable sharding
}

func NewUserProfileRepository(db *gorm.DB) UserProfileRepository {
	return &userProfileRepository{
		db:        db,
		useShards: db == nil, // If no db provided, use sharding
	}
}

// NewShardedUserProfileRepository creates a user profile repository that uses sharding
func NewShardedUserProfileRepository() UserProfileRepository {
	return &userProfileRepository{
		db:        nil,
		useShards: true,
	}
}

func (r *userProfileRepository) Create(profile *models.UserProfile) error {
	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(profile.UserID), func(db *gorm.DB) error {
			return db.Create(profile).Error
		})
	}

	return r.db.Create(profile).Error
}

func (r *userProfileRepository) FindByID(id uint) (*models.UserProfile, error) {
	if r.useShards {
		// Since we don't know the user_id from just the profile ID, we need to search all shards
		var foundProfile *models.UserProfile

		shards := database.Manager.GetAllShards()
		for shardName, db := range shards {
			var profile models.UserProfile
			err := db.First(&profile, id).Error
			if err == nil {
				foundProfile = &profile
				break
			} else if err != gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("error searching shard %s: %v", shardName, err)
			}
		}

		if foundProfile == nil {
			return nil, gorm.ErrRecordNotFound
		}

		return foundProfile, nil
	}

	var profile models.UserProfile
	err := r.db.First(&profile, id).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userProfileRepository) FindByUserID(userID uint) (*models.UserProfile, error) {
	if r.useShards {
		var profile models.UserProfile
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Where("user_id = ?", userID).First(&profile).Error
		})
		if err != nil {
			return nil, err
		}
		return &profile, nil
	}

	var profile models.UserProfile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userProfileRepository) Update(profile *models.UserProfile) error {
	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(profile.UserID), func(db *gorm.DB) error {
			return db.Save(profile).Error
		})
	}

	return r.db.Save(profile).Error
}

func (r *userProfileRepository) Delete(id uint) error {
	if r.useShards {
		// Since we don't know the user_id, we need to find it first
		profile, err := r.FindByID(id)
		if err != nil {
			return err
		}

		return database.Manager.ExecuteOnUserShard(int(profile.UserID), func(db *gorm.DB) error {
			return db.Unscoped().Delete(&models.UserProfile{}, id).Error
		})
	}

	return r.db.Unscoped().Delete(&models.UserProfile{}, id).Error
}

func (r *userProfileRepository) DeleteByUserID(userID uint) error {
	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Unscoped().Where("user_id = ?", userID).Delete(&models.UserProfile{}).Error
		})
	}

	return r.db.Unscoped().Where("user_id = ?", userID).Delete(&models.UserProfile{}).Error
}

func (r *userProfileRepository) Patch(userID uint, data map[string]interface{}) error {
	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			var profile models.UserProfile
			if err := db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
				return err
			}
			return db.Model(&profile).Updates(data).Error
		})
	}

	var profile models.UserProfile
	if err := r.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return err
	}
	return r.db.Model(&profile).Updates(data).Error
}

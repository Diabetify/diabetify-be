package repository

import (
	"diabetify/database"
	"diabetify/internal/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id uint) (*models.User, error)
	PatchUser(id uint, data map[string]interface{}) error
	UpdateUser(user *models.User) error
	DeleteUser(id uint) error
	SetUserVerified(email string) error
	IsUserVerified(email string) (bool, error)
	UpdateLastPredictionTime(userID uint, lastPredictionTime *time.Time) error
}

type userRepository struct {
	db        *gorm.DB
	useShards bool
}

// NewUserRepository creates a new user repository
// If you pass nil for db, it will use sharding mode
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db:        db,
		useShards: db == nil, // If no db provided, use sharding
	}
}

func NewShardedUserRepository() UserRepository {
	return &userRepository{
		db:        nil,
		useShards: true,
	}
}

func (ur *userRepository) CreateUser(user *models.User) error {
	user.Verified = false

	if ur.useShards {
		// For new users with ID = 0, we need special handling
		if user.ID == 0 {
			// Create in shard1 first to get an ID, then potentially move if needed
			return database.Manager.ExecuteOnUserShard(1, func(db *gorm.DB) error {
				if err := db.Create(user).Error; err != nil {
					return err
				}

				// Check if user should be in a different shard based on generated ID
				expectedShard := database.GetShardNameByUserID(int(user.ID))
				if expectedShard != "shard1" {
					// User should be in a different shard, move it
					userData := *user
					// Delete from shard1
					db.Unscoped().Delete(user)
					// Create in correct shard
					return database.Manager.ExecuteOnUserShard(int(user.ID), func(targetDB *gorm.DB) error {
						return targetDB.Create(&userData).Error
					})
				}
				return nil
			})
		}

		return database.Manager.ExecuteOnUserShard(int(user.ID), func(db *gorm.DB) error {
			return db.Create(user).Error
		})
	}

	return ur.db.Create(user).Error
}

func (ur *userRepository) GetUserByEmail(email string) (*models.User, error) {
	if ur.useShards {
		// Email lookups require searching across all shards
		var foundUser *models.User

		shards := database.Manager.GetAllShards()
		for shardName, db := range shards {
			var user models.User
			err := db.Where("email = ?", email).First(&user).Error
			if err == nil {
				foundUser = &user
				break
			} else if err != gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("error searching shard %s: %v", shardName, err)
			}
		}

		if foundUser == nil {
			return nil, gorm.ErrRecordNotFound
		}

		return foundUser, nil
	}

	var user models.User
	err := ur.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (ur *userRepository) GetUserByID(id uint) (*models.User, error) {
	if ur.useShards {
		var user models.User
		err := database.Manager.ExecuteOnUserShard(int(id), func(db *gorm.DB) error {
			return db.First(&user, id).Error
		})
		if err != nil {
			return nil, err
		}
		return &user, nil
	}

	var user models.User
	err := ur.db.First(&user, id).Error
	return &user, err
}

func (ur *userRepository) PatchUser(id uint, data map[string]interface{}) error {
	if ur.useShards {
		return database.Manager.ExecuteOnUserShard(int(id), func(db *gorm.DB) error {
			var user models.User
			if err := db.First(&user, id).Error; err != nil {
				return err
			}
			return db.Model(&user).Updates(data).Error
		})
	}

	var user models.User
	if err := ur.db.First(&user, id).Error; err != nil {
		return err
	}
	return ur.db.Model(&user).Updates(data).Error
}

func (ur *userRepository) UpdateUser(user *models.User) error {
	if ur.useShards {
		return database.Manager.ExecuteOnUserShard(int(user.ID), func(db *gorm.DB) error {
			return db.Save(user).Error
		})
	}

	return ur.db.Save(user).Error
}

func (ur *userRepository) DeleteUser(id uint) error {
	if ur.useShards {
		return database.Manager.ExecuteOnUserShard(int(id), func(db *gorm.DB) error {
			return db.Delete(&models.User{}, id).Error
		})
	}

	return ur.db.Delete(&models.User{}, id).Error
}

func (ur *userRepository) SetUserVerified(email string) error {
	if ur.useShards {
		// Need to find user first to determine shard
		user, err := ur.GetUserByEmail(email)
		if err != nil {
			return err
		}

		return database.Manager.ExecuteOnUserShard(int(user.ID), func(db *gorm.DB) error {
			return db.Model(&models.User{}).Where("email = ?", email).Update("verified", true).Error
		})
	}

	return ur.db.Model(&models.User{}).Where("email = ?", email).Update("verified", true).Error
}

func (ur *userRepository) IsUserVerified(email string) (bool, error) {
	if ur.useShards {
		user, err := ur.GetUserByEmail(email)
		if err != nil {
			return false, err
		}
		return user.Verified, nil
	}

	var user models.User
	err := ur.db.Select("verified").Where("email = ?", email).First(&user).Error
	if err != nil {
		return false, err
	}
	return user.Verified, nil
}

func (ur *userRepository) UpdateLastPredictionTime(userID uint, lastPredictionTime *time.Time) error {
	if ur.useShards {
		return database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Model(&models.User{}).Where("id = ?", userID).Update("last_prediction_at", lastPredictionTime).Error
		})
	}

	return ur.db.Model(&models.User{}).Where("id = ?", userID).Update("last_prediction_at", lastPredictionTime).Error
}

package repository

import (
	"diabetify/database"
	"diabetify/internal/models"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

type ResetPasswordRepository struct {
	db        *gorm.DB // Keep for backward compatibility
	useShards bool     // Flag to enable/disable sharding
}

func NewResetPasswordRepository(db *gorm.DB) *ResetPasswordRepository {
	return &ResetPasswordRepository{
		db:        db,
		useShards: db == nil, // If no db provided, use sharding
	}
}

// NewShardedResetPasswordRepository creates a reset password repository that uses sharding
func NewShardedResetPasswordRepository() *ResetPasswordRepository {
	return &ResetPasswordRepository{
		db:        nil,
		useShards: true,
	}
}

func (rp *ResetPasswordRepository) CreateResetPassword(resetPassword *models.ResetPassword) error {
	if rp.useShards {
		shardKey := resetPassword.GetShardKey()
		return database.Manager.ExecuteOnUserShard(shardKey, func(db *gorm.DB) error {
			return db.Create(resetPassword).Error
		})
	}

	return rp.db.Create(resetPassword).Error
}

func (rp *ResetPasswordRepository) FindByEmailAndCode(email, code string) (*models.ResetPassword, error) {
	if rp.useShards {
		// Use the same hash logic to find the right shard
		tempResetPassword := &models.ResetPassword{Email: email}
		shardKey := tempResetPassword.GetShardKey()

		var resetPassword models.ResetPassword
		err := database.Manager.ExecuteOnUserShard(shardKey, func(db *gorm.DB) error {
			return db.Where("email = ? AND code = ? AND expires_at > ?", email, code, time.Now()).
				First(&resetPassword).Error
		})

		if err == gorm.ErrRecordNotFound {
			// If not found in expected shard, search all shards as fallback
			var foundResetPassword *models.ResetPassword
			shards := database.Manager.GetAllShards()
			for shardName, db := range shards {
				var tempRP models.ResetPassword
				err := db.Where("email = ? AND code = ? AND expires_at > ?", email, code, time.Now()).
					First(&tempRP).Error
				if err == nil {
					foundResetPassword = &tempRP
					break
				} else if err != gorm.ErrRecordNotFound {
					return nil, fmt.Errorf("error searching shard %s: %v", shardName, err)
				}
			}

			if foundResetPassword == nil {
				return nil, gorm.ErrRecordNotFound
			}

			return foundResetPassword, nil
		}

		if err != nil {
			return nil, err
		}
		return &resetPassword, nil
	}

	var resetPassword models.ResetPassword
	err := rp.db.Where("email = ? AND code = ? AND expires_at > ?", email, code, time.Now()).
		First(&resetPassword).Error
	if err != nil {
		return nil, err
	}
	return &resetPassword, nil
}

func (rp *ResetPasswordRepository) DeleteByEmail(email string) error {
	if rp.useShards {
		// Use the same hash logic to find the right shard
		tempResetPassword := &models.ResetPassword{Email: email}
		shardKey := tempResetPassword.GetShardKey()

		err := database.Manager.ExecuteOnUserShard(shardKey, func(db *gorm.DB) error {
			result := db.Unscoped().Where("email = ?", email).Delete(&models.ResetPassword{})
			if result.Error != nil {
				log.Println("Error deleting reset password record:", result.Error)
			}
			return result.Error
		})

		// If no rows affected in expected shard, try all shards as cleanup
		if err == nil {
			shards := database.Manager.GetAllShards()
			for shardName, db := range shards {
				result := db.Unscoped().Where("email = ?", email).Delete(&models.ResetPassword{})
				if result.Error != nil {
					log.Printf("Error deleting reset password from shard %s: %v", shardName, result.Error)
				}
			}
		}

		return err
	}

	result := rp.db.Unscoped().Where("email = ?", email).Delete(&models.ResetPassword{})
	if result.Error != nil {
		log.Println("Error deleting reset password record:", result.Error)
	}
	return result.Error
}

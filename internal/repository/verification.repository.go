package repository

import (
	"diabetify/database"
	"diabetify/internal/models"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

type VerificationRepository struct {
	db        *gorm.DB // Keep for backward compatibility
	useShards bool     // Flag to enable/disable sharding
}

func NewVerificationRepository(db *gorm.DB) *VerificationRepository {
	return &VerificationRepository{
		db:        db,
		useShards: db == nil, // If no db provided, use sharding
	}
}

// NewShardedVerificationRepository creates a verification repository that uses sharding
func NewShardedVerificationRepository() *VerificationRepository {
	return &VerificationRepository{
		db:        nil,
		useShards: true,
	}
}

func (vr *VerificationRepository) CreateVerification(verification *models.Verification) error {
	if vr.useShards {
		shardKey := verification.GetShardKey()
		return database.Manager.ExecuteOnUserShard(shardKey, func(db *gorm.DB) error {
			return db.Create(verification).Error
		})
	}

	return vr.db.Create(verification).Error
}

func (vr *VerificationRepository) FindByEmail(email string) (*models.Verification, error) {
	if vr.useShards {
		// Since verification might be in any shard, we need to search all shards
		// However, we can optimize by using the same hash logic
		tempVerification := &models.Verification{Email: email}
		shardKey := tempVerification.GetShardKey()

		var verification models.Verification
		err := database.Manager.ExecuteOnUserShard(shardKey, func(db *gorm.DB) error {
			return db.Where("email = ?", email).First(&verification).Error
		})

		if err == gorm.ErrRecordNotFound {
			// If not found in the expected shard, search all shards as fallback
			var foundVerification *models.Verification
			shards := database.Manager.GetAllShards()
			for shardName, db := range shards {
				var tempVer models.Verification
				err := db.Where("email = ?", email).First(&tempVer).Error
				if err == nil {
					foundVerification = &tempVer
					break
				} else if err != gorm.ErrRecordNotFound {
					return nil, fmt.Errorf("error searching shard %s: %v", shardName, err)
				}
			}

			if foundVerification == nil {
				return nil, gorm.ErrRecordNotFound
			}

			return foundVerification, nil
		}

		if err != nil {
			return nil, err
		}
		return &verification, nil
	}

	var verification models.Verification
	err := vr.db.Where("email = ?", email).First(&verification).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

func (vr *VerificationRepository) FindByEmailAndCode(email, code string) (*models.Verification, error) {
	if vr.useShards {
		// Use the same hash logic to find the right shard
		tempVerification := &models.Verification{Email: email}
		shardKey := tempVerification.GetShardKey()

		var verification models.Verification
		err := database.Manager.ExecuteOnUserShard(shardKey, func(db *gorm.DB) error {
			return db.Where("email = ? AND code = ? AND expires_at > ?", email, code, time.Now()).
				First(&verification).Error
		})

		if err == gorm.ErrRecordNotFound {
			// If not found in expected shard, search all shards as fallback
			var foundVerification *models.Verification
			shards := database.Manager.GetAllShards()
			for shardName, db := range shards {
				var tempVer models.Verification
				err := db.Where("email = ? AND code = ? AND expires_at > ?", email, code, time.Now()).
					First(&tempVer).Error
				if err == nil {
					foundVerification = &tempVer
					break
				} else if err != gorm.ErrRecordNotFound {
					return nil, fmt.Errorf("error searching shard %s: %v", shardName, err)
				}
			}

			if foundVerification == nil {
				return nil, gorm.ErrRecordNotFound
			}

			return foundVerification, nil
		}

		if err != nil {
			return nil, err
		}
		return &verification, nil
	}

	var verification models.Verification
	err := vr.db.Where("email = ? AND code = ? AND expires_at > ?", email, code, time.Now()).
		First(&verification).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

func (vr *VerificationRepository) DeleteByEmail(email string) error {
	if vr.useShards {
		// Use the same hash logic to find the right shard
		tempVerification := &models.Verification{Email: email}
		shardKey := tempVerification.GetShardKey()

		err := database.Manager.ExecuteOnUserShard(shardKey, func(db *gorm.DB) error {
			result := db.Unscoped().Where("email = ?", email).Delete(&models.Verification{})
			if result.Error != nil {
				log.Println("Error deleting verification:", result.Error)
			}
			return result.Error
		})

		// If no rows affected in expected shard, try all shards as cleanup
		if err == nil {
			shards := database.Manager.GetAllShards()
			for shardName, db := range shards {
				result := db.Unscoped().Where("email = ?", email).Delete(&models.Verification{})
				if result.Error != nil {
					log.Printf("Error deleting verification from shard %s: %v", shardName, result.Error)
				}
			}
		}

		return err
	}

	result := vr.db.Unscoped().Where("email = ?", email).Delete(&models.Verification{})
	if result.Error != nil {
		log.Println("Error deleting verification:", result.Error)
	}
	return result.Error
}

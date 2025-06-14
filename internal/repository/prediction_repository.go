package repository

import (
	"diabetify/database"
	"diabetify/internal/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type PredictionRepository interface {
	SavePrediction(prediction *models.Prediction) error
	GetPredictionsByUserID(userID uint, limit int) ([]models.Prediction, error)
	GetPredictionsByUserIDAndDateRange(userID uint, startDate, endDate time.Time) ([]models.Prediction, error)
	GetPredictionByID(id uint) (*models.Prediction, error)
	DeletePrediction(id uint) error
	GetPredictionScoreByUserIDAndDateRange(userID uint, startDate, endDate time.Time) ([]PredictionScore, error)
	GetLatestPredictionByUserID(userID uint) (*models.Prediction, error)
	UpdatePrediction(prediction *models.Prediction) error
}

type predictionRepository struct {
	db        *gorm.DB // Keep for backward compatibility
	useShards bool     // Flag to enable/disable sharding
}

func NewPredictionRepository(db *gorm.DB) PredictionRepository {
	return &predictionRepository{
		db:        db,
		useShards: db == nil, // If no db provided, use sharding
	}
}

// NewShardedPredictionRepository creates a prediction repository that uses sharding
func NewShardedPredictionRepository() PredictionRepository {
	return &predictionRepository{
		db:        nil,
		useShards: true,
	}
}

// PredictionScore represents the risk score and creation date of a prediction.
type PredictionScore struct {
	RiskScore float64   `json:"risk_score"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *predictionRepository) SavePrediction(prediction *models.Prediction) error {
	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(prediction.UserID), func(db *gorm.DB) error {
			return db.Create(prediction).Error
		})
	}

	return r.db.Create(prediction).Error
}

func (r *predictionRepository) GetPredictionsByUserID(userID uint, limit int) ([]models.Prediction, error) {
	if r.useShards {
		var predictions []models.Prediction
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&predictions).Error
		})
		return predictions, err
	}

	var predictions []models.Prediction
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&predictions).Error
	return predictions, err
}

func (r *predictionRepository) GetPredictionsByUserIDAndDateRange(userID uint, startDate, endDate time.Time) ([]models.Prediction, error) {
	if r.useShards {
		var predictions []models.Prediction
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Raw(`
				SELECT * 
				FROM (
					SELECT *,
						   ROW_NUMBER() OVER (PARTITION BY DATE(created_at) ORDER BY created_at DESC) as rn
					FROM predictions 
					WHERE user_id = ? AND created_at BETWEEN ? AND ?
				) ranked_predictions
				WHERE rn = 1
				ORDER BY created_at DESC
			`, userID, startDate, endDate).Find(&predictions).Error
		})
		return predictions, err
	}

	var predictions []models.Prediction
	err := r.db.Raw(`
		SELECT * 
		FROM (
			SELECT *,
				   ROW_NUMBER() OVER (PARTITION BY DATE(created_at) ORDER BY created_at DESC) as rn
			FROM predictions 
			WHERE user_id = ? AND created_at BETWEEN ? AND ?
		) ranked_predictions
		WHERE rn = 1
		ORDER BY created_at DESC
	`, userID, startDate, endDate).Find(&predictions).Error
	return predictions, err
}

func (r *predictionRepository) GetPredictionByID(id uint) (*models.Prediction, error) {
	if r.useShards {
		// Since we don't know the user_id from just the prediction ID, we need to search all shards
		var foundPrediction *models.Prediction

		shards := database.Manager.GetAllShards()
		for shardName, db := range shards {
			var prediction models.Prediction
			err := db.First(&prediction, id).Error
			if err == nil {
				foundPrediction = &prediction
				break
			} else if err != gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("error searching shard %s: %v", shardName, err)
			}
		}

		if foundPrediction == nil {
			return nil, gorm.ErrRecordNotFound
		}

		return foundPrediction, nil
	}

	var prediction models.Prediction
	err := r.db.First(&prediction, id).Error
	if err != nil {
		return nil, err
	}
	return &prediction, nil
}

func (r *predictionRepository) DeletePrediction(id uint) error {
	if r.useShards {
		// Since we don't know the user_id, we need to find it first
		prediction, err := r.GetPredictionByID(id)
		if err != nil {
			return err
		}

		return database.Manager.ExecuteOnUserShard(int(prediction.UserID), func(db *gorm.DB) error {
			return db.Delete(&models.Prediction{}, id).Error
		})
	}

	return r.db.Delete(&models.Prediction{}, id).Error
}

func (r *predictionRepository) GetPredictionScoreByUserIDAndDateRange(userID uint, startDate, endDate time.Time) ([]PredictionScore, error) {
	if r.useShards {
		var scores []PredictionScore
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Raw(`
				SELECT risk_score, created_at 
				FROM (
					SELECT risk_score, created_at,
						   ROW_NUMBER() OVER (PARTITION BY DATE(created_at) ORDER BY created_at DESC) as rn
					FROM predictions 
					WHERE user_id = ? AND created_at BETWEEN ? AND ?
				) ranked_predictions
				WHERE rn = 1
				ORDER BY created_at DESC
			`, userID, startDate, endDate).Scan(&scores).Error
		})
		return scores, err
	}

	var scores []PredictionScore
	err := r.db.Raw(`
		SELECT risk_score, created_at 
		FROM (
			SELECT risk_score, created_at,
				   ROW_NUMBER() OVER (PARTITION BY DATE(created_at) ORDER BY created_at DESC) as rn
			FROM predictions 
			WHERE user_id = ? AND created_at BETWEEN ? AND ?
		) ranked_predictions
		WHERE rn = 1
		ORDER BY created_at DESC
	`, userID, startDate, endDate).Scan(&scores).Error
	return scores, err
}

func (r *predictionRepository) GetLatestPredictionByUserID(userID uint) (*models.Prediction, error) {
	if r.useShards {
		var prediction models.Prediction
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Where("user_id = ?", userID).Order("created_at DESC").First(&prediction).Error
		})
		if err != nil {
			return nil, err
		}
		return &prediction, nil
	}

	var prediction models.Prediction
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").First(&prediction).Error
	if err != nil {
		return nil, err
	}
	return &prediction, nil
}

func (r *predictionRepository) UpdatePrediction(prediction *models.Prediction) error {
	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(prediction.UserID), func(db *gorm.DB) error {
			return db.Save(prediction).Error
		})
	}

	return r.db.Save(prediction).Error
}

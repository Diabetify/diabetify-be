package repository

import (
	"diabetify/internal/models"
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
}

type predictionRepository struct {
	db *gorm.DB
}

func NewPredictionRepository(db *gorm.DB) PredictionRepository {
	return &predictionRepository{db}
}

func (r *predictionRepository) SavePrediction(prediction *models.Prediction) error {
	return r.db.Create(prediction).Error
}
func (r *predictionRepository) GetPredictionsByUserID(userID uint, limit int) ([]models.Prediction, error) {
	var predictions []models.Prediction
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&predictions).Error
	return predictions, err
}

func (r *predictionRepository) GetPredictionsByUserIDAndDateRange(userID uint, startDate, endDate time.Time) ([]models.Prediction, error) {
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
	var prediction models.Prediction
	err := r.db.First(&prediction, id).Error
	if err != nil {
		return nil, err
	}
	return &prediction, nil
}

func (r *predictionRepository) DeletePrediction(id uint) error {
	return r.db.Delete(&models.Prediction{}, id).Error
}

// PredictionScore represents the risk score and creation date of a prediction.
type PredictionScore struct {
	RiskScore float64   `json:"risk_score"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *predictionRepository) GetPredictionScoreByUserIDAndDateRange(userID uint, startDate, endDate time.Time) ([]PredictionScore, error) {
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

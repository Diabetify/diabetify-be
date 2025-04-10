package repository

import (
	"diabetify/internal/models"

	"gorm.io/gorm"
)

type RecommendationRepository interface {
	Create(rec *models.Recommendation) error
	FindAllByUserID(userID uint) ([]models.Recommendation, error)
	FindByID(id uint) (*models.Recommendation, error)
	Update(rec *models.Recommendation) error
	Delete(id uint) error
}

type recommendationRepository struct {
	db *gorm.DB
}

func NewRecommendationRepository(db *gorm.DB) RecommendationRepository {
	return &recommendationRepository{db}
}

func (r *recommendationRepository) Create(rec *models.Recommendation) error {
	return r.db.Create(rec).Error
}

func (r *recommendationRepository) FindAllByUserID(userID uint) ([]models.Recommendation, error) {
	var recs []models.Recommendation
	err := r.db.Where("user_id = ?", userID).Find(&recs).Error
	return recs, err
}

func (r *recommendationRepository) FindByID(id uint) (*models.Recommendation, error) {
	var rec models.Recommendation
	err := r.db.First(&rec, id).Error
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *recommendationRepository) Update(rec *models.Recommendation) error {
	return r.db.Save(rec).Error
}

func (r *recommendationRepository) Delete(id uint) error {
	return r.db.Delete(&models.Recommendation{}, id).Error
}

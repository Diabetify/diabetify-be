package repository

import (
	"diabetify/database"
	"diabetify/internal/models"
	"log"
	"time"
)

type VerificationRepository struct{}

func NewVerificationRepository() *VerificationRepository {
	return &VerificationRepository{}
}

func (vr *VerificationRepository) CreateVerification(verification *models.Verification) error {
	return database.DB.Create(verification).Error
}

func (vr *VerificationRepository) FindByEmail(email string) (*models.Verification, error) {
	var verification models.Verification
	err := database.DB.Where("email = ?", email).First(&verification).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

func (vr *VerificationRepository) FindByEmailAndCode(email, code string) (*models.Verification, error) {
	var verification models.Verification
	err := database.DB.Where("email = ? AND code = ? AND expires_at > ?", email, code, time.Now()).
		First(&verification).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

func (vr *VerificationRepository) DeleteByEmail(email string) error {
	result := database.DB.Unscoped().Where("email = ?", email).Delete(&models.Verification{})
	if result.Error != nil {
		log.Println("Error deleting verification:", result.Error)
	}
	return result.Error
}

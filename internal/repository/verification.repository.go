package repository

import (
	"diabetify/internal/models"
	"log"
	"time"

	"gorm.io/gorm"
)

type VerificationRepository struct {
	db *gorm.DB
}

func NewVerificationRepository(db *gorm.DB) *VerificationRepository {
	return &VerificationRepository{
		db: db,
	}
}

func (vr *VerificationRepository) CreateVerification(verification *models.Verification) error {
	return vr.db.Create(verification).Error
}

func (vr *VerificationRepository) FindByEmail(email string) (*models.Verification, error) {
	var verification models.Verification
	err := vr.db.Where("email = ?", email).First(&verification).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

func (vr *VerificationRepository) FindByEmailAndCode(email, code string) (*models.Verification, error) {
	var verification models.Verification
	err := vr.db.Where("email = ? AND code = ? AND expires_at > ?", email, code, time.Now()).
		First(&verification).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

func (vr *VerificationRepository) DeleteByEmail(email string) error {
	result := vr.db.Unscoped().Where("email = ?", email).Delete(&models.Verification{})
	if result.Error != nil {
		log.Println("Error deleting verification:", result.Error)
	}
	return result.Error
}

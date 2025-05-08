package repository

import (
	"diabetify/internal/models"
	"log"
	"time"

	"gorm.io/gorm"
)

type ResetPasswordRepository struct {
	db *gorm.DB
}

func NewResetPasswordRepository(db *gorm.DB) *ResetPasswordRepository {
	return &ResetPasswordRepository{
		db: db,
	}
}

func (rp *ResetPasswordRepository) CreateResetPassword(resetPassword *models.ResetPassword) error {
	return rp.db.Create(resetPassword).Error
}

func (rp *ResetPasswordRepository) FindByEmailAndCode(email, code string) (*models.ResetPassword, error) {
	var resetPassword models.ResetPassword
	err := rp.db.Where("email = ? AND code = ? AND expires_at > ?", email, code, time.Now()).
		First(&resetPassword).Error
	if err != nil {
		return nil, err
	}
	return &resetPassword, nil
}

func (rp *ResetPasswordRepository) DeleteByEmail(email string) error {
	result := rp.db.Unscoped().Where("email = ?", email).Delete(&models.ResetPassword{})
	if result.Error != nil {
		log.Println("Error deleting reset password record:", result.Error)
	}
	return result.Error
}

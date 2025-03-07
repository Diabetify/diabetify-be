package repository

import (
	"diabetify/database"
	"diabetify/internal/models"
	"log"
	"time"
)

type ResetPasswordRepository struct{}

func NewResetPasswordRepository() *ResetPasswordRepository {
	return &ResetPasswordRepository{}
}

func (rpr *ResetPasswordRepository) CreateResetPassword(reset_password *models.ResetPassword) error {
	return database.DB.Create(reset_password).Error
}

func (rpr *ResetPasswordRepository) FindByEmail(email string) (*models.ResetPassword, error) {
	var reset_password models.ResetPassword
	err := database.DB.Where("email = ?", email).First(&reset_password).Error
	if err != nil {
		return nil, err
	}
	return &reset_password, nil
}

func (rpr *ResetPasswordRepository) FindByEmailAndCode(email, code string) (*models.ResetPassword, error) {
	var reset_password models.ResetPassword
	err := database.DB.Where("email = ? AND code = ? AND expires_at > ?", email, code, time.Now()).
		First(&reset_password).Error
	if err != nil {
		return nil, err
	}
	return &reset_password, nil
}

func (rpr *ResetPasswordRepository) DeleteByEmail(email string) error {
	result := database.DB.Unscoped().Where("email = ?", email).Delete(&models.ResetPassword{})
	if result.Error != nil {
		log.Println("Error deleting reset password data:", result.Error)
	}
	return result.Error
}

package models

import (
	"time"

	"gorm.io/gorm"
)

// @description Verification model
type Verification struct {
	ID        uint           `gorm:"primaryKey" json:"id" example:"1"`
	CreatedAt time.Time      `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt time.Time      `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-" swaggerignore:"true"`
	Email     string         `json:"email" example:"admin@admin.com" gorm:"unique"`
	Code      string         `json:"code" example:"123456"`
	ExpiresAt time.Time      `json:"expires_at" example:"2023-01-01T00:00:00Z"`
}

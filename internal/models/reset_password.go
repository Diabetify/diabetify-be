package models

import (
	"time"

	"gorm.io/gorm"
)

type ResetPassword struct {
	ID        uint           `gorm:"primaryKey" json:"id" example:"1"`
	CreatedAt time.Time      `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt time.Time      `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-" swaggerignore:"true"`
	Email     string         `gorm:"unique" json:"email" example:"john.doe@example.com"`
	Code      string         `json:"code" example:"123456"`
	ExpiresAt time.Time      `json:"expires_at" example:"2023-01-01T00:00:00Z"`
}

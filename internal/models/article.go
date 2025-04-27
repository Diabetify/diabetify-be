package models

import (
	"time"

	"gorm.io/gorm"
)

type Article struct {
	ID          uint           `gorm:"primaryKey" json:"id" example:"1"`
	CreatedAt   time.Time      `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt   time.Time      `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-" swaggerignore:"true"`
	Title       string         `json:"title" example:"Sample Article Title"`
	Description string         `json:"description" example:"This is a sample article description."`
}

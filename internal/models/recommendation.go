package models

import (
	"time"

	"gorm.io/gorm"
)

type Recommendation struct {
	ID          uint           `gorm:"primaryKey" json:"id" example:"1"`
	CreatedAt   time.Time      `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt   time.Time      `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-" swaggerignore:"true"`
	UserID      uint           `json:"user_id" example:"1"`
	User        User           `gorm:"foreignKey:UserID" json:"-"`
	Title       string         `json:"title" example:"Kurangi konsumsi gula"`
	Description string         `json:"description" example:"Kurangi konsumsi gula sebanyak 10% dari asupan harian Anda untuk menurunkan risiko diabetes."`
	Type        string         `json:"type" example:"nutrition"`
}

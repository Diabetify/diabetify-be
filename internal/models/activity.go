package models

import (
	"time"

	"gorm.io/gorm"
)

type Activity struct {
	ID           uint           `gorm:"primaryKey" json:"id" example:"1"`
	CreatedAt    time.Time      `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt    time.Time      `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-" swaggerignore:"true"`
	UserID       uint           `json:"user_id" example:"1"`
	User         User           `gorm:"foreignKey:UserID" json:"-"`
	ActivityType string         `json:"activity_type" example:"food"`
	ActivityDate time.Time      `json:"activity_date" example:"2023-01-01"`
	Value        int            `json:"value" example:"500"`
}

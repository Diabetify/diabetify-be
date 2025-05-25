package models

import (
	"time"

	"gorm.io/gorm"
)

type UserProfile struct {
	ID              uint           `gorm:"primaryKey" json:"id" example:"1"`
	CreatedAt       time.Time      `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt       time.Time      `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-" swaggerignore:"true"`
	UserID          uint           `gorm:"unique" json:"user_id" example:"1"`
	Hypertension    *bool          `json:"hypertension" example:"false"`
	DisturbedVision *bool          `json:"disturbed_vision" example:"false"`
	Weight          *int           `json:"weight" example:"70"`
	Height          *int           `json:"height" example:"175"`
	BMI             *float64       `json:"bmi" example:"22.9"`
	Smoking         *bool          `gorm:"column:smoking" json:"smoking" example:"false"`
	MacrosomicBaby  *bool          `gorm:"column:macrosomic_baby" json:"macrosomic_baby" example:"false"`
}

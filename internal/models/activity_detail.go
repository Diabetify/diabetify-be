package models

import (
	"time"

	"gorm.io/gorm"
)

type ActivityDetail struct {
	ID         uint           `gorm:"primaryKey" json:"id" example:"1"`
	CreatedAt  time.Time      `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt  time.Time      `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-" swaggerignore:"true"`
	ActivityID uint           `json:"activity_id" example:"1"`
	Activity   Activity       `gorm:"foreignKey:ActivityID" json:"-"`
	FieldName  string         `json:"field_name" example:"calories"`
	Value      string         `json:"value" example:"500"`
}

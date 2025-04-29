package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a system user
// @description User model for the system
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id" example:"1"`
	CreatedAt time.Time      `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt time.Time      `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-" swaggerignore:"true"`
	Name      string         `json:"name" example:"John Doe"`
	Email     string         `gorm:"unique" json:"email" example:"john.doe@example.com"`
	Gender    *string        `gorm:"type:text;check:gender IN ('male', 'female');" json:"gender" example:"male"`
	Password  string         `json:"password" example:"securepassword123"`
	DOB       *string        `gorm:"type:DATE;" json:"dob" example:"2000-01-30"`
	Verified  bool           `gorm:"default:false" json:"verified" example:"false"`
}

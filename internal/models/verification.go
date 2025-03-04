package models

import (
	"time"

	"gorm.io/gorm"
)

type Verification struct {
	gorm.Model
	Email     string `gorm:"unique"`
	Code      string
	ExpiresAt time.Time
}

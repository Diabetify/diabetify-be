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

func (v *Verification) GetShardKey() int {
	hash := 0
	for _, char := range v.Email {
		hash += int(char)
	}
	// Map to shard ranges (1-5000 for shard1, 5001-10000 for shard2)
	if hash%2 == 0 {
		return 2500 // Will go to shard1
	}
	return 7500 // Will go to shard2
}

func (v *Verification) TableName() string {
	return "verifications"
}

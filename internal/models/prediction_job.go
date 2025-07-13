package models

import (
	"time"

	"gorm.io/gorm"
)

type PredictionJob struct {
	ID           string         `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID       uint           `gorm:"not null;index" json:"user_id"`
	Status       string         `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	IsWhatIf     bool           `json:"is_what_if"`
	PredictionID *uint          `gorm:"index" json:"prediction_id,omitempty"`
	ErrorMessage *string        `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt    time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	User       User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Prediction *Prediction `gorm:"foreignKey:PredictionID" json:"prediction,omitempty"`
}

// Job status constants
const (
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusSubmitted  = "submitted"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
	JobStatusCancelled  = "cancelled"
)

func (pj *PredictionJob) TableName() string {
	return "prediction_jobs"
}

// PredictionJobRequest represents a job request for processing
type PredictionJobRequest struct {
	JobID       string       `json:"job_id"`
	UserID      uint         `json:"user_id"`
	WhatIfInput *WhatIfInput `json:"what_if_input,omitempty"`
}

// What if Input
type WhatIfInput struct {
	SmokingStatus             int     `json:"smoking_status" binding:"oneof=0 1 2"`
	YearsOfSmoking            int     `json:"years_of_smoking" binding:"min=0"`
	AvgSmokeCount             int     `json:"avg_smoke_count" binding:"min=0"`
	Weight                    float64 `json:"weight" binding:"min=1"`
	IsHypertension            bool    `json:"is_hypertension"`
	PhysicalActivityFrequency int     `json:"physical_activity_frequency" binding:"min=0"`
	IsCholesterol             bool    `json:"is_cholesterol"`
}

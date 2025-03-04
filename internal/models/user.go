package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name            string
	Email           string `gorm:"unique"`
	Password        string
	Age             int
	Hipertension    bool
	Cholesterol     bool
	DisturbedVision bool
	Weight          int
	Height          int
	Verified        bool `gorm:"default:false"`
}

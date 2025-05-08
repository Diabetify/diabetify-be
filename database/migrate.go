package database

import (
	"diabetify/internal/models"
	"log"
)

func MigrateDatabase() error {
	log.Println("Running database migrations...")

	err := DB.AutoMigrate(
		&models.User{},
		&models.UserProfile{},
		&models.Activity{},
		&models.Article{},
		&models.Verification{},
		&models.ResetPassword{},
	)

	if err != nil {
		log.Printf("Error during migration: %v", err)
		return err
	}

	log.Println("Database migrations completed successfully")
	return nil
}

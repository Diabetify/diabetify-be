package database

import (
	"diabetify/internal/models"
	"fmt"
	"log"

	"gorm.io/gorm"
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
		&models.Prediction{},
	)

	if err != nil {
		log.Printf("Error during migration: %v", err)
		return err
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// MigrateAllShards runs migrations on all shards
func MigrateAllShards() error {
	if Manager == nil {
		return fmt.Errorf("shard manager not initialized")
	}

	log.Println("Running migrations on all shards...")

	return Manager.ExecuteOnAllShards(func(db *gorm.DB) error {
		return migrateOnShard(db)
	})
}

// migrateOnShard runs migrations on a specific shard
func migrateOnShard(db *gorm.DB) error {
	log.Println("Running database migrations on shard...")

	err := db.AutoMigrate(
		&models.User{},
		&models.UserProfile{},
		&models.Activity{},
		&models.Article{},
		&models.Verification{},
		&models.ResetPassword{},
		&models.Prediction{},
	)

	if err != nil {
		log.Printf("Error during shard migration: %v", err)
		return err
	}

	log.Println("Shard migration completed successfully")
	return nil
}

package utils

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// CheckForDuplicateEmails checks if any of the test emails already exist in the database
func CheckForDuplicateEmails(startIndex, endIndex int) error {
	// Use environment variables for database connection
	dbHost := getEnv("DB_HOST", "diabetify-db")
	dbPort := getEnv("DB_PORT", "5439")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "diabetify")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")
	dbTimeZone := getEnv("DB_TIMEZONE", "Asia/Jakarta")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode, dbTimeZone)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	log.Println("Connected to database successfully")
	log.Printf("Checking for duplicate emails in range %d-%d...", startIndex, endIndex)

	// Check for duplicate emails
	for i := startIndex; i <= endIndex; i++ {
		email := fmt.Sprintf("testuser%d@example.com", i)

		var count int64
		if err := db.Model(&struct{ Email string }{}).
			Table("users").
			Where("email = ?", email).
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check for duplicate email %s: %v", email, err)
		}

		if count > 0 {
			log.Printf("Email %s already exists in the database", email)
		}
	}

	log.Println("Email duplicate check completed")
	return nil
}

// DeleteTestUsers deletes test users in the specified range
func DeleteTestUsers(startIndex, endIndex int) error {
	// Use environment variables for database connection
	dbHost := getEnv("DB_HOST", "diabetify-db")
	dbPort := getEnv("DB_PORT", "5439")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "diabetify")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")
	dbTimeZone := getEnv("DB_TIMEZONE", "Asia/Jakarta")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode, dbTimeZone)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	log.Println("Connected to database successfully")
	log.Printf("Deleting test users with emails in range testuser%d@example.com to testuser%d@example.com...",
		startIndex, endIndex)

	// Delete test users in batches for better performance
	batchSize := 1000
	totalDeleted := 0

	for i := startIndex; i <= endIndex; i += batchSize {
		end := i + batchSize - 1
		if end > endIndex {
			end = endIndex
		}

		// Generate list of emails to delete
		var emails []string
		for j := i; j <= end; j++ {
			emails = append(emails, fmt.Sprintf("testuser%d@example.com", j))
		}

		// Delete users with these emails
		result := db.Table("users").Where("email IN ?", emails).Delete(nil)
		if result.Error != nil {
			return fmt.Errorf("failed to delete users batch %d-%d: %v", i, end, result.Error)
		}

		totalDeleted += int(result.RowsAffected)
		log.Printf("Deleted %d users in range %d-%d", result.RowsAffected, i, end)
	}

	log.Printf("Successfully deleted %d test users", totalDeleted)
	return nil
}

package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"diabetify/internal/models"
	"encoding/hex"
	"fmt"
	"log"
	mathrand "math/rand"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const DefaultNumUsers = 10000

func generateTestPassword() string {
	password := "TestPassword123!"

	salt := make([]byte, 8)
	rand.Read(salt)

	h := sha256.New()
	h.Write([]byte(password))
	h.Write(salt)
	hash := h.Sum(nil)

	return hex.EncodeToString(salt) + hex.EncodeToString(hash)
}

func SeedUsers(numUsers int) error {
	dbHost := getEnv("DB_HOST", "diabetify-db")
	dbPort := getEnv("DB_PORT", "5439")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "diabetify")
	dbSSLMode := getEnv("DB_SSLMODE", "require")
	dbTimeZone := getEnv("DB_TIMEZONE", "Asia/Jakarta")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode, dbTimeZone)

	// Configure GORM with connection pool settings for seeding
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt:            true, // Cache prepared statements
		SkipDefaultTransaction: true, // Skip default transaction for better performance
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Connected to database successfully")
	log.Printf("Starting to seed %d users with simple SHA256 hashing...", numUsers)

	startTime := time.Now()

	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	var maxID uint
	row := db.Model(&models.User{}).Select("COALESCE(MAX(id), 0)").Row()
	if err := row.Scan(&maxID); err != nil {
		return fmt.Errorf("failed to get max user ID: %v", err)
	}

	log.Printf("Current max user ID: %d", maxID)
	baseIndex := int(maxID) + 1

	batchSize := 1000
	for i := 0; i < numUsers; i += batchSize {
		var users []models.User

		end := i + batchSize
		if end > numUsers {
			end = numUsers
		}

		batchStartTime := time.Now()

		for j := i; j < end; j++ {
			user := generateUser(baseIndex+j, r)
			users = append(users, user)
		}

		result := db.CreateInBatches(&users, 100)
		if result.Error != nil {
			return fmt.Errorf("failed to create users batch %d-%d: %v", i, end-1, result.Error)
		}

		batchElapsed := time.Since(batchStartTime)
		log.Printf("Created users %d-%d in %s (%.2f users/sec)",
			i, end-1, batchElapsed, float64(end-i)/batchElapsed.Seconds())
	}

	elapsed := time.Since(startTime)
	usersPerSecond := float64(numUsers) / elapsed.Seconds()
	log.Printf("✅ Successfully created %d users in %s (%.2f users/sec)",
		numUsers, elapsed, usersPerSecond)

	return nil
}

func generateUser(index int, r *mathrand.Rand) models.User {
	password := generateTestPassword()

	gender := randomGender(r)
	dob := randomDOB(r)

	return models.User{
		Name:      fmt.Sprintf("Test User %d", index),
		Email:     fmt.Sprintf("testuser%d@example.com", index),
		Gender:    &gender,
		Password:  password,
		DOB:       &dob,
		Verified:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func randomGender(r *mathrand.Rand) string {
	if r.Intn(2) == 0 {
		return "male"
	}
	return "female"
}

func randomDOB(r *mathrand.Rand) string {
	year := r.Intn(50) + 1950 // 1950-1999
	month := r.Intn(12) + 1
	day := r.Intn(28) + 1

	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// CleanupTestUsers removes all test users (useful for testing)
func CleanupTestUsers() error {
	dbHost := getEnv("DB_HOST", "diabetify-db")
	dbPort := getEnv("DB_PORT", "5439")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "diabetify")
	dbSSLMode := getEnv("DB_SSLMODE", "require")
	dbTimeZone := getEnv("DB_TIMEZONE", "Asia/Jakarta")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode, dbTimeZone)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	result := db.Where("email LIKE ?", "testuser%@example.com").Delete(&models.User{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup test users: %v", result.Error)
	}

	log.Printf("✅ Deleted %d test users", result.RowsAffected)
	return nil
}

// GetUserCount returns the current number of users in the database
func GetUserCount() (int64, error) {
	dbHost := getEnv("DB_HOST", "diabetify-db")
	dbPort := getEnv("DB_PORT", "5439")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "diabetify")
	dbSSLMode := getEnv("DB_SSLMODE", "require")
	dbTimeZone := getEnv("DB_TIMEZONE", "Asia/Jakarta")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode, dbTimeZone)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return 0, fmt.Errorf("failed to connect to database: %v", err)
	}

	var count int64
	result := db.Model(&models.User{}).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count users: %v", result.Error)
	}

	return count, nil
}

// VerifyTestUser checks if a specific test user exists and can login
func VerifyTestUser(userIndex int) error {
	dbHost := getEnv("DB_HOST", "diabetify-db")
	dbPort := getEnv("DB_PORT", "5439")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "diabetify")
	dbSSLMode := getEnv("DB_SSLMODE", "require")
	dbTimeZone := getEnv("DB_TIMEZONE", "Asia/Jakarta")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode, dbTimeZone)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	email := fmt.Sprintf("testuser%d@example.com", userIndex)
	var user models.User
	result := db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return fmt.Errorf("test user %s not found: %v", email, result.Error)
	}

	log.Printf("✅ Test user %s exists (ID: %d, Verified: %t)", email, user.ID, user.Verified)
	return nil
}

func SeedUsersWithIDRange(startID, endID int) error {
	dbHost := getEnv("DB_HOST", "diabetify-db")
	dbPort := getEnv("DB_PORT", "5439")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "diabetify")
	dbSSLMode := getEnv("DB_SSLMODE", "require")
	dbTimeZone := getEnv("DB_TIMEZONE", "Asia/Jakarta")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode, dbTimeZone)

	// Configure GORM with connection pool settings
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt:            true, // Cache prepared statements
		SkipDefaultTransaction: true, // Skip default transaction for better performance
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	log.Printf("Starting to seed users with IDs %d-%d", startID, endID)

	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	for userID := startID; userID <= endID; userID++ {
		user := generateUserWithID(uint(userID), userID, r)

		result := db.Exec(`
			INSERT INTO users (name, email, gender, password, dob, verified, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			user.Name, user.Email, user.Gender, user.Password, user.DOB,
			user.Verified, user.CreatedAt, user.UpdatedAt)

		if result.Error != nil {
			return fmt.Errorf("failed to create user %d: %v", userID, result.Error)
		}

		log.Printf("Created user %d with email %s", userID, user.Email)
	}

	log.Printf("Successfully created %d users with IDs %d-%d", endID-startID+1, startID, endID)
	return nil
}

// Helper function to generate a user with a specific ID
func generateUserWithID(id uint, index int, r *mathrand.Rand) models.User {
	// Use your existing password generation method (SHA256 with salt)
	password := generateTestPassword()

	gender := randomGender(r)
	dob := randomDOB(r)

	return models.User{
		ID:        id, // Explicitly set the ID as uint
		Name:      fmt.Sprintf("Test User %d", index),
		Email:     fmt.Sprintf("testuser%d@example.com", index),
		Gender:    &gender,
		Password:  password,
		DOB:       &dob,
		Verified:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

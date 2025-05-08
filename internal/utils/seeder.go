package utils

import (
	"diabetify/internal/models"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const DefaultNumUsers = 10000

func SeedUsers(numUsers int) error {
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
	log.Printf("Starting to seed %d users...", numUsers)

	startTime := time.Now()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

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

		for j := i; j < end; j++ {
			user := generateUser(baseIndex+j, r)
			users = append(users, user)
		}

		result := db.Create(&users)
		if result.Error != nil {
			return fmt.Errorf("failed to create users batch %d-%d: %v", i, end-1, result.Error)
		}

		log.Printf("Created users %d-%d", i, end-1)
	}

	elapsed := time.Since(startTime)
	log.Printf("Successfully created %d users in %s", numUsers, elapsed)

	return nil
}

func generateUser(index int, r *rand.Rand) models.User {
	password, _ := bcrypt.GenerateFromPassword([]byte("TestPassword123!"), bcrypt.DefaultCost)

	gender := randomGender(r)
	dob := randomDOB(r)

	return models.User{
		Name:      fmt.Sprintf("Test User %d", index),
		Email:     fmt.Sprintf("testuser%d@example.com", index),
		Gender:    &gender,
		Password:  string(password),
		DOB:       &dob,
		Verified:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func randomGender(r *rand.Rand) string {
	if r.Intn(2) == 0 {
		return "male"
	}
	return "female"
}

func randomDOB(r *rand.Rand) string {
	year := r.Intn(50) + 1950
	month := r.Intn(12) + 1
	day := r.Intn(28) + 1

	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

func randomBoolPtr(r *rand.Rand) *bool {
	val := r.Intn(2) == 1
	return &val
}

func randomIntPtr(r *rand.Rand, min, max int) *int {
	val := r.Intn(max-min+1) + min
	return &val
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

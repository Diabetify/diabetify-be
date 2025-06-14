package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"diabetify/database"
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

// ==================== ORIGINAL FUNCTIONS (Single Database) ====================

func SeedUsers(numUsers int) error {
	db, err := connectToSingleDatabase()
	if err != nil {
		return err
	}
	return seedUsersToDatabase(db, numUsers, 0, 0, false)
}

func SeedUsersWithIDRange(startID, endID int) error {
	db, err := connectToSingleDatabase()
	if err != nil {
		return err
	}
	return seedUsersWithSpecificIDs(db, startID, endID)
}

func CleanupTestUsers() error {
	db, err := connectToSingleDatabase()
	if err != nil {
		return err
	}

	result := db.Where("email LIKE ?", "testuser%@example.com").Delete(&models.User{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup test users: %v", result.Error)
	}

	log.Printf("✅ Deleted %d test users", result.RowsAffected)
	return nil
}

func GetUserCount() (int64, error) {
	db, err := connectToSingleDatabase()
	if err != nil {
		return 0, err
	}

	var count int64
	result := db.Model(&models.User{}).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count users: %v", result.Error)
	}

	return count, nil
}

func VerifyTestUser(userIndex int) error {
	db, err := connectToSingleDatabase()
	if err != nil {
		return err
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

// ==================== FIXED SHARDED FUNCTIONS ====================

// SeedUsersSharded seeds users with proper global unique IDs across shards
func SeedUsersSharded(numUsers int, shardName string) error {
	if database.Manager == nil {
		return fmt.Errorf("shard manager not initialized")
	}

	if shardName != "" {
		// Seed to specific shard - but with global unique IDs
		db := database.Manager.GetShard(shardName)
		if db == nil {
			return fmt.Errorf("shard %s not found", shardName)
		}
		log.Printf("Seeding %d users to shard: %s with global unique IDs", numUsers, shardName)
		return seedUsersToShardWithGlobalIDs(db, numUsers)
	}

	// Distribute across all shards with global unique IDs
	return SeedUsersShardedWithGlobalIDs(numUsers)
}

// SeedUsersShardedWithGlobalIDs properly distributes users with global unique IDs
func SeedUsersShardedWithGlobalIDs(totalUsers int) error {
	if database.Manager == nil {
		return fmt.Errorf("shard manager not initialized")
	}

	log.Printf("Seeding %d users with global unique IDs across all shards", totalUsers)

	// Create users with sequential global IDs and distribute based on shard key
	for userID := 1; userID <= totalUsers; userID++ {
		// Determine which shard this user should go to
		targetShard := database.GetShardNameByUserID(userID)
		db := database.Manager.GetShard(targetShard)
		if db == nil {
			return fmt.Errorf("shard %s not found for user ID %d", targetShard, userID)
		}

		// Create user with specific ID
		if err := seedSingleUserWithGlobalID(db, userID); err != nil {
			return fmt.Errorf("error seeding user %d to shard %s: %v", userID, targetShard, err)
		}

		if userID%1000 == 0 {
			log.Printf("Seeded user %d to %s", userID, targetShard)
		}
	}

	log.Printf("✅ Successfully seeded %d users with global unique IDs", totalUsers)
	return nil
}

func SeedUsersWithIDRangeSharded(startID, endID int, shardName string) error {
	if database.Manager == nil {
		return fmt.Errorf("shard manager not initialized")
	}

	if shardName != "" {
		// Seed to specific shard
		db := database.Manager.GetShard(shardName)
		if db == nil {
			return fmt.Errorf("shard %s not found", shardName)
		}
		log.Printf("Seeding users %d-%d to shard: %s", startID, endID, shardName)
		return seedUsersWithSpecificIDs(db, startID, endID)
	}

	// Distribute based on ID ranges - FIXED VERSION
	log.Printf("Distributing users %d-%d across shards based on shard key", startID, endID)
	for userID := startID; userID <= endID; userID++ {
		targetShard := database.GetShardNameByUserID(userID)
		db := database.Manager.GetShard(targetShard)
		if db == nil {
			return fmt.Errorf("shard %s not found for user ID %d", targetShard, userID)
		}

		if err := seedSingleUserWithGlobalID(db, userID); err != nil {
			return fmt.Errorf("error seeding user %d to shard %s: %v", userID, targetShard, err)
		}

		if userID%1000 == 0 {
			log.Printf("Seeded user %d to %s", userID, targetShard)
		}
	}

	return nil
}

// CleanupAllShardsAndReseed completely clears all shards and reseeds with proper distribution
func CleanupAllShardsAndReseed(totalUsers int) error {
	log.Println("🧹 Cleaning up all shards...")
	if err := ClearAllDataSharded("all"); err != nil {
		return fmt.Errorf("failed to clear shards: %v", err)
	}

	log.Println("🌱 Reseeding with proper global IDs...")
	return SeedUsersShardedWithGlobalIDs(totalUsers)
}

func GetUserCountSharded() (map[string]int64, error) {
	if database.Manager == nil {
		return nil, fmt.Errorf("shard manager not initialized")
	}

	shards := database.Manager.GetAllShards()
	counts := make(map[string]int64)
	total := int64(0)

	for shardName, db := range shards {
		var count int64
		result := db.Model(&models.User{}).Count(&count)
		if result.Error != nil {
			return nil, fmt.Errorf("failed to count users in shard %s: %v", shardName, result.Error)
		}
		counts[shardName] = count
		total += count
		log.Printf("Shard %s: %d users", shardName, count)
	}

	counts["total"] = total
	log.Printf("Total users across all shards: %d", total)
	return counts, nil
}

func ClearAllDataSharded(shardName string) error {
	if database.Manager == nil {
		return fmt.Errorf("shard manager not initialized")
	}

	if shardName == "all" || shardName == "" {
		// Clear all shards
		shards := database.Manager.GetAllShards()
		for name, db := range shards {
			log.Printf("Clearing all data from shard: %s", name)
			if err := clearDataFromDatabase(db); err != nil {
				return fmt.Errorf("error clearing shard %s: %v", name, err)
			}
		}
		log.Println("✅ All shards cleared successfully")
		return nil
	}

	// Clear specific shard
	db := database.Manager.GetShard(shardName)
	if db == nil {
		return fmt.Errorf("shard %s not found", shardName)
	}

	log.Printf("Clearing all data from shard: %s", shardName)
	return clearDataFromDatabase(db)
}

func DeleteTestUsersSharded(startIndex, endIndex int, shardName string) error {
	if database.Manager == nil {
		return fmt.Errorf("shard manager not initialized")
	}

	if shardName == "all" || shardName == "" {
		// Delete from all shards
		shards := database.Manager.GetAllShards()
		totalDeleted := int64(0)
		for name, db := range shards {
			log.Printf("Deleting users from shard: %s", name)
			deleted, err := deleteUsersFromDatabaseWithCount(db, startIndex, endIndex)
			if err != nil {
				return fmt.Errorf("error deleting from shard %s: %v", name, err)
			}
			totalDeleted += deleted
		}
		log.Printf("✅ Total deleted across all shards: %d users", totalDeleted)
		return nil
	}

	// Delete from specific shard
	db := database.Manager.GetShard(shardName)
	if db == nil {
		return fmt.Errorf("shard %s not found", shardName)
	}

	log.Printf("Deleting users from shard: %s", shardName)
	_, err := deleteUsersFromDatabaseWithCount(db, startIndex, endIndex)
	return err
}

func CheckForDuplicateEmailsSharded(startIndex, endIndex int) error {
	if database.Manager == nil {
		return fmt.Errorf("shard manager not initialized")
	}

	shards := database.Manager.GetAllShards()
	allDuplicates := make(map[string]int)

	for shardName, db := range shards {
		log.Printf("Checking duplicates in shard: %s", shardName)

		var duplicateEmails []string
		err := db.Raw(`
			SELECT email 
			FROM users 
			WHERE email LIKE 'testuser%@example.com'
			GROUP BY email 
			HAVING COUNT(*) > 1
			ORDER BY email
		`).Scan(&duplicateEmails).Error

		if err != nil {
			return fmt.Errorf("error checking shard %s: %v", shardName, err)
		}

		for _, email := range duplicateEmails {
			allDuplicates[email]++
		}
	}

	if len(allDuplicates) > 0 {
		log.Printf("Found %d duplicate emails across all shards:", len(allDuplicates))
		for email, count := range allDuplicates {
			log.Printf("  - %s (found in %d shard(s))", email, count)
		}
	} else {
		log.Println("No duplicate emails found across all shards")
	}

	return nil
}

// GetUserDistributionReport shows how users are distributed across shards
func GetUserDistributionReport() error {
	if database.Manager == nil {
		return fmt.Errorf("shard manager not initialized")
	}

	shards := database.Manager.GetAllShards()
	totalUsers := int64(0)

	log.Println("📊 User Distribution Report:")
	log.Println("==========================")

	for shardName, db := range shards {
		var count int64
		var minID, maxID uint

		// Get count
		db.Model(&models.User{}).Count(&count)

		// Get ID range
		db.Model(&models.User{}).Select("MIN(id)").Row().Scan(&minID)
		db.Model(&models.User{}).Select("MAX(id)").Row().Scan(&maxID)

		log.Printf("Shard %s: %d users (ID range: %d-%d)", shardName, count, minID, maxID)
		totalUsers += count
	}

	log.Println("==========================")
	log.Printf("Total users: %d", totalUsers)
	return nil
}

// VerifyShardingConsistency checks if users are in the correct shards
func VerifyShardingConsistency() error {
	if database.Manager == nil {
		return fmt.Errorf("shard manager not initialized")
	}

	shards := database.Manager.GetAllShards()
	inconsistencies := 0

	log.Println("🔍 Verifying sharding consistency...")

	for shardName, db := range shards {
		var users []models.User
		db.Find(&users)

		for _, user := range users {
			expectedShard := database.GetShardNameByUserID(int(user.ID))
			if expectedShard != shardName {
				log.Printf("❌ User %d is in shard %s but should be in %s", user.ID, shardName, expectedShard)
				inconsistencies++
			}
		}
	}

	if inconsistencies == 0 {
		log.Println("✅ All users are in the correct shards")
	} else {
		log.Printf("❌ Found %d sharding inconsistencies", inconsistencies)
	}

	return nil
}

// MigrateToProperSharding fixes existing data by moving users to correct shards
func MigrateToProperSharding() error {
	if database.Manager == nil {
		return fmt.Errorf("shard manager not initialized")
	}

	log.Println("🔄 Starting migration to proper sharding...")

	// Step 1: Collect all users from all shards
	allUsers := make(map[int]models.User) // userID -> user
	shards := database.Manager.GetAllShards()

	for shardName, db := range shards {
		var users []models.User
		db.Find(&users)

		for _, user := range users {
			if _, exists := allUsers[int(user.ID)]; exists {
				log.Printf("⚠️ Duplicate user ID %d found in shard %s", user.ID, shardName)
			}
			allUsers[int(user.ID)] = user
		}
		log.Printf("Collected %d users from shard %s", len(users), shardName)
	}

	// Step 2: Clear all shards
	log.Println("Clearing all shards...")
	if err := ClearAllDataSharded("all"); err != nil {
		return fmt.Errorf("failed to clear shards: %v", err)
	}

	// Step 3: Redistribute users to correct shards
	log.Printf("Redistributing %d users to correct shards...", len(allUsers))

	for userID, user := range allUsers {
		targetShard := database.GetShardNameByUserID(userID)
		db := database.Manager.GetShard(targetShard)
		if db == nil {
			return fmt.Errorf("shard %s not found for user ID %d", targetShard, userID)
		}

		// Insert user into correct shard
		if err := db.Create(&user).Error; err != nil {
			return fmt.Errorf("failed to create user %d in shard %s: %v", userID, targetShard, err)
		}

		if userID%1000 == 0 {
			log.Printf("Migrated user %d to %s", userID, targetShard)
		}
	}

	log.Printf("✅ Successfully migrated %d users to proper shards", len(allUsers))
	return nil
}

// ==================== CORE DATABASE FUNCTIONS ====================

func connectToSingleDatabase() (*gorm.DB, error) {
	dbHost := getEnv("DB_HOST", "diabetify-db")
	dbPort := getEnv("DB_PORT", "5439")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "diabetify")
	dbSSLMode := getEnv("DB_SSLMODE", "require")
	dbTimeZone := getEnv("DB_TIMEZONE", "Asia/Jakarta")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode, dbTimeZone)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func seedUsersToDatabase(db *gorm.DB, numUsers, startID, endID int, useSpecificIDs bool) error {
	log.Println("Connected to database successfully")

	if useSpecificIDs && startID > 0 && endID > 0 {
		log.Printf("Starting to seed users with IDs %d-%d", startID, endID)
		return seedUsersWithSpecificIDs(db, startID, endID)
	} else {
		log.Printf("Starting to seed %d users with auto-increment IDs", numUsers)
		return seedUsersWithAutoIncrement(db, numUsers)
	}
}

func seedUsersToShardWithGlobalIDs(db *gorm.DB, numUsers int) error {
	log.Printf("Starting to seed %d users with global unique IDs", numUsers)

	// Get the current max ID across all shards
	shards := database.Manager.GetAllShards()
	globalMaxID := uint(0)

	for _, shardDB := range shards {
		var maxID uint
		row := shardDB.Model(&models.User{}).Select("COALESCE(MAX(id), 0)").Row()
		if err := row.Scan(&maxID); err == nil && maxID > globalMaxID {
			globalMaxID = maxID
		}
	}

	startID := int(globalMaxID) + 1
	endID := startID + numUsers - 1

	log.Printf("Seeding users with IDs %d-%d", startID, endID)

	for userID := startID; userID <= endID; userID++ {
		if err := seedSingleUserWithGlobalID(db, userID); err != nil {
			return fmt.Errorf("error seeding user %d: %v", userID, err)
		}

		if userID%1000 == 0 {
			log.Printf("Seeded user %d", userID)
		}
	}

	return nil
}

func seedUsersWithAutoIncrement(db *gorm.DB, numUsers int) error {
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

func seedUsersWithSpecificIDs(db *gorm.DB, startID, endID int) error {
	log.Printf("Starting to seed users with IDs %d-%d", startID, endID)

	for userID := startID; userID <= endID; userID++ {
		if err := seedSingleUserToDatabase(db, userID, true); err != nil {
			return err
		}

		if userID%1000 == 0 {
			log.Printf("Created user %d", userID)
		}
	}

	log.Printf("✅ Successfully created %d users with IDs %d-%d", endID-startID+1, startID, endID)
	return nil
}

func seedSingleUserToDatabase(db *gorm.DB, userID int, useSpecificID bool) error {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	if useSpecificID {
		user := generateUserWithID(uint(userID), userID, r)
		result := db.Exec(`
			INSERT INTO users (name, email, gender, password, dob, verified, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			user.Name, user.Email, user.Gender, user.Password, user.DOB,
			user.Verified, user.CreatedAt, user.UpdatedAt)

		if result.Error != nil {
			return fmt.Errorf("failed to create user %d: %v", userID, result.Error)
		}
	} else {
		user := generateUser(userID, r)
		if err := db.Create(&user).Error; err != nil {
			return fmt.Errorf("failed to create user: %v", err)
		}
	}

	return nil
}

func seedSingleUserWithGlobalID(db *gorm.DB, userID int) error {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano() + int64(userID)))

	password := generateTestPassword()
	gender := randomGender(r)
	dob := randomDOB(r)

	// Use raw SQL to insert with specific ID (bypassing auto-increment)
	result := db.Exec(`
		INSERT INTO users (id, name, email, gender, password, dob, verified, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID,
		fmt.Sprintf("Test User %d", userID),
		fmt.Sprintf("testuser%d@example.com", userID),
		gender,
		password,
		dob,
		true,
		time.Now(),
		time.Now())

	if result.Error != nil {
		return fmt.Errorf("failed to create user %d: %v", userID, result.Error)
	}

	return nil
}

func clearDataFromDatabase(db *gorm.DB) error {
	// Delete in order due to foreign key constraints
	tables := []interface{}{
		&models.Prediction{},
		&models.Activity{},
		&models.UserProfile{},
		&models.Verification{},
		&models.ResetPassword{},
		&models.User{},
	}

	for _, table := range tables {
		if err := db.Unscoped().Where("1 = 1").Delete(table).Error; err != nil {
			return fmt.Errorf("error clearing table %T: %v", table, err)
		}
	}

	log.Println("✅ All tables cleared successfully")
	return nil
}

func deleteUsersFromDatabaseWithCount(db *gorm.DB, startIndex, endIndex int) (int64, error) {
	result := db.Unscoped().Where(
		"email LIKE 'testuser%@example.com' AND id BETWEEN ? AND ?",
		startIndex, endIndex,
	).Delete(&models.User{})

	if result.Error != nil {
		return 0, fmt.Errorf("error deleting users: %v", result.Error)
	}

	log.Printf("Deleted %d users", result.RowsAffected)
	return result.RowsAffected, nil
}

// ==================== HELPER FUNCTIONS ====================

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

func generateUserWithID(id uint, index int, r *mathrand.Rand) models.User {
	password := generateTestPassword()
	gender := randomGender(r)
	dob := randomDOB(r)

	return models.User{
		ID:        id,
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

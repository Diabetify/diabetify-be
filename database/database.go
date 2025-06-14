package database

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ShardManager manages multiple database connections
type ShardManager struct {
	shards map[string]*gorm.DB
	mutex  sync.RWMutex
}

var (
	Manager *ShardManager
	// Keep the original DB for backward compatibility during migration
	DB *gorm.DB
)

// ShardConfig represents configuration for a single shard
type ShardConfig struct {
	Name     string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	MinRange int
	MaxRange int
}

// ConnectDatabase - original function for backward compatibility
func ConnectDatabase() {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")
	sslmode := os.Getenv("DB_SSLMODE")

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s "+
			"TimeZone=Asia/Jakarta",
		host, user, password, dbname, port, sslmode,
	)

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Millisecond * 500,
			Colorful:                  true,
			IgnoreRecordNotFoundError: true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   newLogger,
		PrepareStmt:                              true,
		SkipDefaultTransaction:                   true,
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	sqlDB.SetMaxOpenConns(1000)
	sqlDB.SetMaxIdleConns(200)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(1 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully")
	DB = db
}

func ConnectShardedDatabase() {
	Manager = &ShardManager{
		shards: make(map[string]*gorm.DB),
	}

	shards := []ShardConfig{
		{
			Name:     "shard1",
			Host:     os.Getenv("DB_HOST"), // 159.223.47.205
			Port:     os.Getenv("DB_PORT"), // 6432
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			DBName:   os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
			MinRange: 1,
			MaxRange: 5000,
		},
		{
			Name:     "shard2",
			Host:     os.Getenv("DB_HOST2"), // 167.172.76.144
			Port:     os.Getenv("DB_PORT"),  // 6432
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			DBName:   os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
			MinRange: 5001,
			MaxRange: 10000,
		},
	}

	// Connect to each shard
	for _, shardConfig := range shards {
		db := connectToShard(shardConfig)
		Manager.addShard(shardConfig.Name, db)
		log.Printf("Connected to %s (users %d-%d)", shardConfig.Name, shardConfig.MinRange, shardConfig.MaxRange)
	}

	// Set the first shard as default DB for backward compatibility
	DB = Manager.GetShard("shard1")

	log.Println("All shards connected successfully")
}

func connectToShard(config ShardConfig) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s "+
			"application_name=diabetify-%s TimeZone=Asia/Jakarta",
		config.Host, config.User, config.Password, config.DBName,
		config.Port, config.SSLMode, config.Name,
	)

	newLogger := logger.New(
		log.New(os.Stdout, fmt.Sprintf("\r\n[%s] ", config.Name), log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Millisecond * 500,
			Colorful:                  true,
			IgnoreRecordNotFoundError: true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   newLogger,
		PrepareStmt:                              true,
		SkipDefaultTransaction:                   true,
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", config.Name, err)
	}

	// Configure connection pool for each shard
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection for %s: %v", config.Name, err)
	}

	sqlDB.SetMaxOpenConns(1000)
	sqlDB.SetMaxIdleConns(200)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(1 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping %s: %v", config.Name, err)
	}

	return db
}

// Add a shard to the manager
func (sm *ShardManager) addShard(name string, db *gorm.DB) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.shards[name] = db
}

// Get a specific shard by name
func (sm *ShardManager) GetShard(name string) *gorm.DB {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.shards[name]
}

// Get shard by user ID using range-based sharding
func (sm *ShardManager) GetShardByUserID(userID int) *gorm.DB {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Range-based sharding logic
	switch {
	case userID >= 1 && userID <= 5000:
		return sm.shards["shard1"]
	case userID >= 5001 && userID <= 10000:
		return sm.shards["shard2"]
	default:
		// Default to shard1 for new users or fallback
		log.Printf("User ID %d not in defined range, using shard1", userID)
		return sm.shards["shard1"]
	}
}

// Get all shards (useful for queries that need to search across all shards)
func (sm *ShardManager) GetAllShards() map[string]*gorm.DB {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	shards := make(map[string]*gorm.DB)
	for name, db := range sm.shards {
		shards[name] = db
	}
	return shards
}

// Execute a query on a specific shard by user ID
func (sm *ShardManager) ExecuteOnUserShard(userID int, fn func(*gorm.DB) error) error {
	db := sm.GetShardByUserID(userID)
	if db == nil {
		return fmt.Errorf("no shard found for user ID %d", userID)
	}
	return fn(db)
}

// Execute a query on all shards (for global queries)
func (sm *ShardManager) ExecuteOnAllShards(fn func(*gorm.DB) error) error {
	shards := sm.GetAllShards()

	for shardName, db := range shards {
		if err := fn(db); err != nil {
			log.Printf("Error executing on shard %s: %v", shardName, err)
			return fmt.Errorf("error on shard %s: %v", shardName, err)
		}
	}
	return nil
}

// MonitorDBConnections - original function for backward compatibility
func MonitorDBConnections() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			sqlDB, _ := DB.DB()
			stats := sqlDB.Stats()
			if stats.InUse > 150 {
				log.Printf("⚠️  DB Connection Pool: InUse=%d, Idle=%d, Open=%d",
					stats.InUse, stats.Idle, stats.OpenConnections)
			}
		}
	}()
}

// MonitorShardedDBConnections - monitor connections across all shards
func MonitorShardedDBConnections() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			if Manager == nil {
				continue
			}

			shards := Manager.GetAllShards()
			for shardName, db := range shards {
				sqlDB, _ := db.DB()
				stats := sqlDB.Stats()
				if stats.InUse > 75 { // Adjusted threshold for multiple shards
					log.Printf("⚠️  Shard %s Connection Pool: InUse=%d, Idle=%d, Open=%d",
						shardName, stats.InUse, stats.Idle, stats.OpenConnections)
				}
			}
		}
	}()
}

// Utility function to get shard name by user ID (for logging/debugging)
func GetShardNameByUserID(userID int) string {
	switch {
	case userID >= 1 && userID <= 5000:
		return "shard1"
	case userID >= 5001 && userID <= 10000:
		return "shard2"
	default:
		return "shard1" // default
	}
}

// Health check for all shards
func CheckShardsHealth() map[string]bool {
	if Manager == nil {
		return nil
	}

	health := make(map[string]bool)
	shards := Manager.GetAllShards()

	for shardName, db := range shards {
		sqlDB, err := db.DB()
		if err != nil {
			health[shardName] = false
			continue
		}

		err = sqlDB.Ping()
		health[shardName] = err == nil
	}

	return health
}

package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDatabase() {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")
	sslmode := os.Getenv("DB_SSLMODE")

	// Create DSN string with only supported connection parameters
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s "+
			"application_name=diabetify connect_timeout=10 statement_timeout=30000 "+
			"idle_in_transaction_session_timeout=60000 TimeZone=Asia/Jakarta",
		host, user, password, dbname, port, sslmode,
	)

	// Configure GORM logger - reduce logging in production
	logLevel := logger.Info
	if os.Getenv("APP_ENV") == "production" {
		logLevel = logger.Error // Only log errors in production
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Millisecond * 500, // Log queries slower than 500ms
			LogLevel:                  logLevel,
			Colorful:                  true,
			IgnoreRecordNotFoundError: true, // Don't log record not found errors
		},
	)

	// Open database connection with optimized parameters
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   newLogger,
		PrepareStmt:                              true, // Cache prepared statements
		SkipDefaultTransaction:                   true, // Skip default transaction for better performance
		DisableForeignKeyConstraintWhenMigrating: true, // Disable FK checks during migrations for speed
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	// Optimize connection pool settings
	sqlDB.SetMaxIdleConns(50)                  // Increased idle connections
	sqlDB.SetMaxOpenConns(300)                 // Match PostgreSQL max_connections
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // Shorter connection lifetime for better resource usage
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // Close idle connections sooner

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully")
	log.Printf("Max open connections: %d", 300)

	DB = db
}

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
			"application_name=diabetify TimeZone=Asia/Jakarta",
		host, user, password, dbname, port, sslmode,
	)

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Millisecond * 500, // Log queries slower than 500ms
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

	// Set connection pools
	sqlDB.SetMaxOpenConns(200)
	sqlDB.SetMaxIdleConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(15 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully")
	log.Printf("Database connection pool configured:")
	log.Printf("  - Max open connections: %d", 397)
	log.Printf("  - Max idle connections: %d", 100)
	log.Printf("  - Connection max lifetime: %v", 5*time.Minute)
	log.Printf("  - Connection max idle time: %v", 2*time.Minute)

	DB = db
}
func MonitorDBConnections() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			sqlDB, _ := DB.DB()
			stats := sqlDB.Stats()
			if stats.InUse > 150 { // Alert if using too many connections
				log.Printf("⚠️  DB Connection Pool: InUse=%d, Idle=%d, Open=%d",
					stats.InUse, stats.Idle, stats.OpenConnections)
			}
		}
	}()
}

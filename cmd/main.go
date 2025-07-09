package main

import (
	"context"
	"diabetify/database"
	"diabetify/internal/controllers"
	"diabetify/internal/ml"
	"diabetify/internal/repository"
	"diabetify/routes"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"diabetify/docs"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Check if sharding is enabled
	useSharding := os.Getenv("USE_SHARDING") == "true"
	log.Printf("Sharding mode: %v", useSharding)

	// Swagger Documentation
	docs.SwaggerInfo.Title = "Diabetify API"
	docs.SwaggerInfo.Description = "This is api of Diabetify App with gRPC ML Service."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	// Connect to database based on sharding configuration
	if useSharding {
		database.ConnectShardedDatabase()
		if err := database.MigrateAllShards(); err != nil {
			log.Fatalf("Failed to run database migrations: %v", err)
		}
		database.MonitorShardedDBConnections()
	} else {
		database.ConnectDatabase()
		if err := database.MigrateDatabase(); err != nil {
			log.Fatalf("Failed to run database migrations: %v", err)
		}
		database.MonitorDBConnections()
	}

	// Initialize repositories based on sharding configuration
	var (
		forgotPasswordRepo repository.ResetPasswordRepository
		userRepo           repository.UserRepository
		verificationRepo   repository.VerificationRepository
		activityRepo       repository.ActivityRepository
		profileRepo        repository.UserProfileRepository
		predictionRepo     repository.PredictionRepository
	)

	if useSharding {
		// Use sharded repositories
		forgotPasswordRepo = repository.NewResetPasswordRepository(nil)
		userRepo = repository.NewUserRepository(nil)
		verificationRepo = repository.NewVerificationRepository(nil)
		activityRepo = repository.NewShardedActivityRepository()
		profileRepo = repository.NewShardedUserProfileRepository()
		predictionRepo = repository.NewShardedPredictionRepository()
		log.Println("Initialized sharded repositories")
	} else {
		// Use single database repositories
		forgotPasswordRepo = repository.NewResetPasswordRepository(database.DB)
		userRepo = repository.NewUserRepository(database.DB)
		verificationRepo = repository.NewVerificationRepository(database.DB)
		activityRepo = repository.NewActivityRepository(database.DB)
		profileRepo = repository.NewUserProfileRepository(database.DB)
		predictionRepo = repository.NewPredictionRepository(database.DB)
		log.Println("Initialized single database repositories")
	}

	// Article repository (assuming it doesn't need sharding)
	articleRepo := repository.NewArticleRepository(database.DB)

	// Initialize ML gRPC client
	mlServiceAddress := os.Getenv("ML_SERVICE_ADDRESS")
	if mlServiceAddress == "" {
		mlServiceAddress = "localhost:50051"
	}

	log.Printf("Connecting to ML service via gRPC at %s...", mlServiceAddress)
	mlClient, err := ml.NewGRPCMLClient(mlServiceAddress)
	if err != nil {
		log.Fatal("Failed to create ML gRPC client:", err)
	}
	defer mlClient.Close()

	// Test ML service connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := mlClient.HealthCheck(ctx); err != nil {
		log.Printf("Warning: ML service health check failed: %v", err)
		log.Println("The application will start, but predictions will fail until ML service is available")
	} else {
		log.Println("ML service gRPC connection established successfully")
	}

	// Initialize controllers
	userController := controllers.NewUserController(userRepo, forgotPasswordRepo)
	verificationController := controllers.NewVerificationController(verificationRepo, userRepo)
	oauthController := controllers.NewOauthController(userRepo)
	activityController := controllers.NewActivityController(activityRepo)
	articleController := controllers.NewArticleController(articleRepo)
	profileController := controllers.NewUserProfileController(profileRepo)

	// Updated prediction controller with all required repositories
	predictionController := controllers.NewPredictionController(
		predictionRepo,
		userRepo,
		profileRepo,
		activityRepo,
		mlClient,
	)

	gin.SetMode(gin.ReleaseMode)
	// Setup Gin router
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		response := gin.H{
			"message":    "Diabetify API is running",
			"version":    "1.0.0",
			"status":     "healthy",
			"ml_service": "gRPC",
			"prediction": "Auto-prediction from user profile",
		}

		if useSharding {
			response["database"] = "Sharded PostgreSQL"
			response["shards"] = []string{"shard1 (users 1-5000)", "shard2 (users 5001-10000)"}
		} else {
			response["database"] = "Single PostgreSQL"
		}

		c.JSON(200, response)
	})

	// Register routes
	routes.RegisterUserRoutes(router, userController)
	routes.RegisterVerificationRoutes(router, verificationController)
	routes.RegisterSwaggerRoutes(router)
	routes.RegisterOauthRoutes(router, oauthController)
	routes.RegisterActivityRoutes(router, activityController)
	routes.RegisterArticleRoutes(router, articleController)
	routes.RegisterUserProfileRoutes(router, profileController)
	routes.RegisterPredictionRoutes(router, predictionController)

	// Debug endpoints
	router.GET("/debug/stats", func(c *gin.Context) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		c.JSON(200, gin.H{
			"goroutines": runtime.NumGoroutine(),
			"memory_mb":  m.Alloc / 1024 / 1024,
		})
	})

	// Conditional shard health check endpoint
	if useSharding {
		router.GET("/debug/shards", func(c *gin.Context) {
			shardsHealth := database.CheckShardsHealth()
			c.JSON(200, gin.H{
				"shards_health": shardsHealth,
				"total_shards":  len(shardsHealth),
			})
		})
	} else {
		router.GET("/debug/database", func(c *gin.Context) {
			// Simple database health check for single DB
			sqlDB, err := database.DB.DB()
			if err != nil {
				c.JSON(500, gin.H{
					"database_health": false,
					"mode":            "single_database",
					"error":           err.Error(),
				})
				return
			}

			var result int
			row := sqlDB.QueryRowContext(c.Request.Context(), "SELECT 1")
			err = row.Scan(&result)
			isHealthy := err == nil && result == 1

			c.JSON(200, gin.H{
				"database_health": isHealthy,
				"mode":            "single_database",
			})
		})
	}

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)

	if useSharding {
		log.Printf("Database Shards Health: http://localhost:%s/debug/shards", port)
	} else {
		log.Printf("Database Health: http://localhost:%s/debug/database", port)
	}

	server := &http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Printf("Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

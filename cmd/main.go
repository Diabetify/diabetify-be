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

	// Swagger Documentation
	docs.SwaggerInfo.Title = "Diabetify API"
	docs.SwaggerInfo.Description = "This is api of Diabetify App with gRPC ML Service."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	// Connect to the sharded databases
	database.ConnectShardedDatabase()

	if err := database.MigrateAllShards(); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	database.MonitorShardedDBConnections()

	// Initialize repositories with sharding
	forgotPasswordRepo := repository.NewShardedResetPasswordRepository()
	userRepo := repository.NewShardedUserRepository()
	verificationRepo := repository.NewShardedVerificationRepository()
	activityRepo := repository.NewShardedActivityRepository()
	articleRepo := repository.NewArticleRepository(database.DB) // Article might not need sharding if not user-specific
	profileRepo := repository.NewShardedUserProfileRepository()
	predictionRepo := repository.NewShardedPredictionRepository()

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
		log.Println("✅ ML service gRPC connection established successfully")
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
		predictionRepo, // Prediction repository
		userRepo,       // User repository for getting DOB
		profileRepo,    // Profile repository for getting user profile data
		activityRepo,   // Activity repository for calculating Brinkman index and physical activity
		mlClient,       // gRPC ML client
	)

	gin.SetMode(gin.ReleaseMode)
	// Setup Gin router
	router := gin.Default()

	// Add a root endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":    "Diabetify API is running",
			"version":    "1.0.0",
			"status":     "healthy",
			"ml_service": "gRPC",
			"prediction": "Auto-prediction from user profile",
			"database":   "Sharded PostgreSQL",
			"shards":     []string{"shard1 (users 1-5000)", "shard2 (users 5001-10000)"},
		})
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

	// Shard health check endpoint
	router.GET("/debug/shards", func(c *gin.Context) {
		shardsHealth := database.CheckShardsHealth()
		c.JSON(200, gin.H{
			"shards_health": shardsHealth,
			"total_shards":  len(shardsHealth),
		})
	})

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	log.Printf("📋 API Documentation available at http://localhost:%s/swagger/index.html", port)
	log.Printf("🔗 ML Health check: http://localhost:%s/prediction/predict/health", port)
	log.Printf("🗄️  Database Shards Health: http://localhost:%s/debug/shards", port)

	server := &http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Printf("Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

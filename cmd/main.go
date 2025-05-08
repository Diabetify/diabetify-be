package main

import (
	"diabetify/database"
	"diabetify/internal/controllers"
	"diabetify/internal/repository"
	"diabetify/routes"
	"log"

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
	docs.SwaggerInfo.Description = "This is api of Diabetify App."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	// Connect to the database
	database.ConnectDatabase()
	db := database.DB
	if err := database.MigrateDatabase(); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	forgotPasswordRepo := repository.NewResetPasswordRepository(db)
	userRepo := repository.NewUserRepository(db)
	userController := controllers.NewUserController(userRepo, forgotPasswordRepo)

	verificationRepo := repository.NewVerificationRepository(db)
	verificationController := controllers.NewVerificationController(verificationRepo, userRepo)

	oauthController := controllers.NewOauthController(userRepo)
	router := gin.Default()

	activityRepo := repository.NewActivityRepository(db)
	activityController := controllers.NewActivityController(activityRepo)

	articleRepo := repository.NewArticleRepository(db)
	articleController := controllers.NewArticleController(articleRepo)

	profileRepo := repository.NewUserProfileRepository(db)
	profileController := controllers.NewUserProfileController(profileRepo)

	// Register user routes
	routes.RegisterUserRoutes(router, userController)
	routes.RegisterVerificationRoutes(router, verificationController)
	routes.RegisterSwaggerRoutes(router)
	routes.RegisterOauthRoutes(router, oauthController)
	routes.RegisterActivityRoutes(router, activityController)
	routes.RegisterArticleRoutes(router, articleController)
	routes.RegisterUserProfileRoutes(router, profileController)

	// Start the server
	log.Println("Server is running on port 8080...")
	router.Run(":8080")
}

package main

import (
	"diabetify/database"
	"diabetify/internal/controllers"
	"diabetify/internal/repository"
	"diabetify/routes"
	"log"
	"os"

	"diabetify/docs"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
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

	// Goth Provider
	goth.UseProviders(
		google.New(os.Getenv("GOOGLE_KEY"), os.Getenv("GOOGLE_SECRET"), os.Getenv("GOOGLE_CALLBACK_URL"), "email", "profile"),
	)
	forgotPasswordRepo := repository.NewResetPasswordRepository()
	userRepo := repository.NewUserRepository()
	userController := controllers.NewUserController(userRepo, forgotPasswordRepo)

	verificationRepo := repository.NewVerificationRepository()
	verificationController := controllers.NewVerificationController(verificationRepo, userRepo)

	oauthController := controllers.NewOauthController(userRepo)
	router := gin.Default()

	// Register user routes
	routes.RegisterUserRoutes(router, userController)
	routes.RegisterVerificationRoutes(router, verificationController)
	routes.RegisterSwaggerRoutes(router)
	routes.RegisterOauthRoutes(router, oauthController)

	// Start the server
	log.Println("Server is running on port 8080...")
	router.Run(":8080")
}

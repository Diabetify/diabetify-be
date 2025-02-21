package main

import (
	"diabetify/database"
	"diabetify/internal/controllers"
	"diabetify/internal/repository"
	"diabetify/routes"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// Connect to the database
	database.ConnectDatabase()

	userRepo := repository.NewUserRepository()
	userController := controllers.NewUserController(userRepo)

	router := gin.Default()

	// Register user routes
	routes.RegisterUserRoutes(router, userController)

	// Start the server
	log.Println("Server is running on port 8080...")
	router.Run(":8080")
}

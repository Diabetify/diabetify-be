package routes

import (
	"diabetify/internal/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(router *gin.Engine, userController *controllers.UserController) {
	userRoutes := router.Group("/users")
	{
		userRoutes.POST("/", userController.CreateUser)
		userRoutes.GET("/:id", userController.GetUserByID)
		userRoutes.GET("/email/:email", userController.GetUserByEmail)
		userRoutes.PUT("/:id", userController.UpdateUser)
		userRoutes.DELETE("/:id", userController.DeleteUser)

		// Auth
		userRoutes.POST("/login", userController.LoginUser)
		userRoutes.POST("/reset-password", userController.ForgotPassword)
		userRoutes.POST("/change-password", userController.ResetPassword)
	}
}

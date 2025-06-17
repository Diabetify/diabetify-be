package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(router *gin.Engine, userController *controllers.UserController) {
	userRoutesPublic := router.Group("/users")
	{
		userRoutesPublic.POST("/", userController.CreateUser)
		userRoutesPublic.POST("/login", userController.LoginUser)
		userRoutesPublic.POST("/forgot-password", userController.ForgotPassword)
		userRoutesPublic.POST("/reset-password", userController.ResetPassword)
	}
	userRoutesPrivate := router.Group("/users")
	userRoutesPrivate.Use(middleware.AuthMiddleware())
	{
		userRoutesPrivate.GET("/me", userController.GetCurrentUser)
		userRoutesPrivate.PUT("/me", userController.UpdateUser)
		userRoutesPrivate.PATCH("/me", userController.PatchUser)
	}
}

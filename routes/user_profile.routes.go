package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterUserProfileRoutes(router *gin.Engine, userProfileController *controllers.UserProfileController) {
	profileRoutes := router.Group("/profile")
	profileRoutes.Use(middleware.AuthMiddleware())
	{
		profileRoutes.GET("/", userProfileController.GetUserProfile)
		profileRoutes.POST("/", userProfileController.CreateUserProfile)
		profileRoutes.PUT("/", userProfileController.UpdateUserProfile)
		profileRoutes.DELETE("/", userProfileController.DeleteUserProfile)
		profileRoutes.PATCH("/", userProfileController.PatchUserProfile)
	}
}

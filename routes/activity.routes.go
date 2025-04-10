package routes

import (
	"diabetify/internal/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterActivityRoutes(router *gin.Engine, activityController *controllers.ActivityController) {
	userRoutes := router.Group("/activity")
	{
		userRoutes.POST("/", activityController.CreateActivity)
		userRoutes.GET("/user/:user_id", activityController.GetActivitiesByUserID)
		userRoutes.GET("/:id", activityController.GetActivityByID)
		userRoutes.PUT("/:id", activityController.UpdateActivity)
		userRoutes.DELETE("/:id", activityController.DeleteActivity)
	}
}

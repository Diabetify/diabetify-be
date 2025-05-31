package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterActivityRoutes(router *gin.Engine, activityController *controllers.ActivityController) {
	activityRoutes := router.Group("/activity")
	activityRoutes.Use(middleware.AuthMiddleware())
	{
		activityRoutes.POST("/", activityController.CreateActivity)
		activityRoutes.GET("/:id", activityController.GetActivityByID)
		activityRoutes.PUT("/:id", activityController.UpdateActivity)
		activityRoutes.DELETE("/:id", activityController.DeleteActivity)
		activityRoutes.GET("/me", activityController.GetCurrentUserActivities)
		activityRoutes.GET("/me/date-range", activityController.GetActivitiesByDateRange)
		activityRoutes.GET("/me/count", activityController.CountUserActivities)
	}
}

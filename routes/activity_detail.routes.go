package routes

import (
	"diabetify/internal/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterActivityDetailRoutes(router *gin.Engine, activityDetailController *controllers.ActivityDetailController) {
	activityDetailRoutes := router.Group("/activity-detail")
	{
		activityDetailRoutes.POST("/", activityDetailController.CreateActivityDetail)
		activityDetailRoutes.GET("/activity/:activity_id", activityDetailController.GetActivityDetailsByActivityID)
		activityDetailRoutes.GET("/:id", activityDetailController.GetActivityDetailByID)
		activityDetailRoutes.PUT("/:id", activityDetailController.UpdateActivityDetail)
		activityDetailRoutes.DELETE("/:id", activityDetailController.DeleteActivityDetail)
		activityDetailRoutes.DELETE("/activity/:activity_id", activityDetailController.DeleteActivityDetailsByActivityID)
	}
}

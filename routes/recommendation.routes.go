package routes

import (
	"diabetify/internal/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterRecommendationRoutes(router *gin.Engine, recommendationController *controllers.RecommendationController) {
	userRoutes := router.Group("/recommendation")
	{
		userRoutes.POST("/", recommendationController.CreateRecommendation)
		userRoutes.GET("/user/:user_id", recommendationController.GetRecommendationsByUserID)
		userRoutes.GET("/:id", recommendationController.GetRecommendationByID)
		userRoutes.PUT("/:id", recommendationController.UpdateRecommendation)
		userRoutes.DELETE("/:id", recommendationController.DeleteRecommendation)
	}
}

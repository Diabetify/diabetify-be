package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPredictionRoutes(router *gin.Engine, predictionController *controllers.PredictionController) {
	predictionRoutes := router.Group("/prediction")
	predictionRoutes.GET("/predict/health", predictionController.TestMLConnection)
	predictionRoutes.Use(middleware.AuthMiddleware())
	{
		predictionRoutes.POST("/", predictionController.MakePrediction)
		predictionRoutes.GET("/:id", predictionController.GetPredictionByID)
		predictionRoutes.DELETE("/:id", predictionController.DeletePrediction)
		predictionRoutes.GET("/me", predictionController.GetUserPredictions)
		predictionRoutes.GET("/me/date-range", predictionController.GetPredictionsByDateRange)
		predictionRoutes.GET("/me/score", predictionController.GetPredictionScoreByDate)
		predictionRoutes.GET("/me/explanation", predictionController.GetLatestPredictionExplanation)
	}
}

package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPredictionRoutes(router *gin.Engine, predictionController *controllers.PredictionController) {
	predictionRoutes := router.Group("/prediction")
	predictionRoutes.GET("/health", predictionController.TestMLConnection)
	predictionRoutes.Use(middleware.AuthMiddleware())
	{
		predictionRoutes.POST("/", predictionController.MakePrediction)
		predictionRoutes.POST("/what-if", predictionController.WhatIfPrediction)

		predictionRoutes.GET("/job/:job_id/status", predictionController.GetJobStatus)
		predictionRoutes.GET("/job/:job_id/result", predictionController.GetJobResult)
		predictionRoutes.POST("/job/:job_id/cancel", predictionController.CancelJob)
		predictionRoutes.GET("/jobs", predictionController.GetUserJobs)

		predictionRoutes.GET("/:id", predictionController.GetPredictionByID)
		predictionRoutes.DELETE("/:id", predictionController.DeletePrediction)

		predictionRoutes.GET("/me", predictionController.GetUserPredictions)
		predictionRoutes.GET("/me/date-range", predictionController.GetPredictionsByDateRange)
		predictionRoutes.GET("/me/score", predictionController.GetPredictionScoreByDate)
		predictionRoutes.GET("/me/explanation", predictionController.GetLatestPredictionExplanation)
	}
}

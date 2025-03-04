package routes

import (
	"diabetify/internal/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterVerificationRoutes(router *gin.Engine, verificationController *controllers.VerificationController) {
	verificationRoutes := router.Group("/verification")
	{
		verificationRoutes.POST("/get-code", verificationController.SendVerificationCode)
		verificationRoutes.POST("/verify", verificationController.VerifyCode)
		verificationRoutes.GET("/resend", verificationController.ResendVerificationCode)
	}
}

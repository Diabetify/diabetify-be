package routes

import (
	"diabetify/internal/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterOauthRoutes(router *gin.Engine, oauthController *controllers.OauthController) {
	oauthRoutes := router.Group("/oauth")
	{
		// Google routes
		oauthRoutes.POST("/auth/google", oauthController.GoogleAuth)
	}
}

package routes

import (
	"diabetify/internal/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterOauthRoutes(router *gin.Engine, oauthController *controllers.OauthController) {
	oauthRoutes := router.Group("/oauth")
	{
		oauthRoutes.GET("/auth/google/callback", oauthController.GoogleCallback)
		oauthRoutes.GET("/logout/google", oauthController.Logout)
		oauthRoutes.GET("/auth/google", oauthController.GoogleAuth)
	}
}

package routes

import (
	"diabetify/internal/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterArticleRoutes(router *gin.Engine, articleController *controllers.ArticleController) {
	articleRoutes := router.Group("/article")
	{
		articleRoutes.POST("/", articleController.CreateArticle)
		articleRoutes.GET("/", articleController.GetAllArticles)
		articleRoutes.GET("/:id", articleController.GetArticleByID)
		articleRoutes.PUT("/:id", articleController.UpdateArticle)
		articleRoutes.DELETE("/:id", articleController.DeleteArticle)
	}
}

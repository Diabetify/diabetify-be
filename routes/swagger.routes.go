package routes

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterSwaggerRoutes sets up the Swagger route
func RegisterSwaggerRoutes(router *gin.Engine) {
	// Configure Swagger with options to enable the authorization button
	swaggerHandler := ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/swagger/doc.json"),
		ginSwagger.PersistAuthorization(true),
		ginSwagger.DocExpansion("none"),
	)

	router.GET("/swagger/*any", swaggerHandler)
}

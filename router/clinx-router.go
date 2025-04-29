package router

import (
	"github.com/gin-gonic/gin"
	"one-api/controller"
	"one-api/middleware"
)

func SetClinxRouter(router *gin.Engine) {
	router.Use(middleware.CORS())
	router.Use(middleware.DecompressRequestMiddleware())
	clinxRouter := router.Group("/clinx/v1")
	clinxRouter.POST("/modelList", controller.ModelList)

	relayV1Router := clinxRouter.Group("")
	relayV1Router.Use(middleware.TokenAuth())
	relayV1Router.Use(middleware.ModelRequestRateLimit())
	httpRouter := relayV1Router.Group("")
	httpRouter.Use(middleware.Distribute())
	httpRouter.POST("/chat/completions", controller.Completions)
}

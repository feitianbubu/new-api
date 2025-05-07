package router

import (
	"github.com/gin-gonic/gin"
	"one-api/controller"
	"one-api/middleware"
)

func SetClinxRouter(router *gin.Engine) {
	router.Use(middleware.CORS())
	router.Use(middleware.DecompressRequestMiddleware())
	clinxRouter := router.Group("/clinx")
	clinxRouter.POST("/modelList", controller.ModelList)
	clinxRouter.GET("/mj/image/:id", controller.RelayMidjourneyImage)
	clinxRouter.Use(middleware.ModelRequestRateLimit())

	httpRouter := clinxRouter.Group("/v1")
	httpRouter.Use(middleware.TokenAuth())
	httpRouter.Use(middleware.Distribute())
	httpRouter.POST("/chat/completions", controller.Completions)
	httpRouter.POST("/images/generations", controller.Generations)

	relayMjRouter := clinxRouter.Group("/mj")
	relayMjRouter.Use(middleware.TokenAuth(), middleware.Distribute())
	relayMjRouter.POST("/submit/imagine", controller.SubmitImagine)
}

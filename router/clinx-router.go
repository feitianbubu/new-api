package router

import (
	"github.com/gin-gonic/gin"
	"one-api/controller"
	"one-api/middleware"
)

func SetClinxRouter(router *gin.Engine) {
	router.Use(middleware.CORS())
	router.Use(middleware.DecompressRequestMiddleware())

	legacyRouter := router.Group("")
	legacyRouter.GET("/providers/providersList", controller.ProvidersList)
	legacyRouter.GET("/providers/modelsList/:provider", controller.ModelList)
	legacyRouter.GET("/providers/modelsList", controller.ModelList)
	legacyChatRouter := router.Group("/v1/openai")
	legacyChatRouter.Use(middleware.TokenAuth())
	legacyChatRouter.Use(middleware.Distribute())
	legacyChatRouter.POST("/:provider/chat/completions", controller.Completions)

	clinxRouter := router.Group("/clinx")
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

	SetBillsRouter(clinxRouter)
}

package router

import (
	"github.com/QuantumNous/new-api/common/registry"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func init() {
	registry.RegisterRouter(setOIDCProviderRouter)
}

func setOIDCProviderRouter(router *gin.Engine) {
	// OIDC Provider 标准端点
	router.GET("/.well-known/openid-configuration", controller.OIDCWellKnown)
	router.GET("/.well-known/openid_configuration", controller.OIDCWellKnown)
	router.GET("/oauth/jwks", controller.OIDCJWKS)
	router.GET("/oauth/authorize", controller.OIDCAuthorize)
	router.POST("/oauth/token", controller.OIDCToken)
	router.GET("/oauth/userinfo", controller.OIDCUserInfo)

	// OIDC Provider 管理API (需要管理员权限)
	oidcProviderRouter := router.Group("/api/oidc_provider")
	oidcProviderRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	oidcProviderRouter.Use(middleware.GlobalAPIRateLimit())
	oidcProviderRouter.Use(middleware.RootAuth())
	{
		oidcProviderRouter.POST("/clients", controller.CreateOIDCClient)
		oidcProviderRouter.GET("/clients", controller.GetAllOIDCClients)
		oidcProviderRouter.GET("/clients/:id", controller.GetOIDCClient)
		oidcProviderRouter.PUT("/clients/:id", controller.UpdateOIDCClient)
		oidcProviderRouter.DELETE("/clients/:id", controller.DeleteOIDCClient)
	}
}

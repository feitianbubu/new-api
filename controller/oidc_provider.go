package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// OIDCWellKnown OIDC发现端点
// @Summary 发现端点
// @Description OIDC Well-Known Configuration Provider
// @Tags OIDC
// @Accept json
// @Produce json
// @Router /.well-known/openid-configuration [get]
func OIDCWellKnown(c *gin.Context) {
	if !system_setting.GetOIDCProviderSettings().Enabled {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "OIDC Provider is disabled",
		})
		return
	}

	baseURL := system_setting.ServerAddress
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	config := map[string]interface{}{
		"issuer":                 baseURL,
		"authorization_endpoint": baseURL + "oauth/authorize",
		"token_endpoint":         baseURL + "oauth/token",
		"userinfo_endpoint":      baseURL + "oauth/userinfo",
		"end_session_endpoint":   baseURL + "api/user/logout",
		"jwks_uri":               baseURL + "oauth/jwks",
		"scopes_supported": []string{
			"openid",
			"profile",
			"email",
		},
		"response_types_supported": []string{
			"code",
		},
		"grant_types_supported": []string{
			"authorization_code",
			"refresh_token",
		},
		"subject_types_supported": []string{
			"public",
		},
		"id_token_signing_alg_values_supported": []string{
			"RS256",
		},
		"token_endpoint_auth_methods_supported": []string{
			"client_secret_basic",
			"client_secret_post",
		},
	}

	c.JSON(http.StatusOK, config)
}

// OIDCJWKS JSON Web Key Set端点
// @Summary 获取公钥
// @Description JWK Set for token verification
// @Tags OIDC
// @Accept json
// @Produce json
// @Router /oauth/jwks [get]
func OIDCJWKS(c *gin.Context) {
	if !system_setting.GetOIDCProviderSettings().Enabled {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "OIDC Provider is disabled",
		})
		return
	}

	jwks, err := model.GetOIDCJWKS()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get JWK Set",
		})
		return
	}

	c.JSON(http.StatusOK, jwks)
}

// OIDCAuthorize OIDC授权端点
// @Summary 发起授权
// @Description OIDC Authorization Request Handler
// @Tags OIDC
// @Accept json
// @Produce html
// @Param response_type query string true "response_type"
// @Param client_id query string true "client_id"
// @Param redirect_uri query string true "redirect_uri"
// @Param scope query string true "scope"
// @Param state query string false "state"
// @Param nonce query string false "nonce"
// @Router /oauth/authorize [get]
func OIDCAuthorize(c *gin.Context) {
	if !system_setting.GetOIDCProviderSettings().Enabled {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "OIDC Provider is disabled",
		})
		return
	}

	// 获取授权参数
	responseType := c.Query("response_type")
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	scope := c.Query("scope")
	state := c.Query("state")
	nonce := c.Query("nonce")

	// 验证必需参数
	if responseType != "code" {
		redirectError(c, redirectURI, "unsupported_response_type", "Only authorization_code flow is supported", state)
		return
	}

	if clientID == "" {
		redirectError(c, redirectURI, "invalid_request", "client_id is required", state)
		return
	}

	if redirectURI == "" {
		redirectError(c, redirectURI, "invalid_request", "redirect_uri is required", state)
		return
	}

	// 验证客户端
	client, err := model.GetOIDCClientByClientId(clientID)
	if err != nil {
		redirectError(c, redirectURI, "invalid_client", "Invalid client", state)
		return
	}

	// 验证redirect_uri
	var redirectUris []string
	if err := json.Unmarshal([]byte(client.RedirectUris), &redirectUris); err != nil {
		redirectError(c, redirectURI, "server_error", "Failed to parse redirect URIs", state)
		return
	}

	//validURI := false
	//for _, uri := range redirectUris {
	//	if uri == redirectURI {
	//		validURI = true
	//		break
	//	}
	//}
	//
	//if !validURI {
	//	redirectError(c, redirectURI, "invalid_request", "Invalid redirect URI", state)
	//	return
	//}

	// 验证scope
	if scope == "" {
		redirectError(c, redirectURI, "invalid_request", "scope is required", state)
		return
	}

	scopes := strings.Split(scope, " ")
	hasOpenID := false
	for _, s := range scopes {
		if s == "openid" {
			hasOpenID = true
			break
		}
	}

	if !hasOpenID {
		redirectError(c, redirectURI, "invalid_scope", "Scope must include openid", state)
		return
	}

	// 检查用户是否已登录
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		// 用户未登录，重定向到登录页面
		// 可以存储授权信息到session中，登录后继续
		oauthData := map[string]interface{}{
			"client_id":    clientID,
			"redirect_uri": redirectURI,
			"scope":        scope,
			"state":        state,
			"nonce":        nonce,
		}
		jsonData, _ := json.Marshal(oauthData)
		session.Set("oauth_pending", string(jsonData))
		if err := session.Save(); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message": "无法保存会话信息，请重试",
				"success": false,
			})
			return
		}

		// 重定向到登录页面
		loginURL := "/login?redirect=" + url.QueryEscape(c.Request.URL.String())
		c.Redirect(http.StatusFound, loginURL)
		return
	}

	// 用户已登录，生成JWT格式的授权码
	userID := session.Get("id")
	var userIDInt64 int64
	if uid, ok := userID.(int); ok {
		userIDInt64 = int64(uid)
	} else if uid, ok := userID.(float64); ok {
		userIDInt64 = int64(uid)
	} else {
		redirectError(c, redirectURI, "server_error", "Invalid user ID in session", state)
		return
	}

	authCode, err := model.GenerateOIDCAuthorizationCode(userIDInt64, clientID, redirectURI, scope, nonce)
	if err != nil {
		redirectError(c, redirectURI, "server_error", "Failed to generate authorization code", state)
		return
	}

	// 构造回调URL
	callbackURL, _ := url.Parse(redirectURI)
	params := callbackURL.Query()
	params.Set("code", authCode)
	if state != "" {
		params.Set("state", state)
	}
	callbackURL.RawQuery = params.Encode()

	c.Redirect(http.StatusFound, callbackURL.String())
}

// OIDCToken OIDC令牌端点
// @Summary OIDCToken
// @Summary 交换令牌
// @Description OIDC Token Request Handler
// @Tags OIDC
// @Accept json
// @Produce json
// @Param grant_type formData string true "grant_type (authorization_code | refresh_token)"
// @Param code formData string false "授权码（grant_type=authorization_code 时必填）"
// @Param refresh_token formData string false "刷新令牌（grant_type=refresh_token 时必填）"
// @Param redirect_uri formData string false "redirect_uri（授权码流程时需与授权请求一致）"
// @Param client_id formData string true "client_id"
// @Param client_secret formData string true "client_secret"
// @Param token_ttl formData string false "AccessToken 有效期，默认使用客户端配置的TTL。支持格式: 30m, 2h, 1h30m 等。范围: 1h-720h" example(2h)
// @Router /oauth/token [post]
func OIDCToken(c *gin.Context) {
	if !system_setting.GetOIDCProviderSettings().Enabled {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "OIDC Provider is disabled",
		})
		return
	}
	clientID := c.PostForm("client_id")
	clientSecret := c.PostForm("client_secret")
	client, err := model.GetOIDCClientByClientId(clientID)
	if err != nil || client.ClientSecret != clientSecret {
		logger.LogWarn(c, fmt.Sprintf("OIDC client authentication failed for client_id: %s, client_secret: %s", clientID, clientSecret))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid clientId or clientSecret",
		})
		return
	}
	grantType := c.DefaultPostForm("grant_type", "authorization_code")
	if grantType == "" {
		if rt := c.PostForm("refresh_token"); rt != "" {
			FailWithStatus(c, http.StatusBadRequest, "grant_type is required and must be refresh_token when using refresh_token")
			return
		}
		if code := c.PostForm("code"); code != "" {
			FailWithStatus(c, http.StatusBadRequest, "grant_type is required and must be authorization_code when using code")
			return
		}
		FailWithStatus(c, http.StatusBadRequest, "grant_type is required and must be authorization_code or refresh_token")
		return
	}
	switch grantType {
	case "authorization_code":
		handleAuthorizationCodeGrant(c, client)
	case "refresh_token":
		handleRefreshTokenGrant(c, client)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unsupported_grant_type",
		})
	}
}

// OIDCUserInfo OIDC用户信息端点
// @Summary 用户信息
// @Description OIDC User Information Handler
// @Tags OIDC
// @Accept json
// @Produce json
// @Router /oauth/userinfo [get]
func OIDCUserInfo(c *gin.Context) {
	if !system_setting.GetOIDCProviderSettings().Enabled {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "OIDC Provider is disabled",
		})
		return
	}

	// 从Authorization header获取访问令牌
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid_token",
		})
		return
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := model.ValidateOIDCToken(accessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid_token",
		})
		return
	}

	// 获取用户信息
	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "server_error",
		})
		return
	}

	user, err := model.GetUserById(userID, false)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "user_not_found",
		})
		return
	}

	// 返回用户信息
	userInfo := map[string]interface{}{
		"sub":                claims.Subject,
		"username":           user.Username,
		"preferred_username": user.Username,
	}

	if user.DisplayName != "" {
		userInfo["name"] = user.DisplayName
	}

	if user.Email != "" {
		userInfo["email"] = user.Email
		userInfo["email_verified"] = true
	}

	c.JSON(http.StatusOK, userInfo)
}

func redirectError(c *gin.Context, redirectURI, error, description, state string) {
	if redirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             error,
			"error_description": description,
		})
		return
	}

	callbackURL, _ := url.Parse(redirectURI)
	params := callbackURL.Query()
	params.Set("error", error)
	params.Set("error_description", description)
	if state != "" {
		params.Set("state", state)
	}
	callbackURL.RawQuery = params.Encode()

	c.Redirect(http.StatusFound, callbackURL.String())
}

func handleAuthorizationCodeGrant(c *gin.Context, client *model.OidcClient) {
	code := c.PostForm("code")
	token, err := model.ValidateUserToken(code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	user, err := model.GetUserById(token.UserId, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "server_error",
		})
		return
	}
	tokenTTL := client.GetTokenTTL()
	if tokenTTLStr := c.PostForm("token_ttl"); tokenTTLStr != "" {
		tokenTTL = common.ParseTokenTTL(tokenTTLStr, tokenTTL)
	}
	accessToken, _, err := model.GenerateOIDCAccessToken(user, client.ClientId, tokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "server_error",
		})
		return
	}
	idToken, err := model.GenerateOIDCIDToken(user, client.ClientId, tokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "server_error",
		})
		return
	}

	refreshToken, err := model.CreateRefreshTokenWithClient(user, client.ClientId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "server_error",
		})
		return
	}
	response := map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    tokenTTL,
		"refresh_token": refreshToken,
		"id_token":      idToken,
	}

	c.JSON(http.StatusOK, response)
}

func handleRefreshTokenGrant(c *gin.Context, client *model.OidcClient) {
	refreshToken := c.PostForm("refresh_token")
	claims, err := model.ParseRefreshToken(refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid_grant",
		})
		return
	}

	if claims.ClientID != "" && claims.ClientID != client.ClientId {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid_grant",
		})
		return
	}

	user, err := model.GetUserById(claims.UserId, false)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid_grant",
		})
		return
	}

	tokenTTL := client.GetTokenTTL()
	if tokenTTLStr := c.PostForm("token_ttl"); tokenTTLStr != "" {
		tokenTTL = common.ParseTokenTTL(tokenTTLStr, tokenTTL)
	}

	accessToken, _, err := model.GenerateOIDCAccessToken(user, client.ClientId, tokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "server_error",
		})
		return
	}
	response := map[string]interface{}{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   tokenTTL,
	}

	c.JSON(http.StatusOK, response)
}

// OIDC客户端管理API

// CreateOIDCClient 创建OIDC客户端
// @Summary Create OIDC Client
// @Description Create a new OIDC client
// @Tags OIDC Provider
// @Accept json
// @Produce json
// @Param client body model.OidcClient true "OIDC Client"
// @Success 200 {object} model.OidcClient
// @Router /api/oidc_provider/clients [post]
func CreateOIDCClient(c *gin.Context) {
	var client model.OidcClient
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid parameters: " + err.Error(),
		})
		return
	}

	// 生成客户端ID和密钥
	clientId, err := common.GenerateKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to generate client ID: " + err.Error(),
		})
		return
	}
	clientSecret, err := common.GenerateKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to generate client secret: " + err.Error(),
		})
		return
	}

	client.ClientId = clientId
	client.ClientSecret = clientSecret
	client.Status = 1 // enabled

	if err := model.CreateOIDCClient(&client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create client: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Client created successfully",
		"data":    client,
	})
}

// GetAllOIDCClients 获取所有OIDC客户端
// @Summary Get All OIDC Clients
// @Description Get all OIDC clients
// @Tags OIDC Provider
// @Accept json
// @Produce json
// @Success 200 {array} model.OidcClient
// @Router /api/oidc_provider/clients [get]
func GetAllOIDCClients(c *gin.Context) {
	clients, err := model.GetAllOIDCClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get clients: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    clients,
	})
}

// GetOIDCClient 获取OIDC客户端
// @Summary Get OIDC Client
// @Description Get OIDC client by ID
// @Tags OIDC Provider
// @Accept json
// @Produce json
// @Param id path int true "Client ID"
// @Success 200 {object} model.OidcClient
// @Router /api/oidc_provider/clients/{id} [get]
func GetOIDCClient(c *gin.Context) {
	idStr := c.Param("id")
	_, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid client ID",
		})
		return
	}

	client, err := model.GetOIDCClientByClientId(idStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Client not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    client,
	})
}

// UpdateOIDCClient 更新OIDC客户端
// @Summary Update OIDC Client
// @Description Update OIDC client
// @Tags OIDC Provider
// @Accept json
// @Produce json
// @Param id path int true "Client ID"
// @Param client body model.OidcClient true "OIDC Client"
// @Success 200 {object} model.OidcClient
// @Router /api/oidc_provider/clients/{id} [put]
func UpdateOIDCClient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid client ID",
		})
		return
	}

	var client model.OidcClient
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid parameters: " + err.Error(),
		})
		return
	}

	// 确保ID匹配
	client.Id = id

	if err := model.UpdateOIDCClient(id, &client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update client: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Client updated successfully",
		"data":    client,
	})
}

// DeleteOIDCClient 删除OIDC客户端
// @Summary Delete OIDC Client
// @Description Delete OIDC client
// @Tags OIDC Provider
// @Accept json
// @Produce json
// @Param id path int true "Client ID"
// @Router /api/oidc_provider/clients/{id} [delete]
func DeleteOIDCClient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid client ID",
		})
		return
	}

	if err := model.DeleteOIDCClient(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to delete client: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Client deleted successfully",
	})
}

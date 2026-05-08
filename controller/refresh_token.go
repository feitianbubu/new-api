package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	TokenTTL     string `json:"token_ttl,omitempty"` // 可选，自定义新AccessToken过期时间
}

type RefreshResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresAt        int64  `json:"expires_at"`         // AccessToken过期时间戳
	RefreshExpiresAt int64  `json:"refresh_expires_at"` // RefreshToken过期时间戳
}

// RefreshToken
// @Summary 刷新令牌
// @Description 使用RefreshToken获取新的AccessToken和RefreshToken
// @Tags User
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "刷新令牌请求"
// @Success 200 {object} RefreshResponse "新的令牌信息"
// @Router /api/auth/refresh [post]
func RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 解析RefreshToken
	claims, err := model.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "无效的RefreshToken: " + err.Error(),
		})
		return
	}

	// 获取用户信息
	user, err := model.GetUserById(claims.UserId, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 检查用户状态
	if user.Status != common.UserStatusEnabled {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "用户已被禁用",
		})
		return
	}

	// 设置token_ttl参数到context中
	if req.TokenTTL != "" {
		c.Set("token_ttl", req.TokenTTL)
	}

	// 生成新的AccessToken
	newAccessToken, err := model.CreateUserJWT(c, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成AccessToken失败: " + err.Error(),
		})
		return
	}

	// 生成新的RefreshToken（轮换策略）
	newRefreshToken, err := model.CreateRefreshToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成RefreshToken失败: " + err.Error(),
		})
		return
	}

	// 计算AccessToken过期时间戳
	accessTokenDuration := time.Duration(common.SessionExpirationSeconds) * time.Second
	if req.TokenTTL != "" {
		ttlSeconds := common.ParseTokenTTL(req.TokenTTL, common.SessionExpirationSeconds)
		accessTokenDuration = time.Duration(ttlSeconds) * time.Second
	}

	now := time.Now()
	accessTokenExpiresAt := now.Add(accessTokenDuration).Unix()
	refreshTokenExpiresAt := now.Add(common.MaxTokenTTL).Unix()

	response := RefreshResponse{
		AccessToken:      newAccessToken,
		RefreshToken:     newRefreshToken,
		ExpiresAt:        accessTokenExpiresAt,
		RefreshExpiresAt: refreshTokenExpiresAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "令牌刷新成功",
		"data":    response,
	})
}

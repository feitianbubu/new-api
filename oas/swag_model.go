package oas

import "github.com/gin-gonic/gin"

// Models
// @Summary 模型列表
// @Description 获取当前用户可用的模型列表，支持OpenAI API格式
// @Tags OpenAI
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Router /v1/models [get]
func Models(c *gin.Context) {}

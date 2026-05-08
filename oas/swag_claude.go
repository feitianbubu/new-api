package oas

import (
	"github.com/gin-gonic/gin"
)

// ExampleClaudeRequest Claude 请求示例
type ExampleClaudeRequest struct {
	Model    string `json:"model" example:"claude-sonnet-4-5-20250929"`
	Messages []struct {
		Role    string `json:"role" example:"user"`
		Content string `json:"content" example:"今天上海的天气?"`
	} `json:"messages"`
	Tools []struct {
		Type    string `json:"type" example:"web_search_20250305"`
		Name    string `json:"name" example:"web_search"`
		MaxUses int    `json:"max_uses" example:"3"`
	} `json:"tools"`
}

// ClaudeMessages
// @Summary 发送消息
// @Description 使用 Anthropic Claude Messages API 格式发送消息。支持流式和非流式响应，可选的工具调用和思考模式。
// @Description
// @Description * Claude: https://platform.claude.com/docs/en/api/messages/create
// @Tags Claude
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param anthropic-version header string true "Anthropic API 版本" example(2023-06-01)"
// @Param x-api-key header string false "Anthropic API Key (可选，也可使用 Bearer Token)"
// @Param request body dto.ClaudeRequest true "Claude 消息请求体"
// @Success 200 {object} dto.ClaudeResponse "成功返回 Claude 响应"
// @Router /v1/messages [post]
func ClaudeMessages(c *gin.Context) {}

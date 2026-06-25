package oas

import "github.com/gin-gonic/gin"

// ChatCompletions swagger 聊天补全
// @Summary      聊天补全
// @description.markdown chat-completions
// @Tags         OpenAI
// @Accept       json
// @Produce      json
// @Produce      text/event-stream
// @Param        Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param        request body dto.GeneralOpenAIRequest true "OpenAI 请求体"
// @Success      200 {object} dto.OpenAITextResponse "非流式响应"
// @Success      200 {string} string "流式响应 (text/event-stream)"
// @Failure      400 {object} dto.OpenAIErrorWithStatusCode "无效的请求"
// @Failure      401 {object} dto.OpenAIErrorWithStatusCode "无效的认证"
// @Failure      403 {object} dto.OpenAIErrorWithStatusCode "用户或令牌额度不足"
// @Failure      500 {object} dto.OpenAIErrorWithStatusCode "内部服务器错误"
// @Router       /v1/chat/completions [post]
func ChatCompletions(c *gin.Context) {}

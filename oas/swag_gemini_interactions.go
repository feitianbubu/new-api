package oas

import "github.com/gin-gonic/gin"

// ExampleGeminiInteractionRequest Gemini Interactions 请求示例
type ExampleGeminiInteractionRequest struct {
	Input      string `json:"input" example:"test, reply in 100 tokens"`
	Agent      string `json:"agent" example:"deep-research-pro-preview-12-2025"`
	Background bool   `json:"background" example:"true"`
}

// ExampleGeminiInteractionResponse Gemini Interactions 响应示例
type ExampleGeminiInteractionResponse struct {
	ID      string `json:"id" example:"v1_Chd1d3U2YWZiZ0NwcUswLWtQNXBmMjBBVRIXdXd1NmFmYmdDcHFLMC1rUDVwZjIwQVU"`
	Status  string `json:"status" example:"in_progress"`
	Role    string `json:"role" example:"agent"`
	Created string `json:"created" example:"2026-03-18T02:19:41Z"`
	Updated string `json:"updated" example:"2026-03-18T02:19:41Z"`
	Object  string `json:"object" example:"interaction"`
	Agent   string `json:"agent,omitempty" example:"deep-research-pro-preview-12-2025"`
	Error   any    `json:"error,omitempty"`
}

// GeminiInteractions
// @Summary 创建交互
// @Description * 使用 Gemini Interactions 接口创建一个交互任务，比如深度研究，支持前台同步返回和 `background=true` 的后台异步执行。
// @Description * 文档: `https://ai.google.dev/gemini-api/docs/deep-research`
// @Description
// @Description **示例请求**:
// @Description ```json
// @Description {
// @Description   "input": "test, replay in 100 tokens",
// @Description   "agent": "deep-research-pro-preview-12-2025",
// @Description   "background": true
// @Description }
// @Description ```
// @Tags Gemini
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body oas.ExampleGeminiInteractionRequest true "Gemini Interactions 请求体"
// @Success 200 {object} oas.ExampleGeminiInteractionResponse "成功返回 Interaction 信息"
// @Failure 400 {object} dto.GeneralErrorResponse "Bad Request"
// @Failure 401 {object} dto.GeneralErrorResponse "Unauthorized"
// @Failure 403 {object} dto.GeneralErrorResponse "Forbidden"
// @Failure 429 {object} dto.GeneralErrorResponse "Too Many Requests"
// @Failure 500 {object} dto.GeneralErrorResponse "Internal Server Error"
// @Router /v1beta/interactions [post]
func GeminiInteractions(c *gin.Context) {}

// GeminiRetrieveInteraction
// @Summary 查询交互
// @Description * 按 `interaction_id` 查询 Gemini Interactions 任务状态与结果。对于后台任务，可轮询该接口直到 `status=completed`。
// @Description * 文档: `https://ai.google.dev/gemini-api/docs/deep-research`
// @Tags Gemini
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param interaction_id path string true "Interaction ID" example(v1_Chd1d3U2YWZiZ0NwcUswLWtQNXBmMjBBVRIXdXd1NmFmYmdDcHFLMC1rUDVwZjIwQVU)
// @Success 200 {object} oas.ExampleGeminiInteractionResponse "成功返回 Interaction 状态"
// @Failure 400 {object} dto.GeneralErrorResponse "Bad Request"
// @Failure 401 {object} dto.GeneralErrorResponse "Unauthorized"
// @Failure 403 {object} dto.GeneralErrorResponse "Forbidden"
// @Failure 404 {object} dto.GeneralErrorResponse "Not Found"
// @Failure 429 {object} dto.GeneralErrorResponse "Too Many Requests"
// @Failure 500 {object} dto.GeneralErrorResponse "Internal Server Error"
// @Router /v1beta/interactions/{interaction_id} [get]
func GeminiRetrieveInteraction(c *gin.Context) {}

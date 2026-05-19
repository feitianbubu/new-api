package oas

// Responses godoc
// @Summary 模型调用新[beta版]
// @Description openAI新版Responses接口，用于多模态和更复杂的响应场景。支持web-search等功能。
// @Description **注意**: beta版目前只有openAI官方渠道(gpt系列)和火山官方渠道(豆包系列)支持该接口。 其他渠道需要后续厂商适配。
// @Description
// @Description **文档**: https://developers.openai.com/api/reference/resources/responses/methods/create
// @Description
// @Description **示例**:
// @Description ```json
// @Description {
// @Description   "model": "doubao-seed-1-8-251228",
// @Description   "tools": [{type: "web_search"}],
// @Description   "input": "今天有什么新闻?",
// @Description   "stream": false
// @Description }
// @Description ```
// @Description
// @Description 当 `stream=true` 时，接口返回 `text/event-stream`，每条事件体为 `dto.ResponsesStreamResponse`。
// @Tags OpenAI
// @Accept json
// @Produce json
// @Param request body dto.OpenAIResponsesRequest true "Response request"
// @Success 200 {object} dto.OpenAIResponsesResponse "Response created successfully"
// @Failure 400 {object} dto.GeneralErrorResponse "Bad Request"
// @Failure 401 {object} dto.GeneralErrorResponse "Unauthorized"
// @Failure 403 {object} dto.GeneralErrorResponse "Forbidden"
// @Failure 429 {object} dto.GeneralErrorResponse "Too Many Requests"
// @Failure 500 {object} dto.GeneralErrorResponse "Internal Server Error"
// @Router /v1/responses [post]
//func Responses(c *gin.Context) {}

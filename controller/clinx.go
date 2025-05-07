package controller

import (
	"one-api/relay"
	"strings"

	"github.com/gin-gonic/gin"
)

// ModelList
// @Summary 模型列表
// @Description 获取系统当前模型列表的定价信息，包括模型价格、用户组倍率和可用用户组
// @Tags Clinx
// @Accept json
// @Produce json
// @Success 200 {object} gin.H "成功返回定价信息"
// @Router /clinx/v1/modelList [post]
func ModelList(c *gin.Context) {
	GetPricing(c)
}

// Completions
// @Summary      模型对话
// @Description  接收符合 OpenAI API 格式的文本或聊天补全请求
// @Tags         Clinx
// @Accept       json
// @Produce      json
// @Produce      text/event-stream
// @Param        Authorization header string true "用户认证令牌 (Bearer sk-xxxx)" example(Bearer sk-4No9laxl9cLoEDsPbF2vKpQ7MOVp4FHgXE3Br4zpoNq98Ldm)
// @Param        request body dto.GeneralOpenAIRequest true "OpenAI 请求体"
// @Success      200 {object} dto.OpenAITextResponse "非流式响应"
// @Success      200 {string} string "流式响应 (text/event-stream)"
// @Failure      400 {object} dto.OpenAIErrorWithStatusCode "无效的请求"
// @Failure      401 {object} dto.OpenAIErrorWithStatusCode "无效的认证"
// @Failure      403 {object} dto.OpenAIErrorWithStatusCode "用户或令牌额度不足"
// @Failure      500 {object} dto.OpenAIErrorWithStatusCode "内部服务器错误"
// @Router       /clinx/v1/chat/completions [post]
func Completions(c *gin.Context) {
	clinxRelay(c)
}

// Generations
// @Summary      图像生成
// @Description  接收符合 OpenAI API 格式的图像生成请求
// @Tags         Clinx
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "用户认证令牌 (Bearer sk-xxxx)" example(Bearer sk-4No9laxl9cLoEDsPbF2vKpQ7MOVp4FHgXE3Br4zpoNq98Ldm)
// @Param        request body dto.ImageRequest true "OpenAI 请求体"
// @Router       /clinx/v1/images/generations [post]
func Generations(c *gin.Context) {
	clinxRelay(c)
}

func trimClinxPath(c *gin.Context) {
	c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/clinx")
}

func clinxRelay(c *gin.Context) {
	trimClinxPath(c)
	Relay(c)
}

// SubmitImagine
// @Summary		 图像生成_MJ
// @Description  接收符合 Midjourney API 格式的图像生成请求
// @Tags         Clinx
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "用户认证令牌 (Bearer sk-xxxx)" example(Bearer sk-4No9laxl9cLoEDsPbF2vKpQ7MOVp4FHgXE3Br4zpoNq98Ldm)
// @Param        request body dto.MidjourneyRequest true "Midjourney 请求体"
// @Router       /clinx/mj/submit/imagine [post]
func SubmitImagine(c *gin.Context) {
	trimClinxPath(c)
	RelayMidjourney(c)
}

// RelayMidjourneyImage
// @Summary		 图像获取_MJ
// @Description  获取 Midjourney 图像
// @Tags         Clinx
// @Param        id path string true "图像 ID" example(1746607709831346)
// @Router       /clinx/mj/image/{id} [get]
func RelayMidjourneyImage(c *gin.Context) {
	trimClinxPath(c)
	relay.RelayMidjourneyImage(c)
}

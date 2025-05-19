package controller

import (
	"fmt"
	"net/http"
	"one-api/common"
	"one-api/model"
	"one-api/relay"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ModelList
// @Tags Clinx
// @Summary 模型列表
// @Produce application/json
// @Router /providers/modelsList/{provider} [get]
func ModelList(c *gin.Context) {
	channelId, _ := strconv.Atoi(c.Param("provider"))
	enableAbilities := model.GetAllEnableAbilities()
	type ModelVo struct {
		Provider  string `json:"provider"`
		Model     string `json:"model_name"`
		ModelType string `json:"model_type"`
	}
	mapData := make(map[string]ModelVo) //map[string]ModelVo
	for _, ability := range enableAbilities {
		if channelId == 0 || ability.ChannelId == channelId {
			modelType := ""
			if ability.Tag != nil {
				modelType = *ability.Tag
			}
			name := ability.Model
			mapData[name] = ModelVo{
				Provider:  strconv.Itoa(channelId),
				Model:     ability.Model,
				ModelType: modelType,
			}
		}
	}
	var data []any
	for _, v := range mapData {
		data = append(data, v)
	}
	SuccessPage(c, data)
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
// @Router       /api/v1/chat/completions [post]
func Completions(c *gin.Context) {
	if strings.Contains(c.Request.URL.Path, "/openai") {
		c.Request.URL.Path = "/v1/chat/completions"
	}
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
// @Router       /api/v1/images/generations [post]
func Generations(c *gin.Context) {
	clinxRelay(c)
}

func trimClinxPath(c *gin.Context) {
	c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/api")
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
// @Router       /api/mj/submit/imagine [post]
func SubmitImagine(c *gin.Context) {
	trimClinxPath(c)
	RelayMidjourney(c)
}

// RelayMidjourneyImage
// @Summary		 图像获取_MJ
// @Description  获取 Midjourney 图像
// @Tags         Clinx
// @Param        id path string true "图像 ID" example(1746607709831346)
// @Router       /api/mj/image/{id} [get]
func RelayMidjourneyImage(c *gin.Context) {
	trimClinxPath(c)
	relay.RelayMidjourneyImage(c)
}

// ProvidersList
// Deprecated
// compatible with legacy API to be removed in future
// @Tags Clinx
// @Summary 渠道列表
// @Produce application/json
// @Router /providers/providersList [get]
func ProvidersList(c *gin.Context) {
	common.SysLog("Deprecated ProvidersList called")
	channels, err := model.GetAllChannels(0, 999, false, false)
	if err != nil {
		Fail(c, "get channels failed")
		return
	}
	type ProviderVo struct {
		Id       string `json:"id"`
		Name     string `json:"name"`
		Provider string `json:"provider"`
	}
	var data []any
	for _, channel := range channels {
		if channel.Status != common.ChannelStatusEnabled {
			continue
		}
		data = append(data, ProviderVo{
			Id:       strconv.Itoa(channel.Id),
			Name:     channel.Name,
			Provider: strconv.Itoa(channel.Id),
		})
	}
	SuccessPage(c, data)
}

// Nd99u
// @Summary 99U登录
// @Description 通过 99u 进行用户登录
// @Tags User
// @Accept json
// @Produce json
// @Param code query string true "99u的uckey" example(QXV0aG9yaXphdGlvbjogTUFDIGlkPSI3RjkzOEIyMDVGODc2RkMzNTVGNEY2MTIwN0ZFOTQzRENEMDQ4RURDQjAzRERGNDAwODJDNzY1RTY1RTRBMDhENzMzQTVDQjMzM0NCODc2NUNFOTMzNzVENTcxOEE1OTMiLG5vbmNlPSIxNzQ3MTg4OTAzNTYzOkdTTkxSSE5PIixtYWM9IjdtUXZkQTZ6TlRpNVBCU0RGWE5IcnhVYWJvZnFsaURCeWE5ZGZpcmpyRnM9IixyZXF1ZXN0X3VyaT0iLyIsaG9zdD0idWMtY29tcG9uZW50LmJldGEuMTAxLmNvbSI=)
// @Router /api/oauth/nd99u [get]
func Nd99u(c *gin.Context) {
	OidcAuth(c)
}

// CheckToken
// @Summary 检查认证
// @Description 检查认证
// @Tags User
// @Accept json
// @Produce json
// @Param Authorization header string true "用户认证令牌 (Aeess-Token: sk-xxxx)" example(Access-Token: sk-4No9laxl9cLoEDsPbF2vKpQ7MOVp4FHgXE3Br4zpoNq98Ldm)
// @Router /api/checkToken [get]
// @Success 200 {object} model.User "用户信息"
// @Router /api/checkToken [get]
func CheckToken(c *gin.Context) {
	accessToken := c.Request.Header.Get("Authorization")
	if accessToken == "" {
		Fail(c, "empty token")
		return
	}
	user, err := model.ParseUserJWT(accessToken)
	if err != nil {
		Fail(c, "invalid token")
		return
	}
	if user, err = model.GetUserById(user.Id, false); err != nil {
		Fail(c, "user not found")
		return
	}
	Success(c, user.ToBaseUser())
	return
}

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

func Success(c *gin.Context, data interface{}) {
	common.SysLog(fmt.Sprintf("Success: %+v", data))
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Data: data,
		Msg:  "success",
	})
}
func SuccessPage(c *gin.Context, data []any) {
	common.SysLog(fmt.Sprintf("SuccessPage: %+v", data))
	type PageResult struct {
		List  interface{} `json:"list"`
		Total int64       `json:"total"`
	}
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Data: PageResult{
			List:  data,
			Total: int64(len(data)),
		},
		Msg: "success",
	})
}
func Fail(c *gin.Context, msg string) {
	common.SysLog(fmt.Sprintf("Fail: %s", msg))
	c.JSON(http.StatusOK, Response{
		Code: 10000,
		Data: nil,
		Msg:  msg,
	})
}

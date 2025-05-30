package controller

import (
	"github.com/gin-gonic/gin"
	"one-api/common"
	"one-api/model"
	"strconv"
)

func ModelListLegacy(c *gin.Context) {
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

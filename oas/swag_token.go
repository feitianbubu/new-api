package oas

import "github.com/gin-gonic/gin"

// AddToken godoc
// @Summary 添加令牌
// @Description 创建一个新的API令牌，支持设置名称、额度、过期时间、模型限制、IP白名单、分组等
// @Tags ApiKey
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AddTokenRequest true "令牌信息"
// @Success 200 {object} object{success=bool,message=string,data=AddTokenResponse} "创建成功，返回令牌信息（含完整Key）"
// @Failure 200 {object} object{success=bool,message=string} "创建失败"
// @Router /api/token [post]
func AddToken(c *gin.Context) {}

type AddTokenRequest struct {
	Name               string `json:"name" example:"my-token"`
	ExpiredTime        int64  `json:"expired_time" example:"-1"`
	RemainQuota        int    `json:"remain_quota" example:"500000"`
	UnlimitedQuota     bool   `json:"unlimited_quota" example:"false"`
	ModelLimitsEnabled bool   `json:"model_limits_enabled" example:"false"`
	ModelLimits        string `json:"model_limits" example:""`
	Group              string `json:"group" example:"default"`
}

type AddTokenResponse struct {
	Id                 int    `json:"id" example:"1"`
	UserId             int    `json:"user_id" example:"123"`
	Key                string `json:"key" example:"sk-xxxxxxxxxxxxxxxx"`
	Status             int    `json:"status" example:"1"`
	Name               string `json:"name" example:"my-token"`
	CreatedTime        int64  `json:"created_time" example:"1234567890"`
	ExpiredTime        int64  `json:"expired_time" example:"-1"`
	RemainQuota        int    `json:"remain_quota" example:"500000"`
	UnlimitedQuota     bool   `json:"unlimited_quota" example:"false"`
	ModelLimitsEnabled bool   `json:"model_limits_enabled" example:"false"`
	ModelLimits        string `json:"model_limits" example:""`
	Group              string `json:"group" example:"default"`
}

// GetAllTokens godoc
// @Summary 令牌列表
// @Description 获取当前用户的令牌列表，支持分页
// @Tags ApiKey
// @Produce json
// @Security BearerAuth
// @Param p query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10) maximum(100)
// @Success 200 {object} object{success=bool,message=string,data=object{page=int,page_size=int,total=int,items=[]TokenItem}} "令牌列表"
// @Router /api/token/ [get]
func GetAllTokens(c *gin.Context) {}

type TokenItem struct {
	Id                 int    `json:"id" example:"1"`
	UserId             int    `json:"user_id" example:"123"`
	Key                string `json:"key" example:"sk-abcd**********efgh"`
	Status             int    `json:"status" example:"1"`
	Name               string `json:"name" example:"my-token"`
	CreatedTime        int64  `json:"created_time" example:"1234567890"`
	AccessedTime       int64  `json:"accessed_time" example:"1234567890"`
	ExpiredTime        int64  `json:"expired_time" example:"-1"`
	RemainQuota        int    `json:"remain_quota" example:"500000"`
	UnlimitedQuota     bool   `json:"unlimited_quota" example:"false"`
	ModelLimitsEnabled bool   `json:"model_limits_enabled" example:"false"`
	ModelLimits        string `json:"model_limits" example:""`
	AllowIps           string `json:"allow_ips" example:""`
	UsedQuota          int    `json:"used_quota" example:"0"`
	Group              string `json:"group" example:"default"`
	CrossGroupRetry    bool   `json:"cross_group_retry" example:"false"`
}

// GetTokenKey godoc
// @Summary 查看令牌
// @Description 获取指定令牌的完整API Key（未脱敏）
// @Tags ApiKey
// @Produce json
// @Security BearerAuth
// @Param id path int true "令牌ID"
// @Success 200 {object} object{success=bool,message=string,data=object{key=string}} "完整Key"
// @Failure 200 {object} object{success=bool,message=string} "获取失败"
// @Router /api/token/{id}/key [post]
func GetTokenKey(c *gin.Context) {}

// DeleteToken godoc
// @Summary 删除令牌
// @Description 根据ID删除指定的令牌
// @Tags ApiKey
// @Produce json
// @Security BearerAuth
// @Param id path int true "令牌ID"
// @Success 200 {object} object{success=bool,message=string} "删除成功"
// @Failure 200 {object} object{success=bool,message=string} "删除失败"
// @Router /api/token/{id} [delete]
func DeleteToken(c *gin.Context) {}

package controller

import (
	"github.com/QuantumNous/new-api/relay/channel/task/doubao"
	"github.com/gin-gonic/gin"
)

// RelayListAssets
// @Summary 查询素材资产列表
// @Description 代理转发到豆包(Doubao) Asset API，查询符合筛选条件的素材资产(Asset)列表。
// @Tags Doubao Asset
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param request body doubao.ListAssetsRequest true "素材资产列表查询请求"
// @Success 200 {object} doubao.ListAssetsResponse "素材资产列表"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 502 {object} map[string]string "上游请求失败"
// @Router /doubao/open/ListAssets [post]
func RelayListAssets(c *gin.Context) {
	doubao.HandleListAssets(c)
}

// RelayGetAsset
// @Summary 查询素材资产信息
// @Description 代理转发到豆包(Doubao) Asset API，获取单个素材资产(Asset)信息。
// @Tags Doubao Asset
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param request body doubao.GetAssetRequest true "素材资产查询请求"
// @Success 200 {object} doubao.AssetItem "素材资产信息"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 502 {object} map[string]string "上游请求失败"
// @Router /doubao/open/GetAsset [post]
func RelayGetAsset(c *gin.Context) {
	doubao.HandleGetAsset(c)
}

// RelayCreateAsset
// @Summary 创建素材资产
// @Description 代理转发到豆包(Doubao) Asset API，向指定素材组内创建素材资产（异步接口）。
// @Tags Doubao Asset
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param request body doubao.CreateAssetRequest true "创建素材资产请求"
// @Success 200 {object} map[string]string "返回 Id"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 502 {object} map[string]string "上游请求失败"
// @Router /doubao/open/CreateAsset [post]
func RelayCreateAsset(c *gin.Context) {
	doubao.HandleCreateAsset(c)
}

// RelayUpdateAsset
// @Summary 更新素材资产
// @Description 代理转发到豆包(Doubao) Asset API，更新素材资产名称。
// @Tags Doubao Asset
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param request body doubao.UpdateAssetRequest true "更新素材资产请求"
// @Success 200 {object} map[string]string "返回 Id"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 502 {object} map[string]string "上游请求失败"
// @Router /doubao/open/UpdateAsset [post]
func RelayUpdateAsset(c *gin.Context) {
	doubao.HandleUpdateAsset(c)
}

// RelayDeleteAsset
// @Summary 删除素材资产
// @Description 代理转发到豆包(Doubao) Asset API，删除单个素材资产。
// @Tags Doubao Asset
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param request body doubao.DeleteAssetRequest true "删除素材资产请求"
// @Success 200 {object} map[string]string "成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 502 {object} map[string]string "上游请求失败"
// @Router /doubao/open/DeleteAsset [post]
func RelayDeleteAsset(c *gin.Context) {
	doubao.HandleDeleteAsset(c)
}

// RelayCreateAssetGroup
// @Summary 创建素材组
// @Description 代理转发到豆包(Doubao) Asset API，创建素材组(Asset Group)。
// @Tags Doubao Asset
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param request body doubao.CreateAssetGroupRequest true "创建素材组请求"
// @Success 200 {object} map[string]string "返回 Id"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 502 {object} map[string]string "上游请求失败"
// @Router /doubao/open/CreateAssetGroup [post]
func RelayCreateAssetGroup(c *gin.Context) {
	doubao.HandleCreateAssetGroup(c)
}

// RelayListAssetGroups
// @Summary 查询素材组列表
// @Description 代理转发到豆包(Doubao) Asset API，查询符合筛选条件的素材组列表。
// @Tags Doubao Asset
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param request body doubao.ListAssetGroupsRequest true "素材组列表查询请求"
// @Success 200 {object} doubao.ListAssetGroupsResponse "素材组列表"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 502 {object} map[string]string "上游请求失败"
// @Router /doubao/open/ListAssetGroups [post]
func RelayListAssetGroups(c *gin.Context) {
	doubao.HandleListAssetGroups(c)
}

// RelayGetAssetGroup
// @Summary 查询素材组信息
// @Description 代理转发到豆包(Doubao) Asset API，获取单个素材组信息。
// @Tags Doubao Asset
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param request body doubao.GetAssetGroupRequest true "素材组查询请求"
// @Success 200 {object} doubao.AssetGroupItem "素材组信息"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 502 {object} map[string]string "上游请求失败"
// @Router /doubao/open/GetAssetGroup [post]
func RelayGetAssetGroup(c *gin.Context) {
	doubao.HandleGetAssetGroup(c)
}

// RelayUpdateAssetGroup
// @Summary 更新素材组
// @Description 代理转发到豆包(Doubao) Asset API，更新素材组的名称和描述。
// @Tags Doubao Asset
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "用户认证令牌 (Bearer sk-xxxx)"
// @Param request body doubao.UpdateAssetGroupRequest true "更新素材组请求"
// @Success 200 {object} map[string]string "返回 Id"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 502 {object} map[string]string "上游请求失败"
// @Router /doubao/open/UpdateAssetGroup [post]
func RelayUpdateAssetGroup(c *gin.Context) {
	doubao.HandleUpdateAssetGroup(c)
}

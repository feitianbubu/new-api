package oas

import "github.com/gin-gonic/gin"

// WebSearch godoc
// @Summary 网络搜索
// @Description 调用上游搜索引擎进行网络检索，返回结构化的网页结果（标题、URL、摘要等）。
// @Description
// @Description * 接口格式标准文档: https://docs.tavily.com/documentation/api-reference/endpoint/search
// @Description * 智谱参数参考文档: https://docs.bigmodel.cn/cn/guide/tools/web-search
// @Tags Tools
// @Accept json
// @Produce json
// @Param request body ExampleWebSearchRequest true "Web search request"
// @Success 200 {object} dto.WebSearchResponse
// @Router /v1/web_search [post]
func WebSearch(c *gin.Context) {}

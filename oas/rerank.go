package oas

import "github.com/gin-gonic/gin"

// Rerank godoc
// @Summary 语义重排
// @Description 根据查询语句对候选列表按语义相关性重新排序，返回每个候选项的相关性分数与排序索引
// @Description
// @Description * Cohere: https://docs.cohere.com/reference/rerank
// @Description * 阿里云百炼: https://help.aliyun.com/zh/model-studio/text-rerank-api
// @Tags OpenAI
// @Accept json
// @Produce json
// @Param request body ExampleRerankRequest true "Rerank request"
// @Success 200 {object} dto.RerankResponse
// @Router /v1/rerank [post]
func Rerank(c *gin.Context) {}

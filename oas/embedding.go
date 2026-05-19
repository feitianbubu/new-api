package oas

import "github.com/gin-gonic/gin"

// Embeddings godoc
// @Summary 模型向量
// @Description 将文本、图片或视频转换为向量表示，支持多模态输入
// @Description
// @Description * OpenAI: https://developers.openai.com/api/reference/resources/embeddings/methods/create
// @Description * 豆包多模态: https://www.volcengine.com/docs/82379/1523520
// @Tags OpenAI
// @Accept json
// @Produce json
// @Param request body dto.EmbeddingRequest true "Embedding request"
// @Success 200 {object} dto.EmbeddingResponse
// @Router /v1/embeddings [post]
func Embeddings(c *gin.Context) {}

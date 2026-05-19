package oas

import "github.com/gin-gonic/gin"

// ImagesEdits swagger 图像编辑
// @Summary 图像编辑
// @Description 根据用户提供的源图像、遮罩和文本提示，编辑现有图像
// @Description * OAI文档: https://developers.openai.com/api/reference/resources/images/methods/edit
// @Description * 其他文档参考: <a href="#tag/openai/POST/v1/images/generations">图像生成</a>
// @Tags OpenAI
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "源图像文件 (PNG, WebP, JPEG, 最大 4MB)"
// @Param prompt formData string true "描述所需编辑的文本提示"
// @Param model formData string false "使用的模型名称" example(gpt-image-2)
// @Success 200 {object} dto.ImageResponse 成功响应
// @Router /v1/images/edits [post]
// @Example curl -X POST 'http://localhost/v1/images/edits' \
// @Example     -H 'Content-Type: multipart/form-data' \
// @Example     -H 'Authorization: Bearer YOUR_API_KEY' \
// @Example     -F 'image=@sunlit_lounge.png' \
// @Example     -F 'prompt=A sunlit indoor lounge area with a pool with a blue sky and white clouds'
func ImagesEdits(c *gin.Context) {}

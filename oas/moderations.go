package oas

import "github.com/gin-gonic/gin"

// Moderations godoc
// @Summary 内容审核
// @Description 检测文本内容是否包含潜在的有害信息（仇恨、骚扰、自残、色情、暴力等），返回每个分类的命中结果与置信分数
// @Description
// @Description * OpenAI: https://platform.openai.com/docs/api-reference/moderations/create
// @Tags OpenAI
// @Accept json
// @Produce json
// @Param request body ModerationRequest true "Moderation request"
// @Success 200 {object} ModerationResponse
// @Router /v1/moderations [post]
func Moderations(c *gin.Context) {}

type ModerationRequest struct {
	Model string `json:"model" example:"omni-moderation-latest"`
	Input string `json:"input" example:"I want to hurt them."`
}

type ModerationResponse struct {
	Id      string             `json:"id" example:"modr-0d9740456c391e43c445bf0f010940c7"`
	Model   string             `json:"model" example:"omni-moderation-latest"`
	Results []ModerationResult `json:"results"`
}

type ModerationResult struct {
	Flagged        bool                     `json:"flagged" example:"true"`
	Categories     ModerationCategories     `json:"categories"`
	CategoryScores ModerationCategoryScores `json:"category_scores"`
}

type ModerationCategories struct {
	Harassment            bool `json:"harassment" example:"true"`
	HarassmentThreatening bool `json:"harassment/threatening" example:"true"`
	Hate                  bool `json:"hate" example:"false"`
	HateThreatening       bool `json:"hate/threatening" example:"false"`
	Illicit               bool `json:"illicit" example:"false"`
	IllicitViolent        bool `json:"illicit/violent" example:"false"`
	SelfHarm              bool `json:"self-harm" example:"false"`
	SelfHarmIntent        bool `json:"self-harm/intent" example:"false"`
	SelfHarmInstructions  bool `json:"self-harm/instructions" example:"false"`
	Sexual                bool `json:"sexual" example:"false"`
	SexualMinors          bool `json:"sexual/minors" example:"false"`
	Violence              bool `json:"violence" example:"true"`
	ViolenceGraphic       bool `json:"violence/graphic" example:"false"`
}

type ModerationCategoryScores struct {
	Harassment            float64 `json:"harassment" example:"0.842"`
	HarassmentThreatening float64 `json:"harassment/threatening" example:"0.751"`
	Hate                  float64 `json:"hate" example:"0.013"`
	HateThreatening       float64 `json:"hate/threatening" example:"0.004"`
	Illicit               float64 `json:"illicit" example:"0.002"`
	IllicitViolent        float64 `json:"illicit/violent" example:"0.001"`
	SelfHarm              float64 `json:"self-harm" example:"0.0001"`
	SelfHarmIntent        float64 `json:"self-harm/intent" example:"0.0001"`
	SelfHarmInstructions  float64 `json:"self-harm/instructions" example:"0.0001"`
	Sexual                float64 `json:"sexual" example:"0.0002"`
	SexualMinors          float64 `json:"sexual/minors" example:"0.0001"`
	Violence              float64 `json:"violence" example:"0.673"`
	ViolenceGraphic       float64 `json:"violence/graphic" example:"0.012"`
}

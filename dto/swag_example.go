package dto

type ExampleGeneralOpenAIRequest struct {
	Model    string `json:"model" example:"gpt-4.1"`
	Messages []struct {
		Role    string `json:"role" example:"user"`
		Content string `json:"content" example:"你是?"`
	}
}

type ExampleImageRequest struct {
	Model  string `json:"model" example:"dall-e-2"`
	Prompt string `json:"prompt" example:"可爱的中国小女孩"`
	N      int    `json:"n" example:"1"`
	Size   string `json:"size" example:"256x256"`
	Seed   int    `json:"seed" example:"-1"`
}

type ExampleMidjourneyRequest struct {
	Prompt  string `json:"prompt" example:"Dog"`
	BotType string `json:"botType" example:"MID_JOURNEY"`
}

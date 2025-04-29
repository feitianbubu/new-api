package dto

type ExampleGeneralOpenAIRequest struct {
	Model    string `json:"model" example:"qwen2.5:7b"`
	Messages []struct {
		Role    string `json:"role" example:"user"`
		Content string `json:"content" example:"你是?"`
	}
}

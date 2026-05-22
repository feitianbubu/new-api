package oas

type ExampleGeneralOpenAIRequest struct {
	Model    string `json:"model" example:"gpt-4.1"`
	Messages []struct {
		Role    string `json:"role" example:"user"`
		Content string `json:"content" example:"你是?"`
	}
}

type ExampleImageRequest struct {
	Model          string `json:"model" example:"jimeng_high_aes_general_v21_L"`
	Prompt         string `json:"prompt" example:"可爱的中国小女孩"`
	N              int    `json:"n" example:"1"`
	Size           string `json:"size" example:"256x256"`
	Seed           int    `json:"seed" example:"-1"`
	ResponseFormat string `json:"response_format" example:"url"`
	ExtraFields    any    `json:"extra_fields,omitempty"` // 透传字段,json格式
}

type ExampleMidjourneyRequest struct {
	Prompt  string `json:"prompt" example:"Dog"`
	BotType string `json:"botType" example:"MID_JOURNEY"`
}

type ExampleEmbeddingRequest struct {
	Model          string   `json:"model" example:"text-embedding-v4"`
	Input          []string `json:"input" example:"[\"hi\"]"`
	EncodingFormat string   `json:"encoding_format" example:"float"`
}

type ExampleRerankRequest struct {
	Model     string   `json:"model" example:"qwen3-rerank"`
	Query     string   `json:"query" example:"什么是文本排序模型"`
	Documents []string `json:"documents" example:"[\"文本排序模型广泛用于搜索引擎和推荐系统中\",\"量子计算是计算科学的一个前沿领域\",\"预训练语言模型的发展给文本排序模型带来了新的进展\"]"`
	TopN      int      `json:"top_n,omitempty" example:"3"`
}

type ExampleWebSearchRequest struct {
	Model          string   `json:"model" example:"search_std"`
	Query          string   `json:"query" example:"今天上海天气"`
	MaxResults     int      `json:"max_results,omitempty" example:"10"`
	IncludeDomains []string `json:"include_domains,omitempty" example:"www.example.com"`
	Recency        string   `json:"recency,omitempty" example:"oneWeek"`
	ContentSize    string   `json:"content_size,omitempty" example:"medium"`
	SearchIntent   bool     `json:"search_intent,omitempty" example:"false"`
}

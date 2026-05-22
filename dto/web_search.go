package dto

import (
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// WebSearchRequest 是 new-api 统一的 Web Search 请求 DTO，
// 字段命名参考社区通用习惯（Tavily 风格）。各上游厂商在 adaptor 中
// 把它转换为自家协议字段。
type WebSearchRequest struct {
	// Model 路由依据 + 计费档位（如 search_std / search_pro / tavily-basic 等）
	Model string `json:"model"`
	// Query 必填，搜索关键词
	Query string `json:"query"`

	// MaxResults 返回条数
	MaxResults *int `json:"max_results,omitempty"`
	// IncludeDomains 域名白名单
	IncludeDomains []string `json:"include_domains,omitempty"`
	// Recency 时间过滤：oneDay/oneWeek/oneMonth/oneYear/noLimit
	Recency *string `json:"recency,omitempty"`
	// ContentSize 内容详尽程度：medium/high
	ContentSize *string `json:"content_size,omitempty"`
	// SearchIntent 是否启用意图识别
	SearchIntent *bool `json:"search_intent,omitempty"`

	// UserId / RequestId 透传给上游
	UserId    *string `json:"user_id,omitempty"`
	RequestId *string `json:"request_id,omitempty"`
}

func (r *WebSearchRequest) IsStream(c *gin.Context) bool {
	return false
}

func (r *WebSearchRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{
		CombineText: r.Query,
	}
}

func (r *WebSearchRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}

// WebSearchResult 归一化的单条搜索结果。
type WebSearchResult struct {
	Title       string  `json:"title,omitempty"`
	URL         string  `json:"url,omitempty"`
	Content     string  `json:"content,omitempty"`
	Site        string  `json:"site,omitempty"`
	Icon        string  `json:"icon,omitempty"`
	Refer       string  `json:"refer,omitempty"`
	PublishedAt string  `json:"published_at,omitempty"`
	Score       float64 `json:"score,omitempty"`
}

// WebSearchIntent 归一化的搜索意图项。
type WebSearchIntent struct {
	Query    string `json:"query,omitempty"`
	Intent   string `json:"intent,omitempty"`
	Keywords string `json:"keywords,omitempty"`
}

// WebSearchResponse 是 new-api 对外返回的统一搜索响应。
// Provider 字段暴露上游引擎名，便于客户端区分。
type WebSearchResponse struct {
	Id        string            `json:"id,omitempty"`
	Created   int64             `json:"created,omitempty"`
	RequestId string            `json:"request_id,omitempty"`
	Provider  string            `json:"provider,omitempty"`
	Intents   []WebSearchIntent `json:"intents,omitempty"`
	Results   []WebSearchResult `json:"results"`
	Usage     Usage             `json:"usage"`
}

package zhipu_4v

import (
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// ── 上游协议 DTO ─────────────────────────────────────────────

type zhipuWebSearchRequest struct {
	SearchQuery         string   `json:"search_query"`
	SearchEngine        string   `json:"search_engine"`
	SearchIntent        bool     `json:"search_intent"`
	Count               *int     `json:"count,omitempty"`
	SearchDomainFilter  *string  `json:"search_domain_filter,omitempty"`
	SearchRecencyFilter *string  `json:"search_recency_filter,omitempty"`
	ContentSize         *string  `json:"content_size,omitempty"`
	RequestId           *string  `json:"request_id,omitempty"`
	UserId              *string  `json:"user_id,omitempty"`
}

type zhipuWebSearchIntent struct {
	Query    string `json:"query"`
	Intent   string `json:"intent"`
	Keywords string `json:"keywords"`
}

type zhipuWebSearchResult struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Link        string `json:"link"`
	Media       string `json:"media"`
	Icon        string `json:"icon"`
	Refer       string `json:"refer"`
	PublishDate string `json:"publish_date"`
}

type zhipuWebSearchResponse struct {
	Id           string                 `json:"id"`
	Created      int64                  `json:"created"`
	RequestId    string                 `json:"request_id"`
	SearchIntent []zhipuWebSearchIntent `json:"search_intent"`
	SearchResult []zhipuWebSearchResult `json:"search_result"`
}

// ── 请求转换 ─────────────────────────────────────────────────

// convertWebSearchRequest 把统一 DTO 翻译成智谱 /paas/v4/web_search 协议。
// model 字段作为 search_engine 透传（new-api 把 search_std/search_pro/... 当模型名）。
func convertWebSearchRequest(req dto.WebSearchRequest) zhipuWebSearchRequest {
	out := zhipuWebSearchRequest{
		SearchQuery:         req.Query,
		SearchEngine:        req.Model,
		Count:               req.MaxResults,
		SearchRecencyFilter: req.Recency,
		ContentSize:         req.ContentSize,
		RequestId:           req.RequestId,
		UserId:              req.UserId,
	}
	if req.SearchIntent != nil {
		out.SearchIntent = *req.SearchIntent
	}
	// 智谱 search_domain_filter 仅支持单个域名，多值只取第一个（其余忽略）。
	if len(req.IncludeDomains) > 0 {
		first := req.IncludeDomains[0]
		out.SearchDomainFilter = &first
	}
	return out
}

// ── 响应处理 ─────────────────────────────────────────────────

func handleWebSearchResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.WebSearchResponse, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewError(fmt.Errorf("read web search response body: %w", err), types.ErrorCodeReadResponseBodyFailed)
	}

	var upstream zhipuWebSearchResponse
	if err := common.Unmarshal(body, &upstream); err != nil {
		return nil, types.NewError(fmt.Errorf("unmarshal web search response: %w (body=%s)", err, string(body)), types.ErrorCodeBadResponseBody)
	}

	out := &dto.WebSearchResponse{
		Id:        upstream.Id,
		Created:   upstream.Created,
		RequestId: upstream.RequestId,
		Provider:  ChannelName,
		Results:   make([]dto.WebSearchResult, 0, len(upstream.SearchResult)),
	}
	for _, it := range upstream.SearchIntent {
		out.Intents = append(out.Intents, dto.WebSearchIntent{
			Query:    it.Query,
			Intent:   it.Intent,
			Keywords: it.Keywords,
		})
	}
	for _, r := range upstream.SearchResult {
		out.Results = append(out.Results, dto.WebSearchResult{
			Title:       r.Title,
			URL:         r.Link,
			Content:     r.Content,
			Site:        r.Media,
			Icon:        r.Icon,
			Refer:       r.Refer,
			PublishedAt: r.PublishDate,
		})
	}
	// per-call 计价：Web Search 没有 token 概念。
	// service/text_quota.go 在 TotalTokens==0 时会强制把 Quota 清零并跳过扣费，
	// 所以这里塞一个哨兵值（PromptTokens=1）让 TotalTokens>0，per-call 价格才能生效。
	out.Usage = dto.Usage{PromptTokens: 1, TotalTokens: 1}
	return out, nil
}

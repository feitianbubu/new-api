package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type WikiResponse struct {
	Query struct {
		Pages map[string]struct {
			PageID    int
			Title     string
			Extract   string `json:"extract"`
			Revisions []struct {
				Slots struct {
					Main struct {
						Content       string `json:"*"`
						ContentModel  string
						ContentFormat string
					}
				}
			}
		}
	}
}

func isWikiModel(model string) bool {
	return strings.HasPrefix(model, "wiki")
}

func handleWikiRequest(c *gin.Context, info *relaycommon.RelayInfo) (*http.Response, error) {
	var openAIReq dto.GeneralOpenAIRequest
	body, err := common.GetRequestBodyBytes(c)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	if err := json.Unmarshal(body, &openAIReq); err != nil {
		return nil, fmt.Errorf("failed to parse request body: %w", err)
	}

	query := extractQueryFromMessages(openAIReq.Messages)
	if query == "" {
		return nil, fmt.Errorf("no query found in request")
	}

	model := info.UpstreamModelName
	wikiResp, err := QueryWikipedia(query, model)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	content := extractContentFromWikiResp(wikiResp)
	if mt := openAIReq.GetMaxTokens(); mt > 0 {
		content = lo.Substring(content, 0, mt)
	}
	usage := buildWikiUsage(openAIReq.Messages, model, content)

	openAIResp := dto.OpenAITextResponse{
		Id:      fmt.Sprintf("wiki-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []dto.OpenAITextResponseChoice{
			{
				Index: 0,
				Message: dto.Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: *usage,
	}

	jsonData, err := json.Marshal(openAIResp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal wiki response: %w", err)
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(jsonData)),
	}
	resp.Header.Set("Content-Type", "application/json")

	return resp, nil
}

func extractQueryFromMessages(messages []dto.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			if messages[i].IsStringContent() {
				return messages[i].StringContent()
			} else {
				parts := messages[i].ParseContent()
				for _, part := range parts {
					if part.Type == dto.ContentTypeText {
						return part.Text
					}
				}
			}
		}
	}
	return ""
}

func extractContentFromWikiResp(resp *WikiResponse) string {
	for _, page := range resp.Query.Pages {
		if page.Extract != "" {
			return page.Extract
		}
		if len(page.Revisions) > 0 {
			return page.Revisions[0].Slots.Main.Content
		}
	}
	return ""
}

func buildWikiUsage(messages []dto.Message, model, content string) *dto.Usage {
	var promptBuilder strings.Builder
	for _, m := range messages {
		if m.Role == "user" || m.Role == "system" || m.Role == "assistant" {
			if m.IsStringContent() {
				promptBuilder.WriteString(m.StringContent())
			} else {
				for _, part := range m.ParseContent() {
					if part.Type == dto.ContentTypeText {
						promptBuilder.WriteString(part.Text)
					}
				}
			}
			promptBuilder.WriteByte('\n')
		}
	}

	promptTokens := service.EstimateTokenByModel(model, promptBuilder.String())
	completionTokens := service.EstimateTokenByModel(model, content)

	return &dto.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		InputTokens:      promptTokens,
		OutputTokens:     completionTokens,
	}
}

func handleWikiResponse(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var openaiResp dto.OpenAITextResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	if info.IsStream {
		return handleWikiStreamResponse(c, &openaiResp, info)
	}

	service.IOCopyBytesGracefully(c, resp, body)
	return &openaiResp.Usage, nil
}

func handleWikiStreamResponse(c *gin.Context, openaiResp *dto.OpenAITextResponse, info *relaycommon.RelayInfo) (*dto.Usage, *types.NewAPIError) {
	helper.SetEventStreamHeaders(c)

	responseId := openaiResp.Id
	if responseId == "" {
		responseId = fmt.Sprintf("wiki-%d", time.Now().UnixNano())
	}

	created := int64(openaiResp.Created.(float64))
	if created == 0 {
		created = time.Now().Unix()
	}

	model := info.UpstreamModelName

	start := helper.GenerateStartEmptyResponse(responseId, created, model, nil)
	if data, err := common.Marshal(start); err == nil {
		_ = helper.StringData(c, string(data))
	}

	content := ""
	if len(openaiResp.Choices) > 0 {
		if contentStr, ok := openaiResp.Choices[0].Message.Content.(string); ok {
			content = contentStr
		}
	}

	// 按字符（rune）分块，避免截断 UTF-8 字符
	runes := []rune(content)
	chunkSize := 50 // 每次发送 50 个字符
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		chunk := dto.ChatCompletionsStreamResponse{
			Id:      responseId,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []dto.ChatCompletionsStreamResponseChoice{{
				Index: 0,
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
					Role: "assistant",
				},
			}},
		}
		chunk.Choices[0].Delta.SetContentString(string(runes[i:end]))

		if data, err := common.Marshal(chunk); err == nil {
			_ = helper.StringData(c, string(data))
		}
	}

	finishReason := "stop"
	if len(openaiResp.Choices) > 0 {
		finishReason = openaiResp.Choices[0].FinishReason
	}

	if stop := helper.GenerateStopResponse(responseId, created, model, finishReason); stop != nil {
		if data, err := common.Marshal(stop); err == nil {
			_ = helper.StringData(c, string(data))
		}
	}

	if final := helper.GenerateFinalUsageResponse(responseId, created, model, openaiResp.Usage); final != nil {
		if data, err := common.Marshal(final); err == nil {
			_ = helper.StringData(c, string(data))
		}
	}

	helper.Done(c)

	return &openaiResp.Usage, nil
}

func QueryWikipedia(title, model string) (*WikiResponse, error) {
	apiURL := buildWikiAPIURL(model)

	params := url.Values{}
	params.Set("action", "query")
	params.Set("prop", "extracts")
	params.Set("titles", title)
	params.Set("explaintext", "1")
	params.Set("exsectionformat", "plain")
	params.Set("format", "json")
	params.Set("redirects", "1")

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	client := service.GetHttpClient()

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build wikipedia request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch wikipedia: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("wikipedia api returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var wikiResp WikiResponse
	if err := json.Unmarshal(body, &wikiResp); err != nil {
		return nil, fmt.Errorf("failed to parse wikipedia response: %w", err)
	}

	return &wikiResp, nil
}

func buildWikiAPIURL(model string) string {
	lang := "zh"

	if idx := strings.LastIndexByte(model, '-'); idx >= 0 && idx < len(model)-1 {
		lang = model[idx+1:]
	}

	return fmt.Sprintf("https://%s.wikipedia.org/w/api.php", lang)
}

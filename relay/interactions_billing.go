package relay

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func extractInteractionIDFromRequestPath(path string) string {
	const prefix = "/v1beta/interactions/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(path, prefix))
}

func shouldSettleInteractionsBilling(method string, statusCode int, responseBody []byte, estimatePromptTokens int) (interactionID string, usage *dto.Usage, shouldSettle bool) {
	if statusCode != http.StatusOK {
		return "", nil, false
	}

	var resp interactionsBillingResponse
	if err := common.Unmarshal(responseBody, &resp); err != nil {
		return "", nil, false
	}
	id := strings.TrimSpace(resp.ID)
	status := strings.ToLower(strings.TrimSpace(resp.Status))
	if status == "" && method == http.MethodPost {
		// Synchronous create may omit status in some response variants.
		status = "completed"
	}
	if !isTerminalInteractionStatus(status) {
		return id, nil, false
	}

	return id, extractInteractionsUsage(resp, estimatePromptTokens), true
}

func isTerminalInteractionStatus(status string) bool {
	switch status {
	case "completed", "failed", "cancelled", "canceled", "expired":
		return true
	default:
		return false
	}
}

func extractInteractionsUsage(resp interactionsBillingResponse, estimatePromptTokens int) *dto.Usage {
	if usage := usageFromInteractionsUsage(resp.Usage); usage != nil {
		return usage
	}
	if resp.UsageMetadata != nil {
		return usageFromGeminiMetadata(*resp.UsageMetadata)
	}
	if resp.UsageMetadataSnake != nil {
		return usageFromGeminiMetadata(*resp.UsageMetadataSnake)
	}
	if estimatePromptTokens <= 0 {
		return nil
	}
	// Keep backward-compatible fallback if upstream does not return usage.
	return &dto.Usage{
		PromptTokens: estimatePromptTokens,
		TotalTokens:  estimatePromptTokens,
	}
}

type interactionsBillingResponse struct {
	ID                 string                   `json:"id"`
	Status             string                   `json:"status"`
	Usage              *interactionsUsage       `json:"usage"`
	UsageMetadata      *dto.GeminiUsageMetadata `json:"usageMetadata"`
	UsageMetadataSnake *dto.GeminiUsageMetadata `json:"usage_metadata"`
}

type interactionsUsage struct {
	TotalTokens           int                          `json:"total_tokens"`
	TotalInputTokens      int                          `json:"total_input_tokens"`
	TotalCachedTokens     int                          `json:"total_cached_tokens"`
	TotalOutputTokens     int                          `json:"total_output_tokens"`
	TotalThoughtTokens    int                          `json:"total_thought_tokens"`
	InputTokensByModality []interactionsModalityTokens `json:"input_tokens_by_modality"`
}

type interactionsModalityTokens struct {
	Modality string `json:"modality"`
	Tokens   int    `json:"tokens"`
}

func usageFromInteractionsUsage(usage *interactionsUsage) *dto.Usage {
	if usage == nil {
		return nil
	}
	prompt := usage.TotalInputTokens
	completion := usage.TotalOutputTokens + usage.TotalThoughtTokens
	total := usage.TotalTokens
	if total == 0 {
		total = prompt + completion
	}
	if prompt == 0 && completion == 0 && total == 0 {
		return nil
	}
	mapped := &dto.Usage{
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      total,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: usage.TotalCachedTokens,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{
			ReasoningTokens: usage.TotalThoughtTokens,
		},
	}
	for _, item := range usage.InputTokensByModality {
		switch strings.ToUpper(strings.TrimSpace(item.Modality)) {
		case "TEXT":
			mapped.PromptTokensDetails.TextTokens += item.Tokens
		case "AUDIO":
			mapped.PromptTokensDetails.AudioTokens += item.Tokens
		case "IMAGE":
			mapped.PromptTokensDetails.ImageTokens += item.Tokens
		}
	}
	return mapped
}

func usageFromGeminiMetadata(metadata dto.GeminiUsageMetadata) *dto.Usage {
	prompt := metadata.PromptTokenCount
	completion := metadata.CandidatesTokenCount + metadata.ThoughtsTokenCount
	total := metadata.TotalTokenCount
	if total == 0 {
		total = prompt + completion
	}
	if prompt == 0 && completion == 0 && total == 0 {
		return nil
	}

	usage := &dto.Usage{
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      total,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: metadata.CachedContentTokenCount,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{
			ReasoningTokens: metadata.ThoughtsTokenCount,
		},
	}
	for _, detail := range metadata.PromptTokensDetails {
		switch strings.ToUpper(strings.TrimSpace(detail.Modality)) {
		case "TEXT":
			usage.PromptTokensDetails.TextTokens += detail.TokenCount
		case "AUDIO":
			usage.PromptTokensDetails.AudioTokens += detail.TokenCount
		case "IMAGE":
			usage.PromptTokensDetails.ImageTokens += detail.TokenCount
		}
	}
	return usage
}

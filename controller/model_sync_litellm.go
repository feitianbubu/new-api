package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	litellmModelsURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"
)

type litellmModel struct {
	LitellmProvider                 string   `json:"litellm_provider"`
	MaxInputTokens                  int      `json:"max_input_tokens"`
	MaxOutputTokens                 int      `json:"max_output_tokens"`
	MaxTokens                       int      `json:"max_tokens"`
	Mode                            string   `json:"mode"`
	InputCostPerToken               float64  `json:"input_cost_per_token"`
	OutputCostPerToken              float64  `json:"output_cost_per_token"`
	SupportsVision                  bool     `json:"supports_vision"`
	SupportsFunctionCalling         bool     `json:"supports_function_calling"`
	SupportsParallelFunctionCalling bool     `json:"supports_parallel_function_calling"`
	SupportsSystemMessages          bool     `json:"supports_system_messages"`
	SupportsResponseSchema          bool     `json:"supports_response_schema"`
	SupportsPromptCaching           bool     `json:"supports_prompt_caching"`
	SupportsAudioInput              bool     `json:"supports_audio_input"`
	SupportsAudioOutput             bool     `json:"supports_audio_output"`
	SupportedRegions                []string `json:"supported_regions"`
}

var providerMapping = map[string]string{
	"openai":                 "OpenAI",
	"azure":                  "Azure",
	"anthropic":              "Anthropic",
	"replicate":              "Replicate",
	"cohere":                 "Cohere",
	"huggingface":            "Hugging Face",
	"together_ai":            "Together AI",
	"together-ai":            "Together AI",
	"aleph_alpha":            "Aleph Alpha",
	"aleph-alpha":            "Aleph Alpha",
	"baseten":                "Baseten",
	"openrouter":             "OpenRouter",
	"google":                 "Google",
	"vertex_ai":              "Google Vertex AI",
	"vertex-ai":              "Google Vertex AI",
	"palm":                   "Google PaLM",
	"ai21":                   "AI21",
	"nlp_cloud":              "NLP Cloud",
	"nlp-cloud":              "NLP Cloud",
	"bedrock":                "AWS Bedrock",
	"sagemaker":              "AWS SageMaker",
	"vllm":                   "vLLM",
	"ollama":                 "Ollama",
	"deepinfra":              "DeepInfra",
	"perplexity":             "Perplexity",
	"groq":                   "Groq",
	"mistral":                "Mistral AI",
	"voyage":                 "Voyage AI",
	"deepseek":               "DeepSeek",
	"gemini":                 "Google Gemini",
	"text-completion-openai": "OpenAI",
	"custom_openai":          "Custom OpenAI",
	"custom-openai":          "Custom OpenAI",
	"cloudflare":             "Cloudflare",
	"fireworks_ai":           "Fireworks AI",
	"fireworks-ai":           "Fireworks AI",
	"anyscale":               "Anyscale",
	"watsonx":                "IBM watsonx",
	"triton":                 "NVIDIA Triton",
	"predibase":              "Predibase",
	"databricks":             "Databricks",
	"xinference":             "Xorbits Inference",
	"cerebras":               "Cerebras",
	"github":                 "GitHub Models",
	"volcengine":             "Volcengine",
	"sambanova":              "SambaNova",
	"xai":                    "xAI",
}

func fetchLiteLLMModels(ctx context.Context) (map[string]upstreamModel, error) {
	req, err := httpClient.Get(litellmModelsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch LiteLLM models: %w", err)
	}
	defer req.Body.Close()

	var rawData map[string]json.RawMessage
	if err := json.NewDecoder(req.Body).Decode(&rawData); err != nil {
		return nil, fmt.Errorf("failed to decode LiteLLM models: %w", err)
	}

	result := make(map[string]upstreamModel)
	for modelName, rawModel := range rawData {
		if modelName == "sample_spec" {
			continue
		}

		var lm litellmModel
		if err := json.Unmarshal(rawModel, &lm); err != nil {
			continue // 跳过无法解析的模型
		}

		if lm.Mode != "" && lm.Mode != "chat" && lm.Mode != "completion" {
			continue
		}

		um := convertLiteLLMToUpstream(modelName, lm)
		result[modelName] = um
	}

	return result, nil
}

func convertLiteLLMToUpstream(modelName string, lm litellmModel) upstreamModel {
	vendorName := normalizeVendorName(lm.LitellmProvider)

	description := generateDescription(lm)

	tags := generateTags(lm)

	endpoints := generateEndpoints(lm)

	return upstreamModel{
		ModelName:   modelName,
		VendorName:  vendorName,
		Description: description,
		Icon:        "", // 忽略图标
		Tags:        tags,
		Endpoints:   endpoints,
		Status:      1, // 默认启用
		NameRule:    0, // 默认精确匹配
	}
}

func normalizeVendorName(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if mapped, ok := providerMapping[provider]; ok {
		return mapped
	}
	if provider == "" {
		return ""
	}
	return strings.ToUpper(provider[:1]) + provider[1:]
}

func generateDescription(lm litellmModel) string {
	parts := []string{}

	switch lm.Mode {
	case "chat":
		parts = append(parts, "Chat model")
	case "completion":
		parts = append(parts, "Completion model")
	case "embedding":
		parts = append(parts, "Embedding model")
	case "image_generation":
		parts = append(parts, "Image generation model")
	case "audio_transcription":
		parts = append(parts, "Audio transcription model")
	case "audio_speech":
		parts = append(parts, "Audio speech model")
	case "moderation":
		parts = append(parts, "Moderation model")
	case "rerank":
		parts = append(parts, "Rerank model")
	default:
		if lm.Mode != "" {
			parts = append(parts, lm.Mode+" model")
		} else {
			parts = append(parts, "AI model")
		}
	}

	features := []string{}
	if lm.SupportsVision {
		features = append(features, "vision")
	}
	if lm.SupportsFunctionCalling {
		features = append(features, "function calling")
	}
	if lm.SupportsPromptCaching {
		features = append(features, "prompt caching")
	}
	if lm.SupportsAudioInput {
		features = append(features, "audio input")
	}
	if lm.SupportsAudioOutput {
		features = append(features, "audio output")
	}

	if len(features) > 0 {
		parts = append(parts, "with "+strings.Join(features, ", ")+" support")
	}

	if lm.MaxInputTokens > 0 || lm.MaxOutputTokens > 0 || lm.MaxTokens > 0 {
		maxInput := lm.MaxInputTokens
		if maxInput == 0 {
			maxInput = lm.MaxTokens
		}
		maxOutput := lm.MaxOutputTokens
		if maxOutput == 0 {
			maxOutput = lm.MaxTokens
		}

		if maxInput > 0 {
			parts = append(parts, fmt.Sprintf("(max input: %d tokens", maxInput))
			if maxOutput > 0 && maxOutput != maxInput {
				parts = append(parts, fmt.Sprintf("max output: %d tokens)", maxOutput))
			} else {
				parts[len(parts)-1] += ")"
			}
		}
	}

	return strings.Join(parts, " ")
}

func generateTags(lm litellmModel) string {
	tags := []string{}

	if lm.Mode != "" {
		tags = append(tags, lm.Mode)
	}

	if lm.SupportsVision {
		tags = append(tags, "vision")
	}
	if lm.SupportsFunctionCalling {
		tags = append(tags, "function-calling")
	}
	if lm.SupportsSystemMessages {
		tags = append(tags, "system-messages")
	}
	if lm.SupportsResponseSchema {
		tags = append(tags, "response-schema")
	}
	if lm.SupportsPromptCaching {
		tags = append(tags, "caching")
	}
	if lm.SupportsAudioInput {
		tags = append(tags, "audio-input")
	}
	if lm.SupportsAudioOutput {
		tags = append(tags, "audio-output")
	}

	if lm.LitellmProvider != "" {
		tags = append(tags, lm.LitellmProvider)
	}

	return strings.Join(tags, ",")
}

func generateEndpoints(lm litellmModel) json.RawMessage {
	endpoint := map[string]interface{}{
		"litellm_provider": lm.LitellmProvider,
		"mode":             lm.Mode,
	}

	if lm.MaxInputTokens > 0 {
		endpoint["max_input_tokens"] = lm.MaxInputTokens
	}
	if lm.MaxOutputTokens > 0 {
		endpoint["max_output_tokens"] = lm.MaxOutputTokens
	}
	if lm.MaxTokens > 0 {
		endpoint["max_tokens"] = lm.MaxTokens
	}

	features := make(map[string]bool)
	if lm.SupportsVision {
		features["vision"] = true
	}
	if lm.SupportsFunctionCalling {
		features["function_calling"] = true
	}
	if lm.SupportsSystemMessages {
		features["system_messages"] = true
	}
	if lm.SupportsResponseSchema {
		features["response_schema"] = true
	}
	if len(features) > 0 {
		endpoint["features"] = features
	}

	data, _ := json.Marshal(endpoint)
	return json.RawMessage(data)
}

type MergeLiteLLMModelsResult struct {
	LiteLLMEnabled     bool
	LiteLLMModelsCount int
	MergedCount        int
	AddedCount         int
}

func MergeLiteLLMModels(ctx context.Context, modelByName map[string]upstreamModel, upstreamNames *[]string) MergeLiteLLMModelsResult {
	result := MergeLiteLLMModelsResult{
		LiteLLMEnabled: false,
	}

	litellmEnabled := strings.ToLower(strings.TrimSpace(
		common.GetEnvOrDefaultString("SYNC_LITELLM_ENABLED", "true")))
	if litellmEnabled != "true" {
		return result
	}

	result.LiteLLMEnabled = true

	lmModels, err := fetchLiteLLMModels(ctx)
	if err != nil {
		return result
	}

	result.LiteLLMModelsCount = len(lmModels)

	for modelName, litellmModel := range lmModels {
		if officialModel, exists := modelByName[modelName]; exists {
			merged := litellmModel
			if strings.TrimSpace(officialModel.Description) != "" {
				merged.Description = officialModel.Description
			}
			modelByName[modelName] = merged
			result.MergedCount++
		} else {
			modelByName[modelName] = litellmModel
			if upstreamNames != nil {
				*upstreamNames = append(*upstreamNames, modelName)
			}
			result.AddedCount++
		}
	}

	return result
}

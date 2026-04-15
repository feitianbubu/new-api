package model

import (
	"regexp"
	"strings"
)

var baseModelPrefixMappings = map[string]string{
	// Claude 系列
	"claude-sonnet": "claude-sonnet",
	"claude-opus":   "claude-opus",
	"claude-haiku":  "claude-haiku",

	// OpenAI 系列
	"text-embedding": "text-embedding",
	"dall-e":         "dall-e",
	"gpt-image":      "gpt-image",
	"gpt":            "gpt",
	"sora":           "sora",

	// Google 系列
	"gemini":      "gemini",
	"nano-banana": "nano-banana",

	// 阿里系列
	"qwen3-vl":   "qwen3-vl",
	"qwen3-omni": "qwen3-omni",
	"qwen3-next": "qwen3-next",
	"qwen3":      "qwen3",
	"qwen":       "qwen",
	"wan":        "wan",

	// 字节系列
	"doubao-seedream": "doubao-seedream",
	"doubao":          "doubao",
	"jimeng":          "jimeng",
	"seed-tts":        "seed-tts",

	// 其他系列
	"deepseek": "deepseek",
	"kimi":     "kimi",
	"glm":      "glm",
	"viduq":    "viduq",
	"vidu":     "vidu",
	"kling":    "kling",
	"whisper":  "whisper",
	"yunyi":    "yunyi",
}

func ExtractBaseModel(modelName string, manualBaseModel string) string {
	if manualBaseModel != "" {
		return manualBaseModel
	}
	if modelName == "" {
		return ""
	}
	lowerName := strings.ToLower(modelName)
	var sortedPrefixes []string
	for prefix := range baseModelPrefixMappings {
		sortedPrefixes = append(sortedPrefixes, prefix)
	}
	for i := 0; i < len(sortedPrefixes); i++ {
		for j := i + 1; j < len(sortedPrefixes); j++ {
			if len(sortedPrefixes[i]) < len(sortedPrefixes[j]) {
				sortedPrefixes[i], sortedPrefixes[j] = sortedPrefixes[j], sortedPrefixes[i]
			}
		}
	}
	for _, prefix := range sortedPrefixes {
		if strings.HasPrefix(lowerName, prefix) {
			return baseModelPrefixMappings[prefix]
		}
	}
	return extractBaseModelByRegex(modelName)
}

func extractBaseModelByRegex(modelName string) string {
	name := strings.ToLower(modelName)
	name = regexp.MustCompile(`[-_]?\d{4}-\d{2}-\d{2}|[-_]?\d{8}|[-_]?\d{6}(?:[-_]|$)`).
		ReplaceAllString(name, "")
	re := regexp.MustCompile(`[-_]?(v\d+|\d+\.?\d*|\d+[bkmgtp]|1080p|720p|4k)(?:[-_]|$)`)
	if loc := re.FindStringIndex(name); loc != nil {
		name = name[:loc[0]]
	}
	name = strings.TrimRight(name, "-_")
	return name
}

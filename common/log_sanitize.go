package common

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	logMaskAll  = "***"
	logKeepTail = 4
)

var (
	logSanitizeKeyValuePattern = regexp.MustCompile(`(?i)(password|passwd|pwd|secret|client_secret|refresh_token|access_token|id_token|authorization|cookie|set-cookie|turnstile|captcha|otp|totp|webauthn|api[_-]?key|token|key|bearer|sk)\s*[:=]\s*([^\s,;&]+)`)
	logMaskAllKeys             = map[string]struct{}{
		"password":     {},
		"passwd":       {},
		"pwd":          {},
		"clientsecret": {},
		"secret":       {},
	}
	logMaskPartialKeys = map[string]struct{}{
		"apikey":        {},
		"key":           {},
		"token":         {},
		"bearer":        {},
		"sk":            {},
		"refreshtoken":  {},
		"accesstoken":   {},
		"idtoken":       {},
		"authorization": {},
	}
)

func normalizeLogKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, " ", "")
	return key
}

func shouldMaskAllByKey(key string) bool {
	nk := normalizeLogKey(key)
	if nk == "" {
		return false
	}
	if _, ok := logMaskAllKeys[nk]; ok {
		return true
	}
	return false
}

func shouldMaskPartialByKey(key string) bool {
	nk := normalizeLogKey(key)
	if nk == "" {
		return false
	}
	if _, ok := logMaskPartialKeys[nk]; ok {
		return true
	}
	return false
}

func toLogString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func maskPartial(value string) string {
	if value == "" {
		return logMaskAll
	}
	if len(value) <= logKeepTail {
		return strings.Repeat("*", len(value))
	}
	return strings.Repeat("*", len(value)-logKeepTail) + value[len(value)-logKeepTail:]
}

func maskValueByKey(key string, value any) any {
	if shouldMaskAllByKey(key) {
		return logMaskAll
	}
	if shouldMaskPartialByKey(key) {
		return maskPartial(toLogString(value))
	}
	return value
}

func sanitizeLogValue(value any, key string) any {
	if key != "" && (shouldMaskAllByKey(key) || shouldMaskPartialByKey(key)) {
		return maskValueByKey(key, value)
	}

	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[k] = sanitizeLogValue(item, k)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = sanitizeLogValue(v[i], "")
		}
		return out
	case string:
		return SanitizeStringForLog(v)
	default:
		return v
	}
}

func SanitizeKVForLog(key string, value any) any {
	return sanitizeLogValue(value, key)
}

func SanitizeForLog(value any) any {
	return sanitizeLogValue(value, "")
}

func SanitizeStringForLog(input string) string {
	if input == "" {
		return input
	}
	return logSanitizeKeyValuePattern.ReplaceAllStringFunc(input, func(match string) string {
		parts := logSanitizeKeyValuePattern.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		key := parts[1]
		originalValue := parts[2]
		masked := maskValueByKey(key, originalValue)
		return strings.Replace(match, originalValue, toLogString(masked), 1)
	})
}

func SanitizeJSONStringForLog(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}

	var payload any
	if err := Unmarshal(raw, &payload); err != nil {
		return SanitizeStringForLog(string(raw))
	}

	sanitized := SanitizeForLog(payload)
	body, err := Marshal(sanitized)
	if err != nil {
		return SanitizeStringForLog(string(raw))
	}
	return string(body)
}

package common

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"
)

const (
	truncateMaxFieldSize    = 256
	truncateMaxArrayPreview = 3
	truncateMaxDepth        = 10
)

func TruncateValue(v any) any {
	return truncateValue(v, 0)
}

func truncateValue(v any, depth int) any {
	if depth > truncateMaxDepth {
		return "..."
	}
	switch val := v.(type) {
	case string:
		return TruncateString(val)
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, item := range val {
			out[k] = truncateValue(item, depth+1)
		}
		return out
	case []any:
		n := len(val)
		if n == 0 {
			return v
		}
		preview := min(n, truncateMaxArrayPreview)
		out := make([]any, preview, preview+1)
		for i := 0; i < preview; i++ {
			out[i] = truncateValue(val[i], depth+1)
		}
		if n > truncateMaxArrayPreview {
			out = append(out, fmt.Sprintf("...and %d more items", n-truncateMaxArrayPreview))
		}
		return out
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, bool, nil:
		return val
	}

	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Slice {
		n := rv.Len()
		if n == 0 {
			return v
		}
		preview := min(n, truncateMaxArrayPreview)
		out := make([]any, preview, preview+1)
		for i := 0; i < preview; i++ {
			out[i] = truncateValue(rv.Index(i).Interface(), depth+1)
		}
		if n > truncateMaxArrayPreview {
			out = append(out, fmt.Sprintf("...and %d more items", n-truncateMaxArrayPreview))
		}
		return out
	}

	return TruncateString(fmt.Sprintf("%v", v))
}

func TruncateString(val string) string {
	if strings.HasPrefix(val, "data:") {
		if summary := summarizeDataURL(val); summary != "" {
			return summary
		}
	}
	if len(val) <= truncateMaxFieldSize {
		return val
	}
	end := truncateMaxFieldSize
	for end > 0 && !utf8.RuneStart(val[end]) {
		end--
	}
	return val[:end] + "..."
}

func summarizeDataURL(s string) string {
	rest := s[5:]
	commaIdx := strings.Index(rest, ",")
	if commaIdx == -1 {
		return ""
	}
	header, payload := rest[:commaIdx], rest[commaIdx+1:]
	mime := header
	isBase64 := false
	if semi := strings.Index(header, ";"); semi != -1 {
		mime = header[:semi]
		isBase64 = strings.Contains(header[semi:], "base64")
	}
	if mime == "" {
		mime = "unknown"
	}
	size := len(payload)
	suffix := ""
	if isBase64 {
		size = size * 3 / 4
		suffix = ";base64"
	}
	return fmt.Sprintf("[%s%s, %s]", mime, suffix, formatByteSize(size))
}

func formatByteSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.2fMB", float64(bytes)/1024/1024)
}

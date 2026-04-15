package model

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
)

type Content interface {
	StringContent() string
}
type ResponseContent[T Content] struct {
	Choices []T `json:"choices"`
}

func (m *ResponseContent[T]) StringContent() string {
	if len(m.Choices) > 0 {
		return m.Choices[0].StringContent()
	}
	return ""
}

type Delta struct {
	dto.Message `json:"delta"`
}

func (m Delta) StringContent() string {
	return m.Message.StringContent()
}

type Message struct {
	dto.Message `json:"message"`
}

func (m Message) StringContent() string {
	return m.Message.StringContent()
}
func newResponseContent[T Content](data []byte) (res ResponseContent[T], err error) {
	err = json.Unmarshal(data, &res)
	return
}
func getResponseContentStr[T Content](data []byte) (res string, err error) {
	var response ResponseContent[T]
	response, err = newResponseContent[T](data)
	if err != nil {
		return
	}
	res = response.StringContent()
	return
}
func GetMsgOutput(c *gin.Context) (output string) {
	var err error
	responseBody, ok := c.Get(common.KeyResponseWriter)
	if !ok {
		return
	}
	blw, ok := responseBody.(common.BodyLogWriter)
	if !ok {
		return
	}
	output = blw.String()
	if len(output) == 0 {
		return
	}
	relayMode := constant.Path2RelayMode(c.Request.URL.Path)
	switch relayMode {
	case constant.RelayModeChatCompletions:
		firstData := output[0]
		switch firstData {
		case '{':
			if output, err = getResponseContentStr[Message]([]byte(output)); err != nil {
				logger.LogWarn(c, fmt.Sprintf("unmarshal response failed: %s", err.Error()))
				return
			}
		case 'd':
			resSlice := strings.Split(output, "\n")
			var streamResBuilder strings.Builder
			for _, r := range resSlice {
				r = strings.TrimPrefix(r, "data:")
				r = strings.TrimSpace(r)
				if r == "" || r == "[DONE]" {
					continue
				}
				deltaContent, err := getResponseContentStr[Delta]([]byte(r))
				if err != nil {
					logger.LogWarn(c, fmt.Sprintf("unmarshal stream response failed: %s", err.Error()))
					continue
				}
				streamResBuilder.WriteString(deltaContent)
			}
			output = streamResBuilder.String()
		}
	case constant.RelayModeEmbeddings:
		var embeddingResponse dto.EmbeddingResponse
		if err = json.Unmarshal([]byte(output), &embeddingResponse); err != nil {
			logger.LogWarn(c, fmt.Sprintf("unmarshal embedding response failed: %s", err.Error()))
			return
		}
		// 记录向量输出摘要：模型名称、向量数量、维度信息
		summary := fmt.Sprintf("model=%s, embeddings=%d", embeddingResponse.Model, len(embeddingResponse.Data))
		if len(embeddingResponse.Data) > 0 {
			summary += fmt.Sprintf(", dimensions=%d", len(embeddingResponse.Data[0].Embedding))
		}
		output = summary
	case constant.RelayModeAudioSpeech:
		// TTS 语音合成返回二进制音频数据，只记录摘要
		audioFormat := detectAudioFormat([]byte(output))
		output = fmt.Sprintf("[audio/%s: %d bytes]", audioFormat, len(output))
	default:
	}

	// 处理输出内容：限制最大长度100k，检测二进制/base64图像或视频
	output = processContent(output)
	return
}

func processMultipartForm(form *multipart.Form) string {
	result := make(map[string]interface{})
	result["type"] = "multipart"

	totalSize := 0
	var fileNames []string
	fields := make(map[string]string)

	for _, files := range form.File {
		for _, file := range files {
			totalSize += int(file.Size)
			fileNames = append(fileNames, file.Filename)
		}
	}

	for fieldName, values := range form.Value {
		for _, value := range values {
			totalSize += len(value)
			const maxFieldValueLen = 2048
			displayValue := value
			if len(value) > maxFieldValueLen {
				displayValue = value[:maxFieldValueLen] + "..."
			}
			fields[fieldName] = displayValue
		}
	}

	if len(fileNames) > 0 {
		result["files"] = strings.Join(fileNames, ", ")
		result["file_count"] = len(fileNames)
	}
	if len(fields) > 0 {
		result["fields"] = fields
	}
	result["size"] = totalSize

	if jsonBytes, err := json.Marshal(result); err == nil {
		return string(jsonBytes)
	}
	return fmt.Sprintf(`{"type":"multipart","size":%d}`, totalSize)
}

func processContent(content string) string {
	const maxLength = 10 * 1024
	originalSize := len(content)

	// 检测并总结 multipart form data（文件上传）
	if result := detectMultipartFormData(content); result != nil {
		if jsonBytes, err := json.Marshal(result); err == nil {
			return string(jsonBytes)
		}
	}

	// 检测并总结原始二进制数据
	if result := detectRawBinaryData(content); result != nil {
		if jsonBytes, err := json.Marshal(result); err == nil {
			return string(jsonBytes)
		}
	}

	// 检测并总结媒体内容（base64 图片/视频）
	if result := detectMediaContent(content); result != nil {
		if jsonBytes, err := json.Marshal(result); err == nil {
			return string(jsonBytes)
		}
	}

	// 对于普通文本内容，保持原始格式，只在超长时截断
	if len(content) > maxLength {
		return content[:maxLength] + fmt.Sprintf("... [original length: %d bytes]", originalSize)
	}

	// 返回原始内容
	return content
}

func detectMultipartFormData(content string) map[string]interface{} {
	boundaryPattern := regexp.MustCompile(`(?m)^------WebKitFormBoundary[A-Za-z0-9]+`)
	if !boundaryPattern.MatchString(content) {
		return nil
	}

	filePattern := regexp.MustCompile(`Content-Disposition: form-data; name="([^"]+)"(?:; filename="([^"]+)")?`)
	fileMatches := filePattern.FindAllStringSubmatch(content, -1)

	if len(fileMatches) == 0 {
		return nil
	}

	result := make(map[string]interface{})
	result["type"] = "multipart"
	result["size"] = len(content)

	var fileNames []string
	fields := make(map[string]string)

	boundaryStr := boundaryPattern.FindString(content)
	parts := strings.Split(content, boundaryStr)

	for _, match := range fileMatches {
		if len(match) >= 3 {
			fieldName := match[1]
			filename := match[2]

			if filename != "" {
				fileNames = append(fileNames, filename)
			} else {
				fieldValue := extractFormFieldValue(parts, fieldName)
				if fieldValue != "" {
					const maxFieldValueLen = 2048
					if len(fieldValue) > maxFieldValueLen {
						fieldValue = fieldValue[:maxFieldValueLen] + "..."
					}
					fields[fieldName] = fieldValue
				}
			}
		}
	}

	if len(fileNames) > 0 {
		result["files"] = strings.Join(fileNames, ", ")
		result["file_count"] = len(fileNames)
	}
	if len(fields) > 0 {
		result["fields"] = fields
	}

	return result
}

func extractFormFieldValue(parts []string, fieldName string) string {
	pattern := fmt.Sprintf(`Content-Disposition: form-data; name="%s"`, regexp.QuoteMeta(fieldName))

	for _, part := range parts {
		if strings.Contains(part, pattern) {
			if idx := strings.Index(part, "\r\n\r\n"); idx >= 0 {
				value := part[idx+4:]
				value = strings.TrimSpace(value)
				return value
			}

			if idx := strings.Index(part, "\n\n"); idx >= 0 {
				value := part[idx+2:]
				value = strings.TrimSpace(value)
				return value
			}
		}
	}

	return ""
}

func detectRawBinaryData(content string) map[string]interface{} {
	if len(content) < 100 {
		return nil
	}

	binaryCount := 0
	totalCheck := len(content)
	if totalCheck > 1000 {
		totalCheck = 1000
	}

	for i := 0; i < totalCheck; i++ {
		c := content[i]
		if c < 32 && c != 9 && c != 10 && c != 13 {
			binaryCount++
		}
	}

	if float64(binaryCount)/float64(totalCheck) > 0.2 {
		mediaType := detectBinaryMediaType([]byte(content))

		result := make(map[string]interface{})
		result["type"] = "binary"
		result["size"] = len(content)

		if mediaType != "" && mediaType != "binary" {
			result["format"] = mediaType
		}

		return result
	}

	return nil
}

func detectBinaryMediaType(data []byte) string {
	if len(data) < 12 {
		return ""
	}

	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}

	if len(data) >= 4 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	if len(data) >= 4 && data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return "image/gif"
	}

	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return "image/webp"
	}

	if len(data) >= 12 {
		for i := 0; i < len(data)-4 && i < 20; i++ {
			if data[i] == 0x66 && data[i+1] == 0x74 && data[i+2] == 0x79 && data[i+3] == 0x70 {
				return "video/mp4"
			}
		}
	}

	return "binary"
}

func detectMediaContent(content string) map[string]interface{} {
	dataURLPattern := regexp.MustCompile(`data:(image|video)/([^;]+);base64,([A-Za-z0-9+/=]+)`)
	matches := dataURLPattern.FindAllStringSubmatch(content, -1)

	if len(matches) > 0 {
		result := make(map[string]interface{})
		result["type"] = "media"
		result["size"] = len(content)

		var mediaFormats []string
		totalMediaSize := 0

		for _, match := range matches {
			if len(match) >= 4 {
				mediaType := match[1]
				format := match[2]
				base64Data := match[3]

				dataSize := (len(base64Data) * 3) / 4
				if strings.HasSuffix(base64Data, "==") {
					dataSize -= 2
				} else if strings.HasSuffix(base64Data, "=") {
					dataSize -= 1
				}

				totalMediaSize += dataSize
				mediaFormats = append(mediaFormats, fmt.Sprintf("%s/%s", mediaType, format))
			}
		}

		cleaned := dataURLPattern.ReplaceAllString(content, "")
		cleaned = strings.TrimSpace(cleaned)

		if cleaned != "" && len(cleaned) < 1000 {
			result["text"] = cleaned
		}

		result["media_count"] = len(matches)
		result["media_formats"] = strings.Join(mediaFormats, ", ")
		result["media_size"] = totalMediaSize

		return result
	}

	base64Pattern := regexp.MustCompile(`[A-Za-z0-9+/]{1000,}={0,2}`)
	if base64Pattern.MatchString(content) {
		base64Matches := base64Pattern.FindAllString(content, -1)

		result := make(map[string]interface{})
		mediaCount := 0
		totalMediaSize := 0

		for _, b64str := range base64Matches {
			if decoded, err := base64.StdEncoding.DecodeString(b64str); err == nil {
				if isMediaBinary(decoded) {
					mediaCount++
					totalMediaSize += len(decoded)
				}
			}
		}

		if mediaCount > 0 {
			result["type"] = "media"
			result["size"] = len(content)
			result["media_count"] = mediaCount
			result["media_size"] = totalMediaSize

			cleaned := base64Pattern.ReplaceAllString(content, "")
			cleaned = strings.TrimSpace(cleaned)
			if cleaned != "" && len(cleaned) < 1000 {
				result["text"] = cleaned
			}

			return result
		}
	}

	return nil
}

func isMediaBinary(data []byte) bool {
	if len(data) < 12 {
		return false
	}

	if len(data) >= 4 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return true
	}

	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return true
	}

	if len(data) >= 4 && data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return true
	}

	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return true
	}

	if len(data) >= 12 {
		for i := 0; i < len(data)-4; i++ {
			if data[i] == 0x66 && data[i+1] == 0x74 && data[i+2] == 0x79 && data[i+3] == 0x70 {
				return true
			}
		}
	}

	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x41 && data[9] == 0x56 && data[10] == 0x49 && data[11] == 0x20 {
		return true
	}

	return false
}

func detectAudioFormat(data []byte) string {
	if len(data) < 12 {
		return "unknown"
	}

	if len(data) >= 3 {
		if data[0] == 0xFF && (data[1] == 0xFB || data[1] == 0xF3 || data[1] == 0xFA) {
			return "mp3"
		}
		if data[0] == 0x49 && data[1] == 0x44 && data[2] == 0x33 {
			return "mp3"
		}
	}

	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x41 && data[10] == 0x56 && data[11] == 0x45 {
		return "wav"
	}

	if len(data) >= 4 && data[0] == 0x4F && data[1] == 0x67 && data[2] == 0x67 && data[3] == 0x53 {
		return "ogg"
	}

	if len(data) >= 4 && data[0] == 0x66 && data[1] == 0x4C && data[2] == 0x61 && data[3] == 0x43 {
		return "flac"
	}

	if len(data) >= 2 && data[0] == 0xFF && (data[1] == 0xF1 || data[1] == 0xF9) {
		return "aac"
	}

	if len(data) >= 12 {
		searchLimit := len(data) - 4
		if searchLimit > 20 {
			searchLimit = 20
		}
		for i := 0; i < searchLimit; i++ {
			if data[i] == 0x66 && data[i+1] == 0x74 && data[i+2] == 0x79 && data[i+3] == 0x70 {
				return "m4a"
			}
		}
	}

	if len(data) >= 36 && data[0] == 0x4F && data[1] == 0x67 && data[2] == 0x67 && data[3] == 0x53 {
		if data[28] == 0x4F && data[29] == 0x70 && data[30] == 0x75 && data[31] == 0x73 {
			return "opus"
		}
	}

	return "unknown"
}

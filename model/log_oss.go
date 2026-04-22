package model

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/storage"
	storagecommon "github.com/QuantumNous/new-api/storage/common"
	"github.com/gin-gonic/gin"
)

const ossDiskSpillThreshold = 4 << 20

type OSSUploadTask struct {
	RequestId        string
	BasePath         string
	FileData         []byte
	DiskPath         string
	ContentType      string
	FileName         string
	PreserveFileName bool
}

var ossUploadQueue chan *OSSUploadTask

func getFileExtensionFromContentType(contentType string) string {
	mainType := strings.Split(contentType, ";")[0]
	mainType = strings.TrimSpace(mainType)

	switch mainType {
	case "application/json":
		return ".json"
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	case "image/bmp":
		return ".bmp"
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav":
		return ".wav"
	case "audio/ogg":
		return ".ogg"
	case "audio/webm":
		return ".weba"
	case "audio/aac":
		return ".aac"
	case "audio/flac":
		return ".flac"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "video/ogg":
		return ".ogv"
	case "video/mpeg":
		return ".mpeg"
	case "video/quicktime":
		return ".mov"
	case "text/plain":
		return ".txt"
	case "text/html":
		return ".html"
	case "text/xml":
		return ".xml"
	case "application/xml":
		return ".xml"
	case gin.MIMEMultipartPOSTForm, gin.MIMEPOSTForm:
		return ".form"
	case "application/octet-stream":
		return ".bin"
	default:
		return ".dat"
	}
}

func InitOSSUploadWorker() {
	if !common.OSSLogEnabled {
		return
	}

	// 创建有缓冲的通道，避免阻塞
	ossUploadQueue = make(chan *OSSUploadTask, 100)

	// 启动 3 个并发上传协程
	for i := 0; i < 3; i++ {
		go ossUploadWorker()
	}

	common.SysLog(fmt.Sprintf("OSS log upload worker initialized with 3 concurrent workers, base path: %s", common.OSSLogBasePath))
}

// ossUploadWorker OSS 上传工作协程
func ossUploadWorker() {
	for task := range ossUploadQueue {
		if err := uploadLogToOSS(task); err != nil {
			log.Printf("Failed to upload log to OSS: requestId=%s, error=%v", task.RequestId, err)
		}
	}
}

func parseSSEDataLines(streamData string) []string {
	lines := strings.Split(streamData, "\n")
	dataLines := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		dataLines = append(dataLines, data)
	}

	return dataLines
}

func convertChatCompletionsStreamToJSON(streamData string) (string, error) {
	lines := parseSSEDataLines(streamData)

	var (
		id                string
		model             string
		created           int64
		systemFingerprint string
		finishReason      string
		message           = make(map[string]interface{})
		toolCallsMap      = make(map[int]map[string]interface{})
		topLevelFields    = make(map[string]interface{})
	)

	stringFields := map[string]bool{
		"content": true, "reasoning_content": true, "audio_content": true,
	}

	for _, jsonData := range lines {

		if jsonData == "[DONE]" {
			continue
		}

		var chunk map[string]interface{}
		if err := common.UnmarshalJsonStr(jsonData, &chunk); err != nil {
			continue
		}

		if id == "" {
			if chunkId, ok := chunk["id"].(string); ok {
				id = chunkId
			}
		}
		if model == "" {
			if chunkModel, ok := chunk["model"].(string); ok {
				model = chunkModel
			}
		}
		if created == 0 {
			if chunkCreated, ok := chunk["created"].(float64); ok {
				created = int64(chunkCreated)
			}
		}
		if systemFingerprint == "" {
			if sf, ok := chunk["system_fingerprint"].(string); ok {
				systemFingerprint = sf
			}
		}

		for key, value := range chunk {
			if key == "usage" {
				if value != nil {
					topLevelFields[key] = value
				} else if _, exists := topLevelFields[key]; !exists {
					topLevelFields[key] = nil
				}
				continue
			}
			if key != "id" && key != "object" && key != "created" && key != "model" && key != "choices" && key != "system_fingerprint" {
				if _, exists := topLevelFields[key]; !exists {
					topLevelFields[key] = value
				}
			}
		}

		if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if delta, ok := choice["delta"].(map[string]interface{}); ok {
					for key, value := range delta {
						switch key {
						case "tool_calls":
							if toolCalls, ok := value.([]interface{}); ok {
								for _, tc := range toolCalls {
									if toolCall, ok := tc.(map[string]interface{}); ok {
										index := 0
										if idx, ok := toolCall["index"].(float64); ok {
											index = int(idx)
										}
										if _, exists := toolCallsMap[index]; !exists {
											toolCallsMap[index] = make(map[string]interface{})
										}
										if id, ok := toolCall["id"].(string); ok {
											toolCallsMap[index]["id"] = id
										}
										if typ, ok := toolCall["type"].(string); ok {
											toolCallsMap[index]["type"] = typ
										}
										if function, ok := toolCall["function"].(map[string]interface{}); ok {
											if _, exists := toolCallsMap[index]["function"]; !exists {
												toolCallsMap[index]["function"] = make(map[string]interface{})
											}
											funcMap := toolCallsMap[index]["function"].(map[string]interface{})
											if name, ok := function["name"].(string); ok {
												funcMap["name"] = name
											}
											if args, ok := function["arguments"].(string); ok {
												if existingArgs, exists := funcMap["arguments"].(string); exists {
													funcMap["arguments"] = existingArgs + args
												} else {
													funcMap["arguments"] = args
												}
											}
										}
									}
								}
							}
						case "function_call":
							if functionCall, ok := value.(map[string]interface{}); ok {
								if _, exists := message["function_call"]; !exists {
									message["function_call"] = make(map[string]interface{})
								}
								fcMap := message["function_call"].(map[string]interface{})
								if name, ok := functionCall["name"].(string); ok {
									fcMap["name"] = name
								}
								if args, ok := functionCall["arguments"].(string); ok {
									if existingArgs, exists := fcMap["arguments"].(string); exists {
										fcMap["arguments"] = existingArgs + args
									} else {
										fcMap["arguments"] = args
									}
								}
							}
						case "reasoning_details":
							if details, ok := value.([]interface{}); ok {
								if existing, exists := message["reasoning_details"].([]interface{}); exists {
									message["reasoning_details"] = append(existing, details...)
								} else {
									message["reasoning_details"] = details
								}
							}
						default:
							if strValue, ok := value.(string); ok && stringFields[key] {
								if existingValue, exists := message[key].(string); exists {
									message[key] = existingValue + strValue
								} else {
									message[key] = strValue
								}
							} else if _, exists := message[key]; !exists {
								message[key] = value
							}
						}
					}
				}
				if fr, ok := choice["finish_reason"].(string); ok && fr != "" {
					finishReason = fr
				}
			}
		}
	}

	if len(toolCallsMap) > 0 {
		toolCalls := make([]map[string]interface{}, 0, len(toolCallsMap))
		for i := 0; i < len(toolCallsMap); i++ {
			if tc, exists := toolCallsMap[i]; exists {
				toolCalls = append(toolCalls, tc)
			}
		}
		message["tool_calls"] = toolCalls
	}

	if id == "" && len(message) == 0 {
		return streamData, nil
	}

	completion := map[string]interface{}{
		"id":      id,
		"object":  "chat.completion",
		"created": created,
		"model":   model,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"message":       message,
				"finish_reason": finishReason,
			},
		},
	}

	if systemFingerprint != "" {
		completion["system_fingerprint"] = systemFingerprint
	}

	for key, value := range topLevelFields {
		completion[key] = value
	}

	result, err := common.Marshal(completion)
	if err != nil {
		return "", fmt.Errorf("failed to marshal completion: %w", err)
	}

	return string(result), nil
}

func convertResponsesStreamToJSON(streamData string) (string, error) {
	lines := parseSSEDataLines(streamData)

	var finalResponse map[string]interface{}
	for _, jsonData := range lines {
		if jsonData == "[DONE]" {
			continue
		}

		var event map[string]interface{}
		if err := common.UnmarshalJsonStr(jsonData, &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case "response.done", "response.completed":
			if resp, ok := event["response"].(map[string]interface{}); ok {
				finalResponse = resp
			}
		case "response.created":
			if finalResponse == nil {
				if resp, ok := event["response"].(map[string]interface{}); ok {
					finalResponse = resp
				}
			}
		default:
			if finalResponse == nil {
				if resp, ok := event["response"].(map[string]interface{}); ok {
					finalResponse = resp
				}
			}
		}
	}

	if finalResponse == nil {
		return streamData, nil
	}

	result, err := common.Marshal(finalResponse)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(result), nil
}

func convertClaudeMessagesStreamToJSON(streamData string) (string, error) {
	lines := parseSSEDataLines(streamData)

	var (
		message         map[string]interface{}
		toolInputDeltas = make(map[int]string)
	)

	for _, jsonData := range lines {
		if jsonData == "[DONE]" {
			continue
		}

		var event map[string]interface{}
		if err := common.UnmarshalJsonStr(jsonData, &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case "message_start":
			if msg, ok := event["message"].(map[string]interface{}); ok {
				message = msg
			}
		case "content_block_start":
			if message == nil {
				message = make(map[string]interface{})
			}
			index := getIntField(event, "index")
			contentBlock, _ := event["content_block"].(map[string]interface{})
			if contentBlock == nil {
				contentBlock = make(map[string]interface{})
			}
			content := ensureClaudeContentSlice(message)
			ensureContentIndex(&content, index)
			content[index] = contentBlock
			message["content"] = content
		case "content_block_delta":
			if message == nil {
				message = make(map[string]interface{})
			}
			index := getIntField(event, "index")
			delta, _ := event["delta"].(map[string]interface{})
			if delta == nil {
				continue
			}
			deltaType, _ := delta["type"].(string)
			content := ensureClaudeContentSlice(message)
			if index < 0 || index >= len(content) {
				continue
			}
			contentBlock, _ := content[index].(map[string]interface{})
			if contentBlock == nil {
				contentBlock = make(map[string]interface{})
				content[index] = contentBlock
			}
			switch deltaType {
			case "text_delta":
				if text, ok := delta["text"].(string); ok {
					if existing, ok := contentBlock["text"].(string); ok {
						contentBlock["text"] = existing + text
					} else {
						contentBlock["text"] = text
					}
				}
			case "input_json_delta":
				if partial, ok := delta["partial_json"].(string); ok {
					toolInputDeltas[index] = toolInputDeltas[index] + partial
				}
			}
			message["content"] = content
		case "content_block_stop":
			if message == nil {
				message = make(map[string]interface{})
			}
			index := getIntField(event, "index")
			content := ensureClaudeContentSlice(message)
			if index < 0 || index >= len(content) {
				continue
			}
			contentBlock, _ := content[index].(map[string]interface{})
			if contentBlock == nil {
				continue
			}
			if contentType, ok := contentBlock["type"].(string); ok && contentType == "tool_use" {
				if rawInput, ok := toolInputDeltas[index]; ok && rawInput != "" {
					var parsed interface{}
					if err := common.UnmarshalJsonStr(rawInput, &parsed); err == nil {
						contentBlock["input"] = parsed
					} else {
						contentBlock["input"] = rawInput
					}
				}
			}
			message["content"] = content
		case "message_delta":
			if delta, ok := event["delta"].(map[string]interface{}); ok {
				if message == nil {
					message = make(map[string]interface{})
				}
				for key, value := range delta {
					message[key] = value
				}
			}
		}
	}

	if message == nil {
		return streamData, nil
	}

	result, err := common.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	return string(result), nil
}

func ensureClaudeContentSlice(message map[string]interface{}) []interface{} {
	if message == nil {
		return []interface{}{}
	}
	if content, ok := message["content"].([]interface{}); ok {
		return content
	}
	return []interface{}{}
}

func ensureContentIndex(content *[]interface{}, index int) {
	if index < 0 {
		return
	}
	for len(*content) <= index {
		*content = append(*content, nil)
	}
}

func getIntField(m map[string]interface{}, key string) int {
	if m == nil {
		return -1
	}
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case int:
			return t
		case int64:
			return int(t)
		case float64:
			return int(t)
		}
	}
	return -1
}

func convertStreamToJSONByPath(streamData, path string) (string, error) {
	switch {
	case strings.HasSuffix(path, "/chat/completions"):
		return convertChatCompletionsStreamToJSON(streamData)
	case strings.HasSuffix(path, "/responses"):
		return convertResponsesStreamToJSON(streamData)
	case strings.HasSuffix(path, "/messages"):
		return convertClaudeMessagesStreamToJSON(streamData)
	default:
		return streamData, nil
	}
}

func EnqueueOSSUploadSingle(c *gin.Context, fileData []byte, contentType, fileName string) {
	enqueueOSSUploadFromContext(c, fileData, contentType, fileName, false)
}

func EnqueueOSSUploadSinglePreserveName(c *gin.Context, fileData []byte, contentType, fileName string) {
	enqueueOSSUploadFromContext(c, fileData, contentType, fileName, true)
}

func enqueueOSSUploadFromContext(c *gin.Context, fileData []byte, contentType, fileName string, preserveFileName bool) {
	requestId := common.GetContextKeyString(c, common.RequestIdKey)
	if requestId == "" {
		return
	}
	requestPath := ""
	if c.Request != nil && c.Request.URL != nil {
		requestPath = c.Request.URL.Path
	}
	enqueueOSSUpload(requestId, common.GetOSSFilePath(c), requestPath, fileData, contentType, fileName, preserveFileName)
}

func EnqueueOSSUploadByRequestID(requestID, requestPath string, fileData []byte, contentType, fileName string) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return
	}
	enqueueOSSUpload(requestID, common.GetOSSBasePathByRequestID(requestID), requestPath, fileData, contentType, fileName, false)
}

func enqueueOSSUpload(requestID, basePath, requestPath string, fileData []byte, contentType, fileName string, preserveFileName bool) {
	if len(fileData) == 0 {
		return
	}
	if contentType == "" {
		contentType = "application/json"
	}

	if contentType == "text/event-stream" {
		converted, err := convertStreamToJSONByPath(string(fileData), requestPath)
		if err != nil {
			log.Printf("Failed to convert stream to JSON for requestId %s: %v", requestID, err)
		} else {
			fileData = []byte(converted)
			contentType = "application/json"
		}
	}

	if len(ossUploadQueue) >= cap(ossUploadQueue) {
		log.Printf("OSS upload queue is full, dropping task for requestId: %s, fileName: %s", requestID, fileName)
		return
	}

	task := &OSSUploadTask{
		RequestId:        requestID,
		BasePath:         basePath,
		FileData:         fileData,
		ContentType:      contentType,
		FileName:         fileName,
		PreserveFileName: preserveFileName,
	}

	if len(fileData) >= ossDiskSpillThreshold {
		if path, err := common.WriteDiskCacheFile(common.DiskCacheTypeFile, fileData); err == nil {
			task.DiskPath = path
			task.FileData = nil
		} else {
			log.Printf("Failed to spill OSS task to disk (requestId=%s, fileName=%s): %v", requestID, fileName, err)
		}
	}

	select {
	case ossUploadQueue <- task:
	default:
		if task.DiskPath != "" {
			_ = os.Remove(task.DiskPath)
		}
		log.Printf("OSS upload queue is full, dropping task for requestId: %s, fileName: %s", requestID, fileName)
	}
}

func uploadLogToOSS(task *OSSUploadTask) error {
	ctx := context.Background()

	storageInstance, err := storage.NewStorageFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	objectName := task.FileName
	if !task.PreserveFileName {
		objectName += getFileExtensionFromContentType(task.ContentType)
	}
	objectKey := fmt.Sprintf("%s/%s", task.BasePath, objectName)

	reader, size, cleanup, err := openTaskPayload(task)
	if err != nil {
		return fmt.Errorf("failed to open payload %s: %w", task.FileName, err)
	}
	defer cleanup()

	if err := uploadSingleFile(ctx, storageInstance, objectKey, reader, size, task.RequestId, task.ContentType); err != nil {
		return fmt.Errorf("failed to upload %s: %w", task.FileName, err)
	}

	return nil
}

func openTaskPayload(task *OSSUploadTask) (io.Reader, int64, func(), error) {
	noop := func() {}
	if task.DiskPath != "" {
		f, err := os.Open(task.DiskPath)
		if err != nil {
			_ = os.Remove(task.DiskPath)
			return nil, 0, noop, err
		}
		fi, err := f.Stat()
		if err != nil {
			_ = f.Close()
			_ = os.Remove(task.DiskPath)
			return nil, 0, noop, err
		}
		cleanup := func() {
			_ = f.Close()
			_ = os.Remove(task.DiskPath)
		}
		return f, fi.Size(), cleanup, nil
	}
	return bytes.NewReader(task.FileData), int64(len(task.FileData)), noop, nil
}

func uploadSingleFile(ctx context.Context, storageInstance storage.Storage, objectKey string, reader io.Reader, size int64, requestId, contentType string) error {
	filename := filepath.Base(objectKey)

	opts := storagecommon.UploadOptions{
		Filename:    filename,
		ContentType: contentType,
		Purpose:     "log_content",
		UserID:      0,
		ObjectKey:   objectKey,
		Metadata: map[string]string{
			"original-filename": filename,
			"upload-time":       time.Now().Format(time.RFC3339),
			"request-id":        requestId,
			"timestamp":         fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	_, err := storageInstance.UploadFile(ctx, reader, size, opts)
	if err != nil {
		return fmt.Errorf("failed to upload to OSS: %w", err)
	}

	log.Printf("Successfully uploaded to OSS: %s", objectKey)
	return nil
}

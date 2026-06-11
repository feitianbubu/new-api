package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"resty.dev/v3"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/storage"
)

func applyVideoTaskResultURL(ctx context.Context, task *model.Task, taskResult *relaycommon.TaskInfo) {
	originalURL := strings.TrimSpace(taskResult.Url)
	if strings.HasPrefix(originalURL, "data:") {
		// data: URI (e.g. Vertex base64 encoded video) — keep in Data, not in ResultURL
		task.PrivateData.ResultURL = taskcommon.BuildProxyURL(task.TaskID)
		return
	}
	if originalURL == "" {
		// No URL from adaptor — construct proxy URL using public task ID
		task.PrivateData.ResultURL = taskcommon.BuildProxyURL(task.TaskID)
		return
	}

	// Direct upstream URL (e.g. Kling, Ali, Doubao, etc.)
	task.Url = originalURL
	if uploadedURL, err := uploadVideoOss(ctx, *task, originalURL); err != nil {
		logger.LogError(ctx, fmt.Sprintf("Upload video to OSS failed for task %s: %s", task.TaskID, err.Error()))
	} else if uploadedURL != "" {
		task.Url = uploadedURL
	}
	if task.Url != "" {
		task.PrivateData.ResultURL = task.Url
	} else {
		task.PrivateData.ResultURL = originalURL
	}
}

func uploadVideoOss(ctx context.Context, task model.Task, videoURL string) (string, error) {
	ext := extractURLFileExt(videoURL)
	if ext == "" {
		ext = ".mp4"
	}

	req := resty.New().R().
		SetDoNotParseResponse(true)

	requiresAuth := false
	channel, err := model.CacheGetChannel(task.ChannelId)
	if err != nil {
		return "", errors.Wrapf(err, "get channel %d failed", task.ChannelId)
	}
	switch channel.Type {
	case constant.ChannelTypeOpenAI, constant.ChannelTypeSora:
		videoURL = fmt.Sprintf("%s/v1/videos/%s/content", channel.GetBaseURL(), task.TaskID)
		req.SetHeader("Authorization", "Bearer "+channel.GetKey())
		requiresAuth = true
	case constant.ChannelTypeGemini:
		if videoURL, err = getGeminiVideoURL(channel, &task, task.PrivateData.Key); err != nil {
			return "", errors.Wrap(err, "get gemini video url failed")
		}
		req.Header.Set("x-goog-api-key", channel.GetKey())
		requiresAuth = true
		if parsedURL, parseErr := url.Parse(videoURL); parseErr == nil {
			if key := parsedURL.Query().Get("key"); key != "" {
				requiresAuth = false
			}
		}
	}

	filename := fmt.Sprintf("video_%s%s", task.TaskID, ext)
	objectKey := common.GetOSSFileKey(task.Properties.RequestId, filename)

	storageInstance, err := storage.NewStorageFromEnv()
	if err != nil {
		return "", errors.Wrap(err, "init storage failed")
	}

	presigner, isPresigner := storageInstance.(storage.Presigner)
	if !isPresigner {
		return "", fmt.Errorf("storage provider %q does not support presigned URLs", storageInstance.GetModelName())
	}

	if uploader, ok := storageInstance.(storage.URLUploader); ok && !requiresAuth && isHTTPURL(videoURL) {
		fileObj, uploadErr := uploader.UploadFileByURL(ctx, videoURL, storage.UploadOptions{
			Filename:  filename,
			Purpose:   "video",
			ObjectKey: objectKey,
			UserID:    task.UserId,
		})
		if uploadErr == nil {
			signedURL, err := presigner.PresignURL(ctx, fileObj.Key, 24*60*60)
			if err != nil {
				return "", errors.Wrap(err, "generate presigned url failed")
			}
			return signedURL, nil
		}
		logger.LogWarn(ctx, fmt.Sprintf("storage url fetch failed, falling back to download upload: %v", uploadErr))
	}

	resp, err := req.Get(videoURL)
	if err != nil {
		return "", errors.Wrap(err, "download video failed")
	}
	rawResp := resp.RawResponse
	if rawResp == nil || rawResp.Body == nil {
		return "", fmt.Errorf("download video has no body")
	}
	defer func() { _ = rawResp.Body.Close() }()

	if resp.IsError() {
		return "", fmt.Errorf("download video returned status %d", resp.StatusCode())
	}

	contentType := resp.Header().Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	var (
		reader io.Reader
		size   int64
	)

	if cl := resp.Header().Get("Content-Length"); cl != "" {
		if n, parseErr := strconv.ParseInt(cl, 10, 64); parseErr == nil && n >= 0 {
			size = n
			reader = rawResp.Body
		}
	}

	if reader == nil {
		// Content-Length missing, buffer to determine size
		data, readErr := io.ReadAll(rawResp.Body)
		if readErr != nil {
			return "", errors.Wrap(readErr, "read video body failed")
		}
		size = int64(len(data))
		reader = bytes.NewReader(data)
	}

	fileObj, err := storageInstance.UploadFile(ctx, reader, size, storage.UploadOptions{
		Filename:    filename,
		ContentType: contentType,
		Purpose:     "video",
		ObjectKey:   objectKey,
		UserID:      task.UserId,
	})
	if err != nil {
		return "", errors.Wrap(err, "upload video to storage failed")
	}

	signedURL, err := presigner.PresignURL(ctx, fileObj.Key, 24*60*60)
	if err != nil {
		return "", errors.Wrap(err, "generate presigned url failed")
	}

	return signedURL, nil
}

func getGeminiVideoURL(channel *model.Channel, task *model.Task, apiKey string) (string, error) {
	if channel == nil || task == nil {
		return "", fmt.Errorf("invalid channel or task")
	}

	if uri := extractGeminiVideoURLFromTaskData(task); uri != "" {
		return ensureAPIKey(uri, apiKey), nil
	}

	baseURL := constant.ChannelBaseURLs[channel.Type]
	if channel.GetBaseURL() != "" {
		baseURL = channel.GetBaseURL()
	}

	adaptor := GetTaskAdaptorFunc(constant.TaskPlatform(strconv.Itoa(channel.Type)))
	if adaptor == nil {
		return "", fmt.Errorf("gemini task adaptor not found")
	}

	if apiKey == "" {
		return "", fmt.Errorf("api key not available for task")
	}

	proxy := channel.GetSetting().Proxy
	resp, err := adaptor.FetchTask(baseURL, apiKey, map[string]any{
		"task_id": task.GetUpstreamTaskID(),
		"action":  task.Action,
		"req_key": task.Properties.UpstreamModelName,
	}, proxy)
	if err != nil {
		return "", fmt.Errorf("fetch task failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read task response failed: %w", err)
	}

	taskInfo, parseErr := adaptor.ParseTaskResult(body)
	if parseErr == nil && taskInfo != nil && taskInfo.RemoteUrl != "" {
		return ensureAPIKey(taskInfo.RemoteUrl, apiKey), nil
	}

	if uri := extractGeminiVideoURLFromPayload(body); uri != "" {
		return ensureAPIKey(uri, apiKey), nil
	}

	if parseErr != nil {
		return "", fmt.Errorf("parse task result failed: %w", parseErr)
	}

	return "", fmt.Errorf("gemini video url not found")
}

func extractGeminiVideoURLFromTaskData(task *model.Task) string {
	if task == nil || len(task.Data) == 0 {
		return ""
	}
	var payload map[string]any
	if err := common.Unmarshal(task.Data, &payload); err != nil {
		return ""
	}
	return extractGeminiVideoURLFromMap(payload)
}

func extractGeminiVideoURLFromPayload(body []byte) string {
	var payload map[string]any
	if err := common.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return extractGeminiVideoURLFromMap(payload)
}

func extractGeminiVideoURLFromMap(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if uri, ok := payload["uri"].(string); ok && uri != "" {
		return uri
	}
	if resp, ok := payload["response"].(map[string]any); ok {
		if uri := extractGeminiVideoURLFromResponse(resp); uri != "" {
			return uri
		}
	}
	return ""
}

func extractGeminiVideoURLFromResponse(resp map[string]any) string {
	if resp == nil {
		return ""
	}
	if gvr, ok := resp["generateVideoResponse"].(map[string]any); ok {
		if uri := extractGeminiVideoURLFromGeneratedSamples(gvr); uri != "" {
			return uri
		}
	}
	if videos, ok := resp["videos"].([]any); ok {
		for _, video := range videos {
			if vm, ok := video.(map[string]any); ok {
				if uri, ok := vm["uri"].(string); ok && uri != "" {
					return uri
				}
			}
		}
	}
	if uri, ok := resp["video"].(string); ok && uri != "" {
		return uri
	}
	if uri, ok := resp["uri"].(string); ok && uri != "" {
		return uri
	}
	return ""
}

func extractGeminiVideoURLFromGeneratedSamples(gvr map[string]any) string {
	if gvr == nil {
		return ""
	}
	if samples, ok := gvr["generatedSamples"].([]any); ok {
		for _, sample := range samples {
			if sm, ok := sample.(map[string]any); ok {
				if video, ok := sm["video"].(map[string]any); ok {
					if uri, ok := video["uri"].(string); ok && uri != "" {
						return uri
					}
				}
			}
		}
	}
	return ""
}

func ensureAPIKey(uri, key string) string {
	if key == "" || uri == "" {
		return uri
	}
	if strings.Contains(uri, "key=") {
		return uri
	}
	if strings.Contains(uri, "?") {
		return fmt.Sprintf("%s&key=%s", uri, key)
	}
	return fmt.Sprintf("%s?key=%s", uri, key)
}

func isHTTPURL(videoURL string) bool {
	return strings.HasPrefix(videoURL, "http")
}

func extractURLFileExt(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	ext := strings.ToLower(path.Ext(u.Path))
	if len(ext) < 2 {
		return ""
	}
	return ext
}

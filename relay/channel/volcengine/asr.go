package volcengine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	asrSubmitURL = "https://openspeech.bytedance.com/api/v3/auc/bigmodel/submit"
	asrQueryURL  = "https://openspeech.bytedance.com/api/v3/auc/bigmodel/query"

	asrCodeSuccess    = 20000000
	asrCodeProcessing = 20000001
	asrCodeQueued     = 20000002
	asrCodeSilent     = 20000003

	asrResourceID = "volc.seedasr.auc"

	asrPollInterval = 5 * time.Second
	asrPollTimeout  = 10 * time.Minute

	contextKeyASRRequestID  = "volcengine_asr_request_id"
	contextKeyASRSubmitBody = "volcengine_asr_submit_body"
)

// --- Request types ---

type DoubaoASRSubmitRequest struct {
	User    DoubaoASRUser    `json:"user"`
	Audio   DoubaoASRAudio   `json:"audio"`
	Request DoubaoASRReqInfo `json:"request"`
}

type DoubaoASRUser struct {
	UID string `json:"uid"`
}

type DoubaoASRAudio struct {
	Format   string `json:"format"`
	URL      string `json:"url"`
	Language string `json:"language,omitempty"`
}

type DoubaoASRReqInfo struct {
	ModelName      string `json:"model_name"`
	EnableITN      bool   `json:"enable_itn"`
	EnablePunc     bool   `json:"enable_punc"`
	ShowUtterances bool   `json:"show_utterances"`
}

// --- Response types (query body) ---

type DoubaoASRQueryResponse struct {
	Result    *DoubaoASRResult    `json:"result,omitempty"`
	AudioInfo *DoubaoASRAudioInfo `json:"audio_info,omitempty"`
}

type DoubaoASRResult struct {
	Text       string               `json:"text"`
	Utterances []DoubaoASRUtterance `json:"utterances,omitempty"`
}

type DoubaoASRUtterance struct {
	Text      string `json:"text"`
	StartTime int    `json:"start_time"`
	EndTime   int    `json:"end_time"`
}

type DoubaoASRAudioInfo struct {
	Duration int `json:"duration"` // milliseconds
}

// convertASRRequest extracts audio file from multipart, uploads to storage, and builds submit request body.
func (a *Adaptor) convertASRRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	audioFiles, err := channel.ExtractMultipartFilesFromMultipart(c, []string{"file"})
	if err != nil {
		return nil, fmt.Errorf("failed to extract audio file: %w", err)
	}
	if len(audioFiles) == 0 {
		return nil, fmt.Errorf("no audio file found in request")
	}

	fileHeader := audioFiles[0]
	userID := channel.GetUserIDFromContext(c)

	audioURL, err := channel.UploadMultipartFile(c, fileHeader, userID, channel.ImageUploadOptions{
		Purpose:        "volcengine_asr",
		ExpiresSeconds: 3600,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload audio file: %w", err)
	}

	audioFormat := detectAudioFormat(fileHeader.Filename)

	requestID := uuid.New().String()
	c.Set(contextKeyASRRequestID, requestID)

	audio := DoubaoASRAudio{
		Format: audioFormat,
		URL:    audioURL,
	}
	// Pass through OpenAI language parameter (ISO-639-1) to Doubao language format
	if request.Language != nil {
		var lang string
		if err := common.Unmarshal(request.Language, &lang); err == nil && lang != "" {
			audio.Language = lang
		}
	}

	submitReq := DoubaoASRSubmitRequest{
		User: DoubaoASRUser{
			UID: fmt.Sprintf("user_%d", userID),
		},
		Audio: audio,
		Request: DoubaoASRReqInfo{
			ModelName:      "bigmodel",
			EnableITN:      true,
			EnablePunc:     true,
			ShowUtterances: true,
		},
	}

	jsonData, err := common.Marshal(submitReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ASR submit request: %w", err)
	}

	return bytes.NewReader(jsonData), nil
}

// handleASRResponse performs the full submit → poll → response conversion flow.
func handleASRResponse(c *gin.Context, info *relaycommon.RelayInfo) (any, *types.NewAPIError) {
	submitBodyRaw, exists := c.Get(contextKeyASRSubmitBody)
	if !exists {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("ASR submit body not found in context"),
			types.ErrorCodeBadRequestBody,
			http.StatusInternalServerError,
		)
	}
	submitBody := submitBodyRaw.([]byte)

	requestIDRaw, _ := c.Get(contextKeyASRRequestID)
	requestID := requestIDRaw.(string)

	proxyURL := info.ChannelSetting.Proxy

	// Submit task
	submitCode, submitMsg, err := doASRSubmit(c.Request.Context(), info.ApiKey, requestID, proxyURL, submitBody)
	if err != nil {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("ASR submit failed: %w", err),
			types.ErrorCodeDoRequestFailed,
			http.StatusBadGateway,
		)
	}

	logger.LogInfo(c, fmt.Sprintf("ASR submit: code=%d, message=%s", submitCode, submitMsg))

	if submitCode != asrCodeSuccess && submitCode != asrCodeProcessing && submitCode != asrCodeQueued {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("ASR submit error: code=%d, message=%s", submitCode, submitMsg),
			types.ErrorCodeBadResponse,
			http.StatusBadGateway,
		)
	}

	// Poll for result
	result, err := pollASRResult(c.Request.Context(), info.ApiKey, requestID, proxyURL)
	if err != nil {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("ASR polling failed: %w", err),
			types.ErrorCodeDoRequestFailed,
			http.StatusGatewayTimeout,
		)
	}

	// Get response format
	responseFormat := "json"
	if audioReq, ok := info.Request.(*dto.AudioRequest); ok && audioReq.ResponseFormat != "" {
		responseFormat = audioReq.ResponseFormat
	}

	// Write response
	resultText := ""
	if result.Result != nil {
		resultText = result.Result.Text
	}

	switch responseFormat {
	case "text":
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(resultText))
	case "verbose_json":
		verboseResp := convertToVerboseJSON(result)
		c.JSON(http.StatusOK, verboseResp)
	default: // "json", "srt", "vtt" fallback to json
		c.JSON(http.StatusOK, dto.AudioResponse{Text: resultText})
	}

	// Calculate usage based on audio duration
	usage := &dto.Usage{
		PromptTokens: info.GetEstimatePromptTokens(),
		TotalTokens:  info.GetEstimatePromptTokens(),
	}
	if result.AudioInfo != nil && result.AudioInfo.Duration > 0 {
		durationSeconds := float64(result.AudioInfo.Duration) / 1000.0
		tokens := int(math.Ceil(durationSeconds))
		if tokens < 1 {
			tokens = 1
		}
		usage.PromptTokens = tokens
		usage.TotalTokens = tokens
	}

	return usage, nil
}

// doASRSubmit sends the submit request. Status code is in response header X-Api-Status-Code, body is empty.
func doASRSubmit(ctx context.Context, apiKey, requestID, proxyURL string, body []byte) (code int, message string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, asrSubmitURL, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	setASRSubmitHeaders(req, apiKey, requestID)

	client, err := service.GetHttpClientWithProxy(proxyURL)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get HTTP client: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	code, _ = strconv.Atoi(resp.Header.Get("X-Api-Status-Code"))
	message = resp.Header.Get("X-Api-Message")
	return code, message, nil
}

// doASRQuery sends the query request. Status code is in header, result JSON is in body.
func doASRQuery(ctx context.Context, apiKey, requestID, proxyURL string) (code int, result *DoubaoASRQueryResponse, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, asrQueryURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return 0, nil, err
	}
	setASRQueryHeaders(req, apiKey, requestID)

	client, err := service.GetHttpClientWithProxy(proxyURL)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	code, _ = strconv.Atoi(resp.Header.Get("X-Api-Status-Code"))

	if code == asrCodeSuccess || code == asrCodeSilent {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return code, nil, fmt.Errorf("failed to read ASR query response: %w", err)
		}
		var queryResp DoubaoASRQueryResponse
		if err := common.Unmarshal(respBody, &queryResp); err != nil {
			return code, nil, fmt.Errorf("failed to parse ASR query response: %w", err)
		}
		return code, &queryResp, nil
	}

	message := resp.Header.Get("X-Api-Message")
	return code, &DoubaoASRQueryResponse{}, fmt.Errorf("status: %d, message: %s", code, message)
}

// pollASRResult polls the query endpoint until the ASR task is complete.
func pollASRResult(ctx context.Context, apiKey, requestID, proxyURL string) (*DoubaoASRQueryResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, asrPollTimeout)
	defer cancel()

	ticker := time.NewTicker(asrPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("ASR polling timed out after %v", asrPollTimeout)
		case <-ticker.C:
			code, result, err := doASRQuery(ctx, apiKey, requestID, proxyURL)
			if err != nil && (code == asrCodeProcessing || code == asrCodeQueued || code == 0) {
				// Still processing or network error, continue polling
				continue
			}
			if err != nil {
				return nil, err
			}

			switch code {
			case asrCodeSuccess:
				return result, nil
			case asrCodeSilent:
				return &DoubaoASRQueryResponse{
					Result:    &DoubaoASRResult{Text: ""},
					AudioInfo: result.AudioInfo,
				}, nil
			case asrCodeProcessing, asrCodeQueued:
				continue
			default:
				return nil, fmt.Errorf("ASR query error: code=%d", code)
			}
		}
	}
}

func setASRSubmitHeaders(req *http.Request, apiKey, requestID string) {
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("X-Api-Resource-Id", asrResourceID)
	req.Header.Set("X-Api-Request-Id", requestID)
	req.Header.Set("X-Api-Sequence", "-1")
	req.Header.Set("Content-Type", "application/json")
}

func setASRQueryHeaders(req *http.Request, apiKey, requestID string) {
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("X-Api-Resource-Id", asrResourceID)
	req.Header.Set("X-Api-Request-Id", requestID)
	req.Header.Set("Content-Type", "application/json")
}

func detectAudioFormat(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp3":
		return "mp3"
	case ".wav":
		return "wav"
	case ".ogg":
		return "ogg"
	case ".raw", ".pcm":
		return "raw"
	default:
		return "mp3"
	}
}

func convertToVerboseJSON(resp *DoubaoASRQueryResponse) *dto.WhisperVerboseJSONResponse {
	verboseResp := &dto.WhisperVerboseJSONResponse{
		Task: "transcribe",
	}
	if resp.Result != nil {
		verboseResp.Text = resp.Result.Text
		for i, u := range resp.Result.Utterances {
			verboseResp.Segments = append(verboseResp.Segments, dto.Segment{
				Id:    i,
				Start: float64(u.StartTime) / 1000.0,
				End:   float64(u.EndTime) / 1000.0,
				Text:  u.Text,
			})
		}
	}
	if resp.AudioInfo != nil {
		verboseResp.Duration = float64(resp.AudioInfo.Duration) / 1000.0
	}
	return verboseResp
}

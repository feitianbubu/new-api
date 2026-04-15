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

type DoubaoASRSubmitRequest struct {
	User         DoubaoASRUser    `json:"user"`
	Audio        DoubaoASRAudio   `json:"audio"`
	Request      DoubaoASRReqInfo `json:"request"`
	Callback     string           `json:"callback,omitempty"`
	CallbackData string           `json:"callback_data,omitempty"`
}

type DoubaoASRUser struct {
	UID string `json:"uid"`
}

type DoubaoASRAudio struct {
	Format   string        `json:"format"`
	URL      string        `json:"url"`
	Language string        `json:"language,omitempty"`
	Codec    string        `json:"codec,omitempty"`
	Rate     *dto.IntValue `json:"rate,omitempty"`
	Bits     *dto.IntValue `json:"bits,omitempty"`
	Channel  *dto.IntValue `json:"channel,omitempty"`
}

type DoubaoASRReqInfo struct {
	ModelName              string           `json:"model_name"`
	SSDVersion             string           `json:"ssd_version,omitempty"`
	EnableITN              *dto.BoolValue   `json:"enable_itn,omitempty"`
	EnablePunc             *dto.BoolValue   `json:"enable_punc,omitempty"`
	EnableDDC              *dto.BoolValue   `json:"enable_ddc,omitempty"`
	EnableSpeakerInfo      *dto.BoolValue   `json:"enable_speaker_info,omitempty"`
	EnableChannelSplit     *dto.BoolValue   `json:"enable_channel_split,omitempty"`
	ShowUtterances         *dto.BoolValue   `json:"show_utterances,omitempty"`
	ShowSpeechRate         *dto.BoolValue   `json:"show_speech_rate,omitempty"`
	ShowVolume             *dto.BoolValue   `json:"show_volume,omitempty"`
	EnableLID              *dto.BoolValue   `json:"enable_lid,omitempty"`
	EnableEmotionDetection *dto.BoolValue   `json:"enable_emotion_detection,omitempty"`
	EnableGenderDetection  *dto.BoolValue   `json:"enable_gender_detection,omitempty"`
	VadSegment             *dto.BoolValue   `json:"vad_segment,omitempty"`
	EndWindowSize          *dto.IntValue    `json:"end_window_size,omitempty"`
	SensitiveWordsFilter   string           `json:"sensitive_words_filter,omitempty"`
	EnablePoiFC            *dto.BoolValue   `json:"enable_poi_fc,omitempty"`
	EnableMusicFC          *dto.BoolValue   `json:"enable_music_fc,omitempty"`
	Corpus                 *DoubaoASRCorpus `json:"corpus,omitempty"`
}

type DoubaoASRCorpus struct {
	BoostingTableName string `json:"boosting_table_name,omitempty"`
	CorrectTableName  string `json:"correct_table_name,omitempty"`
	Context           string `json:"context,omitempty"`
}

type DoubaoASRQueryResponse struct {
	Result    *DoubaoASRResult    `json:"result,omitempty"`
	AudioInfo *DoubaoASRAudioInfo `json:"audio_info,omitempty"`
}

type DoubaoASRResult struct {
	Text       string               `json:"text"`
	Utterances []DoubaoASRUtterance `json:"utterances,omitempty"`
}

type DoubaoASRUtterance struct {
	Text      string              `json:"text"`
	StartTime int                 `json:"start_time"`
	EndTime   int                 `json:"end_time"`
	Definite  bool                `json:"definite,omitempty"`
	Words     []DoubaoASRWord     `json:"words,omitempty"`
	Additions *DoubaoASRAdditions `json:"additions,omitempty"`
}

type DoubaoASRWord struct {
	Text          string  `json:"text"`
	StartTime     int     `json:"start_time"`
	EndTime       int     `json:"end_time"`
	BlankDuration int     `json:"blank_duration"`
	Confidence    float64 `json:"confidence,omitempty"`
}

type DoubaoASRAdditions struct {
	Speaker            string  `json:"speaker,omitempty"`
	Emotion            string  `json:"emotion,omitempty"`
	EmotionDegree      string  `json:"emotion_degree,omitempty"`
	EmotionDegreeScore string  `json:"emotion_degree_score,omitempty"`
	EmotionScore       string  `json:"emotion_score,omitempty"`
	Gender             string  `json:"gender,omitempty"`
	GenderScore        string  `json:"gender_score,omitempty"`
	LIDLang            string  `json:"lid_lang,omitempty"`
	SpeechRate         float64 `json:"speech_rate,omitempty"`
	Volume             float64 `json:"volume,omitempty"`
}

type DoubaoASRAudioInfo struct {
	Duration int `json:"duration"` // milliseconds
}

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

	requestID := generateRequestID()
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

	reqInfo := DoubaoASRReqInfo{
		ModelName: "bigmodel",
	}

	submitReq := DoubaoASRSubmitRequest{
		User: DoubaoASRUser{
			UID: fmt.Sprintf("user_%d", userID),
		},
		Audio:   audio,
		Request: reqInfo,
	}

	// Parse extra form fields and map to nested structure
	if c.Request.MultipartForm != nil {
		applyASRFormFields(c.Request.MultipartForm.Value, &submitReq)
	}

	jsonData, err := common.Marshal(submitReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ASR submit request: %w", err)
	}

	return bytes.NewReader(jsonData), nil
}

func parseBoolPtr(s string) *dto.BoolValue {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return nil
	}
	return common.GetPointer(dto.BoolValue(b))
}

func parseIntPtr(s string) *dto.IntValue {
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return common.GetPointer(dto.IntValue(n))
}

func applyASRFormFields(formValues map[string][]string, req *DoubaoASRSubmitRequest) {
	getVal := func(key string) (string, bool) {
		vals, ok := formValues[key]
		if !ok || len(vals) == 0 || vals[0] == "" {
			return "", false
		}
		return vals[0], true
	}

	// string fields
	stringFields := map[string]*string{
		"codec":                  &req.Audio.Codec,
		"ssd_version":            &req.Request.SSDVersion,
		"sensitive_words_filter": &req.Request.SensitiveWordsFilter,
		"callback":               &req.Callback,
		"callback_data":          &req.CallbackData,
	}
	for key, target := range stringFields {
		if v, ok := getVal(key); ok {
			*target = v
		}
	}

	// int fields
	intFields := map[string]**dto.IntValue{
		"rate":            &req.Audio.Rate,
		"bits":            &req.Audio.Bits,
		"channel":         &req.Audio.Channel,
		"end_window_size": &req.Request.EndWindowSize,
	}
	for key, target := range intFields {
		if v, ok := getVal(key); ok {
			if iv := parseIntPtr(v); iv != nil {
				*target = iv
			}
		}
	}

	// bool fields
	boolFields := map[string]**dto.BoolValue{
		"enable_itn":               &req.Request.EnableITN,
		"enable_punc":              &req.Request.EnablePunc,
		"enable_ddc":               &req.Request.EnableDDC,
		"enable_speaker_info":      &req.Request.EnableSpeakerInfo,
		"enable_channel_split":     &req.Request.EnableChannelSplit,
		"show_utterances":          &req.Request.ShowUtterances,
		"show_speech_rate":         &req.Request.ShowSpeechRate,
		"show_volume":              &req.Request.ShowVolume,
		"enable_lid":               &req.Request.EnableLID,
		"enable_emotion_detection": &req.Request.EnableEmotionDetection,
		"enable_gender_detection":  &req.Request.EnableGenderDetection,
		"vad_segment":              &req.Request.VadSegment,
		"enable_poi_fc":            &req.Request.EnablePoiFC,
		"enable_music_fc":          &req.Request.EnableMusicFC,
	}
	for key, target := range boolFields {
		if v, ok := getVal(key); ok {
			if bv := parseBoolPtr(v); bv != nil {
				*target = bv
			}
		}
	}

	// corpus fields
	var corpus DoubaoASRCorpus
	hasCorpus := false
	corpusFields := map[string]*string{
		"boosting_table_name": &corpus.BoostingTableName,
		"correct_table_name":  &corpus.CorrectTableName,
		"context":             &corpus.Context,
	}
	for key, target := range corpusFields {
		if v, ok := getVal(key); ok {
			*target = v
			hasCorpus = true
		}
	}
	if hasCorpus {
		req.Request.Corpus = &corpus
	}
}

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

	requestIDRaw, exists2 := c.Get(contextKeyASRRequestID)
	if !exists2 {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("ASR request ID not found in context"),
			types.ErrorCodeBadRequestBody,
			http.StatusInternalServerError,
		)
	}
	requestID := requestIDRaw.(string)

	client, err := service.GetHttpClientWithProxy(info.ChannelSetting.Proxy)
	if err != nil {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("failed to get HTTP client: %w", err),
			types.ErrorCodeDoRequestFailed,
			http.StatusInternalServerError,
		)
	}

	submitCode, submitMsg, err := doASRSubmit(c.Request.Context(), info.ApiKey, requestID, client, submitBody)
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

	result, err := pollASRResult(c.Request.Context(), info.ApiKey, requestID, client)
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

func doASRSubmit(ctx context.Context, apiKey, requestID string, client *http.Client, body []byte) (code int, message string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, asrSubmitURL, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	setASRHeaders(req, apiKey, requestID)
	req.Header.Set("X-Api-Sequence", "-1")

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	code, _ = strconv.Atoi(resp.Header.Get("X-Api-Status-Code"))
	message = resp.Header.Get("X-Api-Message")
	return code, message, nil
}

var emptyJSONBody = []byte("{}")

func doASRQuery(ctx context.Context, apiKey, requestID string, client *http.Client) (code int, result *DoubaoASRQueryResponse, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, asrQueryURL, bytes.NewReader(emptyJSONBody))
	if err != nil {
		return 0, nil, err
	}
	setASRHeaders(req, apiKey, requestID)

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	code, _ = strconv.Atoi(resp.Header.Get("X-Api-Status-Code"))

	if code == asrCodeSuccess || code == asrCodeSilent {
		var queryResp DoubaoASRQueryResponse
		if err := common.DecodeJson(resp.Body, &queryResp); err != nil {
			return code, nil, fmt.Errorf("failed to parse ASR query response: %w", err)
		}
		return code, &queryResp, nil
	}

	return code, nil, nil
}

func pollASRResult(ctx context.Context, apiKey, requestID string, client *http.Client) (*DoubaoASRQueryResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, asrPollTimeout)
	defer cancel()

	ticker := time.NewTicker(asrPollInterval)
	defer ticker.Stop()

	for {
		code, result, err := doASRQuery(ctx, apiKey, requestID, client)
		if err != nil {
			return nil, err
		}

		switch code {
		case asrCodeSuccess, asrCodeSilent:
			return result, nil
		case asrCodeProcessing, asrCodeQueued:
			// wait for next tick
		default:
			return nil, fmt.Errorf("ASR query error: code=%d", code)
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("ASR polling timed out after %v", asrPollTimeout)
		case <-ticker.C:
		}
	}
}

func setASRHeaders(req *http.Request, apiKey, requestID string) {
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("X-Api-Resource-Id", asrResourceID)
	req.Header.Set("X-Api-Request-Id", requestID)
	req.Header.Set("Content-Type", "application/json")
}

func detectAudioFormat(filename string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	switch ext {
	case "mp3", "wav", "ogg", "raw":
		return ext
	case "pcm":
		return "raw"
	default:
		return "mp3"
	}
}

type doubaoVerboseResponse struct {
	Task     string                 `json:"task"`
	Language string                 `json:"language,omitempty"`
	Duration float64                `json:"duration,omitempty"`
	Text     string                 `json:"text,omitempty"`
	Segments []doubaoVerboseSegment `json:"segments,omitempty"`
}

type doubaoVerboseSegment struct {
	ID        int                 `json:"id"`
	Start     float64             `json:"start"`
	End       float64             `json:"end"`
	Text      string              `json:"text"`
	Words     []DoubaoASRWord     `json:"words,omitempty"`
	Additions *DoubaoASRAdditions `json:"additions,omitempty"`
}

func convertToVerboseJSON(resp *DoubaoASRQueryResponse) *doubaoVerboseResponse {
	verboseResp := &doubaoVerboseResponse{
		Task: "transcribe",
	}
	if resp.Result != nil {
		verboseResp.Text = resp.Result.Text
		for i, u := range resp.Result.Utterances {
			verboseResp.Segments = append(verboseResp.Segments, doubaoVerboseSegment{
				ID:        i,
				Start:     float64(u.StartTime) / 1000.0,
				End:       float64(u.EndTime) / 1000.0,
				Text:      u.Text,
				Words:     u.Words,
				Additions: u.Additions,
			})
		}
	}
	if resp.AudioInfo != nil {
		verboseResp.Duration = float64(resp.AudioInfo.Duration) / 1000.0
	}
	return verboseResp
}

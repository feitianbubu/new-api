package vidu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

const viraBaseUrl = "https://api.gpugeek.com/predictions"

func isVira(c *gin.Context) bool {
	baseUrl := c.GetString("base_url")
	return isViraByUrl(baseUrl)
}
func isViraByUrl(baseUrl string) bool {
	return strings.Contains(baseUrl, viraBaseUrl)
}

func convertViraReq(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	if !isVira(c) {
		return nil
	}
	req.URL, _ = url.Parse(viraBaseUrl)
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	var model string
	switch info.Action {
	case constant.TaskActionGenerate:
		model = "Vira/image2video"
	case constant.TaskActionFirstTailGenerate:
		model = "Vira/startEnd2video"
	case constant.TaskActionReferenceGenerate:
		model = "Vira/reference2video"
	default:
		model = "Vira/text2video"
	}
	input := string(bodyBytes)
	input, err = removeViduPrefix(input)
	if err != nil {
		return err
	}
	body := fmt.Sprintf(`{"model": "%s", "input": %s}`, model, input)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.ContentLength = int64(len(body))
	logger.LogWarn(c, fmt.Sprintf("vira request: %s", body))
	return nil
}

func removeViduPrefix(body string) (string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return "", err
	}
	if modelVal, ok := data["model"].(string); ok {
		if strings.HasPrefix(modelVal, "vidu") {
			data["model"] = strings.TrimPrefix(modelVal, "vidu")
		}
	}
	for key, val := range data {
		if val == nil {
			delete(data, key)
		}
	}
	result, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

type ViraRes struct {
	Input struct {
		CallbackUrl       string   `json:"callback_url"`
		Duration          int      `json:"duration"`
		Images            []string `json:"images"`
		Model             string   `json:"model"`
		MovementAmplitude string   `json:"movement_amplitude"`
		Prompt            string   `json:"prompt"`
		Resolution        string   `json:"resolution"`
	} `json:"input"`
	Output      interface{} `json:"output"`
	Id          string      `json:"id"`
	Version     interface{} `json:"version"`
	CreatedAt   interface{} `json:"created_at"`
	StartedAt   time.Time   `json:"started_at"`
	CompletedAt interface{} `json:"completed_at"`
	Logs        interface{} `json:"logs"`
	Error       interface{} `json:"error"`
	Status      string      `json:"status"`
	Metrics     interface{} `json:"metrics"`
}

func convertViraRes(c *gin.Context, bodyBytes []byte, payload *responsePayload) error {
	if !isVira(c) {
		return nil
	}
	var viraRes ViraRes
	if err := json.Unmarshal(bodyBytes, &viraRes); err != nil {
		return err
	}
	payload.TaskId = viraRes.Id
	payload.State = viraRes.Status
	return nil
}

type ViraTaskResponse struct {
	State     string         `json:"state"`
	ErrCode   string         `json:"err_code"`
	Credits   int            `json:"credits"`
	Payload   string         `json:"payload"`
	Creations []ViraCreation `json:"creations"`
}

type ViraCreation struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	CoverURL string `json:"cover_url"`
}

func convertViraTaskReq(req *http.Request, taskID string) error {
	if !isViraByUrl(req.URL.String()) {
		return nil
	}
	req.URL, _ = url.Parse(fmt.Sprintf("%s/%s", viraBaseUrl, taskID))
	authorization := req.Header.Get("Authorization")
	authorization = strings.ReplaceAll(authorization, "Token ", "Bearer ")
	req.Header.Set("Authorization", authorization)
	return nil
}

func tryParseViraTaskResp(respBody []byte, taskResp *taskResultResponse) {
	if taskResp.State == "" {
		var alt struct {
			ID          string      `json:"id"`
			Input       interface{} `json:"input"`
			Output      interface{} `json:"output"`
			Status      string      `json:"status"`
			Logs        interface{} `json:"logs"`
			Error       interface{} `json:"error"`
			CreatedAt   interface{} `json:"created_at"`
			StartedAt   interface{} `json:"started_at"`
			CompletedAt interface{} `json:"completed_at"`
			Metrics     interface{} `json:"metrics"`
		}
		if err := json.Unmarshal(respBody, &alt); err != nil {
			common.SysError(fmt.Sprintf("tryParseViraTaskResp fail: %s", err))
		} else if alt.Status != "" {
			if alt.Status == "succeeded" {
				alt.Status = "success"
			}
			taskResp.State = alt.Status
			taskResp.Creations = []creation{
				{
					ID:       alt.ID,
					URL:      fmt.Sprintf("%v", alt.Output),
					CoverURL: "",
				},
			}
			taskResp.Payload = ""
			taskResp.ErrCode = fmt.Sprintf("%v", alt.Error)
		}
	}
}

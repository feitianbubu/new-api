package vidu

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"

	"github.com/pkg/errors"
)

// ============================
// Request / Response structures
// ============================

type requestPayload struct {
	Model             string         `json:"model"`
	Images            []string       `json:"images,omitempty"`
	StartImage        string         `json:"start_image,omitempty"`
	ImageSettings     []imageSetting `json:"image_settings,omitempty"`
	Subjects          []subject      `json:"subjects,omitempty"`
	Prompt            string         `json:"prompt,omitempty"`
	Duration          int            `json:"duration,omitempty"`
	Seed              int            `json:"seed,omitempty"`
	AspectRatio       string         `json:"aspect_ratio,omitempty"`
	Resolution        string         `json:"resolution,omitempty"`
	MovementAmplitude string         `json:"movement_amplitude,omitempty"`
	Bgm               bool           `json:"bgm,omitempty"`
	Audio             bool           `json:"audio,omitempty"`
	OffPeak           bool           `json:"off_peak,omitempty"`
	Watermark         bool           `json:"watermark,omitempty"`
	WmURL             string         `json:"wm_url,omitempty"`
	WmPosition        any            `json:"wm_position,omitempty"`
	MetaData          string         `json:"meta_data,omitempty"`
	Payload           string         `json:"payload,omitempty"`
	CallbackUrl       string         `json:"callback_url,omitempty"`
}

type responsePayload struct {
	TaskId            string         `json:"task_id"`
	State             string         `json:"state"`
	Model             string         `json:"model"`
	Images            []string       `json:"images,omitempty"`
	StartImage        string         `json:"start_image,omitempty"`
	ImageSettings     []imageSetting `json:"image_settings,omitempty"`
	Subjects          []subject      `json:"subjects,omitempty"`
	Prompt            string         `json:"prompt,omitempty"`
	Duration          int            `json:"duration,omitempty"`
	Seed              int            `json:"seed,omitempty"`
	AspectRatio       string         `json:"aspect_ratio,omitempty"`
	Resolution        string         `json:"resolution,omitempty"`
	Bgm               bool           `json:"bgm,omitempty"`
	Audio             bool           `json:"audio,omitempty"`
	MovementAmplitude string         `json:"movement_amplitude,omitempty"`
	OffPeak           bool           `json:"off_peak,omitempty"`
	Watermark         bool           `json:"watermark,omitempty"`
	WmURL             string         `json:"wm_url,omitempty"`
	WmPosition        any            `json:"wm_position,omitempty"`
	MetaData          string         `json:"meta_data,omitempty"`
	Payload           string         `json:"payload,omitempty"`
	Credits           int            `json:"credits,omitempty"`
	CreatedAt         string         `json:"created_at,omitempty"`
}

type imageSetting struct {
	Prompt   string `json:"prompt,omitempty"`
	KeyImage string `json:"key_image,omitempty"`
	Duration int    `json:"duration,omitempty"`
}

type subject struct {
	ID      string   `json:"id,omitempty"`
	Images  []string `json:"images,omitempty"`
	VoiceID string   `json:"voice_id,omitempty"`
}

type taskResultResponse struct {
	State     string     `json:"state"`
	ErrCode   string     `json:"err_code"`
	Credits   int        `json:"credits"`
	Payload   string     `json:"payload"`
	Creations []creation `json:"creations"`
}

type creation struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	CoverURL string `json:"cover_url"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	body, err := a.convertToRequestPayload(&req, info)
	if err != nil {
		return nil, err
	}

	//switch info.Action {
	//case constant.TaskActionReferenceGenerate, constant.TaskActionTextGenerate:
	//	// 参考图生视频和文生视频只能用 viduq2 模型, 不能带有pro或turbo后缀 https://platform.vidu.cn/docs/reference-to-video
	//	if strings.Contains(body.Model, "viduq2") {
	//		body.Model = "viduq2"
	//	}
	//case constant.TaskActionGenerate, constant.TaskActionFirstTailGenerate:
	//	// 图生视频和首尾帧生视频只能用 viduq2-turbo 或 viduq2-pro
	//	if body.Model == "viduq2" {
	//		body.Model = "viduq2-turbo"
	//	}
	//}

	action := constant.TaskActionTextGenerate
	if len(body.Images) == 1 {
		action = constant.TaskActionGenerate
	} else if len(body.Images) > 1 {
		action = constant.TaskActionFirstTailGenerate
	}
	if len(body.Subjects) > 0 {
		action = constant.TaskActionReferenceGenerate
	}
	if body.StartImage != "" {
		action = constant.TaskActionMultiFrame
	}
	if metaAction, ok := req.Metadata["action"]; ok {
		action, _ = metaAction.(string)
	}
	info.Action = action

	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	var path string
	switch info.Action {
	case constant.TaskActionGenerate:
		path = "img2video"
	case constant.TaskActionFirstTailGenerate:
		path = "start-end2video"
	case constant.TaskActionReferenceGenerate:
		path = "reference2video"
	case constant.TaskActionMultiFrame:
		path = "multiframe"
	default:
		path = "text2video"
	}
	return fmt.Sprintf("%s/ent/v2/%s", a.baseURL, path), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Token "+info.ApiKey)

	if err := convertViraReq(c, req, info); err != nil {
		return fmt.Errorf("convert vira request failed: %w", err)
	}
	return nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}

	var vResp responsePayload
	err = common.Unmarshal(responseBody, &vResp)
	if err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrap(err, fmt.Sprintf("%s", responseBody)), "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	if vResp.State == "failed" {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("task failed"), "task_failed", http.StatusBadRequest)
		return
	}

	if err = convertViraRes(c, responseBody, &vResp); err != nil {
		return "", nil, service.TaskErrorWrapper(err, "convert_vira_response_failed", http.StatusInternalServerError)
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return vResp.TaskId, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	url := fmt.Sprintf("%s/ent/v2/tasks/%s/creations", baseUrl, taskID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Token "+key)

	if err = convertViraTaskReq(req, taskID); err != nil {
		return nil, fmt.Errorf("convert vira task request failed: %w", err)
	}

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{"viduq2", "viduq1", "vidu2.0", "vidu1.5"}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "vidu"
}

// ============================
// helpers
// ============================

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) (*requestPayload, error) {
	r := requestPayload{
		Model:             taskcommon.DefaultString(info.UpstreamModelName, "viduq1"),
		Images:            req.Images,
		Prompt:            req.Prompt,
		Duration:          taskcommon.DefaultInt(req.Duration, 5),
		Resolution:        taskcommon.DefaultString(req.Size, "1080p"),
		MovementAmplitude: "auto",
		Bgm:               false,
	}
	if err := taskcommon.UnmarshalMetadata(req.Metadata, &r); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}
	return &r, nil
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	taskInfo := &relaycommon.TaskInfo{}

	var taskResp taskResultResponse
	err := common.Unmarshal(respBody, &taskResp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}

	tryParseViraTaskResp(respBody, &taskResp)

	state := taskResp.State
	switch state {
	case "created", "queueing":
		taskInfo.Status = model.TaskStatusSubmitted
	case "processing":
		taskInfo.Status = model.TaskStatusInProgress
	case "success":
		taskInfo.Status = model.TaskStatusSuccess
		if len(taskResp.Creations) > 0 {
			taskInfo.Url = taskResp.Creations[0].URL
		}
		if taskResp.Credits > 0 {
			taskInfo.CompletionTokens = taskResp.Credits
			taskInfo.TotalTokens = taskResp.Credits
		}
	case "failed":
		taskInfo.Status = model.TaskStatusFailure
		if taskResp.ErrCode != "" {
			taskInfo.Reason = taskResp.ErrCode
		}
	default:
		return nil, fmt.Errorf("unknown task state: %s", state)
	}

	return taskInfo, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var viduResp taskResultResponse
	if err := common.Unmarshal(originTask.Data, &viduResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal vidu task data failed")
	}

	openAIVideo := originTask.ToOpenAIVideo()
	if viduResp.State == "failed" && viduResp.ErrCode != "" {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: viduResp.ErrCode,
			Code:    viduResp.ErrCode,
		}
	}

	return common.Marshal(openAIVideo)
}

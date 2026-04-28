package suno

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// OfficialSunoData 官方 Suno API 的数据结构
type OfficialSunoData struct {
	ID         string  `json:"id"`
	AudioURL   string  `json:"audioUrl"`
	StreamURL  string  `json:"streamAudioUrl"`
	ImageURL   string  `json:"imageUrl"`
	Prompt     string  `json:"prompt"`
	ModelName  string  `json:"modelName"`
	Title      string  `json:"title"`
	Tags       string  `json:"tags"`
	CreateTime int64   `json:"createTime"`
	Duration   float64 `json:"duration"`
}

// OfficialSunoResponseData 官方 Suno API 的响应数据
type OfficialSunoResponseData struct {
	TaskID        string `json:"taskId"`
	ParentMusicID string `json:"parentMusicId"`
	Param         string `json:"param"`
	Response      struct {
		TaskID   string             `json:"taskId"`
		SunoData []OfficialSunoData `json:"sunoData"`
	} `json:"response"`
	Status       string  `json:"status"`
	Type         string  `json:"type"`
	ErrorCode    *string `json:"errorCode"`
	ErrorMessage *string `json:"errorMessage"`
	CreateTime   int64   `json:"createTime"`
}

// OfficialSunoResponse 官方 Suno API 的完整响应结构 (导出供 controller 使用)
type OfficialSunoResponse struct {
	Code int                      `json:"code"`
	Msg  string                   `json:"msg"`
	Data OfficialSunoResponseData `json:"data"`
}

func (r *OfficialSunoResponse) IsSuccess() bool {
	return r.Code == 200
}

// ToStandardResponse 将官方 API 响应转换为标准的 TaskResponse 格式
func (r *OfficialSunoResponse) ToStandardResponse() dto.TaskResponse[[]dto.SunoDataResponse] {
	var failReason string
	if r.Data.ErrorMessage != nil {
		failReason = *r.Data.ErrorMessage
	}

	var finishTime int64
	var url string
	if r.Data.Status == "SUCCESS" {
		finishTime = time.Now().Unix()
		if sunoData := r.Data.Response.SunoData; len(sunoData) > 0 {
			url = sunoData[0].AudioURL
		}
	}

	// 将官方数据转换为 JSON 存储
	var dataBytes []byte
	if len(r.Data.Response.SunoData) > 0 {
		dataBytes, _ = json.Marshal(r.Data.Response.SunoData)
	}

	sunoDataResponse := dto.SunoDataResponse{
		TaskID:     r.Data.TaskID,
		Status:     r.Data.Status,
		FailReason: failReason,
		Url:        url,
		SubmitTime: r.Data.CreateTime / 1000, // to seconds
		StartTime:  r.Data.CreateTime / 1000,
		FinishTime: finishTime,
		Data:       dataBytes,
	}

	return dto.TaskResponse[[]dto.SunoDataResponse]{
		Code:    "success",
		Message: "success",
		Data:    []dto.SunoDataResponse{sunoDataResponse},
	}
}

func (a *TaskAdaptor) DoResponseOfficial(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}

	var officialResp OfficialSunoResponse
	if err := common.Unmarshal(responseBody, &officialResp); err != nil {
		err = errors.Wrapf(err, "response body: %s", string(responseBody))
		taskErr = service.TaskErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if !officialResp.IsSuccess() {
		taskErr = service.TaskErrorWrapper(errors.New(officialResp.Msg), fmt.Sprintf("%d", officialResp.Code), http.StatusInternalServerError)
		return
	}

	upstreamTaskID := officialResp.Data.TaskID
	officialResp.Data.TaskID = info.PublicTaskID
	c.JSON(resp.StatusCode, officialResp)

	return upstreamTaskID, nil, nil
}

// FetchTaskOfficial: 官方 record-info 单次只接受一个 taskId，逐个查询后合并为标准 TaskResponse 格式返回。
func (a *TaskAdaptor) FetchTaskOfficial(baseUrl, key string, body map[string]any) (*http.Response, error) {
	ids, ok := body["ids"].([]string)
	if !ok || len(ids) == 0 {
		return nil, fmt.Errorf("ids array is required in body")
	}

	items := make([]dto.SunoDataResponse, 0, len(ids))
	for _, taskId := range ids {
		if taskId == "" {
			continue
		}
		item, err := a.fetchOfficialOneTask(baseUrl, key, taskId)
		if err != nil {
			common.SysLog(fmt.Sprintf("fetch suno official task %s error: %v", taskId, err))
			continue
		}
		items = append(items, item)
	}

	respBody, err := common.Marshal(dto.TaskResponse[[]dto.SunoDataResponse]{
		Code:    dto.TaskSuccessCode,
		Message: "success",
		Data:    items,
	})
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(respBody)),
	}, nil
}

func (a *TaskAdaptor) fetchOfficialOneTask(baseUrl, key, taskId string) (dto.SunoDataResponse, error) {
	var empty dto.SunoDataResponse
	requestUrl := fmt.Sprintf("%s/api/v1/generate/record-info?taskId=%s", baseUrl, taskId)
	req, err := http.NewRequest(http.MethodGet, requestUrl, nil)
	if err != nil {
		return empty, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := service.GetHttpClient().Do(req)
	if err != nil {
		return empty, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return empty, err
	}
	standard, err := ParseResponseItems(bodyBytes)
	if err != nil {
		return empty, err
	}
	if len(standard.Data) == 0 {
		return empty, nil
	}
	return standard.Data[0], nil
}

func ParseResponseItems(responseBody []byte) (dto.TaskResponse[[]dto.SunoDataResponse], error) {
	var responseItems dto.TaskResponse[[]dto.SunoDataResponse]
	var officialResp OfficialSunoResponse
	if err := common.Unmarshal(responseBody, &officialResp); err != nil {
		return responseItems, errors.Wrapf(err, "parse official API response error, body: %s", string(responseBody))
	}
	if !officialResp.IsSuccess() {
		return responseItems, fmt.Errorf("official API error: %s", officialResp.Msg)
	}
	return officialResp.ToStandardResponse(), nil
}

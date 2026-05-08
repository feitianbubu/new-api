package oas

import (
	"github.com/gin-gonic/gin"
)

// JimengAPI
// @Summary 即梦视频生成API
// @Description 即梦官方API的统一入口，通过Action参数区分不同操作
// @Description
// @Description **提交任务**: POST /jimeng/?Action=CVSync2AsyncSubmitTask&Version=2022-08-31
// @Description - 请求体使用JimengSubmitTaskRequest格式
// @Description - 返回JimengSubmitTaskResponse格式
// @Description
// @Description **查询任务**: POST /jimeng/?Action=CVSync2AsyncGetResult&Version=2022-08-31
// @Description - 请求体使用JimengQueryTaskRequest格式
// @Description - 返回JimengQueryTaskResponse格式
// @Description
// @Description 官方文档: https://www.volcengine.com/docs/85621/1544774
// @Tags Origin
// @Accept json
// @Produce json
// @Param Authorization header string true "用户认证令牌 (Access-Token: sk-xxxx)"
// @Param Action query string true "操作类型" Enums(CVSync2AsyncSubmitTask,CVSync2AsyncGetResult)
// @Param Version query string true "API版本，固定值：2022-08-31" Enums(2022-08-31)
// @Param request body JimengUnifiedRequest true "请求参数（根据Action不同使用不同格式）"
// @Success 200 {object} JimengUnifiedResponse "API响应（根据Action不同返回不同格式）"
// @Failure 400 {object} dto.OpenAIError "请求参数错误"
// @Failure 401 {object} dto.OpenAIError "未授权"
// @Failure 403 {object} dto.OpenAIError "无权限"
// @Failure 500 {object} dto.OpenAIError "服务器内部错误"
// @Router /jimeng/ [post]
func JimengAPI(c *gin.Context) {
}

// JimengSubmitTaskRequest 即梦提交任务请求参数
type JimengSubmitTaskRequest struct {
	ReqKey           string   `json:"req_key" binding:"required" example:"jimeng_vgfm_t2v_l20" enums:"jimeng_vgfm_t2v_l20,jimeng_vgfm_i2v_l20"`
	Prompt           string   `json:"prompt,omitempty" example:"一只猫在花园里弹钢琴"`
	BinaryDataBase64 []string `json:"binary_data_base64,omitempty" example:"[]"`
	ImageUrls        []string `json:"image_urls,omitempty" example:["https://example.com/image.jpg"]`
	Seed             int64    `json:"seed,omitempty" example:"-1"`
	AspectRatio      string   `json:"aspect_ratio,omitempty" example:"16:9" enums:"16:9,9:16,1:1"`
}

// JimengQueryTaskRequest 即梦查询任务请求参数
type JimengQueryTaskRequest struct {
	ReqKey string `json:"req_key" binding:"required" example:"jimeng_vgfm_t2v_l20" enums:"jimeng_vgfm_t2v_l20,jimeng_vgfm_i2v_l20"`
	TaskID string `json:"task_id" binding:"required" example:"20231201-123456-abcdef"`
}

// JimengSubmitTaskResponse 即梦提交任务响应
type JimengSubmitTaskResponse struct {
	Code      int    `json:"code" example:"10000"`
	Message   string `json:"message" example:"success"`
	RequestId string `json:"request_id" example:"req_123456789"`
	Data      struct {
		TaskID string `json:"task_id" example:"20231201-123456-abcdef"`
	} `json:"data"`
}

// JimengQueryTaskResponse 即梦查询任务响应
type JimengQueryTaskResponse struct {
	Code int `json:"code" example:"10000"`
	Data struct {
		BinaryDataBase64 []interface{} `json:"binary_data_base64" example:"[]"`
		ImageUrls        interface{}   `json:"image_urls" example:"null"`
		RespData         string        `json:"resp_data" example:""`
		Status           string        `json:"status" example:"done" enums:"in_queue,processing,done,failed"`
		VideoUrl         string        `json:"video_url" example:"https://example.com/video.mp4"`
	} `json:"data"`
	Message     string `json:"message" example:"success"`
	RequestId   string `json:"request_id" example:"req_123456789"`
	Status      int    `json:"status" example:"200"`
	TimeElapsed string `json:"time_elapsed" example:"30.5s"`
}

// JimengUnifiedRequest 即梦统一请求格式
type JimengUnifiedRequest struct {
	// 提交任务时使用以下字段
	ReqKey      string `json:"req_key,omitempty" example:"jimeng_vgfm_t2v_l20"`
	Prompt      string `json:"prompt,omitempty" example:"一只猫在花园里弹钢琴"`
	Seed        int64  `json:"seed,omitempty" example:"-1"`
	AspectRatio string `json:"aspect_ratio,omitempty" example:"16:9"`

	// 查询任务时使用以下字段
	TaskID string `json:"task_id,omitempty" example:"20231201-123456-abcdef"`
}

// JimengUnifiedResponse 即梦统一响应格式
type JimengUnifiedResponse struct {
	Code      int    `json:"code" example:"10000"`
	Message   string `json:"message" example:"success"`
	RequestId string `json:"request_id" example:"req_123456789"`

	// 提交任务响应
	Data *struct {
		TaskID string `json:"task_id,omitempty" example:"20231201-123456-abcdef"`
		// 查询任务响应
		BinaryDataBase64 []interface{} `json:"binary_data_base64,omitempty"`
		ImageUrls        interface{}   `json:"image_urls,omitempty"`
		RespData         string        `json:"resp_data,omitempty"`
		Status           string        `json:"status,omitempty" example:"done"`
		VideoUrl         string        `json:"video_url,omitempty" example:"https://example.com/video.mp4"`
	} `json:"data,omitempty"`

	// 查询任务特有字段
	Status      int    `json:"status,omitempty" example:"200"`
	TimeElapsed string `json:"time_elapsed,omitempty" example:"30.5s"`
}

// JimengText2VideoRequest 即梦文生视频请求 (兼容格式)
type JimengText2VideoRequest struct {
	Model    string `json:"model" example:"jimeng_vgfm_t2v_l20"`
	Prompt   string `json:"prompt" binding:"required" example:"一只猫在花园里弹钢琴"`
	Metadata struct {
		ReqKey      string `json:"req_key" example:"jimeng_vgfm_t2v_l20"`
		Seed        int64  `json:"seed" example:"-1"`
		AspectRatio string `json:"aspect_ratio" example:"16:9"`
	} `json:"metadata"`
}

// JimengImage2VideoRequest 即梦图生视频请求 (兼容格式)
type JimengImage2VideoRequest struct {
	Model    string `json:"model" example:"jimeng_vgfm_i2v_l20"`
	Prompt   string `json:"prompt,omitempty" example:"猫咪在花园里弹钢琴"`
	Image    string `json:"image" binding:"required" example:"https://example.com/cat.jpg"`
	Metadata struct {
		ReqKey      string `json:"req_key" example:"jimeng_vgfm_i2v_l20"`
		Seed        int64  `json:"seed" example:"-1"`
		AspectRatio string `json:"aspect_ratio" example:"16:9"`
	} `json:"metadata"`
}

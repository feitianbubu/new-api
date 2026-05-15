package oas

import "github.com/gin-gonic/gin"

// SunoSubmitConcatReq Request for song concatenation
type SunoSubmitConcatReq struct {
	ClipID   string `json:"clip_id"`   // ID of the song clip to concatenate
	IsInfill bool   `json:"is_infill"` // Whether it's fill mode
}

// SunoSubmitMusicReq Request for music generation
type SunoSubmitMusicReq struct {
	// 歌词内容,自定义模式下必需
	Prompt string `json:"prompt" example:"儿童上学歌曲"`
	// 歌曲风格
	Style string `json:"style" example:"Classical"`
	// 歌曲标题
	Title string `json:"title" example:"hi"`
	// 是否自定义模式
	CustomMode bool `json:"customMode" example:"true"`
	// 模型版本,可选值: V4_5ALL
	Model string `json:"model" example:"V4_5ALL"`
	// 声音性别,m:男性,f:女性
	VocalGender string `json:"vocalGender,omitempty" example:"m"`
	// 风格权重,0-1之间
	StyleWeight float64 `json:"styleWeight,omitempty" example:"0.65"`
	// 奇怪约束,0-1之间
	WeirdnessConstraint float64 `json:"weirdnessConstraint,omitempty" example:"0.65"`
	// 音频权重,0-1之间
	AudioWeight float64 `json:"audioWeight,omitempty" example:"0.65"`
}

// SunoSubmitLyricsReq Request for lyrics generation
type SunoSubmitLyricsReq struct {
	Prompt     string `json:"prompt"`                // Theme or keywords for lyrics
	NotifyHook string `json:"notify_hook,omitempty"` // Callback URL for notification
}

// SunoSubmitUploadCoverReq Request for upload-and-cover-audio
type SunoSubmitUploadCoverReq struct {
	// 上传的音频文件 URL，无论 customMode/instrumental 取值都必填，音频不超过 8 分钟（V4_5ALL 不超过 1 分钟）
	UploadUrl string `json:"uploadUrl" example:"https://example.com/sample.mp3"`
	// 模型版本，可选 V4、V4_5、V4_5PLUS、V4_5ALL、V5、V5_5
	Model string `json:"model" example:"V5_5"`
	// 是否启用自定义模式
	CustomMode bool `json:"customMode" example:"true"`
	// 是否为纯音乐（无歌词）
	Instrumental bool `json:"instrumental" example:"false"`
	// 歌词内容；非自定义模式必填；自定义模式且 instrumental=false 时必填
	Prompt string `json:"prompt,omitempty" example:"一段平静舒缓的钢琴曲，带有柔和的旋律"`
	// 音乐风格；自定义模式必填
	Style string `json:"style,omitempty" example:"古典"`
	// 歌曲标题；自定义模式必填
	Title string `json:"title,omitempty" example:"宁静钢琴冥想"`
	// 排除的音乐风格或特征
	NegativeTags string `json:"negativeTags,omitempty" example:"重金属, 强节奏鼓点"`
	// 人格 ID，仅自定义模式可用
	PersonaId string `json:"personaId,omitempty" example:"persona_123"`
	// Persona 模型类型：style_persona（默认）或 voice_persona（仅 V5）
	PersonaModel string `json:"personaModel,omitempty" example:"style_persona"`
	// 声音性别，m：男性，f：女性
	VocalGender string `json:"vocalGender,omitempty" example:"m"`
	// 风格权重，0-1
	StyleWeight float64 `json:"styleWeight,omitempty" example:"0.65"`
	// 创意发散/奇异度约束，0-1
	WeirdnessConstraint float64 `json:"weirdnessConstraint,omitempty" example:"0.65"`
	// 输入音频影响力权重，0-1
	AudioWeight float64 `json:"audioWeight,omitempty" example:"0.65"`
}

// SunoSubmitMusic
// @Summary 生成歌曲(废弃)
// @Description 根据提示生成新的歌曲，支持灵感模式、自定义模式、续写
// @Description Suno: https://docs.sunoapi.org/cn/suno-api/generate-music
// @Tags Suno
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body SunoSubmitMusicReq true "歌曲生成请求参数"
// @Success 200 {object} dto.GoAPIFetchResponseData "成功响应，返回任务ID"
// @Router /suno/submit/music [post]
// @Example curl -X POST 'http://localhost/suno/submit/music' \
// @Example     -H 'Authorization: Bearer YOUR_API_KEY' \
// @Example     -H 'Content-Type: application/json' \
// @Example     -d '{
// @Example         "prompt": "[Verse]\\nWalking down the streets\\nBeneath the city lights...",
// @Example         "tags": "emotional punk",
// @Example         "mv": "chirp-v4",
// @Example         "title": "City Lights"
// @Example     }'
//func SunoSubmitMusic(c *gin.Context) {}

// SunoGenerate
// @Summary 生成歌曲
// @Description 根据提示生成新的歌曲，支持灵感模式、自定义模式、续写
// @Description Suno: https://docs.sunoapi.org/cn/suno-api/generate-music
// @Tags Suno
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body SunoSubmitMusicReq true "歌曲生成请求参数"
// @Success 200 {object} dto.GoAPIFetchResponseData "成功响应，返回任务ID"
// @Router /suno/api/v1/generate [post]
// @Example curl -X POST 'http://localhost/suno/submit/music' \
// @Example     -H 'Authorization: Bearer YOUR_API_KEY' \
// @Example     -H 'Content-Type: application/json' \
// @Example     -d '{
// @Example         "prompt": "[Verse]\\nWalking down the streets\\nBeneath the city lights...",
// @Example         "tags": "emotional punk",
// @Example         "mv": "chirp-v4",
// @Example         "title": "City Lights"
// @Example     }'
func SunoGenerate(c *gin.Context) {}

// SunoGenerateUploadCover
// @Summary 翻唱歌曲
// @Description 上传一段音频并按新风格翻唱，保留原始旋律
// @Description Suno: https://docs.sunoapi.org/cn/suno-api/upload-and-cover-audio
// @Tags Suno
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body SunoSubmitUploadCoverReq true "翻唱请求参数"
// @Success 200 {object} dto.GoAPIFetchResponseData "成功响应，返回任务ID"
// @Router /suno/api/v1/generate/upload-cover [post]
// @Example curl -X POST 'http://localhost/suno/api/v1/generate/upload-cover' \
// @Example     -H 'Authorization: Bearer YOUR_API_KEY' \
// @Example     -H 'Content-Type: application/json' \
// @Example     -d '{
// @Example         "uploadUrl": "https://example.com/sample.mp3",
// @Example         "customMode": false,
// @Example         "instrumental": false,
// @Example         "model": "V4_5",
// @Example         "prompt": "jazz cover",
// @Example         "callBackUrl": "https://example.com/callback"
// @Example     }'
func SunoGenerateUploadCover(c *gin.Context) {}

// SunoSubmitLyrics swagger 生成歌词
// @Summary 生成歌词
// @Description 根据提示生成歌词
// @Description 基于 Suno-API 实现的Suno代理接口
// @Tags Origin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body SunoSubmitLyricsReq true "歌词生成请求参数"
// @Success 200 {object} dto.GoAPIFetchResponseData "成功响应，返回任务ID"
// @Router /suno/submit/lyrics [post]
// @Example curl -X POST 'http://localhost/suno/submit/lyrics' \
// @Example     -H 'Authorization: Bearer YOUR_API_KEY' \
// @Example     -H 'Content-Type: application/json' \
// @Example     -d '{
// @Example         "prompt": "dance"
// @Example     }'
func SunoSubmitLyrics(c *gin.Context) {}

// SunoUploadAudio swagger 上传音频
// @Summary 上传音频
// @Description 上传音频文件
// @Description 基于 Suno-API 实现的Suno代理接口
// @Tags Origin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param url body string true "要上传的音频文件的 URL 地址" example(http://cdnimg.example.com/ai/2024-06-18/d416d9c3c34eb22c7d8c094831d8dbd0.mp3)
// @Success 200 {object} dto.GoAPIFetchResponseData "成功响应，返回任务ID"
// @Router /suno/uploads/audio-url [post]
// @Example curl -X POST 'http://localhost/suno/uploads/audio-url' \
// @Example     -H 'Authorization: Bearer YOUR_API_KEY' \
// @Example     -H 'Content-Type: application/json' \
// @Example     -d '{
// @Example         "url": "http://cdnimg.example.com/ai/2024-06-18/d416d9c3c34eb22c7d8c094831d8dbd0.mp3"
// @Example     }'
func SunoUploadAudio(c *gin.Context) {}

// SunoSubmitConcat swagger 歌曲拼接
// @Summary 歌曲拼接
// @Description 将多个音频片段拼接为一首完整的歌曲
// @Description 基于 Suno-API 实现的Suno代理接口
// @Tags Origin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body SunoSubmitConcatReq true "拼接请求参数"
// @Success 200 {object} dto.GoAPIFetchResponseData "成功响应，返回任务ID"
// @Router /suno/submit/concat [post]
// @Example curl -X POST 'http://localhost/suno/submit/concat' \
// @Example     -H 'Authorization: Bearer YOUR_API_KEY' \
// @Example     -H 'Content-Type: application/json' \
// @Example     -d '{
// @Example         "clip_id": "extend 后的 歌曲ID",
// @Example         "is_infill": false
// @Example     }'
func SunoSubmitConcat(c *gin.Context) {}

// SunoBatchFetch swagger 批量查询任务状态
// @Summary 批量查询任务状态
// @Description 批量获取多个任务的状态和结果
// @Description 基于 Suno-API 实现的Suno代理接口
// @Tags Origin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param ids query []string true "要查询的任务 ID 列表"
// @Param action query string false "任务类型，可选值: MUSIC、LYRICS" example(MUSIC)
// @Success 200 {object} []dto.SunoDataResponse "成功响应，返回任务对象数组"
// @Router /suno/fetch [post]
// @Example curl -X POST 'http://localhost/suno/fetch' \
// @Example     -H 'Authorization: Bearer YOUR_API_KEY' \
// @Example     -H 'Content-Type: application/json' \
// @Example     -d '{
// @Example         "ids": ["task_id"],
// @Example         "action": "MUSIC"
// @Example     }'
func SunoBatchFetch(c *gin.Context) {}

// SunoSingleFetch swagger 单个查询任务状态
// @Summary 查询任务
// @Description 查询单个任务的状态和结果
// @Description 基于 Suno-API 实现的Suno代理接口
// @Tags Suno
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "任务 ID"
// @Success 200 {object} dto.SunoDataResponse "成功响应，返回任务对象"
// @Router /suno/fetch/{id} [get]
// @Example curl -X GET 'http://localhost/suno/fetch/{{task_id}}' \
// @Example     -H 'Authorization: Bearer YOUR_API_KEY'
func SunoSingleFetch(c *gin.Context) {}

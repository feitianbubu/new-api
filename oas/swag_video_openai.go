package oas

import (
	"github.com/gin-gonic/gin"
)

// OpenAIVideos
// @Summary 视频/3D生成
// @Description 根据文本提示词生成视频/3D。可选择上传参考图片来引导任务生成。
// @Description
// @Description * OpenAI: https://developers.openai.com/api/reference/resources/videos/methods/create
// @Description * 阿里万相: https://www.alibabacloud.com/help/zh/model-studio/image-to-video-api-reference
// @Description * 快乐马: https://help.aliyun.com/zh/model-studio/happyhorse-text-to-video-api-reference
// @Description * 可灵: https://app.klingai.com/cn/dev/document-api/apiReference/model/skillsMap
// @Description * 即梦: https://www.volcengine.com/docs/85621/1792707
// @Description * 即梦数字人: https://www.volcengine.com/docs/85621/1829013
// @Description * 豆包视频: https://www.volcengine.com/docs/82379/1520758
// @Description * Vidu: https://platform.vidu.cn/docs/text-to-video
// @Description * 海螺: https://platform.minimaxi.com/docs/api-reference/video-generation-intro
// @Description * Veo: https://ai.google.dev/gemini-api/docs/video
// @Description * 豆包3D: https://www.volcengine.com/docs/82379/1856293
// @Description * 影眸3D: https://www.volcengine.com/docs/82379/2279947
// @Description * 数美3D: https://www.volcengine.com/docs/82379/2307070
// @Description
// @Description ### 示例请求:
// @Description		```
// @Description     curl https://api.openai.com/v1/videos \
// @Description       -H "Authorization: Bearer $OPENAI_API_KEY" \
// @Description       -F "model=sora-2" \
// @Description       -F "prompt=一只花猫在舞台上弹钢琴" \
// @Description       -F "input_reference=@image.jpg"
// @Description       -F "metadata={"input":{"audio_url":"https://help-static-aliyun-doc.aliyuncs.com/file-manage-files/zh-CN/20250925/ozwpvi/rap.mp3"}}"
// @Description     ```
// @Description ### Vidu 智能多帧: metadata参数示例:
// @Description * 适用模型: `viduq2-turbo`
// @Description ```json
// @Description {
// @Description   "start_image": "https://prod-ss-images.s3.cn-northwest-1.amazonaws.com.cn/vidu-maas/template/image2video.png",
// @Description   "image_settings": [
// @Description     {
// @Description       "prompt": "Smooth transition between key frames.",
// @Description       "key_image": "https://help-static-aliyun-doc.aliyuncs.com/file-manage-files/zh-CN/20250925/iclsnx/input2.png",
// @Description       "duration": 5
// @Description     },
// @Description     {
// @Description       "prompt": "Smooth transition between key frames.",
// @Description       "key_image": "https://help-static-aliyun-doc.aliyuncs.com/file-manage-files/zh-CN/20250925/thtclx/input1.png",
// @Description       "duration": 5
// @Description     }
// @Description   ],
// @Description   "resolution": "540p",
// @Description   "watermark": false,
// @Description   "wm_position": "bottom_left"
// @Description }
// @Description ```
// @Description ### 即梦数字人 1.5: metadata参数示例:
// @Description * 适用模型: `jimeng_realman_avatar_picture_omni_v15`
// @Description ```json
// @Description {
// @Description   "image_url": "https://v1-kling.klingai.com/kcdn/cdn-kcdn112452/kling-qa-test/multi-1.png",
// @Description   "audio_url": "https://example.com/demo/hi.wav"
// @Description }
// @Description ```
// @Description * `同时支持image_url和audio_url通过本地文件上传`
// @Description ### 豆包 Seedance 2.0: metadata参数示例:
// @Description * 适用模型: `doubao-seedance-2-0-fast-260128`
// @Description * 支持视频编辑
// @Description ```json
// @Description {
// @Description   "content": [
// @Description     {
// @Description       "type": "video_url",
// @Description       "video_url": {
// @Description         "url": "https://ark-project.tos-cn-beijing.volces.com/doc_video/r2v_extend_video2.mp4"
// @Description       },
// @Description       "role": "reference_video"
// @Description     }
// @Description   ]
// @Description }
// @Description ```
// @Description * 示例请求:
// @Description ```
// @Description curl http://localhost:30001/v1/videos \
// @Description   --request POST \
// @Description   --header 'Content-Type: multipart/form-data' \
// @Description   --form 'prompt=场景改为赛博朋克风,以玻璃幕墙为主,画框改为显示器' \
// @Description   --form 'model=doubao-seedance-2-0-fast-260128' \
// @Description   --form 'seconds=' \
// @Description   --form 'size=' \
// @Description   --form 'metadata={"content":[{"type":"video_url","video_url":{"url":"https://ark-project.tos-cn-beijing.volces.com/doc_video/r2v_extend_video2.mp4"},"role":"reference_video"}]}'
// @Description ```
// @Tags OpenAI
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param prompt formData string true "描述要生成的视频内容的文本提示词" example:"一只花猫在舞台上弹钢琴"
// @Param input_reference formData file false "可选的参考图片或视频文件，用于引导视频生成"
// @Param model formData string false "使用的视频生成模型" default:"sora-2" example:"sora-2"
// @Param seconds formData string false "视频时长（秒）" default:"4" example:"4"
// @Param size formData string false "输出分辨率，格式为 宽x高" default:"720x1280" example:"720x1280"
// @Param metadata formData string false "厂商自定义参数，以 JSON 字符串形式传递"
// @Success 200 {object} dto.OpenAIVideoResponse "Successfully created video generation task"
// @Router /v1/videos [post]
func OpenAIVideos(c *gin.Context) {}

// OpenAIVideosRetrieve
// @Summary 视频/3D查询
// @Description 根据任务ID查询任务生成的状态和结果。可获取生成进度、输出URL等信息。
// @Description
// @Description 参考文档: https://developers.openai.com/api/reference/resources/videos/methods/retrieve
// @Description
// @Description 示例请求:
// @Description		```
// @Description     curl https://api.openai.com/v1/videos/req_abc123 \
// @Description       -H "Authorization: Bearer $OPENAI_API_KEY"
// @Description     ```
// @Tags OpenAI
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param task_id path string true "视频生成任务的唯一标识符" example:"req_abc123"
// @Router /v1/videos/{task_id} [get]
func OpenAIVideosRetrieve(c *gin.Context) {}

// OpenAIVideosContent
// @Summary 视频/3D下载
// @Description 根据任务ID下载已有效期内的文件。
// @Description
// @Description 注意: 必须等待视频生成任务完成后才能下载
// @Description     ```
// @Tags OpenAI
// @Accept json
// @Security BearerAuth
// @Param task_id path string true "视频生成任务的唯一标识符" example:"req_abc123"
// @Success 200 {file} file "视频文件"
// @Router /v1/videos/{task_id}/content [get]
func OpenAIVideosContent(c *gin.Context) {}

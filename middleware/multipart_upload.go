package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/channel"

	"github.com/gin-gonic/gin"
)

const multipartUploadedFilenamesKey = "multipart_uploaded_filenames"

// channelsNeedingURLConversion 是适配器通过 c.PostFormArray 读取已上传 URL 的通道。
// 其他通道（OpenAI 直通的 audio/transcriptions、images/edits 等）继续读取原始 multipart.FileHeader，
// 跳过上传以避免对象存储白费。新增此类通道时需同步本表。
var channelsNeedingURLConversion = map[int]struct{}{
	constant.ChannelTypeAli:         {},
	constant.ChannelTypeVolcEngine:  {},
	constant.ChannelTypeJimeng:      {},
	constant.ChannelTypeDoubaoVideo: {},
}

// uploadableFields 是上述通道适配器实际通过 c.PostFormArray 读取的字段集合。
// 不在此集合内的文件字段不上传，避免客户端误传冗余文件触发无意义 OSS 写入。
var uploadableFields = map[string]struct{}{
	"input_reference": {},
	"image":           {},
	"image_url":       {},
	"video_url":       {},
	"audio_url":       {},
	"file":            {},
}

// MultipartFileToURL 将 multipart 文件字段统一上传到对象存储后，把生成的 URL
// 回写到同名 PostForm 字符串字段。userID 来源于 c.Get("id")，channel type 来源于
// Distribute()，必须挂在两者之后。
func MultipartFileToURL() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.ContentType() != gin.MIMEMultipartPOSTForm {
			c.Next()
			return
		}
		if _, ok := channelsNeedingURLConversion[common.GetContextKeyInt(c, constant.ContextKeyChannelType)]; !ok {
			c.Next()
			return
		}
		form, err := c.MultipartForm()
		if err != nil {
			abortWithOpenAiMessage(c, http.StatusBadRequest, "parse multipart form failed: "+err.Error())
			return
		}
		if len(form.File) == 0 {
			c.Next()
			return
		}

		userID := channel.GetUserIDFromContext(c)
		options := channel.ImageUploadOptions{
			Purpose:        "multipart_upload",
			ExpiresSeconds: 3600,
		}

		filenames := make(map[string][]string)

		for field, fileHeaders := range form.File {
			base := normalizeMultipartFieldKey(field)
			if _, ok := uploadableFields[base]; !ok {
				continue
			}
			for _, fh := range fileHeaders {
				url, err := channel.UploadMultipartFile(c, fh, userID, options)
				if err != nil {
					logger.LogWarn(c, "upload multipart file failed, fallback to raw file: "+err.Error())
					continue
				}
				c.Request.PostForm.Add(base, url)
				filenames[base] = append(filenames[base], fh.Filename)
			}
		}

		c.Set(multipartUploadedFilenamesKey, filenames)

		c.Next()
	}
}

// normalizeMultipartFieldKey 把 OpenAI SDK 列表字段（image[]、image[0]）归一到基础字段名。
// 对应 openai-python 默认 array_format=brackets 与 openai-node 数组追加 "[]" 的行为。
func normalizeMultipartFieldKey(key string) string {
	if idx := strings.IndexByte(key, '['); idx > 0 {
		return key[:idx]
	}
	return key
}

// MultipartUploadedFilename 返回指定字段对应的第一个原始文件名，
// 用于从扩展名推断 MIME / 格式的下游场景。
func MultipartUploadedFilename(c *gin.Context, field string) string {
	raw, ok := c.Get(multipartUploadedFilenamesKey)
	if !ok {
		return ""
	}
	m, ok := raw.(map[string][]string)
	if !ok {
		return ""
	}
	if names := m[field]; len(names) > 0 {
		return names[0]
	}
	return ""
}

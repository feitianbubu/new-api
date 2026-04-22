package common

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

var ChannelRequestBodyContextEnabled bool
var ChannelRequestBodyContextMaxBytes int64
var OSSLogEnabled = false
var OSSLogBasePath = "logs"

func GetOSSPath() string {
	if !OSSLogEnabled {
		return ""
	}
	bucket := os.Getenv("TOS_GO_SDK_BUCKET")
	region := os.Getenv("TOS_GO_SDK_REGION")
	basePath := OSSLogBasePath

	if bucket == "" || region == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.tos-%s.volces.com/%s", bucket, region, basePath)
}
func GetOSSFilePath(c *gin.Context) string {
	requestId := c.GetString(RequestIdKey)
	return GetOSSBasePathByRequestID(requestId)
}

func GetOSSBasePathByRequestID(requestId string) string {
	subPath := lo.Substring(requestId, 0, 8)
	return fmt.Sprintf("%s/%s/%s", OSSLogBasePath, subPath, requestId)
}
func GetOSSFileKey(requestId string, filename string) string {
	subPath := lo.Substring(requestId, 0, 8)
	return fmt.Sprintf("%s/%s/%s/%s", OSSLogBasePath, subPath, requestId, filename)
}
func GetOSSFileURL(fileKey string) string {
	return fmt.Sprintf("%s/%s", GetOSSPath(), fileKey)
}

func SetChannelRequestBodyIfEnabled(c *gin.Context, jsonData []byte) {
	if !ChannelRequestBodyContextEnabled {
		return
	}
	if ChannelRequestBodyContextMaxBytes > 0 && int64(len(jsonData)) > ChannelRequestBodyContextMaxBytes {
		truncated := truncateChannelRequestBody(jsonData)
		if truncated == "" {
			return
		}
		SetContextKey(c, constant.ContextKeyChannelRequestBody, truncated)
		return
	}
	SetContextKey(c, constant.ContextKeyChannelRequestBody, string(jsonData))
}

func SetChannelRequestBodyFromReader(c *gin.Context, requestBody io.Reader) {
	if !ChannelRequestBodyContextEnabled {
		return
	}
	br, ok := requestBody.(*bytes.Reader)
	if !ok {
		return
	}
	data := make([]byte, br.Len())
	_, _ = io.ReadFull(br, data)
	_, _ = br.Seek(0, io.SeekStart)
	SetChannelRequestBodyIfEnabled(c, data)
}

func truncateChannelRequestBody(jsonData []byte) string {
	var parsed any
	if err := Unmarshal(jsonData, &parsed); err != nil {
		return ""
	}
	wrapper := map[string]any{
		"_truncated":     true,
		"_original_size": len(jsonData),
		"body":           TruncateValue(parsed),
	}
	result, err := Marshal(wrapper)
	if err != nil {
		return ""
	}
	return string(result)
}

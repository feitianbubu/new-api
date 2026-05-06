package model

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/gin-gonic/gin"
)

var (
	serverInstance     string
	serverInstanceOnce sync.Once
)

func getServerInstance() string {
	serverInstanceOnce.Do(func() {
		if v := os.Getenv("PUSHGATEWAY_INSTANCE"); v != "" {
			serverInstance = v
		} else if h, _ := os.Hostname(); h != "" {
			serverInstance = h
		}
	})
	return serverInstance
}

func extractOtherMap(other string) map[string]interface{} {
	if other == "" {
		return nil
	}
	result := make(map[string]interface{})
	if err := common.UnmarshalJsonStr(other, &result); err != nil {
		return nil
	}
	return result
}

type LogMeta struct {
	ClientIP    string
	RequestId   string
	RequestPath string
	ClientId    string
}

func getRequestPath(c *gin.Context) string {
	if c.Request == nil || c.Request.URL == nil {
		return ""
	}
	return fmt.Sprintf("%s | %s%s", common.GetIp(), c.Request.Host, c.Request.URL.Path)
}
func buildLogMetaFromContext(c *gin.Context) LogMeta {
	meta := LogMeta{}
	if c == nil {
		return meta
	}
	meta.ClientIP = c.ClientIP()
	meta.RequestId = c.GetString(common.RequestIdKey)
	if c.Request != nil && c.Request.URL != nil {
		meta.RequestPath = getRequestPath(c)
	}
	return meta
}

func buildLogMetaFromProperties(ctx context.Context, log *Log, properties *Properties) {
	meta := LogMeta{
		ClientIP:    properties.ClientIP,
		RequestId:   properties.RequestId,
		RequestPath: properties.RequestPath,
	}
	buildLogMetaFromMeta(ctx, log, meta)
}
func buildLogMetaFromMeta(ctx context.Context, log *Log, meta LogMeta) {
	log.Ip = meta.ClientIP
	other := make(map[string]interface{})
	err := common.UnmarshalJsonStr(log.Other, &other)
	if err != nil {
		logger.LogWarn(ctx, "failed to unmarshal log other: "+err.Error())
	}
	msgInfo := make(map[string]any)
	msgInfo["rid"] = meta.RequestId
	if log.RequestId == "" {
		log.RequestId = meta.RequestId
	}
	other["msg_info"] = msgInfo
	other["request_path"] = meta.RequestPath
	if meta.ClientId != "" {
		other["client_id"] = meta.ClientId
	}
	log.Other = common.MapToJsonStr(other)
}

func createLog(c *gin.Context, log *Log) error {
	meta := buildLogMetaFromContext(c)
	buildLogMetaFromMeta(c, log, meta)
	common.SetContextKey(c, constant.ContextKeyEnableOssUpload, true)
	return createLogWithMetrics(c, log)
}

func createLogWithMetrics(ctx context.Context, log *Log) error {
	if log != nil {
		common.ObserveLogMetricsFromContext(log.Type, extractOtherMap(log.Other))
		if log.Type == LogTypeConsume {
			common.ObserveConsumeMetrics(common.AppConsumeMetricsParams{
				ModelName:        log.ModelName,
				ChannelId:        log.ChannelId,
				PromptTokens:     log.PromptTokens,
				CompletionTokens: log.CompletionTokens,
				Quota:            log.Quota,
				UseTimeSeconds:   log.UseTime,
				IsStream:         log.IsStream,
			})
		}
	}
	if log.Instance == "" {
		log.Instance = getServerInstance()
	}
	return LOG_DB.Create(log).Error
}

func AppendPropertiesMeta(c *gin.Context, properties *Properties) {
	properties.RequestId = c.GetString(common.RequestIdKey)
	properties.ClientIP = c.ClientIP()
	properties.RequestPath = getRequestPath(c)
	properties.TokenId = common.GetContextKeyInt(c, constant.ContextKeyTokenId)
	properties.TokenName = c.GetString("token_name")
	common.SetContextKey(c, constant.ContextKeyEnableOssUpload, true)
}

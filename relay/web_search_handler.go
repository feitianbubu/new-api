package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func WebSearchHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	wsReq, ok := info.Request.(*dto.WebSearchRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected dto.WebSearchRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(wsReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy web search request: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	if err := helper.ModelMappedHelper(c, info, request); err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	wsAdaptor, ok := adaptor.(channel.WebSearchAdaptor)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("channel %s does not support web_search", adaptor.GetChannelName()), types.ErrorCodeInvalidApiType, http.StatusNotImplemented, types.ErrOptionWithSkipRetry())
	}

	// Web Search 的统一 DTO 字段名（query/model/...）与各厂商上游字段（search_query/search_engine/...）
	// 不一致，PassThrough 透传必然导致上游 400，所以这里始终走转换路径。
	convertedRequest, err := wsAdaptor.ConvertWebSearchRequest(c, info, *request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)
	jsonData, err := common.Marshal(convertedRequest)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	if len(info.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
		if err != nil {
			return newAPIErrorFromParamOverride(err)
		}
	}
	logger.LogDebug(c, "Web search request body: %s", jsonData)
	var requestBody io.Reader = bytes.NewBuffer(jsonData)

	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")
	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		if httpResp.StatusCode != http.StatusOK {
			newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
			service.ResetStatusCode(newAPIError, statusCodeMappingStr)
			return newAPIError
		}
	}

	searchResp, newAPIError := wsAdaptor.DoWebSearchResponse(c, httpResp, info)
	if newAPIError != nil {
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	// 先把响应交给客户端，再做计费/日志；避免计费链路阻塞导致客户端拿不到结果。
	c.JSON(http.StatusOK, searchResp)

	// Web Search 走 per-call 计价（PriceData.UsePrice=true + ModelPrice>0）。
	// adaptor 必须在 searchResp.Usage 里至少填一个非零 token（哨兵值），
	// 否则 service/text_quota.go 会因 TotalTokens==0 把 Quota 清零。
	service.PostTextConsumeQuota(c, info, &searchResp.Usage, nil)
	return nil
}

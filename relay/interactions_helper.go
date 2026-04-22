package relay

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func InteractionsHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)
	if c.Request.Method == http.MethodPost {
		common.SetContextKey(c, constant.ContextKeyEnableOssUpload, true)
	}
	if info.ApiType != constant.APITypeGemini {
		return types.NewErrorWithStatusCode(
			errors.New("interactions is only supported for gemini api type"),
			types.ErrorCodeInvalidApiType,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(errors.New("invalid api type"), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	if c.Request.Method == http.MethodGet {
		interactionID := extractInteractionIDFromRequestPath(c.Request.URL.Path)
		if state, found, err := model.GetInteractionState(interactionID); err != nil {
			common.SysError("failed to load interaction state: " + err.Error())
		} else if found {
			if len(state.ResponseBody) > 0 {
				c.Data(http.StatusOK, "application/json", state.ResponseBody)
				return nil
			}
			StartInteractionPolling(state)
		}
	}

	var requestBody io.Reader
	if c.Request.Method == http.MethodPost {
		req, ok := info.Request.(*dto.InteractionsRequest)
		if !ok {
			return types.NewErrorWithStatusCode(
				errors.New("invalid interactions request"),
				types.ErrorCodeInvalidRequest,
				http.StatusBadRequest,
				types.ErrOptionWithSkipRetry(),
			)
		}

		if err := helper.ModelMappedHelper(c, info, req); err != nil {
			return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
		}

		requestBodyBytes, err := common.GetRequestBodyBytes(c)
		if err != nil {
			return types.NewError(err, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
		}

		var payload map[string]interface{}
		if err = common.Unmarshal(requestBodyBytes, &payload); err != nil {
			return types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
		}

		if bg, ok := payload["background"].(bool); ok && bg {
			if _, exists := payload["store"]; !exists {
				payload["store"] = true
			}
		}

		if strings.TrimSpace(req.Agent) != "" {
			payload["agent"] = req.Agent
		} else if strings.TrimSpace(req.Model) != "" {
			payload["model"] = req.Model
		}

		requestBodyBytes, err = common.Marshal(payload)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		if len(info.ParamOverride) > 0 {
			requestBodyBytes, err = relaycommon.ApplyParamOverrideWithRelayInfo(requestBodyBytes, info)
			if err != nil {
				return newAPIErrorFromParamOverride(err)
			}
		}
		requestBody = bytes.NewReader(requestBodyBytes)
	}

	respAny, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	httpResp := respAny.(*http.Response)
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		statusCodeMappingStr := c.GetString("status_code_mapping")
		newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return types.NewError(err, types.ErrorCodeReadResponseBodyFailed, types.ErrOptionWithSkipRetry())
	}

	if c.Request.Method == http.MethodPost {
		var interaction struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		if err = common.Unmarshal(respBody, &interaction); err == nil && strings.TrimSpace(interaction.ID) != "" {
			state := model.InteractionState{
				InteractionID: interaction.ID,
				ChannelID:     c.GetInt("channel_id"),
				Model:         info.OriginModelName,
				UserID:        c.GetInt("id"),
				Username:      c.GetString("username"),
				TokenID:       info.TokenId,
				TokenName:     c.GetString("token_name"),
				TokenKey:      info.TokenKey,
				UsingGroup:    info.UsingGroup,
				UserGroup:     info.UserGroup,
				RequestID:     info.RequestId,
				RequestPath:   c.Request.URL.Path,
				CreatedAt:     common.GetTimestamp(),
				UpdatedAt:     common.GetTimestamp(),
				Status:        strings.TrimSpace(interaction.Status),
				ResponseBody:  respBody,
				PriceData:     info.PriceData,
			}
			_ = model.SaveInteractionState(interaction.ID, state, 0)
			StartInteractionPolling(state)
		}
	}

	contentType := httpResp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(httpResp.StatusCode, contentType, respBody)
	return nil
}

package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const interactionsPollInterval = 10 * time.Second

var interactionPollers sync.Map

func StartInteractionPolling(state model.InteractionState) {
	interactionID := strings.TrimSpace(state.InteractionID)
	if interactionID == "" {
		return
	}
	if _, loaded := interactionPollers.LoadOrStore(interactionID, struct{}{}); loaded {
		return
	}
	go func() {
		defer interactionPollers.Delete(interactionID)
		pollInteractionUntilTerminal(interactionID)
	}()
}

func pollInteractionUntilTerminal(interactionID string) {
	for {
		state, found, err := model.GetInteractionState(interactionID)
		if err != nil {
			common.SysError("failed to load interaction state: " + err.Error())
			return
		}
		if !found {
			return
		}
		if state.CreatedAt > 0 && time.Now().Unix()-state.CreatedAt >= int64((2*time.Hour).Seconds()) {
			recordInteractionTimeout(state)
			return
		}
		if isTerminalInteractionStatus(strings.ToLower(strings.TrimSpace(state.Status))) && len(state.ResponseBody) > 0 {
			settleInteractionState(state)
			return
		}

		updatedState, err := fetchInteractionState(state)
		if err != nil {
			state.LastError = err.Error()
			state.UpdatedAt = common.GetTimestamp()
			_ = model.SaveInteractionState(interactionID, state, 0)
			time.Sleep(interactionsPollInterval)
			continue
		}
		if err = model.SaveInteractionState(interactionID, updatedState, 0); err != nil {
			common.SysError("failed to save interaction state: " + err.Error())
			return
		}
		if isTerminalInteractionStatus(strings.ToLower(strings.TrimSpace(updatedState.Status))) {
			settleInteractionState(updatedState)
			return
		}
		time.Sleep(interactionsPollInterval)
	}
}

func fetchInteractionState(state model.InteractionState) (model.InteractionState, error) {
	channelModel, err := model.GetChannelById(state.ChannelID, true)
	if err != nil {
		return state, err
	}
	baseURL := channelModel.GetBaseURL()
	if baseURL == "" {
		baseURL = constant.ChannelBaseURLs[channelModel.Type]
	}
	requestURL := relaycommon.GetFullRequestURL(baseURL, "/v1beta/interactions/"+state.InteractionID, channelModel.Type)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return state, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-goog-api-key", channelModel.Key)
	client, err := service.GetHttpClientWithProxy(channelModel.GetSetting().Proxy)
	if err != nil {
		return state, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return state, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return state, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return state, err
	}
	status := extractInteractionStatus(responseBody)
	state.Status = status
	state.ResponseBody = bytes.Clone(responseBody)
	state.UpdatedAt = common.GetTimestamp()
	if isTerminalInteractionStatus(strings.ToLower(strings.TrimSpace(status))) && state.FinishedAt == 0 {
		state.FinishedAt = state.UpdatedAt
	}
	return state, nil
}

func settleInteractionState(state model.InteractionState) {
	if state.BilledAt > 0 {
		uploadInteractionFinalResponseToOSS(&state)
		if state.OSSUploadedAt > 0 {
			state.UpdatedAt = common.GetTimestamp()
			_ = model.SaveInteractionState(state.InteractionID, state, 0)
		}
		return
	}
	_, usage, shouldSettle := shouldSettleInteractionsBilling(http.MethodGet, http.StatusOK, state.ResponseBody, 0)
	if !shouldSettle || usage == nil {
		uploadInteractionFinalResponseToOSS(&state)
		if state.OSSUploadedAt > 0 {
			state.UpdatedAt = common.GetTimestamp()
			_ = model.SaveInteractionState(state.InteractionID, state, 0)
		}
		return
	}

	ctx, info := newInteractionLogContext(state)

	service.PostTextConsumeQuota(ctx, info, usage, nil)
	uploadInteractionFinalResponseToOSS(&state)

	state.BilledAt = common.GetTimestamp()
	state.UpdatedAt = state.BilledAt
	_ = model.SaveInteractionState(state.InteractionID, state, 0)
}

func recordInteractionTimeout(state model.InteractionState) {
	if state.FinishedAt > 0 {
		return
	}
	ctx, info := newInteractionLogContext(state)
	useTimeSeconds := 0
	if state.CreatedAt > 0 {
		useTimeSeconds = int(time.Now().Unix() - state.CreatedAt)
	}
	other := service.GenerateTextOtherInfo(ctx, info,
		state.PriceData.ModelRatio,
		state.PriceData.GroupRatioInfo.GroupRatio,
		state.PriceData.CompletionRatio,
		0,
		state.PriceData.CacheRatio,
		state.PriceData.ModelPrice,
		state.PriceData.GroupRatioInfo.GroupSpecialRatio,
	)
	other["interaction_id"] = state.InteractionID
	other["timeout"] = true
	if state.LastError != "" {
		other["last_error"] = state.LastError
	}
	model.RecordErrorLog(
		ctx,
		state.UserID,
		state.ChannelID,
		state.Model,
		state.TokenName,
		"interaction polling timeout",
		state.TokenID,
		useTimeSeconds,
		false,
		state.UsingGroup,
		other,
	)
	state.Status = "expired"
	state.FinishedAt = common.GetTimestamp()
	state.UpdatedAt = state.FinishedAt
	state.LastError = "interaction polling timeout"
	_ = model.SaveInteractionState(state.InteractionID, state, 0)
}

func newInteractionLogContext(state model.InteractionState) (*gin.Context, *relaycommon.RelayInfo) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	reqPath := state.RequestPath
	if reqPath == "" {
		reqPath = "/v1beta/interactions/" + state.InteractionID
	}
	req, _ := http.NewRequest(http.MethodGet, reqPath, nil)
	req.Host = "localhost"
	ctx.Request = req
	ctx.Set("username", state.Username)
	ctx.Set("token_name", state.TokenName)
	ctx.Set("use_channel", []string{fmt.Sprintf("%d", state.ChannelID)})
	ctx.Set(common.RequestIdKey, state.RequestID)

	startTime := time.Unix(state.CreatedAt, 0)
	if state.CreatedAt == 0 {
		startTime = time.Now()
	}
	info := &relaycommon.RelayInfo{
		RequestId:         state.RequestID,
		UserId:            state.UserID,
		UsingGroup:        state.UsingGroup,
		UserGroup:         state.UserGroup,
		TokenId:           state.TokenID,
		TokenKey:          state.TokenKey,
		StartTime:         startTime,
		FirstResponseTime: startTime,
		OriginModelName:   state.Model,
		RequestURLPath:    reqPath,
		RelayFormat:       types.RelayFormatInteractions,
		PriceData:         state.PriceData,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:   state.ChannelID,
			ChannelType: constant.ChannelTypeGemini,
			ApiType:     constant.APITypeGemini,
		},
	}
	info.InitRequestConversionChain()
	return ctx, info
}

func extractInteractionStatus(responseBody []byte) string {
	var payload struct {
		Status string `json:"status"`
	}
	if err := common.Unmarshal(responseBody, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Status)
}

func uploadInteractionFinalResponseToOSS(state *model.InteractionState) {
	if state == nil || !common.OSSLogEnabled {
		return
	}
	if state.OSSUploadedAt > 0 || strings.TrimSpace(state.RequestID) == "" || len(state.ResponseBody) == 0 {
		return
	}

	model.EnqueueOSSUploadByRequestID(
		state.RequestID,
		state.RequestPath,
		state.ResponseBody,
		"application/json",
		"final_output",
	)
	state.OSSUploadedAt = common.GetTimestamp()
}

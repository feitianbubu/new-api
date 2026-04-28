package model

import (
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/QuantumNous/new-api/types"
	"github.com/samber/hot"
)

const (
	interactionStateCacheNamespace = "new-api:interaction-state:v1"
	interactionStateTTL            = 2 * time.Hour
)

type InteractionState struct {
	InteractionID string          `json:"interaction_id"`
	ChannelID     int             `json:"channel_id"`
	Model         string          `json:"model"`
	UserID        int             `json:"user_id"`
	Username      string          `json:"username"`
	TokenID       int             `json:"token_id"`
	TokenName     string          `json:"token_name"`
	TokenKey      string          `json:"token_key"`
	UsingGroup    string          `json:"using_group"`
	UserGroup     string          `json:"user_group"`
	RequestID     string          `json:"request_id"`
	RequestPath   string          `json:"request_path"`
	CreatedAt     int64           `json:"created_at"`
	UpdatedAt     int64           `json:"updated_at"`
	FinishedAt    int64           `json:"finished_at"`
	BilledAt      int64           `json:"billed_at"`
	OSSUploadedAt int64           `json:"oss_uploaded_at"`
	Status        string          `json:"status"`
	ResponseBody  []byte          `json:"response_body"`
	LastError     string          `json:"last_error"`
	PriceData     types.PriceData `json:"price_data"`
}

var (
	interactionStateCacheOnce sync.Once
	interactionStateCache     *cachex.HybridCache[InteractionState]
)

func getInteractionStateCache() *cachex.HybridCache[InteractionState] {
	interactionStateCacheOnce.Do(func() {
		interactionStateCache = cachex.NewHybridCache[InteractionState](cachex.HybridCacheConfig[InteractionState]{
			Namespace: cachex.Namespace(interactionStateCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[InteractionState]{},
			Memory: func() *hot.HotCache[string, InteractionState] {
				return hot.NewHotCache[string, InteractionState](hot.LRU, 20000).
					WithTTL(interactionStateTTL).
					WithJanitor().
					Build()
			},
		})
	})
	return interactionStateCache
}

func SaveInteractionState(interactionID string, state InteractionState, ttl time.Duration) error {
	interactionID = strings.TrimSpace(interactionID)
	if interactionID == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = interactionStateTTL
	}
	state.InteractionID = interactionID
	if state.CreatedAt == 0 {
		state.CreatedAt = common.GetTimestamp()
	}
	if state.UpdatedAt == 0 {
		state.UpdatedAt = common.GetTimestamp()
	}
	return getInteractionStateCache().SetWithTTL(interactionID, state, ttl)
}

func GetInteractionState(interactionID string) (InteractionState, bool, error) {
	interactionID = strings.TrimSpace(interactionID)
	if interactionID == "" {
		return InteractionState{}, false, nil
	}
	return getInteractionStateCache().Get(interactionID)
}

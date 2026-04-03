package middleware

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
)

const (
	tpmWindowSeconds int64 = 60
	tpmRateLimitMark       = "TPM"
)

var tpmRateLimitScript = `
local key = KEYS[1]
local requested = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local expire = tonumber(ARGV[3])

local current = tonumber(redis.call('GET', key) or '0')
if current + requested > limit then
	return {0, current, limit}
end

local nextValue = redis.call('INCRBY', key, requested)
if nextValue == requested then
	redis.call('EXPIRE', key, expire)
end

return {1, nextValue, limit}
`

type tpmMemoryWindow struct {
	WindowStart int64
	Current     int64
}

type tpmMemoryLimiter struct {
	mu    sync.Mutex
	store map[int]tpmMemoryWindow
}

var localTPMLimiter = &tpmMemoryLimiter{
	store: make(map[int]tpmMemoryWindow),
}

func (l *tpmMemoryLimiter) allow(userID int, windowStart int64, requested int64, limit int64) (bool, int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	v, ok := l.store[userID]
	if !ok || v.WindowStart != windowStart {
		v = tpmMemoryWindow{WindowStart: windowStart, Current: 0}
	}

	if v.Current+requested > limit {
		l.store[userID] = v
		return false, v.Current
	}
	v.Current += requested
	l.store[userID] = v
	return true, v.Current
}

func ModelTPMRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := resolveTPMLimit(c)
		if limit <= 0 {
			c.Next()
			return
		}
		if c.Request.Method != http.MethodPost &&
			c.Request.Method != http.MethodPut &&
			c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}

		userID := c.GetInt("id")
		if userID == 0 {
			c.Next()
			return
		}

		storage, err := common.GetBodyStorage(c)
		if err != nil {
			if common.IsRequestBodyTooLargeError(err) {
				abortWithOpenAiMessage(c, http.StatusRequestEntityTooLarge, err.Error())
				return
			}
			abortWithOpenAiMessage(c, http.StatusBadRequest, err.Error())
			return
		}
		if _, err = storage.Seek(0, io.SeekStart); err != nil {
			abortWithOpenAiMessage(c, http.StatusBadRequest, err.Error())
			return
		}
		c.Request.Body = io.NopCloser(storage)

		requested := storage.Size()
		if requested <= 0 {
			requested = 1
		}
		now := time.Now().Unix()
		windowStart := now - now%tpmWindowSeconds

		var (
			allowed bool
			current int64
		)
		if common.RedisEnabled {
			allowed, current, err = allowTPMWithRedis(context.Background(), userID, windowStart, requested, limit)
			if err != nil {
				abortWithOpenAiMessage(c, http.StatusInternalServerError, "rate_limit_check_failed")
				return
			}
		} else {
			allowed, current = localTPMLimiter.allow(userID, windowStart, requested, limit)
		}

		if !allowed {
			abortWithOpenAiMessage(c, http.StatusTooManyRequests,
				fmt.Sprintf("Rate limit reached for tokens per minute (TPM). Limit: %d TPM, current usage: %d TPM. Please reduce request rate and try again.", limit, current+requested),
				"rate_limit_exceeded")
			return
		}

		c.Next()
	}
}

func allowTPMWithRedis(ctx context.Context, userID int, windowStart int64, requested int64, limit int64) (bool, int64, error) {
	key := fmt.Sprintf("rateLimit:%s:%d:%d", tpmRateLimitMark, userID, windowStart)
	expireSeconds := tpmWindowSeconds + 2

	raw, err := common.RDB.Eval(ctx, tpmRateLimitScript, []string{key}, requested, limit, expireSeconds).Result()
	if err != nil {
		return false, 0, err
	}
	items, ok := raw.([]interface{})
	if !ok || len(items) < 2 {
		return false, 0, fmt.Errorf("invalid tpm script result: %v", raw)
	}
	allowedFlag, ok := items[0].(int64)
	if !ok {
		return false, 0, fmt.Errorf("invalid allowed flag type: %T", items[0])
	}
	current, ok := items[1].(int64)
	if !ok {
		return false, 0, fmt.Errorf("invalid current value type: %T", items[1])
	}
	return allowedFlag == 1, current, nil
}

func resolveTPMLimit(c *gin.Context) int64 {
	userTPM := common.GetContextKeyInt(c, constant.ContextKeyUserTpm)
	if userTPM > 0 {
		return int64(userTPM)
	}

	globalTPM := common.GetEnvOrDefault("MODEL_TPM_LIMIT", 0)
	if globalTPM > 0 {
		return int64(globalTPM)
	}
	return 0
}

package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	defaultAppMetricsJob = "newapi_app"
	envAppPushEnabled    = "PUSHGATEWAY_APP_ENABLED"
	envAppPushJob        = "PUSHGATEWAY_APP_JOB"
)

var appMetricsPushOnce sync.Once

func StartAppMetricsPushTask() {
	appMetricsPushOnce.Do(func() {
		cfg, ok := loadAppMetricsPushConfig()
		if !ok {
			return
		}

		gopool.Go(func() {
			ctx := context.Background()
			logger.LogInfo(ctx, fmt.Sprintf("app-metrics push task started: interval=%s pushgateway=%s job=%s instance=%s",
				cfg.Interval, cfg.PushgatewayURL, cfg.JobName, cfg.Instance))

			ticker := time.NewTicker(cfg.Interval)
			defer ticker.Stop()

			runAppMetricsPushOnce(ctx, cfg)
			for range ticker.C {
				runAppMetricsPushOnce(ctx, cfg)
			}
		})
	})
}

type appMetricsPushConfig struct {
	PushgatewayURL string
	JobName        string
	Instance       string
	GroupingLabels map[string]string
	Interval       time.Duration
	Timeout        time.Duration
	AuthToken      string
}

func loadAppMetricsPushConfig() (appMetricsPushConfig, bool) {
	enabledRaw := strings.TrimSpace(os.Getenv(envAppPushEnabled))
	if enabledRaw == "" {
		enabledRaw = strings.TrimSpace(os.Getenv(envPushEnabled))
	}
	if enabledRaw != "true" {
		return appMetricsPushConfig{}, false
	}

	pushgatewayURL := strings.TrimSpace(os.Getenv(envPushgatewayURL))
	if pushgatewayURL == "" {
		return appMetricsPushConfig{}, false
	}

	jobName := strings.TrimSpace(os.Getenv(envAppPushJob))
	if jobName == "" {
		jobName = defaultAppMetricsJob
	}

	instance := resolvePushgatewayInstance()
	interval := parsePushInterval()
	groupingLabels := parseGroupingLabels(os.Getenv(envPushgatewayGroupingLabels))
	authToken := strings.TrimSpace(os.Getenv(envPushgatewayAuthToken))

	return appMetricsPushConfig{
		PushgatewayURL: pushgatewayURL,
		JobName:        jobName,
		Instance:       instance,
		GroupingLabels: groupingLabels,
		Interval:       interval,
		Timeout:        defaultPushRequestTimeout,
		AuthToken:      authToken,
	}, true
}

func runAppMetricsPushOnce(ctx context.Context, cfg appMetricsPushConfig) {
	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	payload, err := common.GatherAppMetrics()
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("app-metrics push: gather failed: %v", err))
		return
	}
	if len(payload) == 0 {
		return
	}

	pushURL, err := buildPushgatewayURLFromParts(cfg.PushgatewayURL, cfg.JobName, cfg.Instance, cfg.GroupingLabels)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("app-metrics push: build push url failed: %v", err))
		return
	}

	if err := pushMetrics(reqCtx, pushURL, payload, cfg.AuthToken); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("app-metrics push: push failed: %v", err))
	}
}

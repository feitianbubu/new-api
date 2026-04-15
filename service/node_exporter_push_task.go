package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/logger"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	defaultNodeExporterURL = "http://host.docker.internal:9100/metrics"
	defaultPushgatewayJob  = "node_exporter"
	envNodeExporterURL     = "NODE_EXPORTER_URL"
)

var nodeExporterPushOnce sync.Once

func StartNodeExporterPushTask() {
	nodeExporterPushOnce.Do(func() {
		cfg, ok := loadNodeExporterPushConfig()
		if !ok {
			return
		}

		gopool.Go(func() {
			ctx := context.Background()
			logger.LogInfo(ctx, fmt.Sprintf("node-exporter push task started: interval=%s pushgateway=%s job=%s instance=%s",
				cfg.Interval, cfg.PushgatewayURL, cfg.JobName, cfg.Instance))

			ticker := time.NewTicker(cfg.Interval)
			defer ticker.Stop()

			runNodeExporterPushOnce(ctx, cfg)
			for range ticker.C {
				runNodeExporterPushOnce(ctx, cfg)
			}
		})
	})
}

func StartPushgatewayTasks() {
	StartNodeExporterPushTask()
	StartAppMetricsPushTask()
}

type nodeExporterPushConfig struct {
	NodeExporterURL string
	PushgatewayURL  string
	JobName         string
	Instance        string
	GroupingLabels  map[string]string
	Interval        time.Duration
	Timeout         time.Duration
	AuthToken       string
}

func loadNodeExporterPushConfig() (nodeExporterPushConfig, bool) {
	enabledRaw := strings.TrimSpace(os.Getenv(envPushEnabled))
	if enabledRaw != "true" {
		return nodeExporterPushConfig{}, false
	}

	pushgatewayURL := strings.TrimSpace(os.Getenv(envPushgatewayURL))
	if pushgatewayURL == "" {
		return nodeExporterPushConfig{}, false
	}

	nodeExporterURL := strings.TrimSpace(os.Getenv(envNodeExporterURL))
	if nodeExporterURL == "" {
		nodeExporterURL = defaultNodeExporterURL
	}

	jobName := strings.TrimSpace(os.Getenv(envPushgatewayJob))
	if jobName == "" {
		jobName = defaultPushgatewayJob
	}

	instance := strings.TrimSpace(os.Getenv(envPushgatewayInstance))
	if instance == "" {
		instance = resolvePushgatewayInstance()
	}

	interval := parsePushInterval()

	groupingLabels := parseGroupingLabels(os.Getenv(envPushgatewayGroupingLabels))
	authToken := strings.TrimSpace(os.Getenv(envPushgatewayAuthToken))

	return nodeExporterPushConfig{
		NodeExporterURL: nodeExporterURL,
		PushgatewayURL:  pushgatewayURL,
		JobName:         jobName,
		Instance:        instance,
		GroupingLabels:  groupingLabels,
		Interval:        interval,
		Timeout:         defaultPushRequestTimeout,
		AuthToken:       authToken,
	}, true
}

func runNodeExporterPushOnce(ctx context.Context, cfg nodeExporterPushConfig) {
	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	payload, err := fetchNodeExporterMetrics(reqCtx, cfg.NodeExporterURL)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("node-exporter push: fetch failed: %v", err))
		return
	}

	pushURL, err := buildPushgatewayURL(cfg)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("node-exporter push: build push url failed: %v", err))
		return
	}

	if err := pushMetrics(reqCtx, pushURL, payload, cfg.AuthToken); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("node-exporter push: push failed: %v", err))
	}
}

func fetchNodeExporterMetrics(ctx context.Context, metricsURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metricsURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("node-exporter returned status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func buildPushgatewayURL(cfg nodeExporterPushConfig) (string, error) {
	return buildPushgatewayURLFromParts(cfg.PushgatewayURL, cfg.JobName, cfg.Instance, cfg.GroupingLabels)
}

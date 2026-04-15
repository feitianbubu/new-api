package common

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

type AppConsumeMetricsParams struct {
	ModelName        string
	ChannelId        int
	PromptTokens     int
	CompletionTokens int
	Quota            int
	UseTimeSeconds   int
	IsStream         bool
}

type appMetrics struct {
	registry        *prometheus.Registry
	consumeRequests *prometheus.CounterVec
	consumeTokens   *prometheus.CounterVec
	consumeQuota    *prometheus.CounterVec
	consumeLatency  *prometheus.HistogramVec
	logsTotal       *prometheus.CounterVec
	errorTotal      *prometheus.CounterVec
	manageTotal     *prometheus.CounterVec
	loginTotal      *prometheus.CounterVec
}

var (
	appMetricsOnce sync.Once
	appMetricsInst *appMetrics
)

func initAppMetrics() {
	appMetricsOnce.Do(func() {
		registry := prometheus.NewRegistry()
		labels := []string{"model", "channel_id", "is_stream"}

		consumeRequests := prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "newapi",
			Subsystem: "consume",
			Name:      "requests_total",
			Help:      "Total consume requests recorded.",
		}, labels)

		consumeTokens := prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "newapi",
			Subsystem: "consume",
			Name:      "tokens_total",
			Help:      "Total tokens recorded.",
		}, []string{"model", "channel_id", "type"})

		consumeQuota := prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "newapi",
			Subsystem: "consume",
			Name:      "quota_total",
			Help:      "Total quota recorded.",
		}, []string{"model", "channel_id"})

		consumeLatency := prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "newapi",
			Subsystem: "consume",
			Name:      "latency_seconds",
			Help:      "Consume request latency in seconds.",
			Buckets: []float64{
				1, 2, 3, 5, 8, 10,
				15, 20, 30, 45, 60,
				90, 120, 180, 240, 300,
			},
		}, labels)

		logsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "newapi",
			Name:      "logs_total",
			Help:      "Total logs recorded.",
		}, []string{"type"})

		errorTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "newapi",
			Name:      "error_total",
			Help:      "Total error logs recorded.",
		}, []string{"code"})

		manageTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "newapi",
			Name:      "manage_total",
			Help:      "Total manage logs recorded.",
		}, []string{"action"})

		loginTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "newapi",
			Name:      "login_total",
			Help:      "Total login logs recorded.",
		}, []string{"result"})

		registry.MustRegister(consumeRequests, consumeTokens, consumeQuota, consumeLatency, logsTotal, errorTotal, manageTotal, loginTotal)

		appMetricsInst = &appMetrics{
			registry:        registry,
			consumeRequests: consumeRequests,
			consumeTokens:   consumeTokens,
			consumeQuota:    consumeQuota,
			consumeLatency:  consumeLatency,
			logsTotal:       logsTotal,
			errorTotal:      errorTotal,
			manageTotal:     manageTotal,
			loginTotal:      loginTotal,
		}
	})
}

func ObserveConsumeMetrics(params AppConsumeMetricsParams) {
	initAppMetrics()
	if appMetricsInst == nil {
		return
	}

	model := params.ModelName
	if model == "" {
		model = "unknown"
	}
	channelID := strconv.Itoa(params.ChannelId)
	if channelID == "0" {
		channelID = "unknown"
	}
	isStream := strconv.FormatBool(params.IsStream)

	appMetricsInst.consumeRequests.WithLabelValues(model, channelID, isStream).Inc()

	if params.PromptTokens > 0 {
		appMetricsInst.consumeTokens.WithLabelValues(model, channelID, "prompt").Add(float64(params.PromptTokens))
	}
	if params.CompletionTokens > 0 {
		appMetricsInst.consumeTokens.WithLabelValues(model, channelID, "completion").Add(float64(params.CompletionTokens))
	}
	if params.Quota > 0 {
		appMetricsInst.consumeQuota.WithLabelValues(model, channelID).Add(float64(params.Quota))
	}
	if params.UseTimeSeconds > 0 {
		appMetricsInst.consumeLatency.WithLabelValues(model, channelID, isStream).Observe(float64(params.UseTimeSeconds))
	}
}

func GatherAppMetrics() ([]byte, error) {
	initAppMetrics()
	if appMetricsInst == nil {
		return []byte{}, nil
	}

	metricFamilies, err := appMetricsInst.registry.Gather()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	encoder := expfmt.NewEncoder(&buf, expfmt.FmtText)
	for _, mf := range metricFamilies {
		if err := encoder.Encode(mf); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func ObserveLogMetrics(logType, action, code, result string) {
	initAppMetrics()
	if appMetricsInst == nil {
		return
	}

	logType = normalizeLabelValue(logType, "unknown")
	appMetricsInst.logsTotal.WithLabelValues(logType).Inc()

	switch logType {
	case "5", "error":
		code = normalizeLabelValue(code, "unknown")
		appMetricsInst.errorTotal.WithLabelValues(code).Inc()
	case "3", "manage":
		action = normalizeLabelValue(action, "unknown")
		appMetricsInst.manageTotal.WithLabelValues(action).Inc()
	case "11", "login":
		result = normalizeLabelValue(result, "unknown")
		appMetricsInst.loginTotal.WithLabelValues(result).Inc()
	}
}

func normalizeLabelValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func ObserveLogMetricsFromContext(logType int, other map[string]interface{}) {
	logTypeLabel := strconv.Itoa(logType)
	action := ""
	code := ""
	result := ""

	switch logType {
	case 3: // LogTypeManage
		action = logActionLabel(other)
	case 5: // LogTypeError
		code = mapStringValue(other, "error_code")
	case 11: // LogTypeLogin
		result = logResultLabel(other)
	}
	ObserveLogMetrics(logTypeLabel, action, code, result)
}

func mapStringValue(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func logActionLabel(other map[string]interface{}) string {
	if value := mapStringValue(other, "action"); value != "" {
		return value
	}
	if value := mapStringValue(other, "operation"); value != "" {
		return value
	}
	return mapStringValue(other, "op")
}

func logResultLabel(other map[string]interface{}) string {
	if value := mapStringValue(other, "result"); value != "" {
		return value
	}
	if raw, ok := other["success"]; ok {
		if success, ok := raw.(bool); ok {
			if success {
				return "success"
			}
			return "fail"
		}
	}
	return ""
}

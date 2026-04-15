package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	defaultPushInterval       = 30 * time.Second
	defaultPushRequestTimeout = 10 * time.Second
	envPushgatewayURL         = "PUSHGATEWAY_URL"
	envPushgatewayJob         = "PUSHGATEWAY_JOB"
	envPushgatewayInstance    = "PUSHGATEWAY_INSTANCE"
	envPushgatewayGroupingLabels = "PUSHGATEWAY_GROUPING_LABELS"
	envPushIntervalSeconds    = "PUSHGATEWAY_PUSH_INTERVAL_SECONDS"
	envPushEnabled            = "PUSHGATEWAY_PUSH_ENABLED"
	envPushgatewayAuthToken   = "PUSHGATEWAY_AUTH_TOKEN"
)

func parsePushInterval() time.Duration {
	interval := defaultPushInterval
	if intervalRaw := strings.TrimSpace(os.Getenv(envPushIntervalSeconds)); intervalRaw != "" {
		if seconds, err := strconv.Atoi(intervalRaw); err == nil && seconds > 0 {
			interval = time.Duration(seconds) * time.Second
		} else {
			common.SysLog("invalid PUSHGATEWAY_PUSH_INTERVAL_SECONDS, fallback to default: " + intervalRaw)
		}
	}
	return interval
}

func resolvePushgatewayInstance() string {
	instance := strings.TrimSpace(os.Getenv(envPushgatewayInstance))
	if instance != "" {
		return instance
	}

	frontendBaseURL := strings.TrimSpace(os.Getenv("FRONTEND_BASE_URL"))
	if frontendBaseURL != "" {
		parsedURL, err := url.Parse(frontendBaseURL)
		if err != nil || parsedURL.Host == "" {
			parsedURL, err = url.Parse("http://" + frontendBaseURL)
		}
		if err == nil {
			if parsedURL.Host != "" {
				return parsedURL.Host
			}
			if parsedURL.Path != "" {
				return parsedURL.Path
			}
		}
	}

	host, err := os.Hostname()
	if err == nil && host != "" {
		return host
	}

	networkIps := common.GetNetworkIps()
	if len(networkIps) > 0 && networkIps[0] != "" {
		return networkIps[0]
	}

	return "unknown"
}

func parseGroupingLabels(raw string) map[string]string {
	result := make(map[string]string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return result
	}
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			continue
		}
		result[key] = value
	}
	return result
}

func buildPushgatewayURLFromParts(pushgatewayURL, jobName, instance string, groupingLabels map[string]string) (string, error) {
	base := strings.TrimRight(pushgatewayURL, "/")
	if base == "" {
		return "", fmt.Errorf("pushgateway url is empty")
	}

	builder := strings.Builder{}
	builder.WriteString(base)
	builder.WriteString("/metrics/job/")
	builder.WriteString(url.PathEscape(jobName))
	builder.WriteString("/instance/")
	builder.WriteString(url.PathEscape(instance))

	if len(groupingLabels) > 0 {
		keys := make([]string, 0, len(groupingLabels))
		for key := range groupingLabels {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			builder.WriteString("/")
			builder.WriteString(url.PathEscape(key))
			builder.WriteString("/")
			builder.WriteString(url.PathEscape(groupingLabels[key]))
		}
	}

	return builder.String(), nil
}

func pushMetrics(ctx context.Context, pushURL string, payload []byte, authToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pushURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain; version=0.0.4")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pushgateway returned status %d", resp.StatusCode)
	}
	return nil
}

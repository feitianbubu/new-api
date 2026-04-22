package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"golang.org/x/net/proxy"
)

type proxyStatus struct {
	isValid   bool
	lastCheck time.Time
}

var (
	proxyStatusLock    sync.RWMutex
	proxyStatuses      = make(map[string]*proxyStatus)
	proxyCheckInterval = 10 * time.Minute
	proxyTestTimeout   = 5 * time.Second
)

func isValidProxy(proxyURL string) bool {
	proxyStatusLock.RLock()
	status, exists := proxyStatuses[proxyURL]
	proxyStatusLock.RUnlock()

	if exists && time.Since(status.lastCheck) < proxyCheckInterval {
		if !status.isValid {
			common.SysError(fmt.Sprintf("Proxy %s recently failed, skipping", proxyURL))
		}
		return status.isValid
	}

	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		markProxyStatus(proxyURL, false)
		common.SysError(fmt.Sprintf("Proxy %s parse failed: %v", proxyURL, err))
		return false
	}

	var testClient *http.Client

	switch parsedURL.Scheme {
	case "http", "https":
		testClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(parsedURL),
			},
			Timeout: proxyTestTimeout,
		}

	case "socks5", "socks5h":
		var auth *proxy.Auth
		if parsedURL.User != nil {
			auth = &proxy.Auth{
				User: parsedURL.User.Username(),
			}
			if password, ok := parsedURL.User.Password(); ok {
				auth.Password = password
			}
		}

		dialer, err := proxy.SOCKS5("tcp", parsedURL.Host, auth, proxy.Direct)
		if err != nil {
			markProxyStatus(proxyURL, false)
			common.SysError(fmt.Sprintf("Proxy %s SOCKS5 dialer creation failed: %v", proxyURL, err))
			return false
		}

		testClient = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				},
			},
			Timeout: proxyTestTimeout,
		}

	default:
		markProxyStatus(proxyURL, false)
		common.SysError(fmt.Sprintf("Proxy %s unsupported scheme: %s", proxyURL, parsedURL.Scheme))
		return false
	}

	isValid := testProxyConnection(testClient, proxyURL)
	markProxyStatus(proxyURL, isValid)

	return isValid
}

func testProxyConnection(client *http.Client, proxyURL string) bool {
	testURL := "https://www.google.com"

	ctx, cancel := context.WithTimeout(context.Background(), proxyTestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", testURL, nil)
	if err != nil {
		common.SysError(fmt.Sprintf("Failed to create test request for proxy %s: %v", proxyURL, err))
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		common.SysError(fmt.Sprintf("Proxy %s connection test failed: %v", proxyURL, err))
		return false
	}
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	if resp != nil && (resp.StatusCode < 200 || resp.StatusCode >= 500) {
		common.SysError(fmt.Sprintf("Proxy %s returned unexpected status code: %d", proxyURL, resp.StatusCode))
		return false
	}

	return true
}

func markProxyStatus(proxyURL string, isValid bool) {
	proxyStatusLock.Lock()
	defer proxyStatusLock.Unlock()
	proxyStatuses[proxyURL] = &proxyStatus{
		isValid:   isValid,
		lastCheck: time.Now(),
	}

	if isValid {
		common.SysLog(fmt.Sprintf("Proxy %s test passed, will be used", proxyURL))
	} else {
		common.SysError(fmt.Sprintf("Proxy %s test failed, will be skipped", proxyURL))
	}
}

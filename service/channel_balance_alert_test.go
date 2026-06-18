package service

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormatChannelBalanceAlert(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 30, 0, 0, time.UTC)
	below := []channelDaysRemaining{
		{id: 18, name: "阿里官方", balance: 2987.06, avgDaily: 132, avgDays: 7, last24h: 100},
		{id: 7, name: "智谱", balance: 860, avgDaily: 120, avgDays: 7, last24h: 200},
		{id: 22, name: "月之暗面", balance: 30, avgDaily: 10, avgDays: 5, last24h: 0},
		{id: 5, name: "DeepSeek", balance: 200, avgDaily: 15, avgDays: 7, last24h: 18},
	}

	subject, content := formatChannelBalanceAlert(below, 30, now)

	require.Equal(t, "渠道余额预警：4 个通道预计剩余不足 30 天", subject)

	lines := strings.Split(content, "\n")
	require.Equal(t, "⚠️ 渠道余额预警 · 4 个通道", lines[0])

	require.Contains(t, content, "智谱（#7） 余额 ")
	require.Contains(t, content, "月之暗面（#22） 余额 ")

	require.Contains(t, content, "7日均 ")
	require.Contains(t, content, "5日均 ")
	require.Contains(t, content, "约剩 7.2 天")
	require.Contains(t, content, "约剩 4.3 天")
	require.Contains(t, content, "约剩 3.0 天")
	require.Contains(t, content, "约剩 11.1 天")
	require.Contains(t, content, "近24h ")
	require.Contains(t, content, "近24h 无消费")

	require.NotContains(t, content, "阿里官方")
	require.Contains(t, content, "……共 4 个通道低于阈值，仅展示最紧急的 3 个")

	require.Equal(t, "阈值 30 天 · 2026-06-16 10:30", lines[len(lines)-1])
}

func TestFormatDaysRemaining(t *testing.T) {
	require.Equal(t, "不足 1 天", formatDaysRemaining(0.4))
	require.Equal(t, "约剩 7.2 天", formatDaysRemaining(7.2))
	require.Equal(t, "超过 999 天", formatDaysRemaining(1614626))
	require.Equal(t, "超过 999 天", formatDaysRemaining(math.Inf(1)))
}

func TestFormatBalanceAmountNonZero(t *testing.T) {
	require.Equal(t, "$0.01", formatBalanceAmount(0.003))
	require.Equal(t, "$200.00", formatBalanceAmount(200))
}

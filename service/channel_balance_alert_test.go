package service

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormatChannelBalanceAlert(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 30, 0, 0, time.UTC)
	// 4 个低于阈值,乱序;TopN=3,最紧急的应被排序选出,阿里官方(22.6)被截断
	below := []channelDaysRemaining{
		{id: 18, name: "阿里官方", balance: 2987.06, avgDaily: 132, days: 22.6},
		{id: 7, name: "智谱", balance: 860, avgDaily: 120, days: 7.2},
		{id: 22, name: "月之暗面", balance: 30, avgDaily: 10, days: 3.0},
		{id: 5, name: "DeepSeek", balance: 200, avgDaily: 15, days: 13.0},
	}

	subject, content := formatChannelBalanceAlert(below, 30, now)

	require.Equal(t, "渠道余额预警：4 个通道预计剩余不足 30 天", subject)

	lines := strings.Split(content, "\n")
	require.Equal(t, "⚠️ 渠道余额预警 · 共 4 个通道预计不足 30 天", lines[0])

	// 约剩天数上提到标题行
	require.Contains(t, content, "月之暗面（#22） 约剩 3.0 天")
	require.Contains(t, content, "智谱（#7） 约剩 7.2 天")
	require.Contains(t, content, "DeepSeek（#5） 约剩 13.0 天")
	// 每个通道第二行:余额 · 日均(金额随站点币种设置变化,只校验结构)
	require.Contains(t, content, "余额 ")
	require.Contains(t, content, " · 日均 ")
	// 卡片间空行分隔
	require.Contains(t, content, "\n\n")

	// 超过 TopN 的被截断,不出现且有提示
	require.NotContains(t, content, "阿里官方")
	require.Contains(t, content, "……共 4 个通道低于阈值，仅展示最紧急的 3 个")

	// 脚注:阈值 + 时间戳
	require.Contains(t, content, "阈值 30 天 · 2026-06-16 10:30")
}

package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/common/registry"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/bytedance/gopkg/util/gopool"
)

const (
	channelBalanceDaysAlertNotifyType = "channel_balance_days_alert"
	channelBalanceDaysAlertInterval   = 24 * time.Hour
	channelBalanceDaysAlertTopN       = 3
)

func init() {
	registry.RegisterInit(startChannelBalanceDaysAlertTask)
}

// startChannelBalanceDaysAlertTask sends root a daily digest of enabled
// channels whose estimated days remaining fall below
// CHANNEL_BALANCE_DAYS_ALERT_THRESHOLD (days); unset or invalid disables the task.
func startChannelBalanceDaysAlertTask() {
	threshold := common.GetEnvOrDefault("CHANNEL_BALANCE_DAYS_ALERT_THRESHOLD", 0)
	if threshold <= 0 {
		return
	}
	if !common.IsMasterNode {
		return
	}
	gopool.Go(func() {
		common.SysLog(fmt.Sprintf("channel balance days alert task started: threshold=%d days", threshold))
		ticker := time.NewTicker(channelBalanceDaysAlertInterval)
		defer ticker.Stop()
		checkChannelBalanceDaysOnce(threshold)
		for range ticker.C {
			checkChannelBalanceDaysOnce(threshold)
		}
	})
}

// formatBalanceAmount renders a USD amount in the site's quota display
// currency, matching the frontend renderQuotaWithAmount.
func formatBalanceAmount(usd float64) string {
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		return fmt.Sprintf("%.0f", usd*common.QuotaPerUnit)
	}
	rate := operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	return fmt.Sprintf("%s%.2f", operation_setting.GetCurrencySymbol(), usd*rate)
}

type channelDaysRemaining struct {
	id       int
	name     string
	balance  float64
	avgDaily float64
	days     float64
}

// formatChannelBalanceAlert 渲染多行卡片通知;正文以 \n 换行,交由发送层处理换行。
func formatChannelBalanceAlert(below []channelDaysRemaining, threshold int, now time.Time) (string, string) {
	sort.Slice(below, func(i, j int) bool { return below[i].days < below[j].days })
	shown := below[:min(len(below), channelBalanceDaysAlertTopN)]

	lines := []string{fmt.Sprintf("⚠️ 渠道余额预警 · 共 %d 个通道预计不足 %d 天", len(below), threshold)}
	for _, c := range shown {
		lines = append(lines,
			"",
			fmt.Sprintf("%s（#%d） 约剩 %.1f 天", c.name, c.id, c.days),
			fmt.Sprintf("余额 %s · 日均 %s", formatBalanceAmount(c.balance), formatBalanceAmount(c.avgDaily)),
		)
	}
	if len(below) > len(shown) {
		lines = append(lines, "", fmt.Sprintf("……共 %d 个通道低于阈值，仅展示最紧急的 %d 个", len(below), len(shown)))
	}
	lines = append(lines, "", fmt.Sprintf("阈值 %d 天 · %s", threshold, now.Format("2006-01-02 15:04")))

	subject := fmt.Sprintf("渠道余额预警：%d 个通道预计剩余不足 %d 天", len(below), threshold)
	return subject, strings.Join(lines, "\n")
}

func checkChannelBalanceDaysOnce(threshold int) {
	var channels []*model.Channel
	err := model.DB.Select("id", "name", "balance", "balance_snapshot", "used_quota").
		Where("status = ?", common.ChannelStatusEnabled).Find(&channels).Error
	if err != nil {
		common.SysError("channel balance days alert: failed to query channels: " + err.Error())
		return
	}
	channelIds := make([]int, 0, len(channels))
	for _, channel := range channels {
		channelIds = append(channelIds, channel.Id)
	}
	since := common.GetTimestamp() - model.ChannelRecentUsageLookbackDays*86400
	usageMap, err := model.GetChannelsRecentUsage(channelIds, since, model.ChannelRecentUsageActiveDays)
	if err != nil {
		common.SysError("channel balance days alert: failed to query recent usage: " + err.Error())
		return
	}

	var below []channelDaysRemaining
	for _, channel := range channels {
		usage, ok := usageMap[channel.Id]
		if !ok || usage.Quota <= 0 {
			continue
		}
		liveBalance := channel.Balance
		if channel.BalanceSnapshot != nil {
			liveBalance -= float64(channel.UsedQuota-*channel.BalanceSnapshot) / common.QuotaPerUnit
		}
		if liveBalance <= 0 {
			continue
		}
		avgDailyUsd := float64(usage.Quota) / float64(max(usage.ActiveDays, 1)) / common.QuotaPerUnit
		days := liveBalance / avgDailyUsd
		if days < float64(threshold) {
			below = append(below, channelDaysRemaining{channel.Id, channel.Name, liveBalance, avgDailyUsd, days})
		}
	}
	if len(below) == 0 {
		return
	}

	subject, content := formatChannelBalanceAlert(below, threshold, time.Now())
	NotifyRootUser(channelBalanceDaysAlertNotifyType, subject, content)
}

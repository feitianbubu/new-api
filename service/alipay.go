package service

import (
	"context"
	"fmt"
	"net/url"
	"one-api/common"
	"one-api/model"
	"one-api/setting"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/smartwalle/alipay/v3"
)

var (
	alipayClientOnce sync.Once
	alipayClient     *alipay.Client
	alipayInitError  error
)

func GetAlipayClient() (*alipay.Client, error) {
	alipayClientOnce.Do(func() {
		if !setting.AlipayEnabled {
			alipayInitError = fmt.Errorf("alipay payment is not enabled")
			return
		}
		if setting.AlipayAppId == "" || setting.AlipayPrivateKey == "" || setting.AlipayPublicKey == "" {
			alipayInitError = fmt.Errorf("alipay configuration (AppId, PrivateKey, PublicKey) is incomplete")
			return
		}

		client, err := alipay.New(setting.AlipayAppId, setting.AlipayPrivateKey, setting.AlipayIsProduction)
		if err != nil {
			alipayInitError = fmt.Errorf("failed to create alipay client: %w", err)
			return
		}
		if err = client.LoadAliPayPublicKey(setting.AlipayPublicKey); err != nil {
			alipayInitError = fmt.Errorf("failed to load alipay public key: %w", err)
			return
		}
		alipayClient = client
	})

	if alipayInitError != nil {
		return nil, alipayInitError
	}
	if alipayClient == nil && alipayInitError == nil { // Should not happen if logic is correct
		return nil, fmt.Errorf("alipay client is nil without initialization error, check AlipayEnabled and configuration")
	}
	return alipayClient, nil
}

func CreateAlipayTrade(amount float64, tradeNo string) (string, error) {
	client, err := GetAlipayClient()
	if err != nil {
		return "", fmt.Errorf("failed to get alipay client: %w", err)
	}

	var p = alipay.TradePagePay{}
	callBackAddress := GetCallbackAddress()
	p.NotifyURL = callBackAddress + "/api/user/alipay/callback"
	p.ReturnURL = callBackAddress + "/api/user/alipay/return"
	p.Subject = "pay"
	p.OutTradeNo = tradeNo
	p.TotalAmount = fmt.Sprintf("%.2f", amount)
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	result, err := client.TradePagePay(p)
	if err != nil {
		return "", err
	}
	common.SysLog("alipay trade page pay: " + result.String())

	return result.String(), nil
}

func VerifyAlipayCallback(params map[string]string) bool {
	client, err := GetAlipayClient()
	if err != nil {
		common.SysError(fmt.Sprintf("failed to get alipay client for callback verification: %s", err.Error()))
		return false
	}

	values := make(url.Values)
	for k, v := range params {
		values.Set(k, v)
	}

	err = client.VerifySign(values)
	if err != nil {
		common.SysLog(fmt.Sprintf("alipay callback sign verification failed: %s", err.Error()))
	}
	return err == nil
}

func CheckTradeByOutTradeNo(ctx context.Context, outTradeNo string) (ok bool, err error) {
	client, err := GetAlipayClient()
	if err != nil {
		return false, fmt.Errorf("failed to get alipay client: %w", err)
	}

	var p = alipay.TradeQuery{}
	p.OutTradeNo = outTradeNo

	rsp, err := client.TradeQuery(ctx, p)
	if err != nil {
		return false, err
	}

	if rsp.IsFailure() {
		return false, fmt.Errorf("alipay trade query failed: %s", rsp.SubMsg)
	}
	return true, nil
}

// CreateAlipayOrderAndGetPayURL creates an Alipay order and retrieves the payment URL.
func CreateAlipayOrderAndGetPayURL(userId int, originalAmount int64, payMoney float64, tradeNo string) (string, error) {
	if !setting.AlipayEnabled {
		return "", fmt.Errorf("alipay payment is not enabled")
	}

	payUrl, err := CreateAlipayTrade(payMoney, tradeNo)
	if err != nil {
		return "", fmt.Errorf("failed to create Alipay trade: %w", err)
	}
	if payUrl == "" {
		return "", fmt.Errorf("alipay payment URL is empty, configuration might be incorrect or trade creation failed silently")
	}

	amountForDB := originalAmount
	if !common.DisplayInCurrencyEnabled {
		dAmount := decimal.NewFromInt(originalAmount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amountForDB = dAmount.Div(dQuotaPerUnit).IntPart()
	}

	topUp := &model.TopUp{
		UserId:     userId,
		Amount:     amountForDB, // Note: using the converted amount here
		Money:      payMoney,
		TradeNo:    tradeNo,
		CreateTime: time.Now().Unix(),
		Status:     "pending",
	}
	err = topUp.Insert()
	if err != nil {
		return "", fmt.Errorf("failed to create order record: %w", err)
	}
	return payUrl, nil
}

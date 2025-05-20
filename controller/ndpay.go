package controller

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"one-api/common"
	"one-api/model"
	"one-api/service"
	"one-api/setting"
	"resty.dev/v3"
	"slices"
	"time"
)

func CheckUnPaidOrders() {
	ctx := context.TODO()
	for {
		unPaidTopUps := model.GetTopUpByStatusRecent("pending", time.Minute*15)
		for _, topUp := range unPaidTopUps {
			common.LogInfo(ctx, fmt.Sprintf("checkUnPaidOrder: %s", topUp.TradeNo))
			err := checkUnPaidOrder(ctx, topUp)
			if err != nil {
				common.LogWarn(ctx, fmt.Sprintf("checkUnPaidOrder fail: %v", err))
			}
		}
		time.Sleep(time.Duration(common.SyncFrequency) * time.Second)
	}
}
func CheckPendingOrder(c *gin.Context) {
	tradeNo := c.Query("tradeNo")
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		Fail(c, "tradeNo not found")
		return
	}
	if topUp.Status == "success" {
		Success(c, topUp)
		return
	}
	if topUp.Status != "pending" {
		Fail(c, "topUp status not pending")
		return
	}
	err := checkUnPaidOrder(c, topUp)
	CommonResponse(c, topUp, err)
}

const NdDefaultChannel = "alipay_pc_qr"

func checkUnPaidOrder(ctx context.Context, topUp *model.TopUp) error {
	var BaseUrl = setting.NDPayBaseUrl
	var appId = setting.NDPayAppId
	var appKey = setting.NDPayAppKey
	username := ""
	orderNO := topUp.TradeNo
	channel := NdDefaultChannel //TODO get from db
	signStr := fmt.Sprintf("%s%s%s%s%s", appId, username, orderNO, channel, appKey)
	sign := md5.Sum([]byte(signStr))

	uriStr := "%s/query.ashx?appId=%s&username=%s&orderNO=%s&channel=%s&sign=%x"
	args := []any{BaseUrl, appId, username, orderNO, channel, sign}
	uri := fmt.Sprintf(uriStr, args...)
	//deps.Logger(ctx).Infof("uri: %s, signStr: %s", uri, signStr)
	resp, err := resty.New().R().Get(uri)
	if err != nil {
		return errors.WithStack(err)
	}
	body := struct {
		ErrorCode float64 `json:"errorCode"`
		Data      any     `json:"data"`
	}{}
	err = json.Unmarshal(resp.Bytes(), &body)
	if err != nil {
		return errors.WithStack(err)
	}
	code := body.ErrorCode
	if code != 0 {
		if code == -4 {
			return nil
		}
		return errors.New(fmt.Sprintf("topUp query failed %s, has ErrorCode: %v", uri, body))
	}

	common.LogInfo(ctx, fmt.Sprintf("checkUnPaidOrder url: %s body: %+v", uri, body))
	// body = {ErrorCode:0 Data:map[OrderItems:[{"Body":"web3 recharge","AutoID":6744354,"UserName":"757566","OrderMoney":0.01,"OrderNO":"1703674424735043584","CooOrderSerial":null,"AppID":1970,"TradeNO":"202309181537316385796af86-1685-1970","TradeStatus":"WAIT_BUYER_PAY","ClientIp":"192.168.246.41","PaymentTime":null,"UserID":45}] Sign:bbd8a357f5c5738db0eac20fa795647d]}

	type OrderItem struct {
		OrderNO     string `json:"OrderNO"`
		TradeStatus string `json:"TradeStatus"`
	}
	if data, ok := (body.Data).(map[string]any); !ok {
		return errors.New(fmt.Sprintf("topUp query failed %s, orderItems is not list: %v", uri, body))
	} else if orderItemsStr, ok := data["OrderItems"].(string); !ok {
		return errors.New(fmt.Sprintf("topUp query failed %s, orderItems is not string: %v", uri, body))
	} else {
		var orderItems []*OrderItem
		err = json.Unmarshal([]byte(orderItemsStr), &orderItems)
		if err != nil {
			return errors.WithStack(err)
		}
		if len(orderItems) == 0 {
			return nil
		}
		orderItem := orderItems[0]
		if orderItem.OrderNO != topUp.TradeNo {
			return fmt.Errorf("topUp query failed %s, orderItem.TradeNO not match: %v:%v", uri, orderItem, topUp)
		}
		if !slices.Contains([]string{service.TradeFinished, service.TradeSuccess}, orderItem.TradeStatus) {
			return nil
		}
	}
	return setTopUpStatusSuccess(ctx, topUp.TradeNo)
}

func setTopUpStatusSuccess(_ context.Context, tradeNo string) error {
	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		return fmt.Errorf("tradeNo %s not found", tradeNo)
	}

	if topUp.Status == "pending" {
		topUp.Status = "success"
		err := topUp.Update()
		if err != nil {
			return errors.WithStack(err)
		}

		dAmount := decimal.NewFromFloat(float64(topUp.Amount))
		if dAmount.IsZero() {
			userId := topUp.UserId
			if user, err := model.GetUserById(userId, false); err == nil {
				dAmount = getAmountDecimal(topUp.Money, user.Group)
			}
		}
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		quotaToAdd := int(dAmount.Mul(dQuotaPerUnit).IntPart())
		err = model.IncreaseUserQuota(topUp.UserId, quotaToAdd, true)
		if err != nil {
			return errors.WithStack(err)
		}
		var totalQuota int
		if u, err := model.GetUserById(topUp.UserId, false); err == nil {
			totalQuota = u.Quota
		}
		model.RecordLog(topUp.UserId, model.LogTypeTopup, fmt.Sprintf("使用NDPay充值成功，充值: %v，支付：¥%.2f, 余额: %v", common.LogQuota(quotaToAdd), topUp.Money, common.LogQuota(totalQuota)))
	}
	return nil
}

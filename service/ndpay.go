package service

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"net/url"
	"one-api/common"
	"one-api/model"
	"one-api/setting"
	"resty.dev/v3"
	"time"
)

// nd pay docs: http://ndsdn.nd.com.cn/index.php?title=%E6%94%AF%E4%BB%98SDK%E6%8E%A5%E5%85%A5%E6%96%87%E6%A1%A3#.E5.B7.A5.E5.85.B7.E6.A6.82.E8.BF.B0

const TradeFinished = "TRADE_FINISHED"
const TradeSuccess = "TRADE_SUCCESS"

//const WAIT_BUYER_PAY = "WAIT_BUYER_PAY"
//const TRADE_CLOSED = "TRADE_CLOSED"

type NdOrderReq struct {
	UserId    int
	Username  string
	PaySource int32   `json:"pay_source" example:"3"`
	Channel   string  `json:"channel" example:"alipay_pc_qr"`
	Money     float64 `json:"money" example:"0.01"`
	Amount    int64   `json:"amount" example:"1"`
	Currency  string
	Subject   string
	Body      string
	ClientIp  string
	TradeNo   string
}

// NdOrderRes {"PayParams":"https://zhifu.99.com/SDP/PaySdk/AliPayWapV2ForSdp/AppRequest.aspx?OrderSerial=2025052013202266611594b6d-1308-1970\u0026TimeExpire=5F8F54E25B93DA82FAAB0C04578704216ADACF9FFC87469F","PublicKey":"","Channel":"alipay_wapv2","OrderNo":"1747718421433103200","Amount":"7.30","ClientIp":"172.24.131.141","Subject":"pay","Body":"nd pay","TimeExpire":"2025-05-20 13:50:21","Remark":"","Sign":"b658c2823479e043b512aaa593911fe3","Component":"webpay"}
type NdOrderData struct {
	PayParams  string `json:"PayParams"`
	PublicKey  string `json:"PublicKey"`
	Channel    string `json:"Channel"`
	OrderNo    string `json:"OrderNo"`
	Amount     string `json:"Amount"`
	ClientIp   string `json:"ClientIp"`
	Subject    string `json:"Subject"`
	Body       string `json:"Body"`
	TimeExpire string `json:"TimeExpire"`
	Remark     string `json:"Remark"`
	Sign       string `json:"Sign"`
	Component  string `json:"Component"`
}

// {"data":{"PayParams":"https://zhifu.99.com/AliPayForScan/AlipayPcForScan/PayOrderForQrCode.aspx?or=20250520134140894d464bd98-1685-1970\u0026sg=734a3b94d2467a8ab6fdcca951b0dc3d","PublicKey":"","Channel":"alipay_pc_qr","OrderNo":"1747719699744614100","Amount":"7.30","ClientIp":"172.24.131.141","Subject":"pay","Body":"nd pay","TimeExpire":"2025-05-20 14:11:39","Remark":"","Sign":"5fe27c8cfc7e1d9cd3ece5a73da345c4","Component":"alipay_pc_qr"},"errorCode":"0","msg":"ok"}
type NdOrderRes struct {
	ErrorCode string      `json:"errorCode"`
	Msg       string      `json:"msg"`
	Data      NdOrderData `json:"data"`
}

func CreateNdOrder(ndPay NdOrderReq) (*NdOrderData, error) {
	var BaseUrl = setting.NDPayBaseUrl
	var appId = setting.NDPayAppId
	var appKey = setting.NDPayAppKey
	tradeNo := ndPay.TradeNo
	clientIp := ndPay.ClientIp
	notifyUrl := setting.NDPayNotifyUrl
	//ndPay := req.GetNdPay()
	paySource := ndPay.PaySource
	channel := ndPay.Channel
	//point := ndPay.Amount
	//value, err := http.GetPointToFen(ctx, point)
	//if err != nil {
	//	return nil, "", errors.WithStack(err)
	//}
	amount := ndPay.Amount
	money := ndPay.Money
	//amount = utils.PointToSatoshi(ctx, bigInt.New(amount))
	currency := ndPay.Currency
	subject := ndPay.Subject
	body := ndPay.Body
	extra := ""
	timeExpire := time.Now().Add(time.Minute * 30).Format(time.DateTime)
	username := ndPay.Username
	userId := ndPay.UserId

	//string signMy = MD5(appId.ToString() + paySource.ToString() + username.ToString() + userId.ToString() + orderNO + channel + amount.ToString("0.00")+ clientIp + currency + subject + notifyUrl + extra + timeExpireStr + remark + Key)
	signStr := fmt.Sprintf("%s%d%s%d%s%s%.2f%s%s%s%s%s%s%s%s", appId, paySource, username, userId, tradeNo, channel, money, clientIp, currency, subject, notifyUrl, extra, timeExpire, "", appKey)
	sign := md5.Sum([]byte(signStr))

	uriStr := "%s/create.ashx?appId=%s&paySource=%d&username=%s&userId=%d&orderNO=%s&channel=%s&amount=%.2f&clientIp=%s&currency=%s&subject=%s&body=%s&notifyUrl=%s&extra=%s&timeExpire=%s&sign=%x"
	args := []any{BaseUrl, appId, paySource, username, userId, tradeNo, channel, money, clientIp, currency, url.QueryEscape(subject), url.QueryEscape(body), notifyUrl, extra, url.QueryEscape(timeExpire), sign}
	uri := fmt.Sprintf(uriStr, args...)
	data, err := resty.New().R().Get(uri)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var ndRes NdOrderRes
	err = json.Unmarshal(data.Bytes(), &ndRes)
	if err != nil {
		return nil, errors.Wrapf(err, "bytes: %s", data)
	}
	if ndRes.ErrorCode != "0" {
		return nil, errors.New(fmt.Sprintf("CreateNdOrder fail: %s", ndRes.Msg))
	}
	// record db
	if !common.DisplayInCurrencyEnabled {
		dAmount := decimal.NewFromInt(amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}
	topUp := &model.TopUp{
		UserId:     ndPay.UserId,
		Amount:     amount,
		Money:      money,
		TradeNo:    tradeNo,
		CreateTime: time.Now().Unix(),
		Status:     "pending",
	}
	err = topUp.Insert()
	return &ndRes.Data, nil
}

//func Notify(ctx context.Context, req *pb.NdPayNotifyRequest) (*pb.NdPayNotifyResponse, error) {
//	orderNO := req.GetOrderNO()
//	db := deps.ChainDAO(ctx)
//	pOrder, ok, err := db.GetWtOrderByOrderID(ctx, orderNO)
//	if err != nil {
//		return nil, errors.WithStack(err)
//	}
//	if !ok {
//		return nil, errors.New("pOrder not found")
//	}
//	pOrder.SetStatus(int32(dpb.OrderStatus_PAID))
//	_, err = db.SetWtOrder(ctx, pOrder)
//	if err != nil {
//		return nil, errors.WithStack(err)
//	}
//	return &pb.NdPayNotifyResponse{}, nil
//}
//

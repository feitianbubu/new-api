package controller

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"one-api/setting/system_setting"
	"regexp"
	"resty.dev/v3"
)

//type NdUser struct {
//	OpenID            string `json:"openid"`
//	UserId            int    `json:"userId"`
//	Name              string `json:"name"`
//	PreferredUsername string `json:"preferred_username"`
//	Token             string `json:"token"`
//	ExpiresIn         int    `json:"expiresIn"`
//	AccessToken       string `json:"accessToken"`
//	Email             string `json:"email"`
//}

// doc https://wiki.doc.101.com/index.php?title=%E8%BA%AB%E4%BB%BD%E8%AE%A4%E8%AF%81%E4%B8%8E%E6%9D%83%E9%99%90%E9%A1%B9%E7%9B%AE
func getNdUserByUcKey(ucKey string) (*OidcUser, error) {
	//var req TokenRequest
	//err := c.ShouldBindJSON(&req)
	//if err != nil {
	//	return nil, fmt.Errorf("invalid request format: %s", err.Error())
	//}

	// 解析 MacToken
	// 解码base64
	macTokenBytes, err := base64.StdEncoding.DecodeString(ucKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %s", err.Error())
	}
	macToken := string(macTokenBytes)
	//if true { //todo
	//	global.GVA_LOG.Debug("Decoded macToken", zap.String("macToken", macToken))
	//	return nil, fmt.Errorf("test macToken")
	//}
	//Authorization: MAC id="7F938B205F876FC355F4F61207FE943DCD048EDCB03DDF407382190342FA67A8D0365EEB3BFD91ACFF4DCE47D53D71D1",nonce="1745388704449:36V8H83V",mac="EB6utKO6Zlix/DPIrBOYDzjKHV/Ud0TmWb52GmL8mlI=",request_uri="/",host="uc-component.beta.101.com"
	pattern := regexp.MustCompile(`MAC id="([^"]+)",nonce="([^"]+)",mac="([^"]+)",request_uri="([^"]+)",host="([^"]+)"`)
	matches := pattern.FindStringSubmatch(macToken)
	if len(matches) != 6 {
		return nil, fmt.Errorf("invalid macToken format: %s", macToken)
	}
	//uid, err := strconv.ParseInt(matches[1], 10, 64)
	//if err != nil {
	//	response.FailWithMessage("Invalid uid format: "+matches[1], c)
	//	return
	//}

	// Get configuration values
	sdpAppId := system_setting.GetOIDCSettings().ClientId
	ucSdkUri := system_setting.GetOIDCSettings().TokenEndpoint
	userInfoUri := system_setting.GetOIDCSettings().UserInfoEndpoint
	accessToken := matches[1]
	nonce := matches[2]
	mac := matches[3]
	requestUri := matches[4]
	host := matches[5]

	// 参数完整性检查
	if ucSdkUri == "" || sdpAppId == "" || macToken == "" || accessToken == "" {
		return nil, fmt.Errorf("参数不完整")
	}

	// Create Resty client
	client := resty.New()

	// 构建登录请求参数
	type RequestBody struct {
		HttpMethod string `json:"http_method"`
		Host       string `json:"host"`
		Nonce      string `json:"nonce"`
		Mac        string `json:"mac"`
		RequestUri string `json:"request_uri"`
	}
	requestBody := RequestBody{
		HttpMethod: "GET",
		Host:       host,
		Nonce:      nonce,
		Mac:        mac,
		RequestUri: requestUri,
	}

	//{\"account_type\":\"org\",\"account_id\":0,\"user_id\":10020027,\"access_token\":\"7F938B205F876FC355F4F61207FE943DCD048EDCB03DDF40CCFAF2067E728E22C8951F25130BF57D2C4C911C26A7D866\",\"refresh_token\":\"7F938B205F876FC355F4F61207FE943DD271D5EDAD952DEE3522963916E0F06DB533A00498EEE3734CA6A053E6BE2D078C24657EB93F730A7F19F94364AB14770706989695C0D3C5C06869A979803046244DD05A41E6849F7990A44464697F3D6EE87948776BE4796B3CCBF83B69B87D0EE1F7886594109918838BA070141104E865F62ECA151B18D036DCCE3A19CC3C\",\"mac_algorithm\":\"hmac-sha-256\",\"mac_key\":\"Tz8I9tv8ns\",\"expires_at\":\"2025-04-29T17:09:29.320+0800\",\"server_time\":\"2025-04-22T17:09:33.290+0800\",\"region\":\"wx\",\"source_token_account_type\":\"org\",\"first_create_time\":\"2025-04-22T17:09:29.320+08:00\",\"auth_verify_types\":[\"PASSWORD\"],\"tenant_id\":0}
	type NdLoginRes struct {
		AccountType            string   `json:"account_type"`
		AccountId              int64    `json:"account_id"`
		UserId                 int64    `json:"user_id"`
		AccessToken            string   `json:"access_token"`
		RefreshToken           string   `json:"refresh_token"`
		MacAlgorithm           string   `json:"mac_algorithm"`
		MacKey                 string   `json:"mac_key"`
		ExpiresAt              string   `json:"expires_at"`
		ServerTime             string   `json:"server_time"`
		Region                 string   `json:"region"`
		SourceTokenAccountType string   `json:"source_token_account_type"`
		FirstCreateTime        string   `json:"first_create_time"`
		AuthVerifyTypes        []string `json:"auth_verify_types"`
		TenantId               int64    `json:"tenant_id"`
	}

	// 发送登录请求
	var ndLoginRes NdLoginRes
	tokenUrl := fmt.Sprintf("%s/tokens/%s/actions/valid", ucSdkUri, accessToken)

	resp, err := client.R().
		SetHeader("sdp-app-id", sdpAppId).
		SetBody(requestBody).
		//SetResult(&ndLoginRes).
		Post(tokenUrl)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %s, resp:%s", err.Error(), resp.String())
	}
	if resp.IsError() {
		return nil, fmt.Errorf("请求失败: %s, resp:%s", resp.String(), resp.Status())
	}

	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %s", err.Error())
	}

	// 先解析为字符串 beta环境需要, 正式环境不需要
	var jsonStr string
	err = json.Unmarshal(body, &jsonStr)
	if err != nil {
		jsonStr = string(body)
	}

	// 再解析JSON
	err = json.Unmarshal([]byte(jsonStr), &ndLoginRes)
	if err != nil {
		return nil, errors.Wrapf(err, "解析响应体失败: %s", jsonStr)
	}

	userId := ndLoginRes.UserId
	if userId == 0 {
		return nil, fmt.Errorf("用户不存在")
	}
	//macKey := ndLoginRes.MacKey

	// 获取用户信息
	//https://uc-gateway.beta.101.com/v1.1/public/users/10020027
	userInfoUrl := fmt.Sprintf("%s/v1.1/public/users/%d", userInfoUri, userId)
	// body
	//{
	//	"user_id": 2086411836,
	//	"avatar_source":1,//头像源，1：内容服务（CS）
	//	"avatar_data": "source为1时，CS的dentry_id",
	//	"real_name": "是谁842",
	//	"real_name_py": "ss842",
	//	"real_name_pinyin": "shishei842",
	//	"nick_name": "是谁842",
	//	"nick_name_py": "ss842",
	//	"nick_name_pinyin": "shishei842",
	//	"org_user_code": "test1236@SU_NEW_TEST001",
	//	"gender": 1,
	//	"org_id": 489809018582,
	//	"org_name": "学校-889",
	//	"org_code": "abclzfp23",
	//	"account_id": 0,
	//	"node_items": [{
	//"node_id": 481037592599,
	//"node_name": "su_new_test001",
	//"node_path": "", //节点路径，组织节点返回空串
	//"node_type": "SCHOOL",//节点类型
	//"is_org": 1, //是否组织 0：否， 1：是， 默认为0
	//"user_seq": 100000
	//}]
	//}
	type UserInfoRes struct {
		UserId       int64  `json:"user_id"`
		AvatarSource int    `json:"avatar_source"`
		AvatarData   string `json:"avatar_data"`
		RealName     string `json:"real_name"`
		NickName     string `json:"nick_name"`
		OrgId        int64  `json:"org_id"`
		OrgName      string `json:"org_name"`
		OrgCode      string `json:"org_code"`
		AccountId    int64  `json:"account_id"`
	}
	var userInfoRes UserInfoRes
	userInfoResp, err := client.R().
		SetHeader("sdp-app-id", sdpAppId).
		SetResult(&userInfoRes).
		Get(userInfoUrl)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %s, resp:%s", err.Error(), userInfoResp.String())
	}
	if userInfoResp.StatusCode() != 200 {
		return nil, fmt.Errorf("请求失败: %s, resp:%s", userInfoResp.String(), userInfoResp.Status())
	}

	res := OidcUser{}
	//res.UserId = int(userId)
	// 优先real_name 其次nick_name
	res.Name = userInfoRes.RealName
	if res.Name == "" {
		res.Name = userInfoRes.NickName
	}
	res.PreferredUsername = fmt.Sprintf("%d", userInfoRes.UserId)
	//res.AccessToken = accessToken
	openID := fmt.Sprintf("%d", userId)
	res.OpenID = openID
	res.Email = fmt.Sprintf("%s@nd.com", openID)
	return &res, nil
}

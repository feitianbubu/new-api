文档变更说明
时间
- Asset（素材资产）上传类型：新增视频、音频类型，并补充了对应上传文件的限制说明。
- 新增最佳实践：增加了关于如何保证人物一致性的最佳实践说明。
- 废弃字段：废弃 Title 字段。
  2026.3.26
- 新增 DeleteAsset 接口
- 增加视频生成中使用Asset的方式介绍
  2026.3.28
  本文介绍素材资产（Assets）API 接口的参数。您可以使用以下 Assets API 接口创建、管理个人人像素材资产。
  本文档仅限预览及邀测用户使用：
- 不承诺正式 API 上线100%一致。
- 仅限邀测用户阅读，请勿截图/分享给其他人员。
- 上传素材 （CreateAsset） API 为异步接口，系统处理可能出现排队，导致入库时间增加。不承诺上传时间 SLA。
- 素材资产应为虚拟人像，非虚拟人像类素材无需入库。
- 您需确保上传的虚拟人像符合以下条件：
   - 您合法拥有该素材，并享有完整的使用及处分权限。素材不包含未获授权的第三方商标、标识类内容。
   - 素材不得与任何自然人肖像或形象雷同，素材不存在抄袭、盗用情形，不会侵害任何第三方的人格权、知识产权等合法权益。
   - 素材不包含违反法规、违背公序良俗、危害国家安全的内容。

素材资产库结构说明
- Asset Group（素材资产组合）：单个素材文件为一个 Asset，每个 Asset 属于一个 Asset Group。
   - 可以使用素材组自由管理素材，例如可将同一虚拟人物素材放入同一素材组合进行管理。
- Asset（素材资产）：一个素材文件（当前支持上传图像、视频、音频），是方舟 Seedance 2.0 系列模型可直接用于推理的可信资产。
  注意
- 仅需入库推理需使用的素材资产，不需使用的素材资产请勿入库。
- 仅可使用已入库素材资产的 Id (Asset ID) 进行视频生成，同一形象未入库素材无法使用。
- 每个上传的素材资产需经过预处理，可轮询调用 GetAsset 接口查询素材状态（对应参数为 Status），仅当状态变为 Active 后，该素材资产方可用于后续推理使用；若状态为 Failed 则表示处理失败，无法用于后续推理使用。详情可参考：上传素材资产并获取素材资产信息代码示例。

素材资产（Assets）API 接口功能
Asset (Group) 创建接口：
1. CreateAssetGroup：创建素材资产组合。首次创建素材资产组合时需在控制台签署授权函，详情参考 「⚠️保密信息」【申请权限填客户名称】私域虚拟人像素材资产库使用指南（邀测用户版）
2. CreateAsset：创建素材资产。该接口可用于上传个人素材资产，创建素材资产后可利用返回字段中的素材 Id （需处于 Active 状态）用于 Seedance 2.0 系列模型生成视频。

Asset (Group) 管理接口：
- ListAssetGroups：查询素材资产组合列表。
- ListAssets：查询素材资产列表。
- GetAsset：查询素材资产信息。
- GetAssetGroup：查询素材资产组合信息。
- UpdateAssetGroup：更新素材资产组合信息。
- UpdateAsset：更新素材资产信息。
- DeleteAsset：删除单个素材资产。

鉴权方式
调用素材资产（Assets）API 接口需使用 Access Key 鉴权，详情参考 获取 API 访问密钥（AK/SK）。

限流要求
- QPS 限流： API接口每秒查询请求的总数上限。超过此限制的查询请求会报错。
  接口名
  账号维度的 QPS 限流
  CreateAssetGroup
  30
  CreateAsset
  1
  ListAssetGroups
  30
  ListAssets
  30
  GetAsset
  100
  GetAssetGroup
  100
  UpdateAsset
  30
  UpdateAssetGroup
  30
  CreateAssetGroup
  POST /open/CreateAssetGroup
  创建 Asset Group（素材资产组合）组合，用作素材资产管理。
  首次创建 Asset Group（素材资产组合）需在控制台签署授权函，详情参考 「⚠️保密信息」【申请权限填客户名称】私域虚拟人像素材资产库使用指南（邀测用户版）
  请求参数
  Name string 必填
  Asset Group（素材资产组合）的名称，上限为 64 字符。

---
Description string
Asset Group（素材资产组合）的描述，上限为 300 字符。

---
GroupType  string
Asset Group（素材资产组合）的类型。可选值：
- AIGC：虚拟人像。
  当前仅支持 AIGC 类型。


---
ProjectName string
资源所属的项目名称，默认值为default。
若资源不在默认项目中，需填写正确的项目名称，获取项目名称，请查看 文档。

---

返回参数
Id string
Asset Group（素材资产组合）的 Id。

---
返回示例：
{
"Id": "group-2026**********-*****"
}

CreateAsset
POST /open/CreateAsset
向指定的Asset Group（素材资产组合）内创建Asset（素材资产）。
上传素材 （CreateAsset） API 为异步接口，系统处理可能出现排队，导致入库时间增加。不承诺上传时间 SLA。
请求参数
GroupId string 必填
Asset（素材资产）所属的 Asset Group（素材资产组合）的 Id。

---
URL string 必填
传入的Asset（素材资产）的公共可访问地址。

---
Name string
Asset（素材资产）的名称，上限为64个字符。
该字段仅用于使用 ListAssets 接口时模糊搜索素材，不会被带入模型推理。关于如何使用素材生成视频，请参考使用人像素材生成视频和常见问题 4。

---
AssetType string 必填
Asset（素材资产）的类型，支持传入图像、音频、视频。可选值：
- Image：Asset（素材资产）的类型为图像。
- Video：Asset（素材资产）的类型为视频。
- Audio：Asset（素材资产）的类型为音频。
  传入图像、音频、视频素材时，仅支持上传 URL ，不支持 base64。
  传入单个图像要求
- 格式：jpeg、png、webp、bmp、tiff、gif、heic/heif
- 宽高比（宽/高）： (0.4, 2.5)
- 宽高长度（px）：(300, 6000)
- 大小：单张图片小于 30 MB
  传入单个视频要求
- 格式：mp4、mov
- 分辨率：480p、720p
- 时长：单个视频时长 [2, 15] s
- 尺寸：
   - 宽高比（宽/高）：[0.4, 2.5]
   - 宽高长度（px）：[300, 6000]
   - 总像素数：[640×640=409600, 834×1112=927408]，即宽和高的乘积符合 [409600, 927408] 的区间要求。
- 大小：单个视频不超过 50 MB
- 帧率 (FPS)：[24, 60]
  传入单个音频要求
- 格式：wav、mp3
- 时长：单个音频时长 [2, 15] s
- 大小：单个音频不超过 15 MB

---
ProjectName string
资源所属的项目名称，默认值为default。
若资源不在默认项目中，需填写正确的项目名称，获取项目名称，请查看 文档。
需要和待传入的 Asset Group（素材资产组合）的 ProjectName 保持一致。

---

返回参数
Id string
Asset（素材资产）的 Id。

---
返回示例：
{
"Id": "Asset-2026**********-*****"
}

ListAssets
POST /open/ListAssets
查询符合筛选条件的 Assets（素材资产）列表。
请求参数
Filter object 必填
搜索的过滤条件。

---
Filter.GroupIds array
Asset（素材资产）所属的 Asset Group（素材资产组合）的 Id。

---
Filter.GroupType string 必填
Asset Group（素材资产组合）的类型。可选值：
- AIGC：虚拟人像。

---
Filter.Statuses array
任务状态。
- Active：素材资产（Asset）已处理完毕，可以使用。
- Processing：素材资产（Asset）正在预处理，无法使用。
- Failed：素材资产（Asset）处理失败。

---
Filter.Name string
Asset（素材资产）的名称，上限为64个字符。

---
PageNumber int (i64) 必填
搜索页码，可用于列表分页功能，从 1 开始。例如："page_number": 1，即返回第一页的搜索结果。

---
PageSize int (i64) 必填
每页搜索结果的数量，上限为100。

---
SortBy string
用于排序的字段名称，默认值 createTime。支持以下类型：
- CreateTime：根据创建时间排序。
- UpdateTime：根据更新时间排序。
- GroupId：根据资产素材组的 Id 排序。

---
SortOrder string
排序顺序，默认值 Desc。可选值：
- Desc：降序
- Asc：升序

---
ProjectName string
资源所属的项目名称，默认值为default。
若资源不在默认项目中，需填写正确的项目名称，获取项目名称，请查看 文档。

---

返回参数
Items array[]
符合筛选条件的Asset（素材资产）数组。

---
Items.Id string
Asset（素材资产）的 Id。

---
Items.name string
Asset（素材资产）的名称，上限为64个字符。

---
Items.URL string
Asset（素材资产）的公共可访问地址。有效期为12小时，请及时保存。

---
Items.GroupId string
Asset（素材资产）所属的 Asset Group（素材资产组合）的 Id。

---
Items.AssetType string
Asset（素材资产）的类型，支持传入图像、音频、视频。支持类型：
- Image：Asset（素材资产）的类型为图像。
- Video：Asset（素材资产）的类型为视频。
- Audio：Asset（素材资产）的类型为音频。

---
Items.Status string
任务状态。
- Active：素材资产（Asset）已处理完毕，可以使用。
- Processing：素材资产（Asset）正在预处理，无法使用。
- Failed：素材资产（Asset）处理失败。

---
Items.Error object
错误信息。

---
Items.Error.Code string
错误码。

---
Items.Error.Message string
错误信息。

---
Items.ProjectName string
资源所属的项目名称。

---
Items.CreateTime string
创建时间。

---
Items.UpdateTime string
更新时间。

---
TotalCount int (i64)
返回总数。

---
PageNumber int (i64)
返回的页数。

---
PageSize int (i64)
每页搜索结果的数量，上限为100。

ListAssetGroups
POST /open/ListAssetGroups
查询符合筛选条件的Asset Groups（素材资产组合）列表。
请求参数
Filter object 必填
搜索的过滤条件。

---
Filter.name string
Asset Group（素材资产组合）的名称，上限为64个字符。

---
Filter.GroupIds array
Asset（素材资产）所属的 Asset Group（素材资产组合）的 Id。

---
Filter.GroupType string 必填
Asset Group（素材资产组合）的类型。可选值：
- AIGC：虚拟人像。

---
PageNumber int (i64)
搜索页码，可用于列表分页功能，从 1 开始。例如："page_number": 1，即返回第一页的搜索结果。

---
PageSize int (i64)
每页搜索结果的数量，上限为100。

---
SortBy string
用于排序的字段名称，默认值 createTime。支持以下类型：
- CreateTime：根据创建时间排序。
- UpdateTime：根据更新时间排序。

---
SortOrder string
排序顺序，默认值 Desc。可选值：
- Desc：降序
- Asc：升序

---
ProjectName string
资源所属的项目名称，默认值为default。
若资源不在默认项目中，需填写正确的项目名称，获取项目名称，请查看 文档。

---

返回参数
TotalCount int (i64)
返回的 Asset Group（素材资产组合）的总数。

---
Items array[]
符合筛选条件的 Asset Group（素材资产组合）数组。

---
Items.Id string
Asset Group（素材资产组合）的 Id。

---
Items.Name string
Asset Group（素材资产组合）的名称，上限为64个字符。

---
Items.Title string
Asset Group（素材资产组合）的标题。
已废弃，请直接使用参数 Name

---
Items.Description string
Asset Group（素材资产组合）的描述，上限为 300 字符。

---
Items.GroupType string
Asset Group（素材资产组合）的类型。
- AIGC：虚拟人像。

---
Items.ProjectName string
资源所属的项目名称。

---
Items.CreateTime string
创建时间。

---
Items.UpdateTime string
更新时间。

---
PageNumber int (i64)
返回的页数。

---
PageSize int (i64)
每页搜索结果的数量，上限为100。

---

GetAssetGroup
POST /open/GetAssetGroup
获取单个Asset Group（素材资产组合）信息。
请求参数

Id string 必填
Asset Group（素材资产组合）的 Id

---
ProjectName string
需要查询的 Asset Group（素材资产组合）所属的项目名称，默认值为default。
若资源不在默认项目中，需填写正确的项目名称，获取项目名称，请查看 文档。

---

返回参数
Id string
Asset Group（素材资产组合）的 Id。

---
Name string
Asset Group（素材资产组合）的名称，上限为64个字符。

---
Title string
Asset Group（素材资产组合）的标题。
已废弃，请直接使用参数 Name

---
Description string
Asset Group（素材资产组合）的描述，上限为 300 字符。

---
GroupType  string
Asset Group（素材资产组合）的类型。
- AIGC：虚拟人像

---
ProjectName string
资源所属的项目名称。

---
CreateTime string
创建时间。

---
UpdateTime string
更新时间。

---

GetAsset
POST /open/GetAsset
获取单个Asset（素材资产）信息。
请求参数

Id string  必填
Asset（素材资产）的 Id。

---
ProjectName string
需要查询的 Asset（素材资产）所属的项目名称，默认值为default。
若资源不在默认项目中，需填写正确的项目名称，获取项目名称，请查看 文档。

---

返回参数
Id string
Asset（素材资产）的 Id。

---
Name string
Asset（素材资产）的名称，上限为64个字符。

---
URL  string
Asset（素材资产）的访问地址。有效期为12小时，请及时保存。

---
AssetType string
Asset（素材资产）的类型，支持传入图像、音频、视频。支持类型：
- Image：Asset（素材资产）的类型为图像。
- Video：Asset（素材资产）的类型为视频。
- Audio：Asset（素材资产）的类型为音频。

---
GroupId string
Asset（素材资产）所属的 Asset Group（素材资产组合）的 Id。

---
Status string
任务状态。
- Active：素材资产（Asset）已处理完毕，可以使用。
- Processing：素材资产（Asset）正在预处理，无法使用。
- Failed：素材资产（Asset）处理失败。

---
Error object
错误信息。

---
Error.Code string
错误码。

---
Error.Message string
错误信息。

---
CreateTime string
创建时间。

---
UpdateTime  string
更新时间。

---
ProjectName string
资源所属的项目名称。

---

UpdateAssetGroup
POST /open/UpdateAssetGroup
更新单个 Asset Group（素材资产组合）信息。当前仅支持更新 Asset Group（素材资产组合）的 Name 和 Description。
请求参数
Id string 必填
需要更新的 Asset Group（素材资产组合）的 Id。

---
Name string
需要更新的 Asset Group（素材资产组合）的新名称，上限为64个字符。

---
Description string
需要更新的 Asset Group（素材资产组合）的新描述，上限为 300 字符。

---
ProjectName string
需要更新的 Asset Group（素材资产组合）所属的项目名称，默认值为default。
若资源不在默认项目中，需填写正确的项目名称，获取项目名称，请查看 文档。

---

返回参数
Id string
Asset Group（素材资产组合）的 Id。

---


UpdateAsset
POST /open/UpdateAsset
更新单个Asset（素材资产）信息。当前仅支持更新Asset（素材资产）的 Name。
请求参数
Id string 必填
需要更新的 Asset（素材资产）的 Id。

---
Name string
需要更新的 Asset（素材资产）的新名称，上限为64个字符。

---
ProjectName string
需要更新的 Asset（素材资产）所属的项目名称，默认值为default。
若资源不在默认项目中，需填写正确的项目名称，获取项目名称，请查看 文档。

---

返回参数
Id string
Asset（素材资产）的 Id。

---

DeleteAsset
POST /open/DeleteAsset
删除单个 Asset（素材资产）。
请求参数
Id string 必填
需要删除的 Asset（素材资产）的 Id。

---
ProjectName string
需要删除的 Asset（素材资产）所属的项目名称，默认值为default。
若资源不在默认项目中，需填写正确的项目名称，获取项目名称，请查看 文档。

---

返回参数
本接口无返回参数。

---

代码示例：
以下示例为使用 Asset API 创建素材资产并用于视频生成的使用链路：
1. 创建素材资产组合：调用 CreateAssetGroup 接口创建一个素材资产组合（Asset Group），用于对同一项目或人物的素材进行统一管理。首次创建时需在控制台签署授权函。
2. 上传素材资产并等待预处理完成：调用 CreateAsset 接口上传图片/视频/音频素材，传入素材的公共访问URL及所属的Group ID，获得素材资产ID（Asset ID）。
   由于上传的素材资产需经过预处理后才能使用，可轮询调用 GetAsset 接口查询素材状态，直至状态变为 Active。若状态为 Failed 则表示处理失败，无法用于后续推理使用。
3. 在视频生成 API 中使用素材：当素材资产状态为 Active 后，将素材ID按 asset://<asset_ID> 的格式拼接成URL，在视频生成API（如Seedance 2.0系列模型）的请求中，将该URL作为参考图片/视频/音频的 image_url 传入，即可使用该素材资产生成视频。
   API 鉴权方式区别说明
- Asset API：Access Key 鉴权，详情参考 获取 API 访问密钥（AK/SK）。
- 视频生成 API：API Key 鉴权，详情参考  管理 API Key。
  素材库项目（Project）隔离说明
- 向指定的 Asset Group（素材资产组合）内创建或查询 Asset（素材资产）时，需保证两者的 ProjectName 一致
- Asset（素材资产）所属的 ProjectName 需与调用视频生成 API 接口时使用的 API key 所属的 ProjectName 一致
1. 创建素材资产组合
   package main

import (
"fmt"

        "github.com/bytedance/sonic"
        "github.com/volcengine/volcengine-go-sdk/volcengine"
        "github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
        "github.com/volcengine/volcengine-go-sdk/volcengine/session"
        "github.com/volcengine/volcengine-go-sdk/volcengine/universal"
)

func main() {
// 配置认证信息（请替换为您的真实 AK/SK）
ak := "<YOUR_ACCESS_KEY>" // 示例：AKLTYzM0YjI3Nj...
sk := "<YOUR_SECRET_KEY>" // 示例：WldNMk5HWXh...
region := "cn-beijing"

        config := volcengine.NewConfig().
                WithCredentials(credentials.NewStaticCredentials(ak, sk, "")).
                WithRegion(region)

        sess, err := session.NewSession(config)
        if err != nil {
                fmt.Printf("创建 session 失败: %v\n", err)
                return
        }

        // 调用 CreateAssetGroup 接口
        resp, err := universal.New(sess).DoCall(
                universal.RequestUniversal{
                        ServiceName: "ark",
                        Action:      "CreateAssetGroup",
                        Version:     "2024-01-01",
                        HttpMethod:  universal.POST,
                        ContentType: universal.ApplicationJSON,
                },
                // 请求参数（请根据实际情况填写）
                &map[string]any{
                        "Name":        "<NAME>",        // 示例：test
                        "Description": "<DESCRIPTION>", // 示例：test
                        "GroupType":   "<GROUP_TYPE>",  // 示例：AIGC
                },
        )
        if err != nil {
                fmt.Printf("调用 CreateAssetGroup 失败: %v\n", err)
                return
        }
        if resp == nil {
                fmt.Println("响应为空")
                return
        }

        // 打印返回结果
        respData, err := sonic.Marshal(resp)
        if err != nil {
                fmt.Printf("序列化响应失败: %v\n", err)
                return
        }
        fmt.Println(string(respData))
}
返回示例
{"ResponseMetadata":{"RequestId":"20260318155041036F7CB6362358FB40FC","Action":"CreateAssetGroup","Version":"2024-01-01","Service":"ark","Region":"cn-beijing"},"Result":{"Id":"group-2026**********-*****"}}

2. 上传素材资产并获取素材资产信息
   package main

import (
"errors"
"fmt"
"time"

        "github.com/bytedance/sonic"
        "github.com/volcengine/volcengine-go-sdk/volcengine"
        "github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
        "github.com/volcengine/volcengine-go-sdk/volcengine/session"
        "github.com/volcengine/volcengine-go-sdk/volcengine/universal"
)

const (
region = "cn-beijing"
serviceName = "ark"
version     = "2024-01-01"

        // 轮询配置
        pollInterval = 3 * time.Second
        pollTimeout  = 2 * time.Minute
)

func main() {
// TODO: 替换为你的 AK / SK
ak := "<YOUR_ACCESS_KEY>" // 示例：AKLTYzM0YjI3Nj...
sk := "<YOUR_SECRET_KEY>" // 示例：WldNMk5HWXh...

        // TODO: 替换为你的实际参数
        groupID := "<GROUP_ID>" // 示例：group-2026xxxxxxxxx-xxxxx
        assetURL := "<IMAGE_URL>" // 示例：https://example.com/image.jpg
        assetType := "<ASSET_TYPE>" // 可选值：Image、Video、Audio（示例：Image）
        projectName := "<PROJECT_NAME>" // 默认 default（示例：test）

        config := volcengine.NewConfig().
                WithCredentials(credentials.NewStaticCredentials(ak, sk, "")).
                WithRegion(region)

        sess, err := session.NewSession(config)
        if err != nil {
                fmt.Printf("create session failed: %v\n", err)
                return
        }

        client := universal.New(sess)

        // 1. 创建素材资产
        assetID, err := createAsset(client, groupID, assetURL, assetType, projectName)
        if err != nil {
                fmt.Printf("create asset failed: %v\n", err)
                return
        }

        fmt.Printf("asset created, AssetId = %s\n", assetID)

        // 2. 查询素材资产状态
        finalURL, err := waitForAssetActive(client, assetID, pollInterval, pollTimeout)
        if err != nil {
                fmt.Printf("poll asset failed: %v\n", err)
                return
        }

        fmt.Printf("asset is Active, URL = %s\n", finalURL)
}

// createAsset 调用 CreateAsset 并返回 AssetId
func createAsset(client *universal.Universal, groupID, url, assetType, projectName string) (string, error) {
resp, err := client.DoCall(
universal.RequestUniversal{
ServiceName: serviceName,
Action:      "CreateAsset",
Version:     version,
HttpMethod:  universal.POST,
ContentType: universal.ApplicationJSON,
},
&map[string]any{
"GroupId":     groupID,
"URL":         url,
"AssetType":   assetType,
"ProjectName": projectName,
},
)
if err != nil {
return "", err
}
if resp == nil {
return "", errors.New("create asset response is nil")
}

        // 打印原始返回，便于排查
        respData, _ := sonic.Marshal(resp)
        fmt.Printf("CreateAsset response: %s\n", string(respData))

        assetID := extractString(resp, "Result", "Id")
        if assetID == "" {
                assetID = extractString(resp, "Result", "AssetId")
        }
        if assetID == "" {
                assetID = extractString(resp, "Id")
        }
        if assetID == "" {
                assetID = extractString(resp, "AssetId")
        }

        if assetID == "" {
                return "", fmt.Errorf("cannot find AssetId in response: %s", string(respData))
        }

        return assetID, nil
}

// waitForAssetActive 查询 GetAsset，直到 Active / Failed / 超时
func waitForAssetActive(client *universal.Universal, assetID string, interval, timeout time.Duration) (string, error) {
deadline := time.Now().Add(timeout)

        for {
                if time.Now().After(deadline) {
                        return "", fmt.Errorf("polling timeout after %v, assetID=%s", timeout, assetID)
                }

                status, url, errMsg, err := getAssetStatus(client, assetID)
                if err != nil {
                        return "", err
                }

                fmt.Printf("asset status: %s\n", status)

                switch status {
                case "Processing":
                        time.Sleep(interval)
                        continue
                case "Active":
                        if url == "" {
                                return "", fmt.Errorf("asset is Active but URL is empty, assetID=%s", assetID)
                        }
                        return url, nil
                case "Failed":
                        if errMsg == "" {
                                errMsg = "unknown asset processing error"
                        }
                        return "", fmt.Errorf("asset processing failed: %s", errMsg)
                default:
                        // 若返回其他状态，保守处理为继续查询
                        fmt.Printf("unexpected status %q, continue polling...\n", status)
                        time.Sleep(interval)
                }
        }
}

// getAssetStatus 调用 GetAsset，返回 Status / URL / Error
func getAssetStatus(client *universal.Universal, assetID string) (status, url, errMsg string, err error) {
resp, err := client.DoCall(
universal.RequestUniversal{
ServiceName: serviceName,
Action:      "GetAsset",
Version:     version,
HttpMethod:  universal.POST,
ContentType: universal.ApplicationJSON,
},
&map[string]any{
"Id": assetID,
},
)
if err != nil {
return "", "", "", err
}
if resp == nil {
return "", "", "", errors.New("get asset response is nil")
}

        // 打印原始返回，便于排查
        respData, _ := sonic.Marshal(resp)
        fmt.Printf("GetAsset response: %s\n", string(respData))

        // 兼容不同层级的字段位置
        status = extractString(resp, "Result", "Status")
        if status == "" {
                status = extractString(resp, "Status")
        }

        url = extractString(resp, "Result", "URL")
        if url == "" {
                url = extractString(resp, "URL")
        }

        errMsg = extractString(resp, "Result", "Error")
        if errMsg == "" {
                errMsg = extractString(resp, "Error")
        }

        return status, url, errMsg, nil
}

// extractString 从响应中按层级安全提取字符串
func extractString(data any, keys ...string) string {
current := data

        for _, key := range keys {
                switch v := current.(type) {
                case map[string]any:
                        next, ok := v[key]
                        if !ok {
                                return ""
                        }
                        current = next

                case *map[string]any:
                        if v == nil {
                                return ""
                        }
                        next, ok := (*v)[key]
                        if !ok {
                                return ""
                        }
                        current = next

                default:
                        return ""
                }
        }

        switch v := current.(type) {
        case string:
                return v
        case fmt.Stringer:
                return v.String()
        case nil:
                return ""
        default:
                return fmt.Sprintf("%v", v)
        }
}
返回示例
CreateAsset response: {"ResponseMetadata":{"RequestId":"202603181520431F067112A17FC078A6DF","Action":"CreateAsset","Version":"2024-01-01","Service":"ark","Region":"cn-beijing"},"Result":{"Id":"Asset-2026**********-*****"}}
asset created, AssetId = asset-20260318072044-n8bcl
GetAsset response: {"ResponseMetadata":{"Service":"ark","Region":"cn-beijing","RequestId":"202603181520448A995106924553F77D0E","Action":"GetAsset","Version":"2024-01-01"},"Result":{"Name":"","GroupId":"group-2026**********-*****","CreateTime":"2026-03-18T07:20:44Z","ProjectName":"default","Id":"Asset-2026**********-*****","URL":"","AssetType":"Image","Status":"Processing","UpdateTime":"2026-03-18T07:20:44Z"}}
asset status: Processing
GetAsset response: {"Result":{"UpdateTime":"2026-03-18T07:20:47Z","ProjectName":"default","Id":"Asset-2026**********-*****","CreateTime":"2026-03-18T07:20:44Z","Name":"","URL":"","AssetType":"Image","GroupId":"group-2026**********-*****","Status":"Processing"},"ResponseMetadata":{"Version":"2024-01-01","Service":"ark","Region":"cn-beijing","RequestId":"2026031815204766FE2BA543E6FF666F66","Action":"GetAsset"}}
asset status: Processing
GetAsset response: {"ResponseMetadata":{"Version":"2024-01-01","Service":"ark","Region":"cn-beijing","RequestId":"202603181520511F067112A17FC078A75A","Action":"GetAsset"},"Result":{"Name":"","URL":"https://ark-media-asset-stg.tos-cn-beijing.volces.com/xxxx","AssetType":"Image","Status":"Active","Id":"Asset-2026**********-*****","GroupId":"group-2026**********-*****","CreateTime":"2026-03-18T07:20:44Z","UpdateTime":"2026-03-18T07:20:47Z","ProjectName":"default"}}
asset status: Active
asset is Active, URL = https://ark-media-asset-stg.tos-cn-beijing.volces.com/xxxx

更多语言的示例代码详见：
注意替换 Demo 中的 AK与SK，若需调用其他接口如 ListAssets，需替换 ACTION 与对应请求参数。
Python
创建素材资产组合：CreateAssetGroup_Demo.py
上传素材资产并获取素材资产信息：CreateAsset&GetAsset_Demo.py
Java
创建素材资产组合：CreateAssetGroup_Demo.java
上传素材资产并获取素材资产信息：CreateAsset_Demo.java
PHP
创建素材资产组合：CreateAssetGroup_Demo.php
上传素材资产：CreateAsset_Demo.php
composer.json
composer.lock

3. 素材资产用于视频生成
   注意：
   在传入给模型的 Prompt 中，需要使用图片 1、视频 1 的方式指代参考素材，素材序号为素材在请求体中的顺序。请勿直接在 Prompt 中直接使用 Asset ID。
   例：“图片1 里的女孩身着图片2中的服装，正在整理柜台上的物品。图片3中的男孩是一位顾客，他走上前，想要向女孩索要联系方式。”
   调用示例请参考常见问题 4
   当上传的素材资产状态为 Active 时，可将素材 Id 按 asset: //<asset_Id> 的规则拼接 URL，以在 视频生成 API 中使用对应的素材资产生成视频：
   {
   "type": "image_url",
   "image_url": {
   "url": "<YOUR_ASSET_ID>" // 示例：asset://asset-2026**********-*****
   },
   "role": "reference_image"
   },
   使用素材资产生成视频的具体调用方式请参考【申请权限填客户名称】Seedance 2.0 & 2.0 fast API文档（邀测用户版）。

最佳实践--私域素材资产上传最佳案例
在上传素材资产时，若将目标人脸图、全身参考图及细节参考图合并为同一张图片，可能导致各参考元素在画面中占比较小，从而增加模型识别难度，造成生成视频中的人物形象与所上传素材资产出现偏差，或造成生成视频中素人脸被误识别为明星脸而触发风控拦截。
建议在上传素材资产时，将人物面部特写、服装细节等关键内容独立分割为单独的图片进行上传。具体可参考如下规则及示例：

应该
不应该

输入内容

给出背景参考图、人物妆造三视图、人物面部无表情特写图、提示词
[图片]

[图片]
[图片]
背景参考图片1，月白虚影闪过，公子（妆造参考图片2；人物形象严格参考图片3）旋身开合折扇，鎏金扇刃弹出，墨竹扇面翻飞，慢鼓重响 1 声；特写：折扇刃格挡反派长刀，扇骨与刀身相击，公子唇角勾轻佻笑意，眼神却冷冽，指腹轻转扇柄。；慢镜：公子侧身贴地滑步，折扇刃贴反派腿侧划过，带起一道浅痕，锦袍下摆扫过地面，玉簪轻晃。快切：公子旋身抬手，折扇刃飞射而出，擦过反派脖颈，钉入身后木柱，反派僵立不敢动。反转：身后突然传来掌风，公子旋身接掌，指尖相触时借力后跳，折扇刃从木柱飞回手中，眼神警惕。慢镜高光：公子折扇半开，扇刃抵唇侧，抬眸望向身后，碎发被风吹起，眉梢微扬，带一丝桀骜。拉镜：公子立于庭院石台上，折扇轻摇，镜头拉远，庭院四角同时浮现戴面具的黑影（持弯刀，呈合围之势）。定格：公子折扇合起一半，扇刃露鎏金锋芒，抬步向前，画面压暗，只留他的侧影和扇刃光，音效骤停。音效：折扇开合脆响 + 刃风切割声 + 慢鼓卡点（偏沉稳）。

给出背景参考图、人物妆造三视图、提示词
[图片]
[图片]
背景参考图片1，月白虚影闪过，公子（参考图片2）旋身开合折扇，鎏金扇刃弹出，墨竹扇面翻飞，慢鼓重响 1 声；特写：折扇刃格挡反派长刀，扇骨与刀身相击，公子唇角勾轻佻笑意，眼神却冷冽，指腹轻转扇柄。；慢镜：公子侧身贴地滑步，折扇刃贴反派腿侧划过，带起一道浅痕，锦袍下摆扫过地面，玉簪轻晃。快切：公子旋身抬手，折扇刃飞射而出，擦过反派脖颈，钉入身后木柱，反派僵立不敢动。反转：身后突然传来掌风，公子旋身接掌，指尖相触时借力后跳，折扇刃从木柱飞回手中，眼神警惕。慢镜高光：公子折扇半开，扇刃抵唇侧，抬眸望向身后，碎发被风吹起，眉梢微扬，带一丝桀骜。拉镜：公子立于庭院石台上，折扇轻摇，镜头拉远，庭院四角同时浮现戴面具的黑影（持弯刀，呈合围之势）。定格：公子折扇合起一半，扇刃露鎏金锋芒，抬步向前，画面压暗，只留他的侧影和扇刃光，音效骤停。音效：折扇开合脆响 + 刃风切割声 + 慢鼓卡点（偏沉稳）。

输出内容
暂时无法在飞书文档外展示此内容
暂时无法在飞书文档外展示此内容

总结
同样是古风打斗剧情：
左边输入内容包括：背景参考图、人物妆造三视图、人物面部无表情特写图、提示词；中间输入内容包括：背景参考图、人物妆造三视图、提示词；右边输入内容包括：背景参考图、人物妆造正视图、提示词。
左边的输出视频更加还原人物面部特征；右边的人物面部特征一致性遵循不佳。


输入内容
给出背景参考图、人物妆造三视图、人物面部无表情特写图、提示词
[图片]

[图片]
[图片]
[图片]
[图片]
3d动画风格，背景参考图片1。人物A（妆造参考图片2；面部特征严格参考图片3）和人物B（妆造参考图片4；面部特征严格参考图片5）手牵手走在花园的小径上，镜头处于人物背后。镜头切到前面，人物A拿起一朵鲜花，轻轻递给人物B。背景音效：轻柔的风吹动树叶和花朵，鸟鸣声。人物B微笑接过花，轻轻闻了闻花香，然后蹲下低头与人物A微笑对视，轻轻拍拍人物A的头，说：“thank you”。背景音效：风铃轻响。

给出背景参考图、人物妆造三视图、提示词
[图片]
[图片]
[图片]
3d动画风格，背景参考图片1。人物A（参考图片2）和人物B（参考图片3）手牵手走在花园的小径上，镜头处于人物背后。镜头切到前面，人物A拿起一朵鲜花，轻轻递给人物B。背景音效：轻柔的风吹动树叶和花朵，鸟鸣声。人物B微笑接过花，轻轻闻了闻花香，然后蹲下低头与人物A微笑对视，轻轻拍拍人物A的头，说：“thank you”。背景音效：风铃轻响。

给出背景参考图、人物妆造正视图、提示词
[图片]
[图片]
[图片]
3d动画风格，背景参考图片1。人物A（妆造参考图片2；面部特征严格参考图片3）和人物B（妆造参考图片4；面部特征严格参考图片5）手牵手走在花园的小径上，镜头处于人物背后。镜头切到前面，人物A拿起一朵鲜花，轻轻递给人物B。背景音效：轻柔的风吹动树叶和花朵，鸟鸣声。人物B微笑接过花，轻轻闻了闻花香，然后蹲下低头与人物A微笑对视，轻轻拍拍人物A的头，说：“thank you”。背景音效：风铃轻响。

输出内容

暂时无法在飞书文档外展示此内容

暂时无法在飞书文档外展示此内容

暂时无法在飞书文档外展示此内容

总结
同样是温馨亲子剧情：
左边输入内容包括：背景参考图、人物妆造三视图、人物面部无表情特写图、提示词；中间输入内容包括：背景参考图、人物妆造三视图、提示词；右边输入内容包括：背景参考图、人物妆造正面图、提示词。
左边的输出视频更加还原人物面部特征；中间的输出视频人物面部特征一致性遵循不佳；右边人物妆造、面部特征一致性遵循不佳。



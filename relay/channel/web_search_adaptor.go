package channel

import (
	"net/http"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// WebSearchAdaptor 是可选接口：仅支持独立 Web Search 端点的 adaptor 才需要实现。
// 这样可以避免在主 Adaptor 接口里加方法、波及所有 channel。
type WebSearchAdaptor interface {
	// ConvertWebSearchRequest 把统一 DTO 转换成上游 channel 自家的请求结构。
	ConvertWebSearchRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.WebSearchRequest) (any, error)
	// DoWebSearchResponse 解析上游响应，返回归一化结果与计费用 Usage。
	DoWebSearchResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.WebSearchResponse, *types.NewAPIError)
}

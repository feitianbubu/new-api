package router

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
)

// SetBillsRouter 初始化 额度管理 路由信息
func SetBillsRouter(Router *gin.RouterGroup) {
	billRouter := Router.Group("bill")
	billRouter.GET("/subscription/info", SubscriptionInfo) // 获取账单列表
	billRouter.GET("/invoices", Invoices)                  // 获取账单列表
	billRouter.GET("/invoiceList", InvoiceList)            // 获取账单列表
	billRouter.GET("/", InvoiceList)                       // 保底
}

func InvoiceList(c *gin.Context) {
	mockRes(c,
		`{
                    "data":  "this is a mock data"
                }`)
}

func Invoices(c *gin.Context) {
	mockRes(c,
		`{
                    "url":  "http://127.0.0.1:8080"
                }`)
}

func SubscriptionInfo(c *gin.Context) {
	mockRes(c,
		`{
					"subscription": {
                        "plan": "enterprise",
                        "interval": "yearly"
                    },
                    "enabled": true,
                    "members": {
                        "size": 1,
                        "limit": 10
                    },
                    "apps": {
                        "size": 1,
                        "limit": 10
                    },
                    "vector_space": {
                        "size": 1,
                        "limit": 10
                    },
                    "documents_upload_quota": {
                        "size": 1,
                        "limit": 10
                    },
                    "annotation_quota_limit": {
                        "size": 1,
                        "limit": 10
                    }
                }`)
}
func mockRes(c *gin.Context, resStr string) {
	var res map[string]any
	err := json.Unmarshal([]byte(resStr), &res)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse mock response",
		})
		return
	}
	c.JSON(http.StatusOK, res)
}

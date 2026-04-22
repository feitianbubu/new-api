package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/dto"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
	Msg     string      `json:"msg"` // for legacy api, will remove later
}

type PageResult struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Data:    data,
		Message: "success",
		Msg:     "success",
	})
}

func SuccessPage[T any](c *gin.Context, data []T) {
	Success(c, PageResult{
		List:  data,
		Total: int64(len(data)),
	})
}

func Fail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{
		Code:    10000,
		Data:    nil,
		Message: msg,
	})
}

// FailWithStatus returns an error response with the specified HTTP status code.
func FailWithStatus(c *gin.Context, statusCode int, msg string) {
	c.JSON(statusCode, Response{
		Code:    statusCode,
		Data:    nil,
		Message: msg,
	})
}

// FailFileError returns an OpenAI Files API compatible error response.
func FailFileError(c *gin.Context, statusCode int, errorType, errorCode, message string) {
	response := dto.FileError{
		Error: struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		}{
			Message: message,
			Type:    errorType,
			Code:    errorCode,
		},
	}
	c.JSON(statusCode, response)
}

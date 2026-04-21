package dto

import "github.com/QuantumNous/new-api/storage/common"

type ExpiresAfter = common.ExpiresAfter

// FileObject OpenAI Files API 兼容的文件对象
type FileObject struct {
	ID          string `json:"id"`                     // 文件ID
	Object      string `json:"object"`                 // 对象类型，固定为 "file"
	Bytes       int64  `json:"bytes"`                  // 文件大小（字节）
	CreatedAt   int64  `json:"created_at"`             // 创建时间戳
	ExpiresAt   int64  `json:"expires_at"`             // 过期时间戳
	Filename    string `json:"filename"`               // 文件名
	ContentType string `json:"content_type,omitempty"` // 文件内容类型
	Purpose     string `json:"purpose"`                // 文件用途
	Usage       *Usage `json:"usage,omitempty"`        // 文件上传的token消费统计
}

// FileListResponse OpenAI Files API 兼容的文件列表响应
type FileListResponse struct {
	Object  string       `json:"object"`   // 对象类型，固定为 "list"
	Data    []FileObject `json:"data"`     // 文件列表
	HasMore bool         `json:"has_more"` // 是否有更多数据
}

// FileDeleteResponse OpenAI Files API 兼容的文件删除响应
type FileDeleteResponse struct {
	ID      string `json:"id"`      // 文件ID
	Object  string `json:"object"`  // 对象类型，固定为 "file"
	Deleted bool   `json:"deleted"` // 是否已删除
}

// FileError OpenAI Files API 兼容的错误响应
type FileError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

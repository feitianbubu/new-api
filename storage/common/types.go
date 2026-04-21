package common

import (
	"io"
	"time"
)

// ExpiresAfter represents the expiration policy for a file
type ExpiresAfter struct {
	Anchor  string `json:"anchor"`  // Anchor timestamp after which the expiration policy applies (e.g., "created_at")
	Seconds int    `json:"seconds"` // Number of seconds after the anchor time that the file will expire (3600-2592000)
}

// FileObject represents a file object in storage
type FileObject struct {
	ID          string            `json:"id"`
	Object      string            `json:"object"`
	Bytes       int64             `json:"bytes"`
	CreatedAt   time.Time         `json:"created_at"`
	ExpiresAt   time.Time         `json:"expires_at"`
	Filename    string            `json:"filename"`
	Purpose     string            `json:"purpose"`
	ETag        string            `json:"etag,omitempty"`
	Key         string            `json:"key,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// FileListResult represents the result of listing files
type FileListResult struct {
	Files      []FileObject `json:"files"`
	HasMore    bool         `json:"has_more"`
	NextMarker string       `json:"next_marker,omitempty"`
}

// UploadOptions represents options for file upload
type UploadOptions struct {
	Filename     string            `json:"filename"`
	ContentType  string            `json:"content_type"`
	Purpose      string            `json:"purpose"`
	UserID       int               `json:"user_id"`
	ObjectKey    string            `json:"object_key,omitempty"`    // 自定义对象路径，如果为空则使用默认路径生成逻辑
	ExpiresAfter *ExpiresAfter     `json:"expires_after,omitempty"` // OpenAI-compatible parameter
	Metadata     map[string]string `json:"metadata,omitempty"`      // 存储到 OSS 的元数据
}

type ListOptions struct {
	UserID  int    `json:"user_id"`
	Limit   int    `json:"limit"`
	After   string `json:"after,omitempty"`
	Purpose string `json:"purpose,omitempty"`
	Prefix  string `json:"prefix,omitempty"` // 按对象 key 前缀过滤
}

type FileContent struct {
	Content       io.ReadCloser     `json:"-"`
	ContentType   string            `json:"content_type"`
	ContentLength int64             `json:"content_length"`
	TotalLength   int64             `json:"total_length,omitempty"`
	RangeStart    int64             `json:"range_start,omitempty"`
	RangeEnd      int64             `json:"range_end,omitempty"`
	Filename      string            `json:"filename,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

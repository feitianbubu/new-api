package storage

import (
	"context"
	"io"
)

type Storage interface {
	UploadFile(ctx context.Context, reader io.Reader, size int64, opts UploadOptions) (*FileObject, error)

	ListFiles(ctx context.Context, opts ListOptions) (*FileListResult, error)

	GetFileInfo(ctx context.Context, fileID string) (*FileObject, error)

	DeleteFile(ctx context.Context, fileID string) error

	GetFileContent(ctx context.Context, fileID string) (*FileContent, error)

	GetModelName() string

	Close() error
}

type URLUploader interface {
	UploadFileByURL(ctx context.Context, sourceURL string, opts UploadOptions) (*FileObject, error)
}

type Presigner interface {
	PresignURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error)
}

type Config struct {
	// Storage provider type: "tos", "s3", "oss", etc.
	Provider string `json:"provider"`

	// Provider-specific configurations
	Endpoint  string `json:"endpoint,omitempty"`
	Region    string `json:"region,omitempty"`
	AccessKey string `json:"access_key,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`
	Bucket    string `json:"bucket,omitempty"`

	// Additional provider-specific options
	Options map[string]interface{} `json:"options,omitempty"`
}

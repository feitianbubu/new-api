package s3

import (
	"fmt"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// Config represents the configuration for an S3-compatible storage backend
// (AWS S3, Cloudflare R2, Backblaze B2, MinIO, Filebase, etc.).
type Config struct {
	Endpoint     string `json:"endpoint"`
	Region       string `json:"region"`
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key"`
	Bucket       string `json:"bucket"`
	UsePathStyle bool   `json:"use_path_style"` // path-style addressing; required by R2/B2/MinIO/Filebase
}

// NewS3StorageFromEnv creates a new S3 storage instance from environment variables.
func NewS3StorageFromEnv() (*S3Storage, error) {
	config := Config{
		Endpoint:     os.Getenv("S3_ENDPOINT"),
		Region:       common.GetEnvOrDefaultString("S3_REGION", "us-east-1"),
		AccessKey:    os.Getenv("S3_ACCESS_KEY"),
		SecretKey:    os.Getenv("S3_SECRET_KEY"),
		Bucket:       os.Getenv("S3_BUCKET"),
		UsePathStyle: common.GetEnvOrDefaultBool("S3_USE_PATH_STYLE", true),
	}

	return NewS3Storage(config)
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Region == "" {
		return fmt.Errorf("S3 region is required")
	}
	if c.AccessKey == "" {
		return fmt.Errorf("S3 access key is required")
	}
	if c.SecretKey == "" {
		return fmt.Errorf("S3 secret key is required")
	}
	if c.Bucket == "" {
		return fmt.Errorf("S3 bucket is required")
	}
	return nil
}

// normalizeEndpoint ensures the endpoint carries an explicit scheme. The
// aws-sdk-go-v2 BaseEndpoint must be a full URL; a bare host like
// "s3.filebase.com" would fail to parse. HTTPS is assumed when no scheme is
// given (an explicit "http://" is kept, e.g. for a local MinIO).
func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if !strings.Contains(endpoint, "://") {
		return "https://" + endpoint
	}
	return endpoint
}

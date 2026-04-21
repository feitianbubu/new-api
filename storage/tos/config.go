package tos

import (
	"fmt"
	"github.com/QuantumNous/new-api/common"
	"os"
)

// Config represents the configuration for TOS storage
type Config struct {
	Endpoint      string `json:"endpoint"`
	Region        string `json:"region"`
	AccessKey     string `json:"access_key"`
	SecretKey     string `json:"secret_key"`
	Bucket        string `json:"bucket"`
	ObjectExpires int64  `json:"object_expires,omitempty"` // Object expiration in days, default is 30
}

// NewTOSStorageFromEnv creates a new TOS storage instance from environment variables
func NewTOSStorageFromEnv() (*TOSStorage, error) {
	config := Config{
		Endpoint:      os.Getenv("TOS_GO_SDK_ENDPOINT"),
		Region:        os.Getenv("TOS_GO_SDK_REGION"),
		AccessKey:     os.Getenv("TOS_GO_SDK_AK"),
		SecretKey:     os.Getenv("TOS_GO_SDK_SK"),
		Bucket:        os.Getenv("TOS_GO_SDK_BUCKET"),
		ObjectExpires: int64(common.GetEnvOrDefault("TOS_GO_SDK_OBJECT_EXPIRES", 30)),
	}

	return NewTOSStorage(config)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("TOS endpoint is required")
	}
	if c.Region == "" {
		return fmt.Errorf("TOS region is required")
	}
	if c.AccessKey == "" {
		return fmt.Errorf("TOS access key is required")
	}
	if c.SecretKey == "" {
		return fmt.Errorf("TOS secret key is required")
	}
	if c.Bucket == "" {
		return fmt.Errorf("TOS bucket is required")
	}
	return nil
}

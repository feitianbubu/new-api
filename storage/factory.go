package storage

import (
	"fmt"
	"os"

	"github.com/QuantumNous/new-api/storage/s3"
	"github.com/QuantumNous/new-api/storage/tos"
)

func NewStorageFromEnv() (Storage, error) {
	provider := os.Getenv("STORAGE_PROVIDER")
	if provider == "" {
		provider = TOSStorage
	}

	switch provider {
	case TOSStorage:
		tosStorage, err := tos.NewTOSStorageFromEnv()
		if err != nil {
			return nil, err
		}
		return NewTOSStorageWrapper(tosStorage), nil
	case S3Storage:
		s3Storage, err := s3.NewS3StorageFromEnv()
		if err != nil {
			return nil, err
		}
		return NewS3StorageWrapper(s3Storage), nil
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", provider)
	}
}

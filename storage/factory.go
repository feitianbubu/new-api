package storage

import (
	"fmt"
	"github.com/QuantumNous/new-api/storage/tos"
	"os"
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
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", provider)
	}
}

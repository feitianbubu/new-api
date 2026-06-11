package storage

import (
	"fmt"
	"os"
	"sync"

	"github.com/QuantumNous/new-api/storage/s3"
	"github.com/QuantumNous/new-api/storage/tos"
)

var (
	storageMu     sync.Mutex
	sharedStorage Storage
)

// NewStorageFromEnv 返回进程级共享的 Storage 单例。
// TOS client 内部有常驻 DNS 刷新协程，按次创建会泄漏 goroutine。
func NewStorageFromEnv() (Storage, error) {
	storageMu.Lock()
	defer storageMu.Unlock()
	if sharedStorage != nil {
		return sharedStorage, nil
	}
	inst, err := newStorageFromEnv()
	if err != nil {
		return nil, err
	}
	sharedStorage = inst
	return inst, nil
}

func newStorageFromEnv() (Storage, error) {
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

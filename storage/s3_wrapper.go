package storage

import "github.com/QuantumNous/new-api/storage/s3"

// S3StorageWrapper wraps the S3 storage to implement the Storage interface
type S3StorageWrapper struct {
	*s3.S3Storage
}

// NewS3StorageWrapper creates a new S3 storage wrapper
func NewS3StorageWrapper(s3Storage *s3.S3Storage) *S3StorageWrapper {
	return &S3StorageWrapper{S3Storage: s3Storage}
}

func (w *S3StorageWrapper) GetModelName() string {
	return S3Storage
}

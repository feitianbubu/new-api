package storage

import "github.com/QuantumNous/new-api/storage/tos"

// TOSStorageWrapper wraps the TOS storage to implement the Storage interface
type TOSStorageWrapper struct {
	*tos.TOSStorage
}

// NewTOSStorageWrapper creates a new TOS storage wrapper
func NewTOSStorageWrapper(tosStorage *tos.TOSStorage) *TOSStorageWrapper {
	return &TOSStorageWrapper{TOSStorage: tosStorage}
}

func (w *TOSStorageWrapper) GetModelName() string {
	return TOSStorage
}

package model

import (
	"fmt"
	"gorm.io/gorm"
	"one-api/common"
	"strings"
	"sync"
)

type CryptoStrategy interface {
	Encrypt(plain string) (string, error)
	Decrypt(cipher string) (string, error)
	Name() string
}

var (
	cryptoStrategies = make(map[string]CryptoStrategy)
	cryptoMu         sync.RWMutex
)

func RegisterGormCryptoStrategyAES(db *gorm.DB) error {
	strategy := AesCryptoStrategy{}
	RegisterCryptoStrategy(strategy)
	return RegisterCryptoPlugin(db)
}

func RegisterCryptoStrategy(strategy CryptoStrategy) {
	cryptoMu.Lock()
	defer cryptoMu.Unlock()
	cryptoStrategies[strategy.Name()] = strategy
}

func getCryptoStrategy(name string) (CryptoStrategy, bool) {
	cryptoMu.RLock()
	defer cryptoMu.RUnlock()
	s, ok := cryptoStrategies[name]
	return s, ok
}

const defaultCryptoStrategy = "aes"

type AesCryptoStrategy struct{}

func (m AesCryptoStrategy) Encrypt(plain string) (string, error) {
	enc, err := common.EncryptAES(plain, common.SessionSecret)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", m.prefix(), enc), nil
}
func (m AesCryptoStrategy) Decrypt(cipher string) (string, error) {
	prefix := m.prefix()
	if !strings.HasPrefix(cipher, prefix) {
		return cipher, nil
	}
	return common.DecryptAES(strings.TrimPrefix(cipher, prefix), common.SessionSecret)
}
func (AesCryptoStrategy) Name() string {
	return defaultCryptoStrategy
}
func (m AesCryptoStrategy) prefix() string {
	return fmt.Sprintf("{%s}", strings.ToUpper(m.Name()))
}

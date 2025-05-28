package model

import (
	"one-api/common"
	"strings"
	"sync"
)

type CryptoStrategy interface {
	Encrypt(plain string) (string, error)
	Decrypt(cipher string) (string, error)
}

var (
	cryptoStrategies = make(map[string]CryptoStrategy)
	cryptoMu         sync.RWMutex
)

func RegisterCryptoStrategy(name string, strategy CryptoStrategy) {
	cryptoMu.Lock()
	defer cryptoMu.Unlock()
	cryptoStrategies[name] = strategy
}

func getCryptoStrategy(name string) (CryptoStrategy, bool) {
	cryptoMu.RLock()
	defer cryptoMu.RUnlock()
	s, ok := cryptoStrategies[name]
	return s, ok
}

type AesCryptoStrategy struct{}

const aesPrefix = "{AES}"

func (AesCryptoStrategy) Encrypt(plain string) (string, error) {
	enc, err := common.EncryptAES(plain, common.SessionSecret)
	if err != nil {
		return "", err
	}
	return aesPrefix + enc, nil
}
func (AesCryptoStrategy) Decrypt(cipher string) (string, error) {
	if !strings.HasPrefix(cipher, aesPrefix) {
		return cipher, nil
	}
	return common.DecryptAES(strings.TrimPrefix(cipher, aesPrefix), common.SessionSecret)
}

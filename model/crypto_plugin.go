package model

import (
	"log"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

func RegisterCryptoPlugin(db *gorm.DB) (err error) {
	if err = db.Callback().Create().Before("gorm:create").Register("crypto:before_create", encryptCryptoFields); err != nil {
		return
	}
	if err = db.Callback().Update().Before("gorm:update").Register("crypto:before_update", encryptCryptoFields); err != nil {
		return
	}
	if err = db.Callback().Query().After("gorm:after_query").Register("crypto:after_query", decryptCryptoFields); err != nil {
		return
	}
	return
}

func encryptCryptoFields(db *gorm.DB) {
	processCryptoFields(db, true)
}

func decryptCryptoFields(db *gorm.DB) {
	processCryptoFields(db, false)
}

func isEncrypted(s string) bool {
	return strings.HasPrefix(s, "{AES}")
}

func processCryptoFields(db *gorm.DB, encrypt bool) {
	if db.Statement == nil || db.Statement.Schema == nil {
		return
	}
	val := db.Statement.ReflectValue
	// 处理指针
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	switch val.Kind() {
	case reflect.Struct:
		if err := processStructCryptoFields(val, encrypt); err != nil {
			log.Printf("[crypto warn] process struct error: %v", err)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			for elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			if elem.Kind() == reflect.Struct {
				if err := processStructCryptoFields(elem, encrypt); err != nil {
					log.Printf("[crypto warn] process struct in slice error: %v", err)
				}
			}
		}
	default:
	}
}

func processStructCryptoFields(val reflect.Value, encrypt bool) error {
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		cryptoTag := field.Tag.Get("crypto")
		if cryptoTag == "" {
			continue
		}
		strategy, ok := getCryptoStrategy(cryptoTag)
		if !ok {
			log.Printf("[crypto warn] unknown crypto strategy: %s", cryptoTag)
			continue
		}
		fv := val.Field(i)
		if fv.Kind() == reflect.String && fv.CanSet() {
			if encrypt {
				plain := fv.String()
				if plain == "" || isEncrypted(plain) {
					continue
				}
				enc, err := strategy.Encrypt(plain)
				if err != nil {
					return err
				}
				fv.SetString(enc)
			} else {
				cipher := fv.String()
				if cipher == "" || !isEncrypted(cipher) {
					continue
				}
				plain, err := strategy.Decrypt(cipher)
				if err != nil {
					return err
				}
				fv.SetString(plain)
			}
		}
	}
	return nil
}
func cryptoWhere(db *gorm.DB, field, value string) *gorm.DB {
	s, ok := getCryptoStrategy("aes")
	if !ok {
		return db.Where(field+" = ?", value)
	}
	enc, err := s.Encrypt(value)
	if err != nil {
		return db.Where(field+" = ?", value)
	}
	return db.Where(field+" = ?", enc)
}

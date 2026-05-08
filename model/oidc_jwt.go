package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/golang-jwt/jwt/v5"
)

type OIDCClaims struct {
	jwt.RegisteredClaims
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	PreferredName string `json:"preferred_username,omitempty"`
	Picture       string `json:"picture,omitempty"`
	Groups        string `json:"groups,omitempty"`
}

// OIDC 授权码 claims
type OIDCAuthorizationCodeClaims struct {
	jwt.RegisteredClaims
	UserID      int64  `json:"user_id"`
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	Scope       string `json:"scope"`
	Nonce       string `json:"nonce,omitempty"`
	// 使用较短的过期时间，比如10分钟
}

// 获取OIDC签名密钥
func getOIDCSigningKey() []byte {
	privateKeyStr := system_setting.GetOIDCProviderSettings().PrivateKey
	if privateKeyStr != "" {
		return []byte(privateKeyStr)
	}
	// 如果没有配置私钥，使用默认的CryptoSecret
	return []byte(common.CryptoSecret)
}

// 生成OIDC ID Token
func GenerateOIDCIDToken(user *User, clientID string, ttl int) (string, error) {
	// 使用统一的token生成函数
	options := &CreateUserJWTOptions{
		ExpirationSeconds: ttl,
		ClientID:          clientID,
		IncludeOIDCFields: true,
		IsOIDCToken:       true,
	}

	tokenString, err := CreateUserJWTWithOptions(nil, user, options)
	if err != nil {
		return "", fmt.Errorf("failed to generate OIDC ID token: %v", err)
	}

	return tokenString, nil
}

// 生成访问令牌 (JWT格式)
func GenerateOIDCAccessToken(user *User, clientID string, ttl int) (string, int64, error) {
	// 如果未指定TTL，使用默认值
	if ttl <= 0 {
		ttl = 3600
	}

	// 使用统一的token生成函数
	options := &CreateUserJWTOptions{
		ExpirationSeconds: ttl,
		ClientID:          clientID,
		IncludeOIDCFields: true,
		IsOIDCToken:       true,
	}

	tokenString, err := CreateUserJWTWithOptions(nil, user, options)
	if err != nil {
		return "", 0, fmt.Errorf("failed to generate OIDC access token: %v", err)
	}

	expiresAt := time.Now().Add(time.Duration(ttl) * time.Second)
	return tokenString, expiresAt.Unix(), nil
}

// 生成OIDC授权码 (JWT格式)
func GenerateOIDCAuthorizationCode(userID int64, clientID, redirectURI, scope, nonce string) (string, error) {
	signingKey := getOIDCSigningKey()

	now := time.Now()
	expiresAt := now.Add(10 * time.Minute) // 授权码有效期10分钟

	claims := OIDCAuthorizationCodeClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    system_setting.ServerAddress,
			Subject:   fmt.Sprintf("%d", userID),
			Audience:  []string{clientID},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			// 使用授权码的特定标识
			ID: fmt.Sprintf("auth_%d_%d", userID, now.UnixNano()),
		},
		UserID:      userID,
		ClientID:    clientID,
		RedirectURI: redirectURI,
		Scope:       scope,
		Nonce:       nonce,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign authorization code: %v", err)
	}

	return tokenString, nil
}

func GenerateOIDCRefreshToken() string {
	refreshToken, err := common.GenerateKey()
	if err != nil {
		// 如果生成失败，返回一个默认的随机字符串
		return fmt.Sprintf("refresh_%d", time.Now().UnixNano())
	}
	return refreshToken
}

func ValidateOIDCToken(tokenString string) (*OIDCClaims, error) {
	signingKey := GetUnifiedTokenSigningKey()

	token, err := jwt.ParseWithClaims(tokenString, &OIDCClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return signingKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if claims, ok := token.Claims.(*OIDCClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// 获取JWKS公钥 (对于HMAC签名，返回一个通用的对称密钥JWK)
func GetOIDCJWKS() (map[string]interface{}, error) {
	// 对于HMAC签名，JWKS通常不包含实际密钥
	// 这里返回一个通用的对称密钥指示
	jwk := map[string]interface{}{
		"kty": "oct", // 对称密钥类型
		"use": "sig",
		"alg": "HS256",
		"kid": "oidc-hmac-key", // 密钥ID
	}

	return map[string]interface{}{
		"keys": []interface{}{jwk},
	}, nil
}

func extractAudFromJWT(tokenString string) string {
	if strings.Count(tokenString, ".") != 2 {
		return ""
	}
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return GetUnifiedTokenSigningKey(), nil
	})
	if err != nil || !token.Valid || len(claims.Audience) == 0 {
		return ""
	}
	return claims.Audience[0]
}

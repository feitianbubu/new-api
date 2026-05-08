package model

import (
	"errors"
)

type OidcClient struct {
	Id           int    `json:"id" gorm:"primaryKey"`
	ClientId     string `json:"client_id" gorm:"type:varchar(100);uniqueIndex;not null"`
	ClientSecret string `json:"client_secret" gorm:"type:varchar(255);not null"`
	Name         string `json:"name" gorm:"type:varchar(100);not null"`
	RedirectUris string `json:"redirect_uris" gorm:"type:text;not null"` // JSON array
	Scopes       string `json:"scopes" gorm:"type:text"`                 // JSON array
	GrantTypes   string `json:"grant_types" gorm:"type:text"`            // JSON array
	TokenTTL     int    `json:"token_ttl" gorm:"type:int;default:3600"`  // Token有效期(秒), 默认1小时
	CreatedAt    int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    int64  `json:"updated_at" gorm:"autoUpdateTime"`
	Status       int    `json:"status" gorm:"type:int;default:1"` // 1=enabled, 0=disabled
}

// GetTokenTTL 获取有效的Token TTL，如果未配置则返回默认值3600秒
func (c *OidcClient) GetTokenTTL() int {
	if c.TokenTTL <= 0 {
		return 3600 // 默认1小时
	}
	return c.TokenTTL
}

func GetOIDCClientByClientId(clientId string) (*OidcClient, error) {
	var client OidcClient
	err := DB.Where("client_id = ? AND status = 1", clientId).First(&client).Error
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func CreateOIDCClient(client *OidcClient) error {
	if client.ClientId == "" || client.ClientSecret == "" || client.Name == "" || client.RedirectUris == "" {
		return errors.New("client_id, client_secret, name and redirect_uris are required")
	}

	// Set default values if not provided
	if client.Scopes == "" {
		client.Scopes = "openid profile email"
	}
	if client.GrantTypes == "" {
		client.GrantTypes = "authorization_code refresh_token"
	}

	return DB.Create(client).Error
}

func UpdateOIDCClient(id int, client *OidcClient) error {
	return DB.Model(&OidcClient{}).Where("id = ?", id).Updates(client).Error
}

func DeleteOIDCClient(id int) error {
	return DB.Delete(&OidcClient{}, id).Error
}

func GetAllOIDCClients() ([]OidcClient, error) {
	var clients []OidcClient
	err := DB.Order("created_at DESC").Find(&clients).Error
	return clients, err
}

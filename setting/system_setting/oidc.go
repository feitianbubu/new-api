package system_setting

import "github.com/QuantumNous/new-api/common"
import "github.com/QuantumNous/new-api/setting/config"

type OIDCSettings struct {
	Enabled               bool   `json:"enabled"`
	ClientId              string `json:"client_id"`
	ClientSecret          string `json:"client_secret"`
	WellKnown             string `json:"well_known"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"user_info_endpoint"`
}

// OIDC Provider settings - read from environment variables
type OIDCProviderSettings struct {
	Enabled    bool
	PrivateKey string
}

func GetOIDCProviderSettings() *OIDCProviderSettings {
	return &OIDCProviderSettings{
		Enabled:    common.GetEnvOrDefaultBool("OIDC_PROVIDER_ENABLED", false),
		PrivateKey: common.GetEnvOrDefaultString("SESSION_SECRET", ""),
	}
}

// 默认配置
var defaultOIDCSettings = OIDCSettings{}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("oidc", &defaultOIDCSettings)
}

func GetOIDCSettings() *OIDCSettings {
	return &defaultOIDCSettings
}

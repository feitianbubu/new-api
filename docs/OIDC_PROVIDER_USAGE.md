# OIDC Provider 使用说明

本文档介绍如何使用本系统的 OIDC Provider 功能。

## 概述

本系统支持作为 OIDC Provider，为第三方应用提供标准的 OAuth2/OpenID Connect 身份认证服务。

## 功能特性

- ✅ 标准 OIDC Discovery 端点
- ✅ 授权码流程 (Authorization Code Flow)
- ✅ JWT ID Token
- ✅ 用户信息端点
- ✅ JSON Web Key Set (JWKS)
- ✅ 客户端管理 API
- ⏳ Refresh Token (计划中)
- ⏳ PKCE (计划中)

## 端点列表

| 端点 | 方法 | 描述 |
|------|------|------|
| `/.well-known/openid-configuration` | GET | OIDC Discovery 端点(标准) |
| `/oauth/jwks` | GET | JSON Web Key Set |
| `/oauth/authorize` | GET | 授权端点 |
| `/oauth/token` | POST | 令牌端点 |
| `/oauth/userinfo` | GET | 用户信息端点 |

## 快速开始

### 1. 启用 OIDC Provider

在系统配置中启用 OIDC Provider：

```json
{
  "oidc": {
    "provider_enabled": true,
    "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"
  }
}
```

### 2. 创建 OIDC 客户端

使用 API 创建新的 OIDC 客户端：

```bash
curl -X POST http://localhost:3000/api/oidc_provider/clients \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "name": "My Application",
    "redirect_uris": "[\"http://localhost:8080/callback\"]",
    "scopes": "[\"openid\", \"profile\", \"email\"]"
  }'
```

响应示例：
```json
{
  "success": true,
  "message": "Client created successfully",
  "data": {
    "id": 1,
    "client_id": "sk-xxxxxxxx",
    "client_secret": "sk-yyyyyyyy",
    "name": "My Application",
    "redirect_uris": "[\"http://localhost:8080/callback\"]",
    "scopes": "[\"openid\", \"profile\", \"email\"]",
    "status": 1
  }
}
```

### 3. 第三方应用集成

#### 3.1 获取 OIDC 配置

```bash
curl http://localhost:3000/.well-known/openid-configuration
```

#### 3.2 授权流程

1. **构造授权 URL**：
```
http://localhost:3000/oauth/authorize?
  response_type=code&
  client_id=sk-xxxxxxxx&
  redirect_uri=http://localhost:8080/callback&
  scope=openid%20profile%20email&
  state=random_string&
  nonce=random_nonce
```

2. **用户授权**：
用户会被重定向到登录页面，登录成功后授权访问。

3. **获取授权码**：
授权成功后，用户会被重定向到指定的 `redirect_uri`，并携带授权码：
```
http://localhost:8080/callback?code=auth_code_here&state=random_string
```

4. **交换令牌**：
```bash
curl -X POST http://localhost:3000/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d '{
    "grant_type": "authorization_code",
    "code": "auth_code_here",
    "redirect_uri": "http://localhost:8080/callback",
    "client_id": "sk-xxxxxxxx",
    "client_secret": "sk-yyyyyyyy"
  }'
```

响应示例：
```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "refresh_token_here",
  "id_token": "eyJ...",
  "scope": "openid profile email"
}
```

5. **获取用户信息**：
```bash
curl -H "Authorization: Bearer access_token_here" \
  http://localhost:3000/oauth/userinfo
```

响应示例：
```json
{
  "sub": "123",
  "username": "testuser",
  "preferred_username": "testuser",
  "name": "Test User",
  "email": "test@example.com",
  "email_verified": true
}
```

## 客户端管理 API

### 获取所有客户端

```bash
curl -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  http://localhost:3000/api/oidc_provider/clients
```

### 更新客户端

```bash
curl -X PUT http://localhost:3000/api/oidc_provider/clients/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "name": "Updated Application",
    "redirect_uris": "[\"http://localhost:8080/callback\", \"http://localhost:8081/callback\"]"
  }'
```

### 删除客户端

```bash
curl -X DELETE http://localhost:3000/api/oidc_provider/clients/1 \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

## 安全注意事项

1. **HTTPS**：生产环境中必须使用 HTTPS
2. **Redirect URI**：确保注册的 redirect_uri 安全可信
3. **Client Secret**：妥善保管 client_secret
4. **State 参数**：使用 state 参数防止 CSRF 攻击
5. **Nonce 参数**：使用 nonce 参数防止重放攻击

## 故障排除

### 常见错误

1. **`invalid_client`**：客户端认证失败，检查 client_id 和 client_secret
2. **`invalid_request`**：请求参数错误，检查必需参数
3. **`invalid_scope`**：scope 必须包含 `openid`
4. **`unsupported_response_type`**：目前只支持 `code` 流程

### 调试技巧

1. 检查 Discovery 端点是否可访问
2. 验证客户端配置是否正确
3. 查看服务器日志了解详细错误信息

## 示例应用

参考 `examples/oidc-client` 目录中的示例应用，了解如何集成 OIDC 认证。

## 技术支持

如有问题，请提交 Issue 或联系技术支持。
# Quickstart: HTTP Basic Auth 认证

**Feature**: 011-basic-auth  
**Date**: 2024-12-30

## 快速开始

### 1. 启用认证

在 `config.toml` 中添加认证配置：

```toml
[auth]
username = "admin"
password = "your-secure-password"
```

或使用环境变量：

```bash
export CLOUDSYNC_AUTH_USERNAME=admin
export CLOUDSYNC_AUTH_PASSWORD=your-secure-password
```

### 2. 启动服务

```bash
go run ./cmd/cloud-sync serve
```

### 3. 访问系统

打开浏览器访问 `http://localhost:8080`，将看到 HTTP Basic Auth 认证对话框。

输入配置的用户名和密码即可访问系统。

## 验证认证工作

### 测试受保护端点

```bash
# 无认证 - 应返回 401
curl -i http://localhost:8080/api/graphql

# 有认证 - 应返回 200
curl -u admin:your-secure-password http://localhost:8080/api/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __typename }"}'
```

### 测试健康检查端点

```bash
# 无需认证 - 应返回 200
curl -i http://localhost:8080/health
```

## 禁用认证

留空或删除 `[auth]` 配置块：

```toml
# config.toml
# [auth] 块省略或用户名密码都为空
```

系统将保持开放访问（向后兼容）。

## 生产环境建议

⚠️ **重要安全提示**：

1. **使用 HTTPS**：HTTP Basic Auth 凭据以 Base64 编码传输，必须使用 HTTPS 保护传输安全
2. **反向代理**：建议在生产环境使用 Nginx/Caddy 等反向代理提供 TLS 终止
3. **保护配置文件**：配置文件中的密码为明文，请确保配置文件权限为 `600`
4. **强密码**：使用复杂密码，避免使用默认或简单密码

### 使用反向代理示例 (Nginx)

```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 故障排除

### 问题：启动时报错 "Invalid auth configuration"

**原因**：仅设置了用户名或密码，但不是两者都设置。

**解决**：确保用户名和密码要么都设置，要么都为空。

### 问题：认证对话框一直弹出

**可能原因**：
1. 密码输入错误
2. 配置文件中的密码与输入不匹配
3. 环境变量覆盖了配置文件

**排查**：
```bash
# 检查实际使用的配置
echo $CLOUDSYNC_AUTH_USERNAME
echo $CLOUDSYNC_AUTH_PASSWORD
```

### 问题：健康检查也需要认证

**原因**：这是 BUG，健康检查 `/health` 应始终无需认证。

**检查**：确认认证中间件正确配置，`/health` 在认证中间件之前注册。

## 开发测试

运行认证相关测试：

```bash
go test ./internal/api/context/... -v -run TestAuth
go test ./internal/core/config/... -v -run TestAuthConfig
```

## 文件清单

实现此功能需要修改/新增的文件：

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/core/config/config.go` | 修改 | 添加 Auth 结构体 |
| `internal/api/context/auth.go` | 新增 | Basic Auth 中间件 |
| `internal/api/context/auth_test.go` | 新增 | 中间件测试 |
| `internal/api/server.go` | 修改 | 集成认证中间件 |
| `internal/i18n/keys.go` | 修改 | 添加错误消息 key |
| `internal/i18n/locales/en.toml` | 修改 | 英文翻译 |
| `internal/i18n/locales/zh-CN.toml` | 修改 | 中文翻译 |

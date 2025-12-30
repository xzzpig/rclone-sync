# Data Model: HTTP Basic Auth 认证

**Feature**: 011-basic-auth  
**Date**: 2024-12-30  
**Status**: Complete

## Overview

HTTP Basic Auth 功能不涉及数据库存储。凭据存储在配置文件或环境变量中，运行时加载到内存。

## Configuration Entity

### AuthConfig

认证配置，作为主配置结构的一部分。

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| Username | string | No* | "" | 认证用户名 |
| Password | string | No* | "" | 认证密码 |

*注意：Username 和 Password 必须同时为空（禁用认证）或同时非空（启用认证）。

### Validation Rules

1. **启动时验证**:
   - `Username != "" && Password == ""` → ERROR: 必须同时设置用户名和密码
   - `Username == "" && Password != ""` → ERROR: 必须同时设置用户名和密码
   - `Username == "" && Password == ""` → OK: 认证禁用
   - `Username != "" && Password != ""` → OK: 认证启用

2. **运行时行为**:
   - 认证禁用时：所有请求直接通过
   - 认证启用时：验证每个请求的 Basic Auth 凭据

## State Transitions

### Authentication State

```
┌─────────────────┐
│   Disabled      │  (Username == "" && Password == "")
│  (Open Access)  │
└─────────────────┘

        OR

┌─────────────────┐
│    Enabled      │  (Username != "" && Password != "")
│ (Auth Required) │
└─────────────────┘
```

### Request Authentication Flow

```
[Request] 
    │
    ▼
┌──────────────────┐
│ Auth Enabled?    │
└────────┬─────────┘
         │
    No   │   Yes
    │    │    │
    │    │    ▼
    │    │ ┌────────────────────┐
    │    │ │ Has Auth Header?   │
    │    │ └────────┬───────────┘
    │    │          │
    │    │    No    │   Yes
    │    │    │     │    │
    │    │    │     │    ▼
    │    │    │     │ ┌───────────────────┐
    │    │    │     │ │ Credentials Valid?│
    │    │    │     │ └────────┬──────────┘
    │    │    │     │          │
    │    │    │     │    No    │   Yes
    │    │    │     │    │     │    │
    │    │    ▼     │    ▼     │    │
    │    │ ┌────────────────┐  │    │
    │    │ │  401 Unauth    │  │    │
    │    │ │ WWW-Authenticate│ │    │
    │    │ └────────────────┘  │    │
    │    │                     │    │
    ▼    │                     │    ▼
┌────────────────────────────────────┐
│          Request Processed          │
└────────────────────────────────────┘
```

## Go Type Definitions

### Config Extension (config.go)

```go
// Auth contains HTTP Basic Auth configuration
type Auth struct {
    Username string `mapstructure:"username"`
    Password string `mapstructure:"password"`
}

// Config represents the application configuration structure.
type Config struct {
    // ... existing fields ...
    Auth Auth `mapstructure:"auth"`
}

// IsAuthEnabled returns true if authentication is configured
func (c *Config) IsAuthEnabled() bool {
    return c.Auth.Username != "" && c.Auth.Password != ""
}

// ValidateAuth checks if auth configuration is valid
func (c *Config) ValidateAuth() error {
    hasUsername := c.Auth.Username != ""
    hasPassword := c.Auth.Password != ""
    if hasUsername != hasPassword {
        return fmt.Errorf("invalid auth config: username and password must both be set or both be empty")
    }
    return nil
}
```

### Middleware Context

```go
// AuthUserKey is the key for authenticated username in gin.Context
// Uses gin.AuthUserKey for compatibility with Gin's built-in BasicAuth
const AuthUserKey = gin.AuthUserKey  // "user"
```

## Configuration Files

### TOML Format (config.toml)

```toml
[auth]
username = "admin"
password = "your-secure-password"
```

### Environment Variables

| Variable | Maps To | Example |
|----------|---------|---------|
| `CLOUDSYNC_AUTH_USERNAME` | auth.username | `admin` |
| `CLOUDSYNC_AUTH_PASSWORD` | auth.password | `secret123` |

## No Database Changes

此功能不需要数据库 schema 变更：
- 无新表
- 无字段修改
- 无迁移脚本

## Integration Points

| Component | Integration |
|-----------|-------------|
| `internal/core/config/config.go` | 添加 Auth 结构体 |
| `internal/api/server.go` | 添加认证中间件 |
| `internal/api/context/auth.go` | 新增认证中间件 |
| `internal/i18n/keys.go` | 添加错误消息 key |
| `internal/i18n/locales/*.toml` | 添加翻译 |

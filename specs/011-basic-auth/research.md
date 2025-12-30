# Research: HTTP Basic Auth 认证

**Feature**: 011-basic-auth  
**Date**: 2024-12-30  
**Status**: Complete

## Research Tasks

### 1. Gin HTTP Basic Auth 实现方式

**Decision**: 使用 Gin 内置的 `gin.BasicAuth()` 中间件进行认证

**Rationale**:
- Gin 框架原生支持 HTTP Basic Auth，提供 `gin.BasicAuth(gin.Accounts)` 中间件
- 内置实现已经处理了 WWW-Authenticate 响应头和 401 状态码
- 可以通过 `c.MustGet(gin.AuthUserKey)` 获取已认证用户信息
- 无需引入额外依赖

**Alternatives considered**:
1. **手动实现 Basic Auth**: 需要自行解析 Authorization header、Base64 解码、密码比较
   - 拒绝原因: Gin 已有成熟实现，无需重复造轮子
2. **第三方认证中间件**: 如 gin-contrib/sessions
   - 拒绝原因: Basic Auth 无需 session 管理，过于复杂

**Implementation Notes**:
```go
// Gin BasicAuth 示例
authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
    "username": "password",
}))
```

但由于我们需要从配置动态读取凭据，需要自定义中间件而非直接使用 `gin.BasicAuth()`：
```go
func BasicAuthMiddleware(username, password string) gin.HandlerFunc {
    return func(c *gin.Context) {
        user, pass, hasAuth := c.Request.BasicAuth()
        if !hasAuth || user != username || pass != password {
            c.Header("WWW-Authenticate", `Basic realm="Login Required"`)
            c.AbortWithStatus(http.StatusUnauthorized)
            return
        }
        c.Set(gin.AuthUserKey, user)
        c.Next()
    }
}
```

---

### 2. 配置结构设计

**Decision**: 在 `Config` 结构体中添加顶层 `Auth` 字段

**Rationale**:
- 遵循现有配置结构模式（Server、Database、App 等顶层字段）
- 使用 `[auth]` 配置块，语义清晰
- 环境变量前缀 `CLOUDSYNC_AUTH_*` 符合现有命名约定

**Config Structure**:
```go
type Config struct {
    // ... existing fields ...
    Auth struct {
        Username string `mapstructure:"username"`
        Password string `mapstructure:"password"`
    } `mapstructure:"auth"`
}
```

**TOML Configuration**:
```toml
[auth]
username = "admin"
password = "secretpassword"
```

**Environment Variables**:
- `CLOUDSYNC_AUTH_USERNAME`
- `CLOUDSYNC_AUTH_PASSWORD`

**Validation Rules**:
- 启动时验证：两个字段必须同时为空或同时非空
- 仅设置用户名或仅设置密码时，拒绝启动并报错

---

### 3. 中间件集成位置

**Decision**: 在 `/health` 端点之后、其他所有路由之前应用认证中间件

**Rationale**:
- `/health` 必须无需认证（用于健康检查、负载均衡器探测）
- 所有其他端点（API、GraphQL、静态资源）都需要认证
- 使用 `r.Use()` 全局应用，但 `/health` 需要单独注册在认证之前

**Implementation Approach**:
```go
// SetupRouter 中的顺序
r := gin.New()
r.Use(ginLogger(...))
r.Use(gin.Recovery())

// Health check BEFORE auth
r.GET("/health", healthHandler)

// Apply auth middleware AFTER health
if authEnabled {
    r.Use(authMiddleware(username, password))
}

// All other routes (API, static files)
apiGroup := r.Group("/api")
```

---

### 4. 日志记录策略

**Decision**: 使用结构化日志记录认证失败事件

**Rationale**:
- 符合 Constitution V. Observability and Reliability 原则
- 使用 zap 记录结构化字段（IP、时间、用户名）
- 不记录密码（安全要求）

**Log Format**:
```go
authLog().Warn("authentication failed",
    zap.String("ip", c.ClientIP()),
    zap.String("username", attemptedUsername),
    zap.String("path", c.Request.URL.Path),
)
```

---

### 5. 错误消息国际化

**Decision**: 添加新的 i18n key 用于认证相关错误

**Rationale**:
- 符合 Constitution IX. Internationalization Standards
- 启动配置验证失败需要国际化错误消息

**New i18n Keys**:
```go
// keys.go
const (
    ErrAuthConfigInvalid = "error_auth_config_invalid"  // 认证配置无效（仅设置用户名或密码）
)
```

**Locale Files**:
```toml
# en.toml
error_auth_config_invalid = "Invalid auth configuration: username and password must both be set or both be empty"

# zh-CN.toml
error_auth_config_invalid = "认证配置无效：用户名和密码必须同时设置或同时为空"
```

---

### 6. 安全时序攻击防护

**Decision**: 使用 `crypto/subtle.ConstantTimeCompare` 进行密码比较

**Rationale**:
- 防止通过响应时间差异推断密码
- Go 标准库提供常量时间比较函数
- 额外安全层，虽然 Basic Auth 本身安全性有限

**Implementation**:
```go
import "crypto/subtle"

func isValidCredentials(inputUser, inputPass, configUser, configPass string) bool {
    userMatch := subtle.ConstantTimeCompare([]byte(inputUser), []byte(configUser)) == 1
    passMatch := subtle.ConstantTimeCompare([]byte(inputPass), []byte(configPass)) == 1
    return userMatch && passMatch
}
```

---

## Unresolved Items

无 - 所有技术问题已解决。

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| crypto/subtle | stdlib | 常量时间密码比较 |
| gin-gonic/gin | existing | Web 框架 |
| spf13/viper | existing | 配置管理 |
| go.uber.org/zap | existing | 结构化日志 |

## References

- [Gin BasicAuth Documentation](https://github.com/gin-gonic/gin#using-basicauth-middleware)
- [RFC 7617 - HTTP Basic Authentication](https://tools.ietf.org/html/rfc7617)
- [OWASP Authentication Cheatsheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)

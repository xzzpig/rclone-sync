# Quickstart: Rclone 连接配置数据库存储

**Feature**: 004-rclone-config-db  
**Date**: 2025-12-16  
**Estimated Effort**: 4-6 开发日

## 快速概览

本功能将 rclone 连接配置从配置文件迁移到 SQLite 数据库存储，主要变更包括：

1. **新增 Connection 实体** - 存储连接配置到数据库
2. **实现 config.Storage 接口** - 通过 `config.SetData()` 替换默认配置存储
3. **整体配置加密** - AES-256-GCM 加密整个配置 JSON（无需识别敏感字段）
4. **自动令牌刷新** - Storage.SetValue 自动处理 OAuth 令牌更新
5. **导入向导** - 从 rclone.conf 批量导入连接

## 先决条件

- Go 1.25+
- 已安装 Node.js 18+ 和 pnpm
- 已配置 Ent CLI (`go tool ent`)

## 实现步骤

### Step 1: 数据库 Schema

创建 Connection 实体 schema：

```bash
touch internal/core/db/schema/connection.go
```

```go
// internal/core/db/schema/connection.go
package schema

import (
    "time"

    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
    "github.com/google/uuid"
)

type Connection struct {
    ent.Schema
}

func (Connection) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("id", uuid.UUID{}).Default(uuid.New),
        field.String("name").NotEmpty().Unique(),
        field.String("type").NotEmpty(),
        field.Bytes("encrypted_config").
            Comment("AES-GCM encrypted configuration JSON"),
        field.Time("created_at").Default(time.Now).Immutable(),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
    }
}

func (Connection) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("name").Unique(),
        index.Fields("type"),
    }
}

func (Connection) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("tasks", Task.Type).
            Annotations(entsql.OnDelete(entsql.Cascade)),
    }
}
```

同时修改 Task schema，将 `remote_name` 改为 `connection_id` 外键：

```go
// internal/core/db/schema/task.go (修改部分)
func (Task) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("id", uuid.UUID{}).Default(uuid.New),
        field.String("name").NotEmpty(),
        field.String("source_path").NotEmpty(),
        field.UUID("connection_id", uuid.UUID{}).Optional(),  // 新增：外键
        field.String("remote_path").NotEmpty(),
        // ... 其他字段保持不变
        // 注意：移除 field.String("remote_name")
    }
}

func (Task) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("jobs", Job.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
        edge.From("connection", Connection.Type).
            Ref("tasks").
            Unique().
            Field("connection_id"),  // 绑定到外键字段
    }
}
```

生成 Ent 代码：

```bash
go generate ./internal/core/ent
```

### Step 2: 加密模块

创建加密工具包（整体加密配置 JSON）：

```bash
mkdir -p internal/core/crypto
touch internal/core/crypto/crypto.go
touch internal/core/crypto/crypto_test.go
```

关键接口：

```go
// internal/core/crypto/crypto.go
package crypto

// Encryptor 提供整体配置加密/解密能力，支持可选加密
type Encryptor struct {
    key       []byte
    plaintext bool // true 表示不加密
}

// NewEncryptor 创建加密器
// - 密钥为空: plaintext 模式（不加密，适合开发环境）
// - 密钥非空: 使用 SHA-256 转换为 32 字节后进行 AES-256-GCM 加密
func NewEncryptor(key string) (*Encryptor, error)

// EncryptConfig 加密整个配置 map
// - Plaintext 模式: 返回 JSON
// - 加密模式: 返回 AES-GCM 加密后的数据
func (e *Encryptor) EncryptConfig(config map[string]string) ([]byte, error)

// DecryptConfig 解密整个配置
// - Plaintext 模式: 直接解析 JSON
// - 加密模式: AES-GCM 解密后解析 JSON
func (e *Encryptor) DecryptConfig(encrypted []byte) (map[string]string, error)
```

### Step 3: Connection Service 接口

在 ports 中定义 ConnectionService 接口：

```go
// internal/core/ports/interfaces.go - 新增接口

// ConnectionService 定义连接管理服务接口
type ConnectionService interface {
    // ListConnections 列出所有连接
    ListConnections(ctx context.Context) ([]*ent.Connection, error)

    // GetConnectionByName 按名称获取连接
    GetConnectionByName(ctx context.Context, name string) (*ent.Connection, error)

    // CreateConnection 创建新连接
    CreateConnection(ctx context.Context, name, connType string, config map[string]string) (*ent.Connection, error)

    // UpdateConnection 更新连接
    UpdateConnection(ctx context.Context, id uuid.UUID, name, connType string, config map[string]string) error

    // DeleteConnectionByName 按名称删除连接
    DeleteConnectionByName(ctx context.Context, name string) error
}
```

### Step 4: Connection Service 实现

创建连接服务层：

```bash
touch internal/core/services/connection_service.go
touch internal/core/services/connection_service_test.go
```

关键实现：

```go
// internal/core/services/connection_service.go
package services

type connectionService struct {
    db        *ent.Client
    encryptor *crypto.Encryptor
}

func NewConnectionService(db *ent.Client, encryptor *crypto.Encryptor) ports.ConnectionService {
    return &connectionService{db: db, encryptor: encryptor}
}

func (s *connectionService) ListConnections(ctx context.Context) ([]*ent.Connection, error) {
    conns, err := s.db.Connection.Query().All(ctx)
    if err != nil {
        return nil, err
    }
    // 解密每个连接的配置
    for _, conn := range conns {
        conn.Config, _ = s.encryptor.DecryptConfig(conn.EncryptedConfig)
    }
    return conns, nil
}

func (s *connectionService) GetConnectionByName(ctx context.Context, name string) (*ent.Connection, error) {
    conn, err := s.db.Connection.Query().
        Where(connection.NameEQ(name)).
        Only(ctx)
    if err != nil {
        return nil, err
    }
    conn.Config, _ = s.encryptor.DecryptConfig(conn.EncryptedConfig)
    return conn, nil
}

func (s *connectionService) CreateConnection(ctx context.Context, name, connType string, config map[string]string) (*ent.Connection, error) {
    encrypted, err := s.encryptor.EncryptConfig(config)
    if err != nil {
        return nil, err
    }
    return s.db.Connection.Create().
        SetName(name).
        SetType(connType).
        SetEncryptedConfig(encrypted).
        Save(ctx)
}
```

### Step 5: DBStorage 实现 (核心)

创建实现 `config.Storage` 接口的 DBStorage：

```bash
touch internal/rclone/storage.go
touch internal/rclone/storage_test.go
```

```go
// internal/rclone/storage.go
package rclone

import (
    "context"
    "encoding/json"

    "github.com/rclone/rclone/fs/cache"
    "github.com/rclone/rclone/fs/config"

    "your-module/internal/core/ports"
)

// DBStorage 实现 config.Storage 接口，将配置存储到数据库
type DBStorage struct {
    svc ports.ConnectionService
}

// NewDBStorage 创建数据库存储实例
func NewDBStorage(svc ports.ConnectionService) *DBStorage {
    return &DBStorage{svc: svc}
}

// Install 安装 DBStorage 为 rclone 的配置存储
// 注意：调用此方法后，不要再调用 configfile.Install()
func (s *DBStorage) Install() {
    config.SetData(s)
}

func (s *DBStorage) GetSectionList() []string {
    ctx := context.Background()
    conns, err := s.svc.ListConnections(ctx)
    if err != nil {
        return nil
    }
    names := make([]string, len(conns))
    for i, c := range conns {
        names[i] = c.Name
    }
    return names
}

func (s *DBStorage) HasSection(section string) bool {
    ctx := context.Background()
    _, err := s.svc.GetConnectionByName(ctx, section)
    return err == nil
}

func (s *DBStorage) DeleteSection(section string) {
    ctx := context.Background()
    _ = s.svc.DeleteConnectionByName(ctx, section)
    cache.ClearConfig(section)
}

func (s *DBStorage) GetKeyList(section string) []string {
    ctx := context.Background()
    conn, err := s.svc.GetConnectionByName(ctx, section)
    if err != nil {
        return nil
    }
    keys := make([]string, 0, len(conn.Config))
    for k := range conn.Config {
        keys = append(keys, k)
    }
    return keys
}

func (s *DBStorage) GetValue(section, key string) (string, bool) {
    ctx := context.Background()
    conn, err := s.svc.GetConnectionByName(ctx, section)
    if err != nil {
        return "", false
    }
    v, ok := conn.Config[key]
    return v, ok
}

func (s *DBStorage) SetValue(section, key, value string) {
    ctx := context.Background()
    conn, err := s.svc.GetConnectionByName(ctx, section)
    if err != nil {
        // 连接不存在，创建新连接
        config := map[string]string{key: value}
        connType := ""
        if key == "type" {
            connType = value
        }
        _ = s.svc.CreateConnection(ctx, section, connType, config)
        return
    }

    // 更新现有配置
    newConfig := make(map[string]string, len(conn.Config))
    for k, v := range conn.Config {
        newConfig[k] = v
    }
    newConfig[key] = value

    connType := conn.Type
    if key == "type" {
        connType = value
    }
    _ = s.svc.UpdateConnection(ctx, conn.ID, conn.Name, connType, newConfig)
    cache.ClearConfig(section)
}

func (s *DBStorage) DeleteKey(section, key string) bool {
    ctx := context.Background()
    conn, err := s.svc.GetConnectionByName(ctx, section)
    if err != nil {
        return false
    }
    if _, ok := conn.Config[key]; !ok {
        return false
    }
    newConfig := make(map[string]string, len(conn.Config)-1)
    for k, v := range conn.Config {
        if k != key {
            newConfig[k] = v
        }
    }
    _ = s.svc.UpdateConnection(ctx, conn.ID, conn.Name, conn.Type, newConfig)
    cache.ClearConfig(section)
    return true
}

func (s *DBStorage) Load() error {
    return nil  // 数据库无需显式加载
}

func (s *DBStorage) Save() error {
    return nil  // 每次 SetValue 已持久化
}

func (s *DBStorage) Serialize() (string, error) {
    ctx := context.Background()
    conns, err := s.svc.ListConnections(ctx)
    if err != nil {
        return "", err
    }
    result := make(map[string]map[string]string)
    for _, c := range conns {
        result[c.Name] = c.Config
    }
    data, err := json.MarshalIndent(result, "", "  ")
    return string(data), err
}
```

### Step 6: Cache 状态检查辅助函数

创建缓存状态检查辅助函数（用于 API 获取加载状态）：

```bash
touch internal/rclone/cache_helper.go
touch internal/rclone/cache_helper_test.go
```

```go
// internal/rclone/cache_helper.go
package rclone

import (
    "context"
    "errors"

    "github.com/rclone/rclone/fs"
    "github.com/rclone/rclone/fs/cache"
)

// ErrNotInCache 是用于检测缓存未命中的哨兵错误
var ErrNotInCache = errors.New("connection not in cache")

// IsConnectionLoaded 检查连接是否已加载到 rclone cache
func IsConnectionLoaded(ctx context.Context, name string) (fs.Fs, bool) {
    f, err := cache.GetFn(ctx, name+":", func(ctx context.Context, s string) (fs.Fs, error) {
        return nil, ErrNotInCache
    })

    if errors.Is(err, ErrNotInCache) {
        return nil, false
    }
    if err != nil {
        return nil, false
    }
    return f, true
}
```

### Step 7: 应用启动集成

更新应用启动流程：

```go
// cmd/cloud-sync/serve.go
func runServe(cmd *cobra.Command, args []string) error {
    // 初始化数据库
    db, err := db.Open(cfg.DatabasePath)
    if err != nil {
        return err
    }
    defer db.Close()

    // 初始化加密器
    // 密钥为空时启用 plaintext 模式（不加密）
    encryptor, err := crypto.NewEncryptor(cfg.Security.EncryptionKey)
    if err != nil {
        return err
    }

    // 创建 ConnectionService
    connSvc := services.NewConnectionService(db, encryptor)

    // 安装 DBStorage（关键步骤！替换默认的配置文件存储）
    // rclone 会通过 Storage 接口按需读取配置，无需预加载
    dbStorage := rclone.NewDBStorage(connSvc)
    dbStorage.Install()

    // 创建其他服务...
    taskSvc := services.NewTaskService(db)
    jobSvc := services.NewJobService(db)

    // 启动 API server
    server := api.NewServer(cfg, connSvc, taskSvc, jobSvc)
    return server.Run()
}
```

**重要提示**:

- 必须在使用任何 rclone 功能之前调用 `dbStorage.Install()`
- 不要调用 `configfile.Install()`，否则会覆盖我们的 DBStorage

### Step 8: API Handlers

创建连接 API 处理程序：

```bash
touch internal/api/handlers/connection.go
touch internal/api/handlers/connection_test.go
touch internal/api/handlers/import.go
touch internal/api/handlers/import_test.go
```

路由注册：

```go
// internal/api/routes.go
connections := router.Group("/connections")
{
    connections.GET("", connHandler.List)
    connections.POST("", connHandler.Create)
    connections.POST("/test", connHandler.TestUnsavedConfig)
    connections.GET("/:id", connHandler.Get)
    connections.PUT("/:id", connHandler.Update)
    connections.DELETE("/:id", connHandler.Delete)
    connections.GET("/:id/config", connHandler.GetConfig)
    connections.POST("/:id/test", connHandler.Test)
    connections.GET("/:id/quota", connHandler.GetQuota)
}

importGroup := router.Group("/import")
{
    importGroup.POST("/parse", importHandler.Parse)
    importGroup.POST("/execute", importHandler.Execute)
}
```

### Step 9: rclone.conf 解析器

创建导入解析器：

```bash
touch internal/rclone/parser.go
touch internal/rclone/parser_test.go
```

```go
// internal/rclone/parser.go
package rclone

import (
    "bytes"
    "github.com/unknwon/goconfig"
)

type ParsedConnection struct {
    Name   string
    Type   string
    Config map[string]string
}

func ParseRcloneConf(content string) ([]ParsedConnection, error) {
    cfg, err := goconfig.LoadFromReader(bytes.NewReader([]byte(content)))
    if err != nil {
        return nil, err
    }

    var connections []ParsedConnection
    for _, section := range cfg.GetSectionList() {
        if section == "" || section == "DEFAULT" {
            continue
        }

        config := make(map[string]string)
        for _, key := range cfg.GetKeyList(section) {
            if value, err := cfg.GetValue(section, key); err == nil {
                config[key] = value
            }
        }

        connections = append(connections, ParsedConnection{
            Name:   section,
            Type:   config["type"],
            Config: config,
        })
    }

    return connections, nil
}
```

### Step 10: 前端更新

#### 10.1 类型定义

更新 `web/src/lib/types.ts`：

```typescript
export type LoadStatus = "loaded" | "loading" | "error";

export interface Connection {
  id: string;
  name: string;
  type: string;
  created_at: string;
  updated_at: string;
  load_status: LoadStatus;
  load_error?: string;
}

export interface ConnectionRequest {
  name: string;
  type: string;
  config: Record<string, string>;
}

export type ConnectionConfig = Record<string, string>;
```

#### 10.2 API 客户端

更新 `web/src/api/connections.ts`。

#### 10.3 导入向导组件

创建导入向导组件：

```bash
mkdir -p web/src/modules/connections/components/ImportWizard
touch web/src/modules/connections/components/ImportWizard/ImportWizard.tsx
touch web/src/modules/connections/components/ImportWizard/Step1Input.tsx
touch web/src/modules/connections/components/ImportWizard/Step2Preview.tsx
touch web/src/modules/connections/components/ImportWizard/Step3Confirm.tsx
```

### Step 11: i18n 翻译

添加翻译键到 `internal/i18n/locales/*.toml` 和 `web/project.inlang/messages/*.json`。

### Step 12: 应用配置

更新配置结构：

```go
// internal/core/config/config.go
type Config struct {
    // ... existing fields
    Security struct {
        EncryptionKey string `mapstructure:"encryption_key"`
    } `mapstructure:"security"`
}
```

```toml
# config.toml
[security]
encryption_key = ""  # 空 = plaintext模式，或使用环境变量 CLOUDSYNC_SECURITY_ENCRYPTION_KEY
```

**使用方式**:

```bash
# 加密模式（生产环境推荐）
export CLOUDSYNC_SECURITY_ENCRYPTION_KEY="my-secret-passphrase"

# Plaintext 模式（开发/测试环境）
# 不设置密钥或设置为空字符串
export CLOUDSYNC_SECURITY_ENCRYPTION_KEY=""
```

## 测试清单

### 单元测试

- [ ] `crypto_test.go` - 加密/解密功能
- [ ] `connection_service_test.go` - CRUD 操作
- [ ] `storage_test.go` - DBStorage 接口实现
- [ ] `cache_helper_test.go` - IsConnectionLoaded 缓存状态检查
- [ ] `parser_test.go` - rclone.conf 解析

### 集成测试

- [ ] API 端点测试
- [ ] 数据库迁移测试
- [ ] 令牌刷新同步测试（通过 Storage.SetValue）

### E2E 测试

- [ ] 创建连接流程
- [ ] 导入向导流程
- [ ] 删除确认流程

## 迁移说明

由于应用尚未正式发布，不需要自动迁移。用户可以通过以下方式迁移现有配置：

1. 打开 "导入连接" 对话框
2. 粘贴现有 `rclone.conf` 文件内容
3. 预览并确认导入

## 回滚策略

如果需要回滚：

1. 使用 `rclone config show > rclone.conf` 导出当前配置
2. 回滚代码到上一版本
3. 将导出的 `rclone.conf` 放回配置目录

## 相关文档

- [spec.md](./spec.md) - 功能规格说明
- [research.md](./research.md) - 技术研究
- [data-model.md](./data-model.md) - 数据模型设计
- [contracts/openapi.yaml](./contracts/openapi.yaml) - API 规格

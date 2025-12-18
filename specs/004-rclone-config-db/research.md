# Research: Rclone 连接配置数据库存储

**Feature**: 004-rclone-config-db  
**Date**: 2025-12-16  
**Status**: Complete

## Research Topics

### 1. rclone 配置系统架构分析

**发现**: rclone 配置系统通过 `config.Storage` 接口实现解耦，可以通过替换 Storage 实现来自定义配置存储

**技术调研**:

1. **全局函数指针** (`fs/config.go`)

```go
// 这些函数指针将配置实现与 fs 包解耦
var (
    ConfigFileGet = func(section, key string) (string, bool) { return "", false }
    ConfigFileSet = func(section, key, value string) (err error) { ... }
    ConfigFileHasSection = func(section string) bool { return false }
)
```

2. **configmap.Mapper 使用函数指针** (`fs/configmap.go`)

```go
// getConfigFile 实现 configmap.Getter，调用 ConfigFileGet
type getConfigFile string
func (section getConfigFile) Get(key string) (value string, ok bool) {
    value, ok = ConfigFileGet(string(section), key)
    return value, ok
}

// setConfigFile 实现 configmap.Setter，调用 ConfigFileSet
type setConfigFile string
func (section setConfigFile) Set(key, value string) {
    err := ConfigFileSet(string(section), key, value)
    // ...
}
```

3. **Storage 接口设置函数指针** (`fs/config/config.go`)

```go
func init() {
    // 将函数指针指向 Storage 实现的方法
    fs.ConfigFileGet = FileGetValue
    fs.ConfigFileSet = SetValueAndSave
    fs.ConfigFileHasSection = func(section string) bool {
        return LoadedData().HasSection(section)
    }
}

// 可以通过 SetData 替换 Storage 实现
func SetData(newData Storage) {
    data = newData
    dataLoaded = false
}
```

4. **Storage 接口定义**

```go
type Storage interface {
    GetSectionList() []string
    HasSection(section string) bool
    DeleteSection(section string)
    GetKeyList(section string) []string
    GetValue(section string, key string) (value string, found bool)
    SetValue(section string, key string, value string)
    DeleteKey(section string, key string) bool
    Load() error
    Save() error
    Serialize() (string, error)
}
```

---

### 2. 实现方案决策：Storage 接口 vs 函数指针

**Decision**: 实现 `config.Storage` 接口，通过 `config.SetData()` 替换默认存储

**对比分析**:

| 维度                | 替换函数指针 (3 个)    | 实现 Storage (10 个方法)         |
| ------------------- | ---------------------- | -------------------------------- |
| **实现工作量**      | 少（3 个函数）         | 多（10 个方法）                  |
| **官方性**          | 半官方（公开的扩展点） | 更官方（设计好的接口）           |
| **功能完整性**      | 仅覆盖读写和检查       | 完整（包括删除、列表、序列化等） |
| **rclone CLI 兼容** | 部分                   | 完整                             |
| **未来兼容性**      | 可能遗漏新增的依赖     | 自动兼容                         |

**Rationale**:

1. **功能更完整**: 需要 `GetSectionList()` 列出所有连接，`DeleteSection()` 删除连接
2. **更规范**: 这是 rclone 官方提供的扩展点
3. **自动处理令牌刷新**: rclone 刷新 OAuth 令牌时自动调用 `SetValue`
4. **工作量差异不大**: 大部分方法都是简单的数据库 CRUD

---

### 3. DBStorage 实现方案

**Decision**: 实现 DBStorage 结构体，依赖 ConnectionService 接口（遵循项目分层架构）

**架构设计**:

```go
// internal/rclone/storage.go
package rclone

import (
    "context"
    "encoding/json"

    "github.com/rclone/rclone/fs/config"
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
// 注意：不要调用 configfile.Install()
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
    cache.ClearConfig(section)  // 清除 rclone 缓存
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
    // 获取现有配置或创建新配置
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

    // 如果是 type 字段，也更新 Type
    connType := conn.Type
    if key == "type" {
        connType = value
    }
    _ = s.svc.UpdateConnection(ctx, conn.ID, conn.Name, connType, newConfig)
    cache.ClearConfig(section)  // 清除缓存以便重新加载
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
    // 返回 JSON 格式（用于 rclone config show --json 等命令）
    result := make(map[string]map[string]string)
    for _, c := range conns {
        result[c.Name] = c.Config
    }
    data, err := json.MarshalIndent(result, "", "  ")
    return string(data), err
}
```

**安装时机**:

```go
// cmd/cloud-sync/serve.go
func runServe(cmd *cobra.Command, args []string) error {
    // ...初始化数据库...

    // 创建 ConnectionService
    connSvc := services.NewConnectionService(db, encryptor)

    // 安装 DBStorage（替换默认的配置文件存储）
    // rclone 会通过 Storage 接口按需读取配置，无需预加载
    dbStorage := rclone.NewDBStorage(connSvc)
    dbStorage.Install()

    // ...启动 API server...
}
```

**优势**:

1. **零侵入现有代码**: 现有的 `fs.NewFs("remote:")` 调用无需修改
2. **自动令牌刷新**: rclone 刷新 OAuth 令牌时自动调用 `SetValue`，保存到数据库
3. **rclone CLI 兼容**: 如果集成 rclone CLI，配置命令会自动使用数据库

---

### 4. 敏感信息加密方案

**Decision**: 使用 AES-256-GCM 对整个配置 JSON 进行整体加密，密钥从环境变量或配置文件获取

**Rationale**:

- **整体加密**: 将整个 config JSON 序列化后加密存储，无需识别敏感字段
- **AES-256-GCM**: 提供认证加密（AEAD），同时保证机密性和完整性
- **实现简单**: 只需一对 encrypt/decrypt 函数，无需维护敏感字段列表
- **更安全**: 所有配置信息都被保护，包括可能泄露业务信息的非敏感字段

**数据模型**:

```go
type Connection struct {
    ID              uuid.UUID `json:"id"`
    Name            string    `json:"name"`
    Type            string    `json:"type"`           // provider 类型，明文（用于显示图标等）
    EncryptedConfig []byte    `json:"-"`              // AES-GCM 加密的完整配置 JSON
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

**加密流程**:

```
创建/更新连接:
  config map[string]string → JSON 序列化 → AES-GCM 加密 → 存储到 encrypted_config

读取连接:
  encrypted_config → AES-GCM 解密 → JSON 反序列化 → config map[string]string
```

**加密实现**:

```go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "io"
)

// Encryptor 处理配置的加密/解密，支持可选加密
type Encryptor struct {
    key       []byte
    plaintext bool // true 表示不加密（空密钥）
}

// NewEncryptor 创建加密器
// - 密钥为空: plaintext 模式（不加密，适合开发环境）
// - 密钥非空: 使用 SHA-256 转换为 32 字节后进行 AES-256-GCM 加密
func NewEncryptor(key string) (*Encryptor, error) {
    if key == "" {
        // 空密钥 = plaintext 模式
        return &Encryptor{plaintext: true}, nil
    }

    // 使用 SHA-256 将任意长度密钥转换为 32 字节
    hash := sha256.Sum256([]byte(key))
    return &Encryptor{key: hash[:], plaintext: false}, nil
}

// EncryptConfig 加密整个配置 map
// - Plaintext 模式: 返回 JSON
// - 加密模式: 返回 AES-GCM 加密后的数据
func (e *Encryptor) EncryptConfig(config map[string]string) ([]byte, error) {
    plaintext, err := json.Marshal(config)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal config: %w", err)
    }

    // Plaintext 模式直接返回 JSON
    if e.plaintext {
        return plaintext, nil
    }

    // 加密模式
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return nil, err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptConfig 解密整个配置
// - Plaintext 模式: 直接解析 JSON
// - 加密模式: AES-GCM 解密后解析 JSON
func (e *Encryptor) DecryptConfig(encrypted []byte) (map[string]string, error) {
    var plaintext []byte

    if e.plaintext {
        // Plaintext 模式，数据已经是 JSON
        plaintext = encrypted
    } else {
        // 加密模式，需要解密
        block, err := aes.NewCipher(e.key)
        if err != nil {
            return nil, err
        }
        gcm, err := cipher.NewGCM(block)
        if err != nil {
            return nil, err
        }
        nonceSize := gcm.NonceSize()
        if len(encrypted) < nonceSize {
            return nil, fmt.Errorf("ciphertext too short")
        }
        nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
        plaintext, err = gcm.Open(nil, nonce, ciphertext, nil)
        if err != nil {
            return nil, fmt.Errorf("failed to decrypt: %w", err)
        }
    }

    var config map[string]string
    if err := json.Unmarshal(plaintext, &config); err != nil {
        return nil, err
    }
    return config, nil
}
```

**密钥配置**:

```toml
# config.toml
[security]
encryption_key = ""  # 空 = plaintext模式，或从环境变量 CLOUDSYNC_SECURITY_ENCRYPTION_KEY 读取
```

**使用示例**:

```bash
# 加密模式（生产环境推荐）
export CLOUDSYNC_SECURITY_ENCRYPTION_KEY="my-secret-passphrase"

# Plaintext 模式（开发/测试环境）
# 不设置密钥或设置为空字符串
export CLOUDSYNC_SECURITY_ENCRYPTION_KEY=""
```

**API 响应策略**:

为了平衡安全性和可用性，将连接信息分为两个接口：

1. `GET /connections/{id}` - 返回连接基本信息（id, name, type 等），**不包含配置详情**
2. `GET /connections/{id}/config` - 返回解密后的完整配置（仅用于编辑场景）

---

### 5. rclone.conf 解析方案

**Decision**: 使用 rclone 依赖的 `github.com/unknwon/goconfig` 库解析配置

**Rationale**:

rclone 自身使用 `goconfig` 库解析 INI 格式的配置文件，我们直接复用这个库可以保证最大的兼容性。

rclone.conf 使用标准 INI 格式：

```ini
[remote_name]
type = onedrive
token = {"access_token":"...","refresh_token":"...","expiry":"2024-..."}
drive_id = abc123
drive_type = personal
```

**解析实现**:

```go
package rclone

import (
    "bytes"
    "github.com/unknwon/goconfig"
)

// ParsedConnection 表示从 rclone.conf 解析的连接
type ParsedConnection struct {
    Name   string
    Type   string
    Config map[string]string
}

// ParseRcloneConf 使用 rclone 依赖的 goconfig 库解析配置
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

---

### 6. 包装类型兼容性（alias, crypt, compress 等）

**Decision**: 无需特殊处理，rclone 通过 Storage 接口按需读取配置

**Rationale**:

rclone 有多种"包装类型"后端（alias, crypt, compress, combine, union 等），它们会引用其他 remote：

```ini
[my-crypt]
type = crypt
remote = my-gdrive:secret-folder
password = xxx

[my-alias]
type = alias
remote = my-gdrive:documents
```

使用 Storage 接口后，rclone 的 `fs.NewFs()` 会自动从数据库读取配置。当包装类型需要解析依赖的 remote 时：

1. 包装类型后端（如 crypt）调用 `cache.Get("my-gdrive:")`
2. `cache.Get` 内部调用 `fs.NewFs` 创建依赖的 fs.Fs
3. `fs.NewFs` 通过 `ConfigFileGet` → `DBStorage.GetValue` 从数据库读取配置

整个过程是按需的、递归的，无需启动时预加载。

**优势**:

1. **简化实现**: 无需维护预加载逻辑
2. **按需加载**: 只有使用到的连接才会被加载
3. **自动处理依赖**: rclone 内部会递归解析依赖

---

## Summary

| Topic            | Decision                    | Key Points                                       |
| ---------------- | --------------------------- | ------------------------------------------------ |
| 配置存储机制     | 实现 config.Storage 接口    | 通过 config.SetData() 替换默认存储               |
| DBStorage 依赖   | 依赖 ConnectionService      | 遵循项目分层架构，不直接使用 ent.Client          |
| 加密方案         | AES-256-GCM                 | 整体配置 JSON 加密，密钥从环境变量获取           |
| rclone.conf 解析 | 使用 rclone 依赖的 goconfig | 与 rclone 100% 兼容，支持验证和冲突检测          |
| 令牌刷新同步     | Storage.SetValue 自动处理   | rclone 刷新令牌时调用 SetValue，自动保存到数据库 |
| 包装类型兼容     | 无需特殊处理                | rclone 通过 Storage 接口按需读取，自动递归解析   |

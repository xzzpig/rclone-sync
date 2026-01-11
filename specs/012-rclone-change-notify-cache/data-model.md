# Data Model: ChangeNotify 缓存加速同步

**Feature Branch**: `012-rclone-change-notify-cache`  
**Created**: 2026-01-08

---

## Entity Changes

### 主数据库变更

Connection 表新增 `options` JSON 字段，用于存储连接级扩展配置（包括缓存配置）。

**Migration**: 新增 `options` 列（可选，JSON 类型）

| 表 | 字段 | 类型 | 默认值 | 说明 |
|----|------|------|--------|------|
| `connections` | `options` | `JSON` | `null` | 连接扩展选项（包含缓存配置） |

### 独立缓存数据库

每个启用缓存的连接有独立的 SQLite 缓存文件，存储在 `app_data/cache/<connection_id>.db`。

**特点**：
- 使用 WAL 模式支持并发读取
- Schema 版本管理 + 条件重建（缓存可丢弃，无需复杂迁移）
- 与主数据库完全隔离，损坏时不影响核心功能

---

## ConnectionOptions 结构

### 存储格式

参考 Task.Options 的设计模式，使用嵌套 JSON 结构：

```json
{
  "cache": {
    "enabled": true,
    "infoAge": "6h",
    "changeNotifyPoll": "1m"
  }
}
```

### Go 类型定义

```go
// internal/api/graphql/model/connection_options.go

// ConnectionOptions 连接扩展选项
type ConnectionOptions struct {
    Cache *ConnectionCacheOptions `json:"cache,omitempty"`
    // 未来可扩展其他选项，如：
    // Bandwidth *BandwidthOptions `json:"bandwidth,omitempty"`
    // Retry *RetryOptions `json:"retry,omitempty"`
}

// ConnectionCacheOptions 元数据缓存配置
type ConnectionCacheOptions struct {
    // 是否启用元数据缓存
    Enabled bool `json:"enabled"`
    
    // 缓存 TTL（Go duration 格式，如 "6h", "24h"）
    // 空字符串表示使用默认值（6小时），0 或负数表示永不过期
    InfoAge string `json:"infoAge,omitempty"`
    
    // ChangeNotify 轮询间隔（Go duration 格式，如 "1m", "5m"）
    // 空字符串表示使用默认值（1分钟）
    ChangeNotifyPoll string `json:"changeNotifyPoll,omitempty"`
}
```

### 字段说明

| 字段名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `cache.enabled` | `bool` | `false` | 是否启用元数据缓存 |
| `cache.infoAge` | `string` | `"6h"` | 缓存条目 TTL，空值使用默认值（6h），0 或负数表示永不过期 |
| `cache.changeNotifyPoll` | `string` | `"1m"` | ChangeNotify 轮询间隔，空值使用默认 |

---

## CacheEntry Schema（独立缓存数据库）

### 表结构

```sql
-- 元数据表（用于 schema 版本管理）
CREATE TABLE cache_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT INTO cache_meta (key, value) VALUES ('schema_version', '1');

-- 缓存条目表
CREATE TABLE cache_entries (
    -- 主键：文件/目录路径（相对于连接根）
    path TEXT PRIMARY KEY,
    
    -- 父目录路径（用于 Fs.List 高效查询）
    parent TEXT NOT NULL,             -- 父目录路径，根目录子项为 ""
    
    -- 核心元数据（rclone sync 比对时使用）
    mod_time INTEGER NOT NULL,        -- 修改时间（Unix timestamp，纳秒精度）
    is_dir BOOLEAN NOT NULL,          -- 是否为目录
    size INTEGER,                     -- 文件大小（字节），目录为 NULL
    hash TEXT,                        -- Hash 值（格式：算法:值，如 "md5:abc123"）
    
    -- 目录专属字段
    dir_loaded BOOLEAN DEFAULT FALSE, -- 子项是否已完整列举
    
    -- 缓存管理字段
    cached_at INTEGER NOT NULL        -- 缓存时间（Unix timestamp）
);

-- 索引
CREATE INDEX idx_parent ON cache_entries(parent);    -- 用于 Fs.List 查询
CREATE INDEX idx_cached_at ON cache_entries(cached_at);
CREATE INDEX idx_is_dir ON cache_entries(is_dir);
```

> **设计说明**：
> 1. `parent` 字段是实现 `Fs.List` 的关键，允许 O(1) 查询某目录的直接子项
> 2. 其他字段（ModTime、Size、Hash）用于 rclone sync/bisync 文件比对
> 3. 移除了不必要的字段（mime_type, dir_count, etag, version）

### 字段说明

| 字段 | 类型 | 必须 | 说明 |
|------|------|------|------|
| `path` | TEXT | ✅ | 主键，文件/目录路径（相对于连接根） |
| `parent` | TEXT | ✅ | 父目录路径，用于 List 查询。根目录子项的 parent 为 `""` |
| `mod_time` | INTEGER | ✅ | 修改时间（Unix timestamp 纳秒），用于变更检测 |
| `is_dir` | BOOLEAN | ✅ | 是否为目录 |
| `size` | INTEGER | 文件必须 | 文件大小（字节），目录为 NULL，用于变更检测 |
| `hash` | TEXT | 可选 | Hash 值（格式：`算法:值`），用于内容校验 |
| `dir_loaded` | BOOLEAN | 可选 | 目录子项是否已完整列举 |
| `cached_at` | INTEGER | ✅ | 缓存时间（Unix timestamp），用于 TTL 过期检查 |

### Fs.List 实现

```go
// List 返回目录的直接子项
func (f *Fs) List(ctx context.Context, dir string) (fs.DirEntries, error) {
    // 1. 检查目录是否已缓存且未过期
    dirEntry, err := f.cache.Get(f.cachePath(dir))
    if err == nil && dirEntry.DirLoaded && !dirEntry.IsExpired(f.opt.InfoAge) {
        // 2. 从缓存查询子项
        entries, err := f.cache.ListChildren(f.cachePath(dir))
        if err == nil {
            return f.toDirEntries(entries), nil
        }
    }
    
    // 3. 缓存未命中或过期，从远程获取
    entries, err := f.wrapped.List(ctx, dir)
    if err != nil {
        return nil, err
    }
    
    // 4. 更新缓存
    f.cache.SetDirEntries(f.cachePath(dir), entries)
    
    return entries, nil
}

// ListChildren 查询某目录的直接子项
func (s *CacheStore) ListChildren(parent string) ([]*CacheEntry, error) {
    rows, err := s.db.Query(`
        SELECT path, parent, mod_time, is_dir, size, hash, dir_loaded, cached_at
        FROM cache_entries 
        WHERE parent = ? AND cached_at > 0
    `, parent)
    // ... 处理结果
}
```

### 存储开销估算

| 文件数量 | 数据大小 | 索引大小 | 总计 |
|---------|---------|---------|------|
| 10,000 | ~1.6 MB | ~0.5 MB | ~2 MB |
| 100,000 | ~16 MB | ~5 MB | ~21 MB |
| 1,000,000 | ~160 MB | ~50 MB | ~210 MB |

---

## Schema 版本管理策略

### 核心思想

缓存数据可丢弃，采用简单的"版本号 + 条件重建"策略，无需复杂迁移脚本。

### 实现流程

```go
const CurrentCacheSchemaVersion = 1

func NewCacheStore(dbPath string) (*CacheStore, error) {
    // 1. 尝试打开数据库
    db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
    if err != nil {
        return nil, err
    }
    
    // 2. 检查 schema 版本
    version, err := getCacheSchemaVersion(db)
    if err != nil || version != CurrentCacheSchemaVersion {
        // 版本不匹配或无法读取，删除重建
        db.Close()
        os.Remove(dbPath)
        
        db, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
        if err != nil {
            return nil, err
        }
        
        if err := initCacheSchema(db); err != nil {
            db.Close()
            return nil, err
        }
    }
    
    return &CacheStore{db: db, dbPath: dbPath}, nil
}
```

### 版本升级流程

```
场景: 从 v1 升级到 v2（如添加新字段）

1. 更新 CurrentCacheSchemaVersion = 2
2. 更新 initCacheSchema() 包含新字段
3. 应用启动时自动检测版本不匹配
4. 删除旧数据库文件，创建新数据库
5. 缓存数据通过正常使用逐步重建
```

---

## GraphQL Schema Changes

详细定义请参考 [contracts/schema.graphql](./contracts/schema.graphql)

### 变更概览

| 类型 | 变更类型 | 说明 |
|------|----------|------|
| `Connection` | 扩展 | 新增 `options` 字段 |
| `ConnectionOptions` | 新增 | 连接扩展选项类型 |
| `ConnectionCacheOptions` | 新增 | 缓存配置类型 |
| `ConnectionCacheOptionsInput` | 新增 | 缓存配置输入类型 |
| `ConnectionCacheStatus` | 新增 | 缓存运行时状态类型 |
| `ConnectionMutation` | 扩展 | 新增 `clearCache` mutation |
| `CreateConnectionInput` | 扩展 | 新增 `options` 字段 |
| `UpdateConnectionInput` | 扩展 | 新增 `options` 字段 |

---

## Ent Schema Changes

### Connection Schema 扩展

```go
// internal/core/db/schema/connection.go

func (Connection) Fields() []ent.Field {
    return []ent.Field{
        // ... 现有字段
        
        // 新增 options 字段
        field.JSON("options", &model.ConnectionOptions{}).
            Optional().
            Comment("连接扩展选项（JSON，包含缓存配置等）"),
    }
}
```

---

## Validation Rules

| 字段 | 验证规则 |
|------|----------|
| `cache.enabled` | 布尔值，无额外验证 |
| `cache.infoAge` | 如果非空，必须是有效的 Go duration 格式（如 "6h", "24h", "30m"）。空值使用默认值（6h），0 或负数表示永不过期。 |
| `cache.changeNotifyPoll` | 如果非空，必须是有效的 Go duration 格式，且 >= 10s（防止过于频繁） |

### 验证实现

```go
func validateCacheOptions(opts *ConnectionCacheOptions) error {
    if opts == nil {
        return nil
    }
    
    if opts.InfoAge != "" {
        if _, err := time.ParseDuration(opts.InfoAge); err != nil {
            return fmt.Errorf("invalid infoAge format: %w", err)
        }
    }
    
    if opts.ChangeNotifyPoll != "" {
        d, err := time.ParseDuration(opts.ChangeNotifyPoll)
        if err != nil {
            return fmt.Errorf("invalid changeNotifyPoll format: %w", err)
        }
        if d < 10*time.Second {
            return fmt.Errorf("changeNotifyPoll must be at least 10s")
        }
    }
    
    return nil
}
```

---

## Data Flow

### 启用缓存时

```
[用户在 UI 启用缓存]
    ↓
[GraphQL Mutation: connection.update]
    ↓
[ConnectionService.Update() 验证配置]
    ↓
[更新 Connection.Options JSON]
    ↓
[保存到数据库]
    ↓
[PinConnection() 创建 MetaCache Fs]
    ↓
[MetaCache.NewFs() 初始化缓存存储]
    ↓
[订阅 ChangeNotify（如果后端支持）]
```

### 同步执行时

```
[读取 Task 关联的 Connection]
    ↓
[检查 Connection.Options.Cache.Enabled]
    ↓ (启用)
[使用 "connection-cache:" 获取 MetaCache Fs]
    ↓
[MetaCache.List() 检查缓存]
    ↓ (缓存命中且未过期)
[直接返回缓存数据]
    ↓ (缓存未命中或过期)
[调用 wrapped Fs 获取远程数据]
    ↓
[更新缓存]
    ↓
[返回数据]
```

### ChangeNotify 工作流

```
[ChangeNotify 收到变更通知]
    ↓
[receiveChangeNotify() 回调]
    ↓
[CacheStore.MarkStale(path)]
    ↓
[设置 cached_at = 0（标记为已过期）]
    ↓
[下次访问该路径时自动从远程刷新]
```

### 关闭缓存时

```
[用户在 UI 关闭缓存]
    ↓
[GraphQL Mutation: connection.update]
    ↓
[更新 Connection.Options.Cache.Enabled = false]
    ↓
[保存到数据库]
    ↓
[UnpinConnection() 释放 MetaCache Fs]
    ↓
[关闭 pollIntervalChan 停止 ChangeNotify]
    ↓
[释放 CacheStore 引用]
```

---

## Dependencies

| 组件 | 依赖 | 说明 |
|------|------|------|
| MetaCache Backend | `github.com/rclone/rclone/fs` | 实现 fs.Fs 接口 |
| MetaCache Backend | `github.com/rclone/rclone/fs/cache` | Fs 缓存和 Pin 机制 |
| CacheStore | `github.com/mattn/go-sqlite3` | 独立缓存数据库（已在项目中使用） |
| DBStorage | `internal/rclone/storage.go` | 透明注入虚拟连接配置 |
| ConnectionService | `internal/core/ports` | 连接 CRUD 服务 |

---

## Runtime Components

### 全局组件

| 组件 | 职责 | 生命周期 |
|------|------|----------|
| `cacheStores` | 管理共享的 CacheStore 实例（按 connection ID） | 应用生命周期 |
| `pinnedFsMap` | 管理 Pin 住的 MetaCache Fs 实例 | 应用生命周期 |

### 每连接组件

| 组件 | 职责 | 生命周期 |
|------|------|----------|
| `CacheStore` | 管理单个连接的缓存数据库 | 缓存启用期间 |
| `MetaCacheFs` | 包装远程 Fs，提供缓存层 | 缓存启用期间 |
| `pollIntervalChan` | 控制 ChangeNotify 轮询 | 缓存启用期间 |

---

## Migration Steps

### 主数据库迁移

1. 添加 `options` JSON 列到 `connections` 表
2. 默认值为 `NULL`（现有连接不受影响）

```sql
-- Migration: 20260108_add_connection_options.sql
ALTER TABLE connections ADD COLUMN options TEXT;
```

### 缓存数据库初始化

无需迁移，首次访问时自动创建。

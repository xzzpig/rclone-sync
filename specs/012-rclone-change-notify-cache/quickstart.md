# Quickstart: ChangeNotify 缓存加速同步

**Feature Branch**: `012-rclone-change-notify-cache`  
**Created**: 2026-01-08

---

## 概述

本功能通过 rclone 的 ChangeNotify 机制和元数据缓存优化大目录同步性能。对于支持变更通知的远程存储（如 OneDrive、Google Drive），可以将 10 万+ 文件目录的同步时间从 5 分钟降至 2 秒。

---

## 开发前准备

### 环境要求

- Go 1.21+
- 已安装项目依赖（`go mod download`）
- 一个支持 ChangeNotify 的远程存储账号（如 OneDrive）用于测试

### 相关文件位置

```
internal/
├── rclone/
│   ├── backend/
│   │   └── metacache/       # [新增] MetaCache 后端
│   │       ├── metacache.go     # 后端注册和 Fs 实现
│   │       ├── cache_store.go   # SQLite 缓存存储
│   │       └── cache_entry.go   # 缓存条目类型
│   ├── pin_manager.go       # [新增] Fs Pin 管理
│   └── storage.go           # [修改] DBStorage 透明注入
├── api/graphql/
│   ├── model/
│   │   └── connection_options.go  # [新增] ConnectionOptions 类型
│   └── schema/
│       └── connection.graphql     # [修改] Connection schema 扩展
├── core/
│   ├── ent/schema/
│   │   └── connection.go    # [修改] 添加 options 字段
│   └── ports/
│       └── interfaces.go    # [修改] ConnectionService 接口扩展
app_data/
└── cache/                   # 缓存数据库存储目录
    └── <connection_id>.db   # 每个连接的缓存文件
```

---

## 核心组件

### 1. CacheStore - 缓存存储

管理单个连接的 SQLite 缓存数据库。

```go
// internal/rclone/backend/metacache/cache_store.go

type CacheStore struct {
    db      *sql.DB
    dbPath  string
    inUse   atomic.Int32
}

// 核心方法
func NewCacheStore(dbPath string) (*CacheStore, error)
func (s *CacheStore) Get(path string) (*CacheEntry, error)
func (s *CacheStore) Set(path string, entry *CacheEntry) error
func (s *CacheStore) MarkStale(path string) error
func (s *CacheStore) Clear() (int, error)
func (s *CacheStore) Close() error
```

### 2. MetaCache Fs - 缓存包装器

实现 `fs.Fs` 接口，包装远程 Fs 提供缓存层。

```go
// internal/rclone/backend/metacache/metacache.go

type Fs struct {
    fs.Fs                          // 嵌入被包装的 Fs
    
    name             string
    root             string
    connectionID     string
    wrapped          fs.Fs
    cache            *CacheStore
    opt              Options
    features         *fs.Features
    pollIntervalChan chan time.Duration
}

// 关键接口实现
func (f *Fs) List(ctx context.Context, dir string) (fs.DirEntries, error)
func (f *Fs) NewObject(ctx context.Context, remote string) (fs.Object, error)
func (f *Fs) Shutdown(ctx context.Context) error
```

### 3. Pin Manager - Fs 生命周期管理

确保启用缓存的连接的 Fs 常驻内存。

```go
// internal/rclone/pin_manager.go

func PinConnection(conn *ent.Connection) error
func UnpinConnection(connID string)
func InitPinnedConnections(connSvc ports.ConnectionQuery)
```

---

## 开发步骤

### Step 1: 添加 Connection.options 字段

**位置**: `internal/core/db/schema/connection.go`（或 ent schema 目录）

```go
import (
    "github.com/xzzpig/rclone-sync/internal/api/graphql/model"
    // ...
)

func (Connection) Fields() []ent.Field {
    return []ent.Field{
        // 现有字段...
        
        field.JSON("options", &model.ConnectionOptions{}).
            Optional().
            Comment("连接扩展选项"),
    }
}
```

**生成代码**:
```bash
go generate ./internal/core/ent
```

**数据库迁移**:
```bash
./scripts/gen-migration.sh "add connection options"
```

### Step 2: 实现 CacheStore

**位置**: `internal/rclone/backend/metacache/cache_store.go`

```go
package metacache


import (
    "database/sql"
    "os"
    "path/filepath"
    "sync"
    "sync/atomic"
    "time"
    
    _ "github.com/mattn/go-sqlite3"
)

const CurrentCacheSchemaVersion = 1

var (
    cacheStoreMu sync.Mutex
    cacheStores  = map[string]*CacheStore{}
)

type CacheStore struct {
    db      *sql.DB
    dbPath  string
    inUse   atomic.Int32
}

type CacheEntry struct {
    Path      string
    Parent    string    // 父目录路径，用于 List 查询
    ModTime   time.Time
    IsDir     bool
    Size      int64
    Hash      string
    DirLoaded bool
    CachedAt  time.Time
}

// GetCacheStore 获取或创建缓存存储（单例模式）
func GetCacheStore(connectionID, dbPath string) (*CacheStore, error) {
    cacheStoreMu.Lock()
    defer cacheStoreMu.Unlock()
    
    if store, ok := cacheStores[connectionID]; ok {
        store.inUse.Add(1)
        return store, nil
    }
    
    store, err := NewCacheStore(dbPath)
    if err != nil {
        return nil, err
    }
    store.inUse.Store(1)
    cacheStores[connectionID] = store
    return store, nil
}

func NewCacheStore(dbPath string) (*CacheStore, error) {
    // 确保目录存在
    if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
        return nil, err
    }
    
    db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
    if err != nil {
        return nil, err
    }
    
    // 检查 schema 版本
    version, err := getCacheSchemaVersion(db)
    if err != nil || version != CurrentCacheSchemaVersion {
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

// 辅助函数定义
func getCacheSchemaVersion(db *sql.DB) (int, error) {
    var version int
    err := db.QueryRow("SELECT value FROM cache_meta WHERE key = 'schema_version'").Scan(&version)
    return version, err
}

func initCacheSchema(db *sql.DB) error {
    _, err := db.Exec(`
        CREATE TABLE cache_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
        INSERT INTO cache_meta (key, value) VALUES ('schema_version', '1');
        CREATE TABLE cache_entries (
            path TEXT PRIMARY KEY,
            parent TEXT NOT NULL,
            mod_time INTEGER NOT NULL,
            is_dir BOOLEAN NOT NULL,
            size INTEGER,
            hash TEXT,
            dir_loaded BOOLEAN DEFAULT FALSE,
            cached_at INTEGER NOT NULL
        );
        CREATE INDEX idx_parent ON cache_entries(parent);
    `)
    return err
}

// Get 获取缓存条目
func (s *CacheStore) Get(path string) (*CacheEntry, error) {
    var entry CacheEntry
    var modTimeNano, cachedAtUnix int64
    
    err := s.db.QueryRow(`
        SELECT path, mod_time, is_dir, size, hash, dir_loaded, cached_at
        FROM cache_entries WHERE path = ?
    `, path).Scan(
        &entry.Path, &modTimeNano, &entry.IsDir, &entry.Size, &entry.Hash,
        &entry.DirLoaded, &cachedAtUnix,
    )
    if err != nil {
        return nil, err
    }
    
    entry.ModTime = time.Unix(0, modTimeNano)
    entry.CachedAt = time.Unix(cachedAtUnix, 0)
    return &entry, nil
}

// MarkStale 标记条目为过期
func (s *CacheStore) MarkStale(path string) error {
    _, err := s.db.Exec(
        "UPDATE cache_entries SET cached_at = 0 WHERE path = ? OR path LIKE ?",
        path, path+"/%",
    )
    return err
}

// 其他方法...
```

### Step 3: 实现 MetaCache Backend

**位置**: `internal/rclone/backend/metacache/metacache.go`

```go
package metacache

import (
    "context"
    "fmt"
    "path"
    "sync"
    "time"
    
    "github.com/rclone/rclone/fs"
    "github.com/rclone/rclone/fs/cache"
    "github.com/rclone/rclone/fs/config/configmap"
    "github.com/rclone/rclone/fs/config/configstruct"
    "github.com/rclone/rclone/fs/fspath"
)

const (
    DefInfoAge          = 6 * time.Hour
    DefChangeNotifyPoll = time.Minute
)

// 注册后端
func init() {
    fs.Register(&fs.RegInfo{
        Name:        "metacache",
        Description: "Metadata cache wrapper for remote backends",
        NewFs:       NewFs,
        Options: []fs.Option{
            {Name: "remote", Help: "Remote to cache metadata for.", Required: true},
            {Name: "info_age", Help: "How long to cache file structure information.", Default: fs.Duration(DefInfoAge)},
            {Name: "change_notify_poll", Help: "ChangeNotify polling interval.", Default: fs.Duration(DefChangeNotifyPoll)},
            {Name: "db_path", Help: "Path to SQLite cache database.", Default: ""},
            {Name: "connection_id", Help: "Connection ID for cache sharing.", Default: ""},
        },
    })
}

type Options struct {
    Remote           string      `config:"remote"`
    InfoAge          fs.Duration `config:"info_age"`
    ChangeNotifyPoll fs.Duration `config:"change_notify_poll"`
    DbPath           string      `config:"db_path"`
    ConnectionID     string      `config:"connection_id"`
}

type Fs struct {
    fs.Fs
    name             string
    root             string
    connectionID     string
    wrapped          fs.Fs
    cache            *CacheStore
    opt              Options
    features         *fs.Features
    pollIntervalChan chan time.Duration
}

func NewFs(ctx context.Context, name, rootPath string, m configmap.Mapper) (fs.Fs, error) {
    opt := new(Options)
    if err := configstruct.Set(m, opt); err != nil {
        return nil, err
    }
    
    // 获取共享的缓存存储
    store, err := GetCacheStore(opt.ConnectionID, opt.DbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open cache store: %w", err)
    }
    
    // 获取被包装的 Fs
    remotePath := fspath.JoinRootPath(opt.Remote, rootPath)
    wrappedFs, err := cache.Get(ctx, remotePath)
    if err != nil {
        ReleaseCacheStore(opt.ConnectionID)
        return nil, fmt.Errorf("failed to get remote %q: %w", remotePath, err)
    }
    
    f := &Fs{
        Fs:               wrappedFs,
        name:             name,
        root:             rootPath,
        connectionID:     opt.ConnectionID,
        wrapped:          wrappedFs,
        cache:            store,
        opt:              *opt,
        pollIntervalChan: make(chan time.Duration, 1),
    }
    
    f.features = (&fs.Features{
        CanHaveEmptyDirectories: true,
    }).Fill(ctx, f).Mask(ctx, wrappedFs).WrapsFs(f, wrappedFs)
    
    // 订阅 ChangeNotify
    if doChangeNotify := wrappedFs.Features().ChangeNotify; doChangeNotify != nil {
        f.pollIntervalChan <- time.Duration(opt.ChangeNotifyPoll)
        doChangeNotify(ctx, f.receiveChangeNotify, f.pollIntervalChan)
    }
    
    return f, nil
}

func (f *Fs) receiveChangeNotify(pathStr string, entryType fs.EntryType) {
    fullPath := path.Join(f.root, pathStr)
    _ = f.cache.MarkStale(fullPath)
}

func (f *Fs) Shutdown(ctx context.Context) error {
    close(f.pollIntervalChan)
    ReleaseCacheStore(f.connectionID)
    return nil
}

// Name, Root, Features, UnWrap 等接口实现...
```

### Step 4: 修改 DBStorage 透明注入

**位置**: `internal/rclone/storage.go`

```go
const CacheSuffix = "-cache"

type DBStorage struct {
    svc     ports.ConnectionQuery
    dataDir string // 基础数据目录
    mu      sync.RWMutex
}

func NewDBStorage(svc ports.ConnectionQuery, dataDir string) *DBStorage {
    return &DBStorage{
        svc:     svc,
        dataDir: dataDir,
    }
}

func (s *DBStorage) HasSection(section string) bool {
    if strings.HasSuffix(section, CacheSuffix) {
        realName := strings.TrimSuffix(section, CacheSuffix)
        ctx := context.Background()
        conn, err := s.svc.GetConnectionByName(ctx, realName)
        return err == nil && conn.Options != nil && 
               conn.Options.Cache != nil && conn.Options.Cache.Enabled
    }
    // 原逻辑...
}

func (s *DBStorage) GetValue(section, key string) (string, bool) {
    if strings.HasSuffix(section, CacheSuffix) {
        return s.getCacheValue(section, key)
    }
    // 原逻辑...
}

func (s *DBStorage) getCacheValue(section, key string) (string, bool) {
    realName := strings.TrimSuffix(section, CacheSuffix)
    
    ctx := context.Background()
    conn, err := s.svc.GetConnectionByName(ctx, realName)
    if err != nil || conn.Options == nil || conn.Options.Cache == nil || !conn.Options.Cache.Enabled {
        return "", false
    }
    
    cache := conn.Options.Cache
    switch key {
    case "type":
        return "metacache", true
    case "remote":
        return realName + ":", true
    case "info_age":
        if cache.InfoAge != "" {
            return cache.InfoAge, true
        }
        return "", false
    case "change_notify_poll":
        if cache.ChangeNotifyPoll != "" {
            return cache.ChangeNotifyPoll, true
        }
        return "", false
    case "db_path":
        return filepath.Join(s.dataDir, "cache", conn.ID.String()+".db"), true
    case "connection_id":
        return conn.ID.String(), true
    default:
        return "", false
    }
}
```

### Step 5: 更新 GraphQL Schema

**位置**: `internal/api/graphql/schema/connection.graphql`

参考 `specs/012-rclone-change-notify-cache/contracts/schema.graphql` 添加新类型和字段。

---

## 测试

### 单元测试

```go
// internal/rclone/backend/metacache/cache_store_test.go

func TestCacheStore_GetSet(t *testing.T) {
    tmpDir := t.TempDir()
    dbPath := filepath.Join(tmpDir, "test.db")
    
    store, err := NewCacheStore(dbPath)
    require.NoError(t, err)
    defer store.Close()
    
    entry := &CacheEntry{
        Path:     "/test/file.txt",
        ModTime:  time.Now(),
        IsDir:    false,
        Size:     1024,
        CachedAt: time.Now(),
    }
    
    err = store.Set(entry.Path, entry)
    require.NoError(t, err)
    
    got, err := store.Get(entry.Path)
    require.NoError(t, err)
    assert.Equal(t, entry.Path, got.Path)
    assert.Equal(t, entry.Size, got.Size)
}

func TestCacheStore_MarkStale(t *testing.T) {
    // ...
}
```

### 集成测试

```go
// internal/rclone/backend/metacache/integration_test.go

func TestMetaCache_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // 需要真实的 OneDrive 连接进行测试
    // ...
}
```

### 手动测试步骤

1. 创建 OneDrive 连接
2. 在 UI 中启用缓存
3. 上传 1000+ 文件到远程目录
4. 创建同步任务并执行
5. 在 Web 界面修改一个文件
6. 再次执行同步，验证仅同步变更的文件
7. 关闭缓存，验证后台服务停止

---

## 常见问题

### Q: 缓存数据库损坏怎么办？

A: 删除 `app_data/cache/<connection_id>.db` 文件，下次访问时自动重建。

### Q: 如何验证 ChangeNotify 正在工作？

A: 查看 `Connection.cacheStatus.lastNotifyTime` 字段，或查看应用日志中的 `ChangeNotify` 相关记录。

### Q: 为什么某个连接不能启用缓存？

A: 可能是该存储后端不支持 ChangeNotify。可以通过 `rclone backend features <remote>:` 查看是否支持。

---

## 参考资料

- [research.md](./research.md) - 详细技术研究
- [data-model.md](./data-model.md) - 数据模型设计
- [contracts/schema.graphql](./contracts/schema.graphql) - GraphQL API 定义
- [rclone ChangeNotify 接口](https://github.com/rclone/rclone/blob/master/fs/features.go)
- [rclone cache backend](https://github.com/rclone/rclone/tree/master/backend/cache)（已废弃，仅参考）

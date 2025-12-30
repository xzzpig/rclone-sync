# Research: Rclone Fs Cache API

**Feature**: 010-rclone-fs-cache  
**Date**: 2025-12-30  
**Status**: Complete

## Research Tasks

### 1. rclone cache 包 API 研究

**任务**: 研究 rclone/fs/cache 包的 API 及使用方式

**发现**:

rclone 的 `fs/cache` 包提供了以下主要函数：

| 函数 | 签名 | 描述 |
|------|------|------|
| `cache.Get` | `func Get(ctx context.Context, remote string) (fs.Fs, error)` | 获取缓存的 Fs，若不存在则创建并缓存 |
| `cache.GetArr` | `func GetArr(ctx context.Context, remotes []string) ([]fs.Fs, error)` | 批量获取多个 Fs |
| `cache.Put` | `func Put(remote string, f fs.Fs)` | 手动将 Fs 放入缓存 |
| `cache.Clear` | `func Clear()` | 清除所有 Fs 缓存（无参数） |
| `cache.ClearConfig` | `func ClearConfig(name string) (deleted int)` | 清除以 `name:` 为前缀的 Fs 缓存，返回删除条目数 |
| `cache.Entries` | `func Entries() int` | 返回缓存条目数 |

**cache.Get vs fs.NewFs 对比**:

| 特性 | `fs.NewFs` | `cache.Get` |
|------|-----------|-------------|
| 缓存复用 | ❌ 每次创建新实例 | ✅ 复用已缓存实例 |
| 线程安全 | ✅ | ✅ |
| 自动 Put | N/A | ✅ 创建后自动入缓存 |
| 错误处理 | 返回 error | 返回 error |

**Decision**: 使用 `cache.Get` 替换 `fs.NewFs`，因为它自动处理缓存的获取和存储。

**Rationale**: 
- `cache.Get` 在 Fs 不存在时会调用 `fs.NewFs` 创建，然后自动 `cache.Put`
- 后续调用直接返回缓存实例，避免重复创建开销
- 项目中已有使用 `cache.Get` 的先例（`cache_helper.go`、`testutil/slowfs.go`）

**Alternatives considered**:
- 手动维护缓存 map：增加复杂度，且 rclone 已提供完善的缓存机制
- 使用 `fs.NewFs` + `cache.Put`：等价于 `cache.Get`，但更繁琐

---

### 2. 本地路径缓存策略

**任务**: 确定如何区分本地路径和远程路径

**发现**:

根据规范要求，需要区分两种情况：
1. **直接本地路径**（如 `/aaa/bbb`）：不使用缓存，始终用 `fs.NewFs`
2. **配置的 remote**（如 `myremote:path` 或 `local-remote:path`）：使用缓存

**实现策略**:

采用三参数设计 `GetFs(ctx, remote, path)`，通过 `remote` 参数显式区分：

```go
// GetFs(ctx, remote, path) 参数说明：
// - remote 为空字符串 → 本地路径，使用 fs.NewFs 不缓存
// - remote 非空 → 远程存储，使用 cache.Get 缓存

// 示例调用：
GetFs(ctx, "", "/home/user/data")           // 本地路径
GetFs(ctx, "myremote", "path/to/folder")    // 远程路径
```

**Decision**: 使用三参数设计，由调用方显式指定 remote 名称，避免解析路径字符串

**Rationale**: 
- 本地文件系统访问开销极低，无需缓存优化
- 规范明确要求"通过直接路径创建的本地 Fs 始终使用 fs.NewFs"
- 调用方在调用时已经知道是操作本地还是远程，无需解析路径
- 避免本地路径占用缓存空间

**Alternatives considered**:
- 通过解析路径字符串判断（如检查是否包含 `:`）：增加复杂度，且在 Windows 上有歧义
- 统一使用 cache.Get：违反规范要求

---

### 3. 缓存失效时机

**任务**: 确定何时需要主动失效缓存

**发现**:

需要失效缓存的场景：

| 场景 | 当前实现 | 需要的变更 |
|------|----------|-----------|
| 配置更新 (`SetValue`) | ✅ 已调用 `cache.ClearConfig` | 需添加 `cache.Clear` |
| 配置删除 (`DeleteSection`) | ✅ 已调用 `cache.ClearConfig` | 需添加 `cache.Clear` |
| 键删除 (`DeleteKey`) | ✅ 已调用 `cache.ClearConfig` | 需添加 `cache.Clear` |
| GraphQL Update mutation | ❌ 未处理 Fs 缓存 | 需添加 `cache.Clear` |
| GraphQL Delete mutation | ❌ 未处理 Fs 缓存 | 需添加 `cache.Clear` |

**cache.ClearConfig 说明**:
- `cache.ClearConfig(name)`: 清除以 `name:` 为前缀的 Fs 缓存条目，返回删除的条目数
- `cache.Clear()`: 清除所有缓存（无参数，不适用于清除特定 remote）

当前 `storage.go` 已调用了 `ClearConfig`，这足以清除特定 remote 的缓存。

**Decision**: 在 `storage.go` 的 `SetValue`、`DeleteSection`、`DeleteKey` 中保持使用 `cache.ClearConfig`；在 resolver 的 Update 和 Delete mutation 中添加 `cache.ClearConfig` 调用。

**Rationale**: 确保配置变更后，后续请求获取到新创建的 Fs 实例

**Alternatives considered**:
- 仅在 resolver 层处理：不够全面，storage.go 也可能被 rclone token refresh 等场景调用

---

### 4. 同步操作的缓存策略

**任务**: 确定同步操作中 Fs 的缓存使用方式

**发现**:

当前 `sync.go` 中的 `RunTask` 函数：
```go
// Source Fs (本地路径)
fSrc, err := fs.NewFs(statsCtx, task.SourcePath)

// Destination Fs (远程)
f2Path := fmt.Sprintf("%s:%s", connectionName, task.RemotePath)
fDst, err := fs.NewFs(statsCtx, f2Path)
```

**分析**:
1. `task.SourcePath` 可能是直接本地路径（如 `/home/user/sync`），需要判断
2. `f2Path` 始终是 `remote:path` 格式，应使用 `cache.Get`

**Decision**: 
- 对 `task.SourcePath` 进行路径判断，直接本地路径用 `fs.NewFs`，remote 格式用 `cache.Get`
- 对远程端 `f2Path` 始终使用 `cache.Get`

**Rationale**: 符合规范要求，同时确保远程连接可复用

**Alternatives considered**:
- 始终使用 fs.NewFs：无法享受缓存优化

---

### 5. 线程安全性

**任务**: 确认 rclone cache 的线程安全性

**发现**:

查看 rclone 源码 `fs/cache/cache.go`:
```go
var (
    cacheMu sync.Mutex
    cache   = map[string]*cacheEntry{}
    // ...
)
```

rclone cache 包使用 `sync.Mutex` 保护缓存 map，所有公开方法都在内部加锁，确保线程安全。

**Decision**: 依赖 rclone 内置的线程安全保证，无需在应用层添加额外同步

**Rationale**: 规范已明确"依赖 rclone 内置的线程安全保证"

**Alternatives considered**:
- 添加应用层锁：过度设计，增加复杂度

---

## Summary

所有 NEEDS CLARIFICATION 已解决：

1. ✅ **cache.Get API**: 直接替换 `fs.NewFs`，自动处理缓存获取和存储
2. ✅ **本地路径判断**: 通过路径格式判断，直接本地路径不缓存
3. ✅ **缓存失效**: 在 `storage.go` 和 resolver 层添加 `cache.Clear` 调用
4. ✅ **同步操作**: 远程端使用缓存，本地直接路径不缓存
5. ✅ **线程安全**: 依赖 rclone 内置机制

## Implementation Notes

### 创建辅助函数

建议在 `internal/rclone` 包中创建统一的 Fs 获取函数：

```go
// GetFs returns a cached Fs for remote paths, or creates a new Fs for direct local paths.
// Parameters:
//   - ctx: context for the operation
//   - remote: the remote name (e.g., "myremote") or empty string for local paths
//   - path: the path within the remote or local filesystem
//
// When remote is empty, path is treated as a direct local filesystem path and
// fs.NewFs is used without caching (per FR-009).
// When remote is non-empty, cache.Get is used to reuse existing Fs instances.
func GetFs(ctx context.Context, remote string, path string) (fs.Fs, error) {
    if remote == "" {
        // Direct local path - no caching
        return fs.NewFs(ctx, path)
    }
    // Remote path - use cache
    fsPath := remote + ":" + path
    return cache.Get(ctx, fsPath)
}

// ClearFsCache clears the Fs cache for the given remote name.
// This should be called when a remote configuration is updated or deleted.
// The remoteName should be just the name, without the colon (e.g., "myremote" not "myremote:").
func ClearFsCache(remoteName string) {
    if remoteName == "" {
        return
    }
    cache.ClearConfig(remoteName)
}
```

### 修改点清单

1. **internal/rclone/remote.go** - `ListRemoteDir`:
   - 将 `fs.NewFs(ctx, fsPath)` 替换为使用 `GetFs` 和 `BasePath` 缓存策略
   - **缓存策略优化**: 当 `opts.BasePath` 设置时，使用 `remote:BasePath` 作为缓存键（而非 `remote:Path`）
   - 这样浏览同一任务下的不同子目录时可以复用同一个 Fs 实例
   - 通过 `f.List(ctx, relativePath)` 访问子目录内容

2. **internal/rclone/about.go** - `GetRemoteQuota`:
   - 将 `fs.NewFs(ctx, remoteName+":")` 替换为 `GetFs(ctx, remoteName, "")`

3. **internal/rclone/sync.go** - `RunTask`:
   - 源路径：使用 `GetFs(ctx, "", task.SourcePath)`（本地路径，remote 为空）
   - 目标路径：使用 `GetFs(ctx, connectionName, task.RemotePath)`

4. **internal/rclone/storage.go**:
   - 已使用 `cache.ClearConfig(section)` 清除 Fs 缓存，无需额外修改

5. **internal/api/graphql/resolver/connection.resolvers.go**:
   - 在 `Update` mutation 中添加 `rclone.ClearFsCache(oldName)` 调用
   - 在 `Delete` mutation 中添加 `rclone.ClearFsCache(connName)` 调用

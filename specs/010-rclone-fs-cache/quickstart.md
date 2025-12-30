# Quickstart: Rclone Fs Cache Optimization

**Feature**: 010-rclone-fs-cache  
**Date**: 2025-12-30  
**Prerequisite**: research.md completed

## Implementation Overview

本功能的实现分为 4 个主要步骤：

1. 创建 Fs 获取辅助函数
2. 修改现有代码使用缓存
3. 添加缓存失效逻辑
4. 更新测试

## Step 1: 创建 Fs 获取辅助函数

在 `internal/rclone/cache_helper.go`（新文件）中创建统一的 Fs 获取函数：

```go
package rclone

import (
    "context"

    "github.com/rclone/rclone/fs"
    "github.com/rclone/rclone/fs/cache"
)

// GetFs returns a cached Fs for remote paths, or creates a new Fs for direct local paths.
// Parameters:
//   - ctx: context for the operation
//   - remote: the remote name (e.g., "myremote") or empty string for local paths
//   - path: the path within the remote or local filesystem
//
// When remote is empty, path is treated as a direct local filesystem path and
// fs.NewFs is used without caching (per FR-009).
// When remote is non-empty, cache.Get is used to reuse existing Fs instances.
//
// This follows FR-009: Direct local paths should always use fs.NewFs, not caching.
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
    // ClearConfig clears all cache entries with the prefix "remoteName:"
    cache.ClearConfig(remoteName)
}
```

## Step 2: 修改现有代码使用缓存

### 2.1 修改 remote.go - ListRemoteDir

**缓存策略优化**: 当 `opts.BasePath` 设置时，使用 `remote:BasePath` 作为缓存键（而非 `remote:Path`）。
这样浏览同一任务下的不同子目录时可以复用同一个 Fs 实例。

```go
// Before:
f, err := fs.NewFs(ctx, fsPath)
entries, err := f.List(ctx, "")

// After:
// 确定 Fs 根路径：优先使用 BasePath，否则使用 Path
fsRootPath := opts.BasePath
if fsRootPath == "" {
    fsRootPath = opts.Path
}

// 计算相对路径：opts.Path 相对于 fsRootPath
listPath := ""
if opts.BasePath != "" && opts.Path != opts.BasePath {
    basePath := strings.TrimSuffix(opts.BasePath, "/")
    currentPath := strings.TrimSuffix(opts.Path, "/")
    if strings.HasPrefix(currentPath, basePath+"/") {
        listPath = strings.TrimPrefix(currentPath, basePath+"/")
    } else if currentPath != basePath {
        // opts.Path 不在 opts.BasePath 下，回退到使用 opts.Path
        fsRootPath = opts.Path
        listPath = ""
    }
}

f, err := GetFs(ctx, opts.RemoteName, fsRootPath)
entries, err := f.List(ctx, listPath)

// entry.Remote() 返回相对于 Fs 根（BasePath）的完整路径
// 例如：列出 "subdir" 时，entry.Remote() 返回 "subdir/file.txt"
entryRemote := entry.Remote()

// 提取文件名用于显示
entryName := entryRemote
if lastSlash := strings.LastIndex(entryRemote, "/"); lastSlash >= 0 {
    entryName = entryRemote[lastSlash+1:]
}

// 过滤时直接使用 entryRemote，因为它已经是相对于 BasePath 的路径
if fi != nil && !fi.IncludeRemote(entryRemote) {
    continue
}
```

完整修改位置：`internal/rclone/remote.go` 的 `ListRemoteDir` 函数

**关键点**:
- 使用 `BasePath`（如果设置）作为 Fs 缓存键
- 通过 `f.List(ctx, relativePath)` 访问子目录
- `entry.Remote()` 返回相对于 Fs 根（BasePath）的完整路径，可直接用于过滤匹配
- 需要提取最后一个路径段作为文件名用于显示

### 2.2 修改 about.go - GetRemoteQuota

```go
// Before:
f, err := fs.NewFs(ctx, remoteName+":")

// After:
f, err := GetFs(ctx, remoteName, "")
```

完整修改位置：`internal/rclone/about.go` 的 `GetRemoteQuota` 函数

### 2.3 修改 sync.go - RunTask

```go
// Before (source - local path):
fSrc, err := fs.NewFs(statsCtx, task.SourcePath)

// After (source - local path, remote 参数为空):
fSrc, err := GetFs(statsCtx, "", task.SourcePath)

// Before (destination - remote):
f2Path := fmt.Sprintf("%s:%s", connectionName, task.RemotePath)
fDst, err := fs.NewFs(statsCtx, f2Path)

// After (destination - remote, 使用 GetFs):
fDst, err := GetFs(statsCtx, connectionName, task.RemotePath)
```

完整修改位置：`internal/rclone/sync.go` 的 `RunTask` 函数

注意：可以移除 `f2Path` 变量的构建，因为 GetFs 会在内部处理路径拼接。

## Step 3: 添加缓存失效逻辑

### 3.1 修改 storage.go

在以下三个方法中添加 `cache.Clear` 调用：

#### SetValue 方法

```go
func (s *DBStorage) SetValue(section, key, value string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    // ... existing code ...
    
    // Clear cache so rclone reloads the config
    // ClearConfig clears both config cache and Fs cache entries with prefix "section:"
    cache.ClearConfig(section)
}
```

#### DeleteSection 方法

```go
func (s *DBStorage) DeleteSection(section string) {
    s.mu.Lock()
    defer s.mu.Unlock()

    ctx := context.Background()
    _ = s.svc.DeleteConnectionByName(ctx, section)

    // Clear rclone cache for this remote
    // ClearConfig clears both config cache and Fs cache entries with prefix "section:"
    cache.ClearConfig(section)
}
```

#### DeleteKey 方法

```go
func (s *DBStorage) DeleteKey(section, key string) bool {
    // ... existing code ...
    
    // Clear cache
    // ClearConfig clears both config cache and Fs cache entries with prefix "section:"
    cache.ClearConfig(section)

    return true
}
```

### 3.2 修改 connection.resolvers.go

#### Update mutation

```go
func (r *connectionMutationResolver) Update(ctx context.Context, obj *model.ConnectionMutation, id uuid.UUID, input model.UpdateConnectionInput) (*model.Connection, error) {
    // Get old connection name before update (for cache invalidation)
    oldConn, err := r.deps.ConnectionService.GetConnectionByID(ctx, id)
    if err != nil {
        return nil, err
    }
    oldName := oldConn.Name
    
    err = r.deps.ConnectionService.UpdateConnection(ctx, id, input.Name, nil, input.Config)
    if err != nil {
        return nil, err
    }

    // Clear Fs cache for old name (in case name changed)
    rclone.ClearFsCache(oldName)
    // If name changed, also clear new name (though it shouldn't exist yet)
    if input.Name != nil && *input.Name != oldName {
        rclone.ClearFsCache(*input.Name)
    }

    // Fetch updated connection
    entConn, err := r.deps.ConnectionService.GetConnectionByID(ctx, id)
    if err != nil {
        return nil, err
    }
    return entConnectionToModel(entConn), nil
}
```

#### Delete mutation

```go
func (r *connectionMutationResolver) Delete(ctx context.Context, obj *model.ConnectionMutation, id uuid.UUID) (*model.Connection, error) {
    // Get connection before delete to return it and get name
    entConn, err := r.deps.ConnectionService.GetConnectionByID(ctx, id)
    if err != nil {
        return nil, err
    }
    conn := entConnectionToModel(entConn)
    connName := entConn.Name  // Save name for cache invalidation

    // Check if connection has dependent tasks
    // ... existing validation code ...

    err = r.deps.ConnectionService.DeleteConnectionByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Clear Fs cache for deleted connection
    rclone.ClearFsCache(connName)

    return conn, nil
}
```

## Step 4: 更新测试

### 4.1 创建新测试文件 cache_test.go

```go
package rclone

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestGetFs_LocalPath(t *testing.T) {
    // Test that GetFs with empty remote uses fs.NewFs (local path)
    ctx := context.Background()
    tempDir := t.TempDir()
    
    f, err := GetFs(ctx, "", tempDir)
    assert.NoError(t, err)
    assert.NotNil(t, f)
    assert.Equal(t, tempDir, f.Root())
}

func TestGetFs_RemotePath(t *testing.T) {
    // Test that GetFs with non-empty remote uses cache.Get
    // Note: This test requires a configured remote, so it's more suitable
    // for integration tests. Here we just verify the function doesn't panic.
    // A proper test would use a mock or test fixture.
}

func TestClearFsCache(t *testing.T) {
    // This test verifies ClearFsCache doesn't panic with various inputs
    tests := []string{
        "",
        "myremote",
        "another-remote",
        "local-storage",
    }

    for _, name := range tests {
        t.Run(name, func(t *testing.T) {
            // Should not panic
            ClearFsCache(name)
        })
    }
}
```

### 4.2 更新现有测试

确保以下测试文件仍然通过：
- `internal/rclone/remote_test.go`
- `internal/rclone/about_test.go`  
- `internal/rclone/sync_test.go`
- `internal/rclone/storage_test.go`
- `internal/api/graphql/resolver/connection_test.go`

## Verification Checklist

- [ ] 创建 `internal/rclone/cache.go` 文件（包含 `GetFs(ctx, remote, path)` 和 `ClearFsCache(remoteName)`）
- [ ] 创建 `internal/rclone/cache_test.go` 文件
- [ ] 修改 `internal/rclone/remote.go` - 使用 `GetFs(ctx, opts.RemoteName, opts.Path)`
- [ ] 修改 `internal/rclone/about.go` - 使用 `GetFs(ctx, remoteName, "")`
- [ ] 修改 `internal/rclone/sync.go` - 源路径使用 `GetFs(ctx, "", path)`，目标使用 `GetFs(ctx, remote, path)`
- [ ] 验证 `internal/rclone/storage.go` - 已使用 `cache.ClearConfig`，无需额外修改
- [ ] 修改 `internal/api/graphql/resolver/connection.resolvers.go` - 添加 `ClearFsCache` 调用
- [ ] 运行 `go test ./internal/rclone/...` 验证测试通过
- [ ] 运行 `go test ./internal/api/graphql/resolver/...` 验证测试通过
- [ ] 手动测试：浏览远程目录，验证重复访问响应更快

## Notes

### GetFs 参数设计说明

`GetFs(ctx, remote, path)` 使用三个参数的设计：
- `remote` 为空字符串时，表示本地路径，直接使用 `fs.NewFs(ctx, path)` 不缓存
- `remote` 非空时，表示远程存储，使用 `cache.Get(ctx, remote+":"+path)` 缓存

这种设计的优点：
1. 调用方明确知道自己在操作本地还是远程
2. 不需要解析路径字符串来判断类型
3. 符合 FR-009 规范：直接本地路径不缓存

### sync.go 中源路径的处理

`task.SourcePath` 目前始终是本地路径（如 `/home/user/sync`），所以调用 `GetFs(ctx, "", task.SourcePath)` 时 remote 参数为空。

如果将来支持远程到远程的同步，需要同时提供源远程名和路径。

### 边界情况处理

1. **空的 remoteName**: `ClearFsCache` 会直接返回，不做任何操作
2. **空的 path**: `GetFs(ctx, "myremote", "")` 会正确处理，生成 `"myremote:"` 路径

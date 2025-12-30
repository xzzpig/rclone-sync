# Research: Task 扩展选项配置

**Feature Branch**: `009-task-extended-options`  
**Created**: 2025-12-28

---

## 1. rclone Filter 语法与实现

### 问题

如何实现文件过滤器功能？rclone filter 语法如何在 Go 代码中使用？

### 研究

查看 rclone 源码 `fs/filter/filter.go`：

```go
// Filter 结构体
type Filter struct {
    // ... 私有字段
}

// NewFilter 创建一个新的 Filter
// opt 为 nil 时使用默认选项
func NewFilter(opt *Options) (*Filter, error)

// AddRule 添加过滤规则
// 规则格式: "+ pattern" 或 "- pattern"
func (f *Filter) AddRule(rule string) error

// Include 检查文件是否应该被包含
func (f *Filter) Include(remote string, size int64, modTime time.Time, metadata fs.Metadata) bool
```

rclone filter 语法：
- `+` 前缀表示 include（包含）
- `-` 前缀表示 exclude（排除）
- 规则按顺序匹配，第一个匹配的规则生效
- 支持通配符：`*`（单层）、`**`（多层）、`?`（单字符）

示例规则：
```
- node_modules/**     # 排除 node_modules 目录
- .git/**             # 排除 .git 目录
- *.tmp               # 排除所有 .tmp 文件
+ **                  # 包含其他所有文件
```

### 在 sync/bisync 中应用 Filter

查看 `fs/sync/sync.go`：

```go
// CopyDir 函数接受 filter 选项
// rclone 使用 filter.GetConfig(ctx) 获取当前 context 中的 filter 配置
```

rclone 的 filter 是通过全局配置设置的：

```go
// 在运行 sync 前设置 filter
import "github.com/rclone/rclone/fs/filter"

// 方式1：通过 --filter 命令行选项（我们不使用这种方式）
// 方式2：通过代码设置

// 创建 filter 并添加规则
fi, err := filter.NewFilter(nil)
if err != nil {
    return err
}

// 添加规则
for _, rule := range filterRules {
    err = fi.AddRule(rule)  // 如 "- node_modules/**" 或 "+ **"
    if err != nil {
        return err
    }
}

// 将 filter 注入到 context 中
ctx = filter.ReplaceConfig(ctx, fi)
```

### 决策

**使用 rclone filter 包实现过滤**：
1. 在 `TaskSyncOptions` 中添加 `filters` 字段（字符串数组，每个元素为一条规则）
2. 在 `runOneWay` 和 `runBidirectional` 执行前，解析 filter 规则并注入到 context
3. 前端使用可视化规则列表 UI，每条规则包含类型（Include/Exclude）和模式
4. 保存时将规则列表序列化为 rclone filter 格式字符串数组（`+/-` + 空格 + 模式）

**规则格式转换**：
- 前端 UI: `[{type: "Exclude", pattern: "node_modules/**"}, ...]`
- 存储格式: `["- node_modules/**", "- .git/**", "+ **"]`

---

## 2. 保留删除文件 (No Delete) 实现

### 问题

如何在单向同步时不删除目标端的多余文件？

### 研究

查看 `fs/sync/sync.go`：

```go
// Sync 函数签名
func Sync(ctx context.Context, fdst, fsrc fs.Fs, copyEmptySrcDirs bool) error

// 在 rclone 中，默认行为是删除目标端多余的文件
// 要禁用删除，需要使用 CopyDir 而不是 Sync
func CopyDir(ctx context.Context, fdst, fsrc fs.Fs, copyEmptySrcDirs bool) error
```

关键区别：
- `Sync`: 将源完全同步到目标（包括删除目标端多余文件）
- `CopyDir`: 仅复制源到目标（不删除目标端多余文件）

对于 bisync（双向同步）：
- bisync 设计上会同步双方的删除操作
- "保留删除文件" 选项在双向同步模式下无意义（删除是双向传播的）
- 因此该选项仅在单向同步模式下有效

### 决策

**使用 CopyDir 替代 Sync 实现不删除**：
1. 在 `TaskSyncOptions` 中添加 `noDelete` 字段（布尔值，默认 false）
2. 修改 `runOneWay` 函数：
   - 当 `noDelete = false`（默认）：使用 `rclonesync.Sync`
   - 当 `noDelete = true`：使用 `rclonesync.CopyDir`
3. **双向同步模式下忽略该选项**（UI 层面隐藏该选项）

---

## 3. 并行传输数量 (Transfers) 配置

### 问题

如何配置同步时的并发传输数量？

### 研究

查看 rclone 配置 `fs/config.go`：

```go
// ConfigInfo 包含 transfers 设置
type ConfigInfo struct {
    Transfers int `config:"transfers"` // 并行传输数量
    // ...(其他配置字段)
}
```

设置方式：

```go
import "github.com/rclone/rclone/fs"

// 方式1：获取全局配置的副本并注入 context
ctx, ci := fs.AddConfig(ctx)
ci.Transfers = 8  // 设置并行传输数量
// ctx 现在包含了修改后的配置
```

**注意**: rclone 的 `Transfers` 控制的是并发上传/下载的文件数量，不是连接数。

### 决策

**通过 rclone config 设置 transfers**：
1. 在 `TaskSyncOptions` 中添加 `transfers` 字段（整数，可选，范围 1-64）
2. 在 `internal/core/config/config.go` 中添加全局默认值配置：
   ```toml
   [sync]
   transfers = 4  # 全局默认并行传输数量
   ```
3. 在 `RunTask` 执行前：
   - 如果任务设置了 `transfers`，使用任务级别的值
   - 否则使用配置文件中的全局默认值
   - 全局默认值未设置时使用 rclone 内置默认值 4
4. 通过 `ctx, ci := fs.AddConfig(ctx)` 获取可修改的配置副本，设置 `ci.Transfers` 后使用新的 ctx

---

## 4. 过滤器规则验证

### 问题

如何验证用户输入的过滤器规则是否合法？

### 研究

rclone 的 `filter.AddRule()` 方法会返回错误，如果规则格式不正确。

```go
fi, _ := filter.NewFilter(nil)
err := fi.AddRule("- node_modules/**")
if err != nil {
    // 规则格式错误
}
```

### 决策

**创建独立的验证函数**：
1. 创建 `ValidateFilterRules(rules []string) error` 函数
2. 尝试创建 Filter 并添加所有规则
3. 如果任何规则添加失败，返回详细错误信息
4. 在保存任务前调用此函数进行验证

---

## 5. 过滤器预览功能

### 问题

如何实现"预览过滤后的文件列表"功能？

### 研究

现有的 `ListDirectory` 函数可以列出目录中的文件。需要扩展该函数以支持过滤器。

```go
// 扩展 ListDirectoryOptions
type ListDirectoryOptions struct {
    Remote       string
    Path         string
    Filters      []string  // 新增：过滤器规则
    IncludeFiles bool      // 新增：是否显示被包含的文件（true）或被排除的文件（false）
}
```

### 决策

**扩展 ListDirectory 支持过滤器预览**：
1. 在 `ListDirectoryOptions` 中添加 `Filters` 和 `IncludeFiles` 字段
2. 使用 `filter.NewFilter()` 和 `filter.AddRule()` 创建过滤器
3. 使用 `filter.ReplaceConfig()` 将过滤器注入到 context
4. 调用现有的 `ListDirectory` 逻辑
5. 前端通过 GraphQL 查询 `file.remote` 时传递 `filters` 参数

---

## Summary

| 研究项 | 决策 | 理由 |
|--------|------|------|
| **rclone Filter 实现** | 使用 filter 包 AddRule + ReplaceConfig | rclone 原生支持，规则存储为字符串数组 |
| **保留删除文件** | CopyDir 替代 Sync | 简洁有效，仅单向同步模式可用 |
| **并行传输数量** | fs.AddConfig 注入 | rclone 原生配置，支持任务级和全局默认值 |
| **规则验证** | 使用 filter.AddRule 返回值 | 复用 rclone 内置验证逻辑 |
| **预览功能** | 扩展 ListRemoteDir | 复用现有文件列表逻辑（connection.go），添加过滤参数 |

---

## 6. 实现详细代码参考

### 6.1 filter_validator.go 完整实现

```go
package rclone

import (
    "fmt"
    "github.com/rclone/rclone/fs/filter"
)

// ValidateFilterRules 校验过滤器规则语法
// 返回 nil 表示所有规则有效，否则返回第一个无效规则的错误信息
func ValidateFilterRules(rules []string) error {
    fi, err := filter.NewFilter(nil)
    if err != nil {
        return fmt.Errorf("failed to create filter: %w", err)
    }
    
    for i, rule := range rules {
        // AddRule 会解析规则并返回语法错误
        if err := fi.AddRule(rule); err != nil {
            return fmt.Errorf("规则 #%d %q 无效: %w", i+1, rule, err)
        }
    }
    return nil
}
```

### 6.2 TaskService.validateSyncOptions 实现

```go
func (s *TaskService) validateSyncOptions(opts *TaskSyncOptionsInput) error {
    // 1. 校验 filters 规则语法
    if len(opts.Filters) > 0 {
        if err := rclone.ValidateFilterRules(opts.Filters); err != nil {
            return errs.NewValidationError("syncOptions.filters", err.Error())
        }
    }
    
    // 2. 校验 transfers 范围
    if opts.Transfers != nil && (*opts.Transfers < 1 || *opts.Transfers > 64) {
        return errs.NewValidationError("syncOptions.transfers", "must be between 1 and 64")
    }
    
    return nil
}
```

### 6.3 SyncOptions 扩展结构体

```go
type SyncOptions struct {
    // ... 现有字段
    Filters   []string  // 过滤器规则列表
    NoDelete  bool      // 是否保留目标端删除的文件（默认 false）
    Transfers int       // 并行传输数量 (1-64)
}
```

### 6.4 Sync 方法中应用选项

```go
// Filters: 使用 rclone filter 包
if len(opts.Filters) > 0 {
    fi, err := filter.NewFilter(nil)
    if err != nil {
        return err
    }
    for _, rule := range opts.Filters {
        if err := fi.AddRule(rule); err != nil {
            return fmt.Errorf("invalid filter rule '%s': %w", rule, err)
        }
    }
    ctx = filter.ReplaceConfig(ctx, fi)
}

// Transfers: 使用 fs.AddConfig
if opts.Transfers > 0 {
    ci := fs.GetConfig(ctx)
    ci.Transfers = opts.Transfers
    ctx = fs.AddConfig(ctx, ci)
}

// NoDelete: 使用 CopyDir 替代 Sync
if opts.NoDelete {
    return sync.CopyDir(ctx, dstFs, srcFs, false)
}
return sync.Sync(ctx, dstFs, srcFs, false)
```

### 6.5 ListRemoteDir 扩展

> ⚠️ **重要发现**：rclone 的 `fs.Fs.List()` 是底层 Fs 接口方法，**不会**自动应用 context 中的过滤器配置。过滤器是在更高层的 `fs/operations` 或 `fs/walk` 包中应用的（如 `rclone ls` 命令使用的是 `operations.List` 而不是直接调用 `fs.List`）。
>
> 解决方案：创建 Filter 对象后，手动对每个 entry 调用 `fi.IncludeRemote(entry.Remote())` 进行过滤。

```go
// ListRemoteDirWithOptions 列出远程目录内容（支持过滤器）
// 注意: 此函数在 connection.go 中实现
func ListRemoteDirWithOptions(ctx context.Context, opts ListRemoteDirOptions) ([]DirEntry, error) {
    // 1. 如果有 filters，创建过滤器（注意：不是注入到 context，而是保存引用用于手动过滤）
    var fi *filter.Filter
    if len(opts.Filters) > 0 {
        fi, err = filter.NewFilter(nil)
        if err != nil {
            return nil, err
        }
        for _, rule := range opts.Filters {
            if err := fi.AddRule(rule); err != nil {
                return nil, fmt.Errorf("invalid filter rule '%s': %w", rule, err)
            }
        }
        // 注意：不使用 filter.ReplaceConfig(ctx, fi)，因为 fs.List() 不会读取它
    }
    
    // 2. 调用 rclone 列出目录
    entries, err := remoteFs.List(ctx, opts.Path)
    
    // 3. 手动应用过滤器 - 因为 fs.List() 不会自动应用 context 中的过滤器
    var result []DirEntry
    for _, entry := range entries {
        if fi != nil && !fi.IncludeRemote(entry.Remote()) {
            continue  // 被过滤器排除
        }
        result = append(result, convertEntry(entry))
    }
    
    return result, nil
}
```

### 6.6 Transfers 三层回退逻辑

```go
// 确定最终使用的 transfers 值
func determineTransfers(taskTransfers int, globalTransfers int) int {
    // 1. 优先使用任务级配置
    if taskTransfers > 0 {
        return taskTransfers
    }
    // 2. 其次使用全局配置
    if globalTransfers > 0 {
        return globalTransfers
    }
    // 3. 最后使用内置默认值
    return 4
}
```

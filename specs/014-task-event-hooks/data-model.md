# Data Model: Task Event Hooks

**Feature Branch**: `014-task-event-hooks`  
**Date**: 2026-01-17  
**Status**: Draft

## Overview

本文档定义 Task Event Hooks 功能的数据模型，包括实体定义、字段规范、关系映射和验证规则。

---

## Entity: Hook

Hook 表示一个钩子配置，可关联到 Task 或 Connection。

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | UUID | ✅ | `uuid.New()` | 主键 |
| `enabled` | Boolean | ✅ | `true` | 是否启用此 Hook |
| `priority` | Int | ❌ | `nil` | 执行优先级（升序，nil 排最后） |
| `event` | Enum(HookEvent) | ✅ | - | 触发事件类型 |
| `type` | Enum(HookType) | ✅ | - | Hook 类型 (HTTP/COMMAND) |
| `on_error` | Enum(HookOnError) | ✅ | `IGNORE` | 错误处理行为 |
| `config` | JSON(*HookConfig) | ✅ | - | Hook 配置（强类型结构体） |
| `task_id` | UUID | ❌ | - | 关联的任务 ID（与 connection_id 互斥） |
| `connection_id` | UUID | ❌ | - | 关联的连接 ID（与 task_id 互斥） |
| `created_at` | DateTime | ✅ | `time.Now()` | 创建时间 |
| `updated_at` | DateTime | ✅ | `time.Now()` | 更新时间 |

### HookConfig 结构体

定义在 `internal/api/graphql/model/hook.go`：

```go
// HookConfig 统一的 Hook 配置结构体
// 根据 Hook.Type 使用不同的字段子集
type HookConfig struct {
    // HTTP Hook 配置
    URL     string            `json:"url,omitempty"`     // 请求 URL（支持模板）
    Method  string            `json:"method,omitempty"`  // HTTP 方法，默认 POST
    Headers map[string]string `json:"headers,omitempty"` // 自定义请求头
    Body    string            `json:"body,omitempty"`    // 请求体模板

    // Command Hook 配置
    Command string `json:"command,omitempty"` // Shell 命令（支持模板）
    WorkDir string `json:"workDir,omitempty"` // 工作目录
    Timeout int    `json:"timeout,omitempty"` // 超时时间（秒），默认 30
}
```

**设计说明**：
- 使用单一结构体而非多态，与现有 `TaskSyncOptions`、`ConnectionOptions` 保持一致
- 字段使用 `omitempty`，HTTP Hook 只存储 HTTP 相关字段，Command Hook 只存储命令相关字段
- 强类型确保编译期类型检查，避免 `map[string]interface{}` 的运行时类型断言

### Enums

#### HookEvent
```go
type HookEvent string

const (
    HookEventOnStart   HookEvent = "ON_START"   // 任务开始执行前
    HookEventOnSuccess HookEvent = "ON_SUCCESS" // 任务成功完成后
    HookEventOnFailure HookEvent = "ON_FAILURE" // 任务失败后
    HookEventOnEnd     HookEvent = "ON_END"     // 任务结束后（无论成功或失败）
)
```

#### HookType
```go
type HookType string

const (
    HookTypeHTTP    HookType = "HTTP"    // HTTP 请求
    HookTypeCommand HookType = "COMMAND" // Shell 命令
)
```

#### HookOnError
```go
type HookOnError string

const (
    HookOnErrorIgnore HookOnError = "IGNORE" // 忽略错误，继续执行后续 hooks
    HookOnErrorCancel HookOnError = "CANCEL" // 停止任务，Job 标记为 CANCELLED
    HookOnErrorFatal  HookOnError = "FATAL"  // 停止任务，Job 标记为 FAILED
)
```

### Relationships (Edges)

| Edge | Direction | Target | Cardinality | Cascade | Description |
|------|-----------|--------|-------------|---------|-------------|
| `task` | From | Task | N:1 | - | 关联的任务（可选） |
| `connection` | From | Connection | N:1 | - | 关联的连接（可选） |

**反向 Edges（需添加到现有实体）**：

| Entity | Edge | Direction | Target | Cascade |
|--------|------|-----------|--------|---------|
| Task | `hooks` | To | Hook | CASCADE |
| Connection | `hooks` | To | Hook | CASCADE |

### Indexes

| Fields | Type | Description |
|--------|------|-------------|
| `task_id` | Index | 按任务查询 hooks |
| `connection_id` | Index | 按连接查询 hooks |
| `task_id, event` | Index | 按任务+事件类型查询 |
| `connection_id, event` | Index | 按连接+事件类型查询 |
| `created_at` | Index | 按创建时间排序 |

### Validation Rules

1. **互斥关联**: `task_id` 和 `connection_id` 必须有且仅有一个有值
2. **Config 完整性**: `config` 字段必须包含对应 `type` 的必填字段（HTTP 需要 url，Command 需要 command）

### Ent Schema

```go
// internal/core/db/schema/hook.go
package schema

import (
    "time"

    "entgo.io/ent"
    "entgo.io/ent/dialect/entsql"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
    "github.com/google/uuid"
    "github.com/xzzpig/rclone-sync/internal/api/graphql/model"
)

// Hook holds the schema definition for the Hook entity.
type Hook struct {
    ent.Schema
}

// Fields of the Hook.
func (Hook) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("id", uuid.UUID{}).
            Default(uuid.New),
        field.Bool("enabled").
            Default(true),
        field.Int("priority").
            Optional().
            Nillable(),
        field.Enum("event").
            GoType(model.HookEvent("")),
        field.Enum("type").
            GoType(model.HookType("")),
        field.Enum("on_error").
            GoType(model.HookOnError("")).
            Default(string(model.HookOnErrorIgnore)),
        // Config - 强类型结构体，与 Task.options、Connection.options 设计一致
        field.JSON("config", &model.HookConfig{}).
            Comment("Hook configuration (HTTP or Command settings)"),
        // Association fields (polymorphic)
        field.UUID("task_id", uuid.UUID{}).
            Optional().
            Nillable(),
        field.UUID("connection_id", uuid.UUID{}).
            Optional().
            Nillable(),
        // Timestamps
        field.Time("created_at").
            Default(time.Now).
            Immutable(),
        field.Time("updated_at").
            Default(time.Now).
            UpdateDefault(time.Now),
    }
}

// Indexes of the Hook.
func (Hook) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("task_id"),
        index.Fields("connection_id"),
        index.Fields("task_id", "event"),
        index.Fields("connection_id", "event"),
        index.Fields("created_at"),
    }
}

// Edges of the Hook.
func (Hook) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("task", Task.Type).
            Ref("hooks").
            Unique().
            Field("task_id"),
        edge.From("connection", Connection.Type).
            Ref("hooks").
            Unique().
            Field("connection_id"),
    }
}
```

---

## Entity: JobLog (Extension)

复用现有 JobLog 实体记录 Hook 执行结果。

### LogAction Enum Extension

新增 `HOOK` 值到现有 `LogAction` 枚举：

```go
// internal/api/graphql/model/enums.go
const (
    LogActionUpload   LogAction = "UPLOAD"
    LogActionDownload LogAction = "DOWNLOAD"
    LogActionDelete   LogAction = "DELETE"
    LogActionMove     LogAction = "MOVE"
    LogActionError    LogAction = "ERROR"
    LogActionUnknown  LogAction = "UNKNOWN"
    LogActionHook     LogAction = "HOOK"     // NEW: Hook 执行记录
)
```

### Hook Execution Log Format

当 `what = HOOK` 时，字段语义如下：

| Field | Semantic | Example |
|-------|----------|---------|
| `level` | INFO=成功, ERROR=失败 | `INFO` |
| `path` | Hook 标识符 | `hook:<uuid>:<event>` 如 `hook:abc123:on_success` |
| `size` | 执行耗时(ms) 或 负值状态码 | `150` (成功, 150ms) / `-500` (HTTP 500) / `-1` (exit code 1) |

---

## Runtime Type: HookContext

Hook 执行时传递给模板引擎的上下文数据。

### Structure

```go
// internal/core/hook/context.go
type HookContext struct {
    Task     TaskInfo          `json:"task"`     // 任务信息
    Job      JobInfo           `json:"job"`      // 执行信息
    Event    string            `json:"event"`    // 事件类型
    Error    string            `json:"error"`    // 错误信息（仅失败时有值）
    Duration time.Duration     `json:"duration"` // 执行耗时
    Stats    TransferStats     `json:"stats"`    // 传输统计
    Env      map[string]string `json:"env"`      // 环境变量映射
}

type TaskInfo struct {
    ID         uuid.UUID `json:"id"`
    Name       string    `json:"name"`
    SourcePath string    `json:"sourcePath"`
    RemotePath string    `json:"remotePath"`
    Direction  string    `json:"direction"`
}

type JobInfo struct {
    ID        uuid.UUID `json:"id"`
    Status    string    `json:"status"`
    Trigger   string    `json:"trigger"`
    StartTime time.Time `json:"startTime"`
    EndTime   time.Time `json:"endTime"`
}

type TransferStats struct {
    FilesTransferred int64 `json:"filesTransferred"`
    BytesTransferred int64 `json:"bytesTransferred"`
    FilesDeleted     int64 `json:"filesDeleted"`
    ErrorCount       int64 `json:"errorCount"`
}
```

### Template Variables

模板中可用的变量：

| Variable | Type | Description |
|----------|------|-------------|
| `.Task.ID` | string | 任务 UUID |
| `.Task.Name` | string | 任务名称 |
| `.Task.SourcePath` | string | 本地源路径 |
| `.Task.RemotePath` | string | 远程目标路径 |
| `.Task.Direction` | string | 同步方向 |
| `.Job.ID` | string | Job UUID |
| `.Job.Status` | string | Job 状态 |
| `.Job.Trigger` | string | 触发方式 |
| `.Job.StartTime` | time.Time | 开始时间 |
| `.Job.EndTime` | time.Time | 结束时间 |
| `.Event` | string | 事件类型 (on_start/on_success/on_failure/on_end) |
| `.Error` | string | 错误信息 |
| `.Duration` | time.Duration | 执行耗时 |
| `.Stats.FilesTransferred` | int64 | 传输文件数 |
| `.Stats.BytesTransferred` | int64 | 传输字节数 |
| `.Stats.FilesDeleted` | int64 | 删除文件数 |
| `.Stats.ErrorCount` | int64 | 错误数 |
| `.Env` | map[string]string | 环境变量 |

### Template Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `FormatTime` | `func(time.Time) string` | 格式化时间为 RFC3339 |
| `FormatDuration` | `func(time.Duration) string` | 格式化耗时 |
| `FormatSizeBytes` | `func(int64) string` | 格式化字节大小 |
| `JsonMarshal` | `func(interface{}) string` | 转换为 JSON |
| `Summary` | `func(*HookContext) string` | 生成默认摘要 |

---

## State Transitions

### Hook Execution Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                        Task Execution                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────┐                                                     │
│  │ PENDING │                                                     │
│  └────┬────┘                                                     │
│       │                                                          │
│       ▼                                                          │
│  ┌─────────────────┐                                             │
│  │ Execute on_start│──────────────────────────────┐              │
│  │     hooks       │                              │              │
│  └────────┬────────┘                              │              │
│           │                                       │              │
│      ┌────┴────┐                                  │              │
│      │ Error?  │                                  │              │
│      └────┬────┘                                  │              │
│    No     │     Yes (CANCEL/FATAL)                │              │
│     ▼     └─────────────────────┐                 │              │
│  ┌─────────┐                    │                 │              │
│  │ RUNNING │                    │                 │              │
│  └────┬────┘                    │                 │              │
│       │                         ▼                 │              │
│       │                  ┌──────────────┐         │              │
│  ┌────┴────┐             │  CANCELLED   │         │              │
│  │ Sync    │             └──────────────┘         │              │
│  │ Execute │                    │                 │              │
│  └────┬────┘                    │                 │              │
│       │                         │                 │              │
│  ┌────┴────┐                    │                 │              │
│  │ Result? │                    │                 │              │
│  └────┬────┘                    │                 │              │
│   ┌───┴───┐                     │                 │              │
│   ▼       ▼                     │                 │              │
│ ┌───────┐ ┌───────┐             │                 │              │
│ │SUCCESS│ │FAILED │◄────────────┘                 │              │
│ └───┬───┘ └───┬───┘                               │              │
│     │         │                                   │              │
│     ▼         ▼                                   │              │
│ ┌───────────────────┐   ┌───────────────────┐     │              │
│ │Execute on_success │   │Execute on_failure │     │              │
│ │     hooks         │   │     hooks         │     │              │
│ └─────────┬─────────┘   └─────────┬─────────┘     │              │
│           │                       │               │              │
│           └───────────┬───────────┘               │              │
│                       ▼                           │              │
│               ┌───────────────────┐               │              │
│               │ Execute on_end    │◄──────────────┘              │
│               │     hooks         │                              │
│               └───────────────────┘                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Migration Strategy

### Migration File: YYYYMMDDHHMMSS_add_hooks_table.up.sql

```sql
-- Create hooks table
CREATE TABLE hooks (
    id TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 1,
    priority INTEGER,
    event TEXT NOT NULL,
    type TEXT NOT NULL,
    on_error TEXT NOT NULL DEFAULT 'IGNORE',
    config TEXT NOT NULL,
    task_id TEXT,
    connection_id TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE CASCADE,
    CHECK ((task_id IS NULL) != (connection_id IS NULL))
);

-- Create indexes
CREATE INDEX idx_hooks_task_id ON hooks(task_id);
CREATE INDEX idx_hooks_connection_id ON hooks(connection_id);
CREATE INDEX idx_hooks_task_id_event ON hooks(task_id, event);
CREATE INDEX idx_hooks_connection_id_event ON hooks(connection_id, event);
CREATE INDEX idx_hooks_created_at ON hooks(created_at);
```

### Migration File: YYYYMMDDHHMMSS_add_hooks_table.down.sql

```sql
DROP INDEX IF EXISTS idx_hooks_created_at;
DROP INDEX IF EXISTS idx_hooks_connection_id_event;
DROP INDEX IF EXISTS idx_hooks_task_id_event;
DROP INDEX IF EXISTS idx_hooks_connection_id;
DROP INDEX IF EXISTS idx_hooks_task_id;
DROP TABLE IF EXISTS hooks;
```

---

## Summary

| Entity | Type | Purpose |
|--------|------|---------|
| Hook | New | 存储 Hook 配置 |
| JobLog | Extended | 复用记录 Hook 执行日志（新增 HOOK action） |
| HookContext | Runtime | Hook 执行时的模板上下文 |

---

## Design Notes

### HookExecution 实现方式

spec.md 中的 `HookExecution` 概念实体通过复用现有 `JobLog` 表实现，而非创建独立表。具体映射：
- `JobLog.action` = `HOOK`
- `JobLog.path` = `hook:<uuid>:<event>` 格式
- `JobLog.level` = 执行结果（INFO=成功，ERROR=失败）
- `JobLog.size` = 执行耗时（毫秒）或 HTTP 状态码

这样设计的优点：复用现有基础设施，Hook 执行记录随 Job 自动保留/清理。


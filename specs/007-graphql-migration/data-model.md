# Data Model: GraphQL Migration

**Feature Branch**: `007-graphql-migration`  
**Date**: 2024-12-20

## Overview

本项目的数据模型通过 GraphQL Schema 定义。所有类型、字段、枚举和验证规则的权威定义请参见：

**📄 [contracts/schema.graphql](./contracts/schema.graphql)**

本文档仅补充 schema 中无法表达的设计决策和映射关系。

## Entity Relationships

```
┌──────────────┐      1:N      ┌──────────────┐
│  Connection  │◄──────────────│     Task     │
└──────────────┘               └──────────────┘
                                     │
                                     │ 1:N
                                     ▼
                               ┌──────────────┐
                               │     Job      │
                               └──────────────┘
                                     │
                                     │ 1:N
                                     ▼
                               ┌──────────────┐
                               │   JobLog     │
                               └──────────────┘
```

| From | To | Cardinality | Description |
|------|-----|-------------|-------------|
| Connection | Task | 1:N | 一个连接可被多个任务使用 |
| Task | Job | 1:N | 一个任务有多个执行记录 |
| Job | JobLog | 1:N | 一个作业有多条日志 |

## Ent ↔ GraphQL Mapping

| Ent Entity | GraphQL Type | Notes |
|------------|--------------|-------|
| `ent.Task` | `Task` | 直接映射，`options` JSON → `TaskSyncOptions` 类型 |
| `ent.Connection` | `Connection` | `encrypted_config` 解密后返回为 `config` |
| `ent.Job` | `Job` | `task_jobs` FK 映射为 `task` 关系 |
| `ent.JobLog` | `JobLog` | `job_logs` FK 映射为 `job` 关系 |

## Custom Types

### TaskSyncOptions

Task 的同步选项，从 JSON 映射为强类型：

```graphql
enum ConflictResolution {
  NEWER   # 保留较新文件，重命名较旧文件（默认）
  LOCAL   # 保留本地文件，删除远程
  REMOTE  # 保留远程文件，删除本地
  BOTH    # 保留两者，添加冲突后缀
}

type TaskSyncOptions {
  conflictResolution: ConflictResolution
}
```

对应后端 `internal/rclone/sync.go` 中的 `getConflictResolutionFromOptions` 函数。

### ConnectionLoadStatus

Connection 的运行时加载状态（非持久化）：

```graphql
enum ConnectionLoadStatus {
  LOADED   # 已加载
  LOADING  # 加载中
  ERROR    # 加载失败
}

type Connection {
  loadStatus: ConnectionLoadStatus!
  loadError: String  # 仅当 loadStatus = ERROR 时有值
}
```

## Resolver 字段标记

使用 `@goField(forceResolver: true)` 标记需要自定义 resolver 的字段：

| 类型 | 字段 | 原因 |
|------|------|------|
| Task | options | JSON → TaskSyncOptions 类型转换 |
| Task | connection, jobs, latestJob | ent edge / 分页查询 |
| Connection | config | 需要解密处理 |
| Connection | loadStatus, loadError | 运行时状态计算 |
| Connection | tasks, quota | 分页查询 / rclone API |
| Job | task, logs | ent edge / 分页查询 |
| JobLog | job | ent edge |
| Query/Mutation | 命名空间字段 | 返回空对象实例 |

## State Transitions

### Job Status Flow

```
     ┌─────────┐
     │ PENDING │
     └────┬────┘
          │ start
          ▼
     ┌─────────┐
     │ RUNNING │
     └────┬────┘
          │
    ┌─────┼─────┐
    │     │     │
    ▼     ▼     ▼
┌───────┐ ┌──────┐ ┌─────────┐
│SUCCESS│ │FAILED│ │CANCELLED│
└───────┘ └──────┘ └─────────┘
```

## Validation Rules

> 详细字段约束见 schema.graphql 中的 Input 类型定义。

### 业务规则（代码实现）

1. **Task**
   - `schedule`: 空或有效的 cron 表达式 (5/6 字段)
   - `connectionId`: 必须引用存在的 Connection

2. **Connection**
   - `name`: 系统唯一
   - `type`: 必须是有效的 Provider name
   - `config`: 必须包含该 type 要求的所有必填选项

3. **Job**
   - 状态转换必须有效: PENDING → RUNNING → (SUCCESS | FAILED | CANCELLED)
   - `endTime` 只能在 RUNNING → 终态转换时设置

## Security Considerations

1. **敏感数据**: `Connection.config` 在数据库中加密存储（`encrypted_config`），GraphQL 返回解密后的数据
2. **无权限控制**: 单机单用户应用，不实现字段级权限
3. **日志过滤**: 确保错误信息不包含敏感凭证

## Non-Persisted Types

以下 GraphQL 类型为运行时数据，不存储在数据库：

- `Provider` / `ProviderOption` - 从 rclone 运行时获取
- `FileEntry` - 实时文件系统查询
- `ConnectionQuota` - 实时远程查询
- `JobProgressEvent` - 运行时进度流

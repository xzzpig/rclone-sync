# Data Model: 同步任务多项优化

**Feature**: 013-sync-optimizations  
**Date**: 2026-01-16  
**Status**: Complete

## Overview

本功能涉及以下数据模型变更：
1. **Task 实体**：新增 `enabled` 字段
2. **TransferItem 类型**：新增 `direction` 字段
3. **JobProgressEvent 类型**：新增 `filesChecking` 字段

---

## 1. Task 实体变更

### 现有字段
| 字段 | 类型 | 说明 |
|-----|------|-----|
| id | UUID | 主键 |
| name | String | 任务名称 |
| source_path | String | 本地路径 |
| connection_id | UUID | 关联连接 |
| remote_path | String | 远程路径 |
| direction | Enum | 同步方向 |
| schedule | String? | Cron 表达式 |
| realtime | Boolean | 实时同步开关 |
| options | JSON | 同步选项 |
| created_at | DateTime | 创建时间 |
| updated_at | DateTime | 更新时间 |

### 新增字段
| 字段 | 类型 | 默认值 | 说明 |
|-----|------|-------|-----|
| **enabled** | Boolean | true | 任务启用状态 |

### Ent Schema 变更

```go
// internal/core/db/schema/task.go
func (Task) Fields() []ent.Field {
    return []ent.Field{
        // ... 现有字段 ...
        field.Bool("enabled").
            Default(true).
            Comment("任务启用状态，禁用后不响应定时/实时触发"),
    }
}
```

### 数据库迁移

```sql
-- 迁移文件: XXXXXX_add_task_enabled.up.sql
ALTER TABLE tasks ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- 回滚文件: XXXXXX_add_task_enabled.down.sql
ALTER TABLE tasks DROP COLUMN enabled;
```

### 索引策略
- **无需添加索引**：`enabled` 是低基数布尔字段，且不用于频繁过滤查询

### 业务规则
1. 新创建的任务默认 `enabled = true`
2. `enabled = false` 时：
   - 定时触发 (scheduler) 跳过此任务
   - 实时触发 (watcher) 跳过此任务
   - 手动触发仍可正常执行
3. 状态变更通过 `setEnabled` mutation 实现

---

## 2. TransferDirection 枚举（新增）

```graphql
"""
传输方向
"""
enum TransferDirection {
    """
    上传 (本地 → 远程)
    """
    UPLOAD
    """
    下载 (远程 → 本地)
    """
    DOWNLOAD
}
```

### 方向判断逻辑

| 任务方向 | 传输方向 |
|---------|---------|
| UPLOAD | 始终为 UPLOAD |
| DOWNLOAD | 始终为 DOWNLOAD |
| BIDIRECTIONAL | 根据文件操作类型判断 |

对于双向同步：
- 文件从本地同步到远程 → UPLOAD
- 文件从远程同步到本地 → DOWNLOAD

---

## 3. TransferItem 类型变更

### 现有字段
| 字段 | 类型 | 说明 |
|-----|------|-----|
| name | String | 文件路径 |
| size | BigInt | 文件大小 |
| bytes | BigInt | 已传输字节 |

### 新增字段
| 字段 | 类型 | 说明 |
|-----|------|-----|
| **direction** | TransferDirection | 传输方向 |

### GraphQL Schema

```graphql
type TransferItem {
    name: String!
    size: BigInt!
    bytes: BigInt!
    direction: TransferDirection!  # 新增
}
```

---

## 4. JobProgressEvent 类型变更

### 现有字段
| 字段 | 类型 | 说明 |
|-----|------|-----|
| jobId | ID | 作业 ID |
| taskId | ID | 任务 ID |
| connectionId | ID | 连接 ID |
| status | JobStatus | 作业状态 |
| filesTransferred | Int | 已传输文件数 |
| bytesTransferred | BigInt | 已传输字节数 |
| filesTotal | Int | 总文件数 |
| bytesTotal | BigInt | 总字节数 |
| filesDeleted | Int | 已删除文件数 |
| errorCount | Int | 错误数量 |
| startTime | DateTime | 开始时间 |
| endTime | DateTime? | 结束时间 |

### 新增字段
| 字段 | 类型 | 说明 |
|-----|------|-----|
| **filesChecking** | Int | 正在检查的文件数 |

### GraphQL Schema

```graphql
type JobProgressEvent {
    # ... 现有字段 ...
    """
    正在检查（扫描）的文件数量
    """
    filesChecking: Int!
}
```

### 数据来源
- 从 rclone 的 `accounting.StatsInfo.Checking` 字段获取
- 在 `processStats` 方法中采集并包含在进度事件中

---

## 5. 空任务判断标准（逻辑定义）

空任务 (Empty Job) 定义：
```go
type EmptyJobCriteria struct {
    Status           JobStatus  // 必须为 SUCCESS
    FilesTransferred int        // 必须为 0
    BytesTransferred int64      // 必须为 0
    FilesDeleted     int        // 必须为 0
    ErrorCount       int        // 必须为 0
}
```

### 清理规则
1. 当 `auto_delete_empty_jobs = true` 时执行
2. Job N 完成后，查询 Job N-1（同任务，按 end_time 降序的上一条记录）
3. 如果 Job N-1 符合空任务标准 → 删除 Job N-1
4. 始终保留 Job N

---

## 6. 缓存数据库大小（逻辑修改）

### 涉及文件
- 主数据库: `{connection_id}.db`
- WAL 文件: `{connection_id}.db-wal`
- SHM 文件: `{connection_id}.db-shm`

### 大小计算公式
```
total_size = size(.db) + size(.db-wal) + size(.db-shm)
```

对于不存在的文件，计为 0。

---

## 7. 概览页面数据加载模型（逻辑变更）

### 现有模型
前端通过单个 GraphQL 字段 `connection.get(id)` 获取包含 `quota` 和 `cacheStatus` 的完整对象。

### 变更后模型
前端将逻辑拆分为两个并发的异步操作，对应同一个数据源但不同的字段集。

#### 存储使用 (Storage Usage)
- **数据源**: `Connection.quota`
- **解析器**: `quota` resolver
- **加载行为**: 异步延迟，显示 Skeleton/Loader
- **错误处理**: 独立显示 ERROR 状态

#### 缓存状态 (Cache Status)
- **数据源**: `Connection.cacheStatus`
- **解析器**: `cacheStatus` resolver
- **加载行为**: 立即展示（本地计算），不受 `quota` 阻塞
- **实时更新**: 继续支持 WebSocket 订阅更新

---

## Entity Relationship Diagram

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│   Connection    │       │      Task       │       │      Job        │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id              │◄──────┤ connection_id   │       │ id              │
│ name            │       │ id              │◄──────┤ task_id         │
│ ...             │       │ name            │       │ status          │
└─────────────────┘       │ enabled [NEW]   │       │ filesTransferred│
                          │ ...             │       │ ...             │
                          └─────────────────┘       └─────────────────┘
```

---

## Validation Rules

### Task.enabled
- 类型: Boolean
- 必填: 是
- 默认: true
- 验证: 无特殊验证

### TransferItem.direction
- 类型: Enum (UPLOAD | DOWNLOAD)
- 必填: 是
- 验证: 必须是有效的枚举值

### JobProgressEvent.filesChecking
- 类型: Int
- 必填: 是
- 默认: 0
- 验证: >= 0

---

## State Transitions

### Task.enabled 状态转换
```
[Enabled] ──setEnabled(false)──▶ [Disabled]
                                     │
                                     │ setEnabled(true)
                                     ▼
                               [Enabled]
```

禁用状态下：
- ❌ 定时触发不执行
- ❌ 实时触发不执行
- ✅ 手动触发可执行

---

*Data model design complete*

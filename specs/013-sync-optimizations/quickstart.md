# Quickstart: 同步任务多项优化

**Feature**: 013-sync-optimizations  
**Date**: 2026-01-16

## 概述

本功能包含 6 个独立的优化项，可分别实现和测试：

| # | 功能 | 优先级 | 复杂度 |
|---|-----|-------|-------|
| 1 | 任务完成 Toast 提醒 | P1 | 低 |
| 2 | 传输进度区分上传/下载 | P1 | 中 |
| 3 | 显示检查中的文件数 | P2 | 低 |
| 4 | 允许禁用任务 | P2 | 中 |
| 5 | 空任务清理优化 | P3 | 低 |
| 6 | 缓存数据库大小修复 | P3 | 低 |
| 7 | 概览卡片数据分离加载 | P2 | 中 |

---

## 快速开始

### 前置条件

1. Go 1.25+ 已安装
2. Node.js 和 pnpm 已安装
3. 项目已克隆并可正常构建

### 开发环境启动

```bash
# 后端
go run ./cmd/rclone-sync serve

# 前端 (另一个终端)
cd web && pnpm dev
```

---

## 实现指南

### 1. 任务完成 Toast 提醒 (P1)

**涉及文件**:
- `web/src/store/jobProgress.tsx` - 添加完成检测逻辑
- `web/project.inlang/messages/en.json` - 添加 Toast 文案
- `web/project.inlang/messages/zh-CN.json` - 添加 Toast 文案

**实现步骤**:
1. 在 `jobProgress.tsx` 中监听 `JobProgressEvent.status` 变化
2. 检测状态从 `RUNNING` 变为终态 (SUCCESS/FAILED/CANCELLED)
3. 调用 `toast()` 显示通知

**代码示例**:
```tsx
// 在订阅回调中
if (prevStatus === 'RUNNING' && ['SUCCESS', 'FAILED', 'CANCELLED'].includes(event.status)) {
  const variant = event.status === 'SUCCESS' ? 'success' 
                : event.status === 'FAILED' ? 'error' 
                : 'warning';
  toast({ title: taskName, description: m.job_completed_status({ status: event.status }), variant });
}
```

**测试验证**:
- 启动任意同步任务，完成后应弹出 Toast

---

### 2. 传输进度区分上传/下载 (P1)

**涉及文件**:
- `internal/api/graphql/schema/job.graphql` - 添加 TransferDirection 枚举和字段
- `internal/api/graphql/model/models_gen.go` - 重新生成
- `internal/rclone/sync.go` - 在 `broadcastTransferProgress` 中设置 direction
- `web/src/modules/connections/components/ActiveTransfersCard.tsx` - 显示方向图标

**GraphQL 变更**:
```graphql
enum TransferDirection { UPLOAD, DOWNLOAD }

type TransferItem {
  # ... 现有字段 ...
  direction: TransferDirection!
}
```

**后端逻辑**:
```go
// 根据任务方向确定传输方向
func getTransferDirection(taskDirection model.SyncDirection, isUpload bool) model.TransferDirection {
    switch taskDirection {
    case model.SyncDirectionUpload:
        return model.TransferDirectionUpload
    case model.SyncDirectionDownload:
        return model.TransferDirectionDownload
    case model.SyncDirectionBidirectional:
        if isUpload {
            return model.TransferDirectionUpload
        }
        return model.TransferDirectionDownload
    }
    return model.TransferDirectionUpload
}
```

**前端显示**:
```tsx
<Show when={transfer.direction === 'UPLOAD'}>
  <IconArrowUp class="h-4 w-4 text-green-500" />
</Show>
<Show when={transfer.direction === 'DOWNLOAD'}>
  <IconArrowDown class="h-4 w-4 text-blue-500" />
</Show>
```

---

### 3. 显示检查中的文件数 (P2)

**涉及文件**:
- `internal/api/graphql/schema/job.graphql` - 添加 filesChecking 字段
- `internal/api/graphql/model/models_gen.go` - 重新生成
- `internal/rclone/sync.go` - 在 `processStats` 中获取 checking 数量
- `web/src/modules/connections/components/RunningJobsCard.tsx` - 显示检查中文件数

**GraphQL 变更**:
```graphql
type JobProgressEvent {
  # ... 现有字段 ...
  filesChecking: Int!
}
```

**后端获取**:
```go
// 在 processStats 中
stats := accounting.GlobalStats()
event.FilesChecking = stats.GetChecking()
```

**前端显示**:
```tsx
<Show when={progress.filesChecking > 0}>
  <p class="text-sm text-muted-foreground">
    {m.checking_files({ count: progress.filesChecking })}
  </p>
</Show>
```

---

### 4. 允许禁用任务 (P2)

**涉及文件**:
- `internal/core/db/schema/task.go` - 添加 enabled 字段
- `internal/core/db/migrations/` - 创建迁移文件
- `internal/api/graphql/schema/task.graphql` - 添加 enabled 字段
- `internal/core/scheduler/scheduler.go` - 检查 enabled 状态
- `internal/core/watcher/watcher.go` - 检查 enabled 状态
- `web/src/modules/connections/views/Tasks.tsx` - 添加启用/禁用按钮

**数据库迁移**:
```bash
# 创建迁移文件
atlas migrate diff add_task_enabled \
  --dir "file://internal/core/db/migrations" \
  --to "ent://internal/core/db/schema" \
  --dev-url "sqlite://file?mode=memory"
```

**Ent Schema**:
```go
field.Bool("enabled").
    Default(true).
    Comment("任务启用状态")
```

**Scheduler 检查**:
```go
func (s *Scheduler) shouldRunTask(task *ent.Task) bool {
    if !task.Enabled {
        s.logger.Debug("跳过禁用的任务", zap.String("task", task.Name))
        return false
    }
    return true
}
```

**前端按钮**:
```tsx
<Button
  variant="ghost"
  size="icon"
  onClick={() => toggleEnabled(task.id, !task.enabled)}
  title={task.enabled ? m.disable_task() : m.enable_task()}
>
  <Show when={task.enabled} fallback={<IconPause />}>
    <IconPlay />
  </Show>
</Button>
```

---

### 5. 空任务清理优化 (P3)

**涉及文件**:
- `internal/rclone/sync.go` - 修改清理逻辑
- `internal/core/db/query/job_query.go` - 添加查询方法

**实现逻辑**:
```go
// 在 RunTask 结束处
if cfg.App.Job.AutoDeleteEmptyJobs {
    // 查询上一个 Job
    prevJob, err := e.jobQuery.GetPreviousJob(ctx, job.TaskID, job.ID)
    if err == nil && prevJob != nil && isEmptyJob(prevJob) {
        e.jobQuery.DeleteJob(ctx, prevJob.ID)
    }
}

func isEmptyJob(job *ent.Job) bool {
    return job.Status == model.JobStatusSuccess &&
           job.FilesTransferred == 0 &&
           job.BytesTransferred == 0 &&
           job.FilesDeleted == 0 &&
           job.ErrorCount == 0
}
```

**查询方法**:
```go
func (q *JobQuery) GetPreviousJob(ctx context.Context, taskID, currentJobID uuid.UUID) (*ent.Job, error) {
    return q.client.Job.Query().
        Where(
            job.TaskIDEQ(taskID),
            job.IDNEQ(currentJobID),
            job.EndTimeNotNil(),
        ).
        Order(ent.Desc(job.FieldEndTime)).
        First(ctx)
}
```

---

### 6. 缓存数据库大小修复 (P3)

**涉及文件**:
- `internal/rclone/backend/metacache/cache_store.go` - 修改 GetDBSize 方法

**实现代码**:
```go
func (s *CacheStore) GetDBSize() (int64, error) {
    var totalSize int64
    
    files := []string{s.dbPath, s.dbPath + "-wal", s.dbPath + "-shm"}
    for _, file := range files {
        info, err := os.Stat(file)
        if err != nil {
            if os.IsNotExist(err) {
                continue
            }
            return 0, err
        }
        totalSize += info.Size()
    }
    
    return totalSize, nil
}
```

**测试验证**:
1. 创建一个连接并启用缓存
2. 执行同步操作生成 WAL 文件
3. 检查 UI 显示的大小与 `ls -la app_data/cache/` 的总大小一致

---

### 7. 概览卡片数据分离加载 (P2)

**涉及文件**:
- `web/src/api/graphql/queries/connections.ts` - 拆分 GraphQL 查询
- `web/src/modules/connections/views/Overview.tsx` - 修改数据加载逻辑

**实现步骤**:
1. 在 `queries/connections.ts` 中更新 `ConnectionGetQuotaQuery` 并定义 `ConnectionGetCacheStatusQuery`。
2. 在 `Overview.tsx` 中：
   - 使用 `ConnectionGetQuotaQuery` 用于 `quota` 字段。
   - 新增一个 `createQuery` 调用 `ConnectionGetCacheStatusQuery` 用于 `cacheStatus` 字段。
   - 分别处理两者的加载状态。

**代码示例**:
```tsx
// 1. Quota 查询 (更新后不再包含 cacheStatus)
const [quotaResult] = createQuery({
  query: ConnectionGetQuotaQuery,
  variables: () => ({ id: connectionId()! }),
});

// 2. CacheStatus 查询 (新增)
const [cacheStatusResult] = createQuery({
  query: ConnectionGetCacheStatusQuery,
  variables: () => ({ id: connectionId()! }),
});

// 渲染时
<Show when={!quotaResult.fetching} fallback={<Skeleton />}>
  {/* 渲染配额卡片 */}
</Show>

<CacheStatusCard
  status={realtimeCacheStatus() ?? cacheStatusResult.data?.connection?.get?.cacheStatus}
  loading={cacheStatusResult.fetching}
/>
```

---

## 代码生成命令

```bash
# 重新生成 Ent 代码
go generate ./internal/core/ent

# 重新生成 GraphQL 代码
go generate ./internal/api/graphql

# 编译前端 i18n
cd web && pnpm paraglide
```

---

## 测试清单

| 功能 | 测试方法 |
|-----|---------|
| Toast 提醒 | 启动任务，完成后检查 Toast |
| 传输方向 | 执行上传/下载任务，检查图标 |
| 检查文件数 | 同步大目录，检查数字更新 |
| 禁用任务 | 禁用后验证定时/实时不触发 |
| 空任务清理 | 连续执行两次无变动同步，检查历史 |
| 缓存大小 | 对比 UI 显示与文件系统实际大小 |

---

*Quickstart guide complete*

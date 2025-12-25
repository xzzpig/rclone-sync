# Research: UI Detail Improvements

**Feature Branch**: `008-ui-detail-improvements`  
**Created**: 2024-12-24

---

## 1. rclone 进度信息获取

### 问题

Spec 要求显示"总文件数和已传输文件数"、"总字节数和已传输字节数"，但当前实现只有已传输的数据。

### 研究

查看 rclone 源码 `fs/accounting/stats.go`：

```go
// StatsInfo 结构体提供的方法
func (s *StatsInfo) GetTransfers() int64      // 已完成的传输数
func (s *StatsInfo) GetBytes() int64          // 已传输的字节数
func (s *StatsInfo) GetChecks() int64         // 已完成的检查数
func (s *StatsInfo) InProgress() []*Transfer  // 当前进行中的传输

// 关键方法 - 返回总进度信息
func (s *StatsInfo) RemoteStats(short bool) (out rc.Params, err error)

// 内部计算方法 calculateTransferStats() 返回：
// - totalChecks = checkQueue + checks + checking
// - totalTransfers = transferQueue + transfers + transferring  
// - totalBytes = transferQueueSize + bytes + transferringBytesTotal - transferringBytesDone
```

关键发现：
1. `RemoteStats(false)` 返回 `rc.Params`，包含：
   - `totalChecks`: 总检查数（队列 + 已完成 + 进行中）
   - `totalTransfers`: 总传输数（队列 + 已完成 + 进行中）
   - `totalBytes`: 总字节数（队列大小 + 已传输 + 正在传输的总大小）
   - `bytes`: 已传输字节数
   - `transfers`: 已完成传输数
   - `checks`: 已完成检查数

2. rclone 在同步过程中会动态设置队列信息（通过 `SetTransferQueue`, `SetCheckQueue`）

### 决策

**可以实现总进度信息**：
- 使用 `Stats.RemoteStats(false)` 获取完整的进度统计
- 提取 `totalTransfers`, `totalBytes` 作为总数
- 提取 `transfers`, `bytes` 作为已完成数
- **注意**: 总数会随着扫描进行而动态增加，不是一开始就固定的

**实现方案**：
- `JobProgressEvent` 新增 `filesTotal`, `bytesTotal` 字段
- 前端显示 "45/128 files" 格式
- 百分比计算 `bytes / bytesTotal * 100`

---

## 2. 当前传输文件详情

### 问题

如何获取正在传输的文件名称和单文件进度？

### 研究

查看 rclone 源码 `fs/accounting/stats.go`：

**注意**: `StatsInfo` **没有**公开的 `InProgress()` 方法来获取当前传输列表。

rclone 的 `StatsInfo` 内部结构：
```go
type StatsInfo struct {
    mu                sync.RWMutex
    startedTransfers  []*Transfer   // 私有字段 - 所有已开始的传输（包括进行中和已完成）
    transferring      *transferMap  // 私有字段
    // ...
}
```

`Transfer.Snapshot()` 返回的快照：
```go
type TransferSnapshot struct {
    Name        string
    Size        int64     // 文件总大小
    Bytes       int64     // 已传输字节
    StartedAt   time.Time
    CompletedAt time.Time
    Error       error
    What        string    // "transferring", "checking", etc.
    // ...
}
```

### 现有实现

项目在 `internal/rclone/sync.go` 中已使用**反射**获取私有字段：

```go
// getStatsInternals 使用 unsafe 反射访问 rclone 私有字段
func getStatsInternals(s *accounting.StatsInfo) (*sync.RWMutex, *[]*accounting.Transfer, error) {
    statsVal := reflect.ValueOf(s).Elem()
    
    // 获取 'mu' 互斥锁
    muField := statsVal.FieldByName("mu")
    mu := (*sync.RWMutex)(unsafe.Pointer(muField.UnsafeAddr()))
    
    // 获取 'startedTransfers' 切片
    transfersField := statsVal.FieldByName("startedTransfers")
    transfers := (*[]*accounting.Transfer)(unsafe.Pointer(transfersField.UnsafeAddr()))
    
    return mu, transfers, nil
}
```

已有单元测试 `TestPollStatsReflection` 验证与当前 rclone 版本的兼容性。

### 决策

**复用现有反射方案获取当前传输**：
1. 使用已有的 `getStatsInternals()` 获取 `startedTransfers` 切片
2. 过滤出 `!tr.IsDone()` 的传输作为"正在传输的文件"
3. 调用 `tr.Snapshot()` 获取 Name, Size, Bytes 构建 `TransferItem`
4. 通过独立的 `TransferProgressBus` 广播给前端

**优势**：
- 复用现有代码，无需引入新的 unsafe 代码
- 在同一个 `processStats()` 循环中处理已完成和进行中的传输
- 已有测试保证与 rclone 版本的兼容性

---

## 3. 配额信息完整性

### 问题

`ConnectionQuota` 当前只有 total, used, free，缺少 trashed, other, objects。

### 研究

查看现有实现 `internal/rclone/about.go`：

```go
func GetRemoteQuota(ctx context.Context, remoteName string) (*AboutInfo, error) {
    // ...
    return &AboutInfo{
        Total:   usage.Total,    // *int64
        Used:    usage.Used,     // *int64
        Trashed: usage.Trashed,  // *int64 - 已有但未暴露
        Other:   usage.Other,    // *int64 - 已有但未暴露
        Free:    usage.Free,     // *int64
        Objects: usage.Objects,  // *int64 - 已有但未暴露
    }, nil
}
```

`fs.Usage` 类型定义：
```go
type Usage struct {
    Total   *int64 `json:"total,omitempty"`
    Used    *int64 `json:"used,omitempty"`
    Trashed *int64 `json:"trashed,omitempty"`
    Other   *int64 `json:"other,omitempty"`
    Free    *int64 `json:"free,omitempty"`
    Objects *int64 `json:"objects,omitempty"`
}
```

### 决策

**扩展 GraphQL schema**：
- 所有字段改为可空类型 `BigInt`
- 后端已有数据，只需修改 resolver 映射

---

## 4. 日志清理策略

### 问题

如何实现按连接独立统计的日志清理？

### 研究

当前数据库结构：
```
Connection (1) --< Task (N) --< Job (N) --< JobLog (N)
```

日志查询需要通过 3 层关系：
```sql
SELECT jl.* FROM job_logs jl
JOIN jobs j ON jl.job_id = j.id
JOIN tasks t ON j.task_id = t.id
WHERE t.connection_id = ?
ORDER BY jl.time ASC
```

### 决策

**清理策略**：
1. 遍历所有连接
2. 对每个连接执行子查询统计日志数
3. 如果超过限制，删除最旧的日志

**SQL 实现**：
```sql
-- 获取连接的日志数
SELECT COUNT(*) FROM job_logs jl
JOIN jobs j ON jl.job_id = j.id
JOIN tasks t ON j.task_id = t.id
WHERE t.connection_id = ?;

-- 删除超出限制的最旧日志
DELETE FROM job_logs WHERE id IN (
    SELECT jl.id FROM job_logs jl
    JOIN jobs j ON jl.job_id = j.id
    JOIN tasks t ON j.task_id = t.id
    WHERE t.connection_id = ?
    ORDER BY jl.time ASC
    LIMIT ?
);
```

---

## 5. 定时任务调度

### 问题

如何实现日志清理的定时任务？

### 研究

当前 `Scheduler` 实现：
- 使用 `robfig/cron/v3`
- **专门用于同步任务调度**（通过 `AddTask(task *ent.Task)` 添加任务）
- 设计上与 Task 实体紧密耦合

### 决策

**使用独立的 cron 实例**：
1. 在 `LogCleanupService` 中创建独立的 `robfig/cron` 实例
2. 不修改现有 `Scheduler`（保持其专用于同步任务的职责）
3. `LogCleanupService` 提供 `Start(schedule)` 和 `Stop()` 方法管理生命周期

**理由**：
- Scheduler 设计专用于执行 Task，日志清理不是 Task
- 独立 cron 实例职责清晰，避免污染现有 Scheduler
- `robfig/cron` 轻量级，多实例无性能问题

---

---

## 6. 自动删除无活动作业

### 问题

如何判断作业是否"无活动"？删除作业时如何处理关联的日志记录？

### 研究

查看现有实现 `internal/core/ent/schema/job.go`：

```go
// Job schema 中定义了与 JobLog 的关系
func (Job) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("logs", JobLog.Type),
    }
}
```

查看 ent 的级联删除行为：
- ent ORM 默认支持通过边（Edge）定义的关系进行级联删除
- 当删除 Job 时，关联的 JobLog 会自动删除

### 决策

**"无活动"判定标准**：
- `filesTransferred = 0`（未传输任何文件）
- `bytesTransferred = 0`（未传输任何字节）
- `filesDeleted = 0`（未删除任何文件）
- `errorCount = 0`（未发生任何错误）
- `status = SUCCESS`（作业状态为成功完成）
- 注：`filesChecked` 不作为判断条件

**删除逻辑**：
- 在同步完成后检查是否满足删除条件
- 使用 JobService.DeleteJob() 方法删除作业
- 利用 ent ORM 的级联删除自动清理关联的 JobLog 记录

**失败作业保留**：
- 即使无活动，失败的作业也会保留
- 便于管理员排查问题

---

## Summary

| 研究项 | 决策 | 理由 |
|--------|------|------|
| 总文件数/字节数 | 使用 RemoteStats() 实现 | rclone 提供 totalTransfers/totalBytes |
| 当前传输详情 | 复用现有反射方案 | 项目已使用 getStatsInternals() 获取 startedTransfers |
| 配额完整字段 | 扩展 ConnectionQuota | 后端已有数据 |
| 日志清理 SQL | 子查询方式 | 支持 SQLite，性能可接受 |
| 定时任务 | 独立 cron 实例 | Scheduler 专用于同步任务，日志清理使用独立 cron |
| 自动删除无活动作业 | 同步完成后检查并删除 | 使用 ent 级联删除，逻辑简单 |

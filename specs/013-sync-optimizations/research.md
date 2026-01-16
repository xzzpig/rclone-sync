# Research: 同步任务多项优化

**Feature**: 013-sync-optimizations  
**Date**: 2026-01-16  
**Status**: Complete

## Research Summary

本文档记录了实现 6 个优化项所需的技术研究结果。

---

## 1. 任务完成 Toast 提醒

### 决策：使用现有 Toast 系统 + 全局 Job 进度订阅

**Rationale**: 
- 项目已有完整的 Toast 系统 (`web/src/components/ui/toast.tsx`)，基于 Kobalte 实现
- 已有全局 JobProgress 订阅 (`web/src/store/jobProgress.tsx`)
- 只需在订阅回调中检测任务完成状态，调用 toast 函数

**实现方案**:
1. 在 `jobProgress.tsx` 中监听 `status` 变化
2. 当 `status` 从 `RUNNING` 变为 `SUCCESS`/`FAILED`/`CANCELLED` 时触发 toast
3. Toast 样式：success (绿色) / error (红色) / warning (黄色)
4. Toast 内容：包含任务名称和状态

**Alternatives Considered**:
- 独立的通知系统 → 过度设计，现有 Toast 完全满足需求
- 浏览器 Notification API → 需要用户授权，且用户可能在应用内，Toast 更合适

---

## 2. 传输进度区分上传/下载

### 决策：在 TransferItem 类型中添加 direction 字段

**Rationale**:
- rclone 的 `Transfer` 对象可以通过上下文判断方向
- 对于单向同步 (UPLOAD/DOWNLOAD)，方向即为任务方向
- 对于双向同步 (BIDIRECTIONAL)，需要根据文件操作类型判断

**实现方案**:
1. GraphQL Schema: 在 `TransferItem` 添加 `direction: TransferDirection!` 字段
2. 新增枚举 `TransferDirection { UPLOAD, DOWNLOAD }`
3. 后端：在 `broadcastTransferProgress` 中根据任务方向和操作类型计算
4. 前端：根据 direction 显示 ↑ 或 ↓ 图标（使用 lucide-icons）

**Alternatives Considered**:
- 仅前端根据任务方向推断 → 双向同步时无法正确显示
- 使用不同颜色区分 → 图标更直观，符合 spec 要求

---

## 3. 同步时展示检查中的文件数

### 决策：在 JobProgressEvent 中添加 filesChecking 字段

**Rationale**:
- rclone 的 `accounting.StatsInfo` 提供了 `checking` 计数
- 可以通过现有的 `pollStats` 机制采集

**实现方案**:
1. GraphQL Schema: 在 `JobProgressEvent` 添加 `filesChecking: Int!`
2. 后端：在 `processStats` 中从 rclone stats 获取 checking 数量
3. 前端：在任务状态卡片中单独一行显示 "正在检查 X 个文件"

**技术细节**:
- rclone stats 结构中有 `Checking` 字段，直接读取即可
- 不需要反射，是公开 API

---

## 4. 允许禁用任务

### 决策：在 Task 实体添加 enabled 布尔字段

**Rationale**:
- 最简单直接的方案
- 禁用任务不会自动触发，但可以手动运行
- 需要在 scheduler 和 watcher 中检查此字段

**实现方案**:
1. Ent Schema: 在 `internal/core/db/schema/task.go` 添加 `field.Bool("enabled").Default(true)`
2. GraphQL Schema: 在 `Task` 类型添加 `enabled: Boolean!`
3. GraphQL Mutation: 添加 `setEnabled(id: ID!, enabled: Boolean!): Task!`
4. 后端逻辑:
   - `scheduler.go`: 执行定时任务前检查 `enabled`
   - `watcher.go`: 文件变更触发前检查 `enabled`
5. 前端：任务列表中显示启用/禁用图标按钮

**数据库迁移**:
- 新增字段 `enabled BOOLEAN DEFAULT TRUE`
- 使用 golang-migrate 版本化迁移

**Alternatives Considered**:
- 删除任务配置 → 用户需要重新创建，体验差
- 暂停状态字段 → enabled 更简洁，语义明确

---

## 5. 空任务清理优化（滚动替换）

### 决策：修改 auto_delete_empty_jobs 逻辑为滚动替换

**Rationale**:
- 现有逻辑：当前 Job 为空时删除当前 Job
- 新逻辑：当前 Job 完成后，检查并删除"上一个"空 Job
- 这样始终保留最新的执行记录，让用户确认系统在运行

**实现方案**:
1. 在 `internal/rclone/sync.go` 的 `RunTask` 结束处修改逻辑
2. 查询同一任务的上一个已完成 Job (按 end_time 排序)
3. 如果 Job N-1 是空任务 (filesTransferred=0, bytesTransferred=0, filesDeleted=0, errorCount=0, status=SUCCESS)，则删除它
4. 无论 Job N 是否为空，始终保留 Job N

**空任务判断条件**:
```go
func isEmptyJob(job *ent.Job) bool {
    return job.Status == model.JobStatusSuccess &&
           job.FilesTransferred == 0 &&
           job.BytesTransferred == 0 &&
           job.FilesDeleted == 0 &&
           job.ErrorCount == 0
}
```

**Alternatives Considered**:
- 保留最近 N 条记录 → 增加配置复杂度
- 只在有变动时删除旧空记录 → 新 spec 要求始终检查

---

## 6. 缓存数据库大小统计修复

### 决策：修改 GetDBSize 以包含 WAL 和 SHM 文件

**Rationale**:
- SQLite WAL 模式会生成 `.db-wal` 和 `.db-shm` 文件
- 当前 `GetDBSize` 只统计 `.db` 文件
- 导致用户看到的大小与实际磁盘占用不符

**实现方案**:
1. 修改 `internal/rclone/backend/metacache/cache_store.go` 的 `GetDBSize` 方法
2. 计算三个文件的总大小：`.db` + `.db-wal` + `.db-shm`
3. 对于不存在的文件，视为大小 0，不报错

**代码示例**:
```go
func (s *CacheStore) GetDBSize() (int64, error) {
    var totalSize int64
    
    files := []string{s.dbPath, s.dbPath + "-wal", s.dbPath + "-shm"}
    for _, file := range files {
        info, err := os.Stat(file)
        if err != nil {
            if os.IsNotExist(err) {
                continue // 文件不存在时跳过
            }
            return 0, err
        }
        totalSize += info.Size()
    }
    
    return totalSize, nil
}
```

**Alternatives Considered**:
- 使用 PRAGMA database_size → 只返回页面数，不是实际文件大小
- 定期执行 VACUUM → 会影响性能，不适合实时显示

---

## 7. 概览卡片数据分离加载

### 决策：拆分前端 GraphQL 查询

**Rationale**:
- 目前存储配额 (`quota`) 和缓存状态 (`cacheStatus`) 在同一个查询中，导致受限于 `quota` (rclone about) 的远程调用速度。
- 拆分为两个独立的查询可实现并行加载，让快速的本地缓存状态先展示，慢速的远程配额数据异步加载并显示 loading 状态。
- 符合 FR-030 “使用独立的 API 请求获取” 的要求。

**实现方案**:
1. **前端 Query 拆分**:
   - 在 `web/src/api/graphql/queries/connections.ts` 中更新 `ConnectionGetQuotaQuery` (仅包含 `quota`)。
   - 在 `web/src/api/graphql/queries/connections.ts` 中新增 `ConnectionGetCacheStatusQuery` (仅包含 `cacheStatus`)。
2. **Overview.tsx 修改**:
   - 使用两个独立的 `createQuery` 调用。
   - 保持原来的 `Skeleton` 逻辑，但仅绑定到 `quota` 的加载状态。
   - `CacheStatusCard` 绑定到 `cacheStatus` 的加载状态。

**Alternatives Considered**:
- **使用 GraphQL @defer**: 需要后端框架和中间件支持，实现较复杂，且拆分 Query 已经能完全满足需求。
- **后端聚合异步返回**: 增加后端复杂度，且 GraphQL 天生支持独立查询。

---

## 技术依赖确认

| 依赖 | 当前状态 | 需要操作 |
|-----|---------|---------|
| Toast 组件 | ✅ 已存在 | 直接使用 |
| JobProgress 订阅 | ✅ 已存在 | 扩展字段 |
| TransferProgress 订阅 | ✅ 已存在 | 扩展 TransferItem 类型 |
| Task Ent Schema | ✅ 已存在 | 添加 enabled 字段 |
| golang-migrate | ✅ 已配置 | 创建新迁移文件 |
| rclone stats API | ✅ 可用 | 读取 Checking 字段 |

---

*Research completed: All NEEDS CLARIFICATION resolved*

# Implementation Plan: UI Detail Improvements

**Feature Branch**: `008-ui-detail-improvements`  
**Created**: 2024-12-24  
**Status**: Planning

---

## Technical Context

| Category | Current Implementation | Changes Needed |
|----------|----------------------|----------------|
| **API Layer** | GraphQL with gqlgen | 扩展 schema 添加新字段和类型 |
| **Sync Engine** | `internal/rclone/sync.go` 使用 rclone accounting | 扩展进度信息获取 |
| **Quota API** | `internal/rclone/about.go` 返回完整 fs.Usage | 暴露更多字段到 GraphQL |
| **Configuration** | `internal/core/config/config.go` 使用 Viper | 添加日志配置项 |
| **Job Service** | `internal/core/services/job_service.go` | 添加日志清理方法 |
| **Frontend** | SolidJS + urql + Tailwind | 更新 UI 组件 |

---

## Constitution Check

| Principle | Compliance | Notes |
|-----------|------------|-------|
| I. Rclone-First | ✅ | 所有配额和进度信息通过 rclone API 获取 |
| II. Web-First | ✅ | 所有功能通过 Web UI 展示 |
| III. TDD (Backend) | ✅ | 需要为所有后端变更编写测试 |
| VI. Modern Component | ✅ | 使用 SolidJS 组件架构 |
| IX. i18n | ✅ | 所有新 UI 文本需要翻译 |
| X. Schema-First | ✅ | 先定义 GraphQL Schema，再生成代码 |

---

## Phase 0: Research

### 研究项 1: rclone 总文件数/字节数获取

**问题**: 当前 `JobProgressEvent` 只有 `filesTransferred` 和 `bytesTransferred`，缺少总数信息。

**研究结果**:
- rclone 的 `accounting.Stats.RemoteStats(false)` 返回完整的进度统计
- 包含 `totalTransfers`, `totalBytes` 作为总数
- 包含 `transfers`, `bytes` 作为已完成数
- **决策**: 使用 `RemoteStats()` 实现，新增 `filesTotal`, `bytesTotal` 字段
- **注意**: 总数会随着扫描进行而动态增加，不是一开始就固定的

### 研究项 2: 当前传输文件详情

**问题**: 如何获取正在传输的文件名和单文件进度？

**研究结果**:
- rclone `StatsInfo` **没有**公开的 `InProgress()` 方法
- 项目已在 `internal/rclone/sync.go` 中使用**反射**获取私有字段 `startedTransfers`
- 通过 `getStatsInternals()` 获取传输列表，过滤 `!tr.IsDone()` 得到进行中的传输
- `accounting.Transfer.Snapshot()` 返回 `TransferSnapshot`，包含 Name, Size, Bytes
- **决策**: 复用现有反射方案，创建独立的 `transferProgress` Subscription

### 研究项 3: 配额信息字段

**问题**: 如何获取回收站、其他空间、对象数量？

**研究结果**:
- `rclone/about.go` 的 `GetRemoteQuota()` 已返回完整的 `fs.Usage`
- 包含: Total, Used, Free, Trashed, Other, Objects（都是 `*int64`，可能为 nil）
- **决策**: 扩展 `ConnectionQuota` GraphQL 类型，添加可空字段

---

## Phase 1: Design & Implementation Plan

### 任务 1: 扩展 ConnectionQuota (P2 - Story 3)

**后端变更**:
1. 修改 `internal/api/graphql/schema/connection.graphql`:
   ```graphql
   type ConnectionQuota {
     total: BigInt
     used: BigInt
     free: BigInt
     trashed: BigInt          # 新增
     other: BigInt            # 新增
     objects: BigInt          # 新增
   }
   ```

2. 修改 `internal/api/graphql/resolver/connection.resolvers.go`:
   - 更新 `Quota()` resolver 返回完整字段

**前端变更**:
1. 更新 `web/src/api/graphql/queries/connections.ts` 查询
2. 更新 `web/src/modules/connections/views/Overview.tsx`:
   - 添加 Trashed、Other、Objects 显示
   - 添加不可用字段的优雅降级

**测试**:
- 后端: `internal/api/graphql/resolver/connection_test.go`
- 前端: 手动测试

**i18n 新增 keys**:
- `overview.trashed`, `overview.other`, `overview.objects`, `overview.quotaUnavailable`

---

### 任务 2a: 扩展 JobProgressEvent 总进度 (P1 - Story 1)

**后端变更**:
1. 修改 `internal/api/graphql/schema/job.graphql`:
   ```graphql
   type JobProgressEvent {
     # ... 现有字段
     filesTotal: Int!              # 新增 - 总文件数（队列+已完成+进行中）
     bytesTotal: BigInt!           # 新增 - 总字节数
   }
   ```

2. 修改 `internal/rclone/sync.go`:
   - 在 `processStats()` 中调用 `accounting.Stats.RemoteStats(false)`
   - 提取 `totalTransfers`, `totalBytes` 作为总数
   - 更新 `broadcastJobUpdate()` 调用

3. 修改 `internal/api/graphql/model/models_gen.go`（自动生成）

**前端变更**:
1. 更新 `web/src/api/graphql/queries/subscriptions.ts` - 添加 filesTotal/bytesTotal
2. 更新 `web/src/modules/connections/views/Overview.tsx`:
   - 显示 "45/128 files (35%)" 格式的进度

**测试**:
- 后端: `internal/rclone/sync_test.go`, `internal/api/graphql/resolver/job_test.go`

---

### 任务 2b: 新增 transferProgress Subscription (P1 - Story 2)

**后端变更**:
1. 修改 `internal/api/graphql/schema/job.graphql`:
   ```graphql
   """
   传输列表项 - 表示一个正在传输的文件
   前端以列表形式展示所有正在传输的文件
   """
   type TransferItem {
     name: String!        # 文件名称（含路径）
     size: BigInt!        # 文件总大小（字节）
     bytes: BigInt!       # 已传输大小（字节）
     # 前端计算: percentage = bytes / size * 100
   }
   
   type TransferProgressEvent {
     jobId: ID!
     taskId: ID!
     connectionId: ID!
     transfers: [TransferItem!]!
   }
   
   extend type Subscription {
     transferProgress(
       connectionId: ID
       taskId: ID
       jobId: ID
     ): TransferProgressEvent!
   }
   ```

2. 新建 `internal/api/graphql/subscription/transfer_progress_bus.go`:
   - 参考 `job_progress_bus.go` 实现
   - 支持按 connectionId/taskId/jobId 筛选
   - **增量推送机制**：
     - 维护上一次推送的传输列表状态
     - 仅推送有变化的传输项（新增、进度更新）
     - 每个文件传输完成时推送一次（bytes == size）
     - 前端根据 name 合并/更新传输列表，bytes == size 时移除该项

3. 修改 `internal/rclone/sync.go`:
   - 复用现有 `getStatsInternals()` 获取 `startedTransfers`
   - 过滤 `!tr.IsDone()` 的传输，调用 `Snapshot()` 构建 `TransferItem` 列表
   - 对比上次状态，筛选变化的项
   - 每个传输完成时额外触发一次推送
   - 通过 `TransferProgressBus` 广播增量数据

4. 修改 `internal/api/graphql/resolver/subscription.resolvers.go`:
   - 新增 `TransferProgress` resolver

**前端变更**:
1. 更新 `web/src/api/graphql/queries/subscriptions.ts` - 添加 transferProgress subscription
2. 更新 `web/src/modules/connections/views/Overview.tsx`:
   - **以列表形式展示**当前连接下所有活跃传输任务
   - 每个列表项显示:
     - 文件名称（含路径）
     - 文件大小（人类可读格式，如 128 MB）
     - 已传输大小（人类可读格式，如 45 MB）
     - 传输进度百分比（如 35%）
     - 可视化进度条
   - 空状态处理：无传输时显示 "暂无传输中的文件"

**测试**:
- 后端: `internal/api/graphql/subscription/transfer_progress_bus_test.go`
- 后端: `internal/api/graphql/resolver/subscription_test.go`

**i18n 新增 keys**:
- `overview.activeTransfers`, `overview.transferProgress`, `common.noActiveTransfers`

---

### 任务 3: 日志数量限制配置 (P2 - Story 4)

**后端变更**:
1. 修改 `internal/core/config/config.go`:
   ```go
   type Config struct {
     // ... 现有字段
     Log struct {
       Level               string `mapstructure:"level"`
       MaxLogsPerConnection int    `mapstructure:"max_logs_per_connection"` // 新增
       CleanupSchedule     string `mapstructure:"cleanup_schedule"`         // 新增
     } `mapstructure:"log"`
   }
   ```
   - 默认值: `max_logs_per_connection = 1000`, `cleanup_schedule = "0 * * * *"`

2. 新建 `internal/core/services/log_cleanup_service.go`:
   ```go
   type LogCleanupService struct {
     client     *ent.Client
     logger     *zap.Logger
     maxLogs    int
     cron       *cron.Cron  // 独立的 cron 实例
   }
   
   func NewLogCleanupService(...) *LogCleanupService
   func (s *LogCleanupService) Start(schedule string) error  // 启动定时任务
   func (s *LogCleanupService) Stop()                        // 停止定时任务
   func (s *LogCleanupService) CleanupLogs(ctx context.Context) error
   func (s *LogCleanupService) CleanupLogsForConnection(ctx context.Context, connectionID uuid.UUID) error
   ```
   - 使用独立的 `robfig/cron` 实例，不修改现有 Scheduler（Scheduler 专用于同步任务）

3. 修改 `internal/core/ports/interfaces.go`:
   - 添加 `LogCleanupService` 接口

**测试**:
- `internal/core/services/log_cleanup_service_test.go`

---

### 任务 4: 日志清理定时任务执行 (P2 - Story 4)

**后端变更**:
1. 修改 `cmd/cloud-sync/serve.go`:
   - 初始化 `LogCleanupService`
   - 调用 `Start(schedule)` 启动独立的 cron 定时任务
   - 在 shutdown 时调用 `Stop()` 清理资源

2. 修改 `internal/core/services/job_service.go`:
   - 添加 `DeleteOldLogsForConnection(ctx, connectionID, keepCount)` 方法
   - **必须使用 ent API 实现**，不能使用原生 SQL（保持与项目其他数据库操作一致）
   - 逻辑说明（以 SQL 形式描述）:
     ```sql
     -- 保留最新的 keepCount 条，删除其余所有（使用 OFFSET 跳过要保留的）
     DELETE FROM job_logs 
     WHERE id IN (
       SELECT id FROM job_logs 
       WHERE job_id IN (SELECT id FROM jobs WHERE task_id IN (SELECT id FROM tasks WHERE connection_id = ?))
       ORDER BY time DESC   -- 按时间降序，最新的在前
       OFFSET keepCount     -- 跳过要保留的最新 keepCount 条
     )
     ```
   - ent 实现要点：
     ```go
     // 1. 查询要删除的日志 ID（跳过最新的 keepCount 条）
     idsToDelete, err := client.JobLog.Query().
         Where(joblog.HasJobWith(
             job.HasTaskWith(
                 task.ConnectionID(connectionID),
             ),
         )).
         Order(joblog.ByTime(sql.OrderDesc())).  // 按时间降序
         Offset(keepCount).                       // 跳过要保留的
         Select(joblog.FieldID).                  // 只查 ID
         IDs(ctx)
     
     // 2. 批量删除（如果有需要删除的）
     if len(idsToDelete) > 0 {
         _, err = client.JobLog.Delete().
             Where(joblog.IDIn(idsToDelete...)).
             Exec(ctx)
     }
     ```
   - 优势：省去 COUNT 查询，逻辑更直观（保留最新 N 条，删除其余）

**测试**:
- `internal/core/services/job_service_test.go`

---

### 任务 5: 层级日志级别配置 (P2 - Story 6)

**背景**: 管理员需要能够对不同模块的日志输出进行精细化控制。全局日志级别可能产生过多日志或遗漏关键信息，按模块层级设置日志级别可以在不影响其他模块的情况下，对目标模块进行详细日志输出。

**层级匹配规则** (匹配过程**区分大小写**):

| 日志名称 | 配置项 | 匹配优先级 | 说明 |
|----------|--------|------------|------|
| `core.db.query` | `core.db.query` | 1 (最高) | 精确匹配 |
| `core.db.query` | `core.db` | 2 | 匹配父级 |
| `core.db.query` | `core` | 3 | 匹配更高父级 |
| `core.db.query` | 全局 level | 4 (最低) | 使用默认全局级别 |

**配置示例**:

```toml
[log]
level = "info"                    # 全局日志级别

[log.levels]
"core.db" = "debug"               # core.db 及其子模块使用 debug 级别
"core.scheduler" = "warn"         # core.scheduler 及其子模块使用 warn 级别
"rclone" = "error"                # rclone 及其子模块使用 error 级别
```

**后端变更**:

1. **修改 `internal/core/config/config.go`**:
   ```go
   type Config struct {
     // ... 现有字段
     Log struct {
       Level                string            `mapstructure:"level"`
       Levels               map[string]string `mapstructure:"levels"`  // 新增：层级日志级别配置
       MaxLogsPerConnection int               `mapstructure:"max_logs_per_connection"`
       CleanupSchedule      string            `mapstructure:"cleanup_schedule"`
     } `mapstructure:"log"`
   }
   ```
   - 在 `setDefaults()` 中添加 `viper.SetDefault("log.levels", map[string]string{})`

2. **新建 `internal/core/logger/level.go`**:
   ```go
   package logger

   import (
       "strings"
       "sync"
       "go.uber.org/zap/zapcore"
   )

   // levelCache 使用 sync.Map 实现无锁并发缓存
   // Key: logger name (string), Value: zapcore.Level
   var levelCache sync.Map

   // levelConfig 存储层级日志级别配置
   var (
       levelConfigMu    sync.RWMutex
       levelConfigMap   map[string]string  // 配置的层级级别映射
       globalLevel      zapcore.Level      // 全局默认级别
   )

   // InitLevelConfig 初始化层级日志级别配置
   // 在 InitLogger 中调用，传入配置文件中的 levels map
   func InitLevelConfig(levels map[string]string, defaultLevel zapcore.Level) {
       levelConfigMu.Lock()
       defer levelConfigMu.Unlock()
       levelConfigMap = levels
       globalLevel = defaultLevel
       // 清空缓存，因为配置已变更
       levelCache = sync.Map{}
   }

   // GetLevelForName 根据日志名称查找最匹配的日志级别
   // 使用按需缓存策略：首次计算后缓存，后续直接查表
   // 匹配过程区分大小写
   func GetLevelForName(name string) zapcore.Level {
       // 1. 先查缓存
       if cached, ok := levelCache.Load(name); ok {
           return cached.(zapcore.Level)
       }

       // 2. 计算匹配的级别
       level := computeLevelForName(name)

       // 3. 存入缓存
       levelCache.Store(name, level)

       return level
   }

   // computeLevelForName 计算日志名称对应的级别（不使用缓存）
   func computeLevelForName(name string) zapcore.Level {
       levelConfigMu.RLock()
       defer levelConfigMu.RUnlock()

       if levelConfigMap == nil || len(levelConfigMap) == 0 {
           return globalLevel
       }

       // 空字符串直接返回全局级别
       if name == "" {
           return globalLevel
       }

       // 1. 精确匹配
       if levelStr, ok := levelConfigMap[name]; ok {
           if level, err := ParseLevel(levelStr); err == nil {
               return level
           }
       }

       // 2. 按 "." 拆分后逐级向上匹配父级
       parts := strings.Split(name, ".")
       for i := len(parts) - 1; i > 0; i-- {
           prefix := strings.Join(parts[:i], ".")
           if levelStr, ok := levelConfigMap[prefix]; ok {
               if level, err := ParseLevel(levelStr); err == nil {
                   return level
               }
           }
       }

       // 3. 返回全局级别
       return globalLevel
   }

   // ParseLevel 解析日志级别字符串（不区分大小写）
   // 支持: debug, info, warn, error
   func ParseLevel(levelStr string) (zapcore.Level, error) {
       var level zapcore.Level
       err := level.UnmarshalText([]byte(strings.ToLower(levelStr)))
       return level, err
   }
   ```

3. **修改 `internal/core/logger/logger.go`**:
   ```go
   // 修改 InitLogger 函数签名，添加 levels 参数
   func InitLogger(environment Environment, logLevel LogLevel, levels map[string]string) {
       // ... 现有代码 ...

       // 在创建 logger 后，初始化层级级别配置
       InitLevelConfig(levels, getZapLevel(string(logLevel)))
   }

   // 修改 Named 函数，应用层级日志级别
   func Named(name string) *zap.Logger {
       baseLogger := Get()
       namedLogger := baseLogger.Named(name)

       // 获取该名称对应的日志级别
       level := GetLevelForName(name)

       // 使用 zap.WrapCore 包装核心，应用自定义级别过滤
       return namedLogger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
           return &levelFilterCore{
               Core:  core,
               level: level,
           }
       }))
   }

   // levelFilterCore 是一个包装的 zapcore.Core，用于过滤日志级别
   type levelFilterCore struct {
       zapcore.Core
       level zapcore.Level
   }

   // Enabled 检查给定级别是否应该被记录
   func (c *levelFilterCore) Enabled(lvl zapcore.Level) bool {
       return lvl >= c.level && c.Core.Enabled(lvl)
   }

   // Check 检查日志条目是否应该被记录
   func (c *levelFilterCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
       if c.Enabled(ent.Level) {
           return c.Core.Check(ent, ce)
       }
       return ce
   }

   // With 创建带有额外字段的新 Core
   func (c *levelFilterCore) With(fields []zapcore.Field) zapcore.Core {
       return &levelFilterCore{
           Core:  c.Core.With(fields),
           level: c.level,
       }
   }
   ```

4. **修改 `cmd/cloud-sync/serve.go`**:
   ```go
   // 更新 InitLogger 调用，传入 levels 配置
   logger.InitLogger(
       logger.Environment(cfg.App.Environment),
       logger.LogLevel(cfg.Log.Level),
       cfg.Log.Levels,  // 新增参数
   )
   ```

**实现要点**:
- **大小写敏感**: 日志名称匹配区分大小写，配置键必须与代码中定义的 Logger Name 完全一致
- **级别值不区分大小写**: `DEBUG`, `Debug`, `debug` 都有效
- **无锁并发缓存**: 使用 `sync.Map` 实现，避免锁竞争
- **按需缓存**: 首次调用 `GetLevelForName` 时计算并缓存，后续直接返回缓存值
- **仅支持四级**: debug, info, warn, error（无 trace/fatal）
- **无效级别处理**: 如果配置了无效的日志级别值，使用全局级别并记录警告

**测试**:
- `internal/core/logger/level_test.go` - 测试层级匹配算法:
  - 精确匹配测试
  - 父级匹配测试
  - 多级父级匹配测试
  - 全局级别回退测试
  - 大小写敏感测试
  - 空字符串名称测试
  - 无效级别值测试
  - 缓存行为测试
- `internal/core/logger/logger_test.go` - 测试 Named Logger 级别控制:
  - Named logger 使用正确级别
  - 级别过滤生效测试

**无前端变更**: 此功能仅涉及后端配置和日志输出。

---

### 任务 6: 概览页展示进行中的任务列表 (P1 - Story 5)

**背景**: 用户在概览页只能看到文件同步详细进度，缺少对"正在运行哪些任务"的整体视图。需要增加一个卡片来展示当前连接下正在进行中的作业（Job）列表。

**前端变更**:
1. 新建 `web/src/modules/connections/components/RunningJobsCard.tsx`:
   ```tsx
   // 进行中任务卡片组件
   // 功能：
   // - 复用现有 jobProgress subscription，按 connectionId 筛选
   // - 使用 jobProgressStore 获取实时进度数据
   // - 以卡片形式展示任务列表
   // - 无进行中任务时隐藏整个卡片
   // - 点击任务项跳转到日志页面并筛选该任务
   ```

2. 卡片内每个任务项显示:
   - **任务名称**: 从关联的 Task 获取
   - **作业状态**: 状态徽章（如 "同步中"）
   - **开始时间**: 作业开始执行的时间（如 "10:30:45"）
   - **文件进度**: 文件数进度（如 "45/128 files"）
   - **字节进度**: 字节数进度（如 "256 MB / 1.2 GB"）
   - **进度条**: 以已传输字节数为基准的可视化进度条（显示百分比，如 35%）

3. 修改 `web/src/modules/connections/views/Overview.tsx`:
   - 集成 `RunningJobsCard` 组件
   - 放置在合适的位置（如 Storage Usage 卡片附近）

**实现要点**:
- **无需后端变更**: 复用现有 `jobProgress` subscription
- 使用 `jobProgressStore` 来获取连接下的实时作业进度
- 筛选 `status === 'RUNNING'` 的作业
- **无进行中任务时隐藏整个卡片**（而非显示空状态）
- 任务完成后自动从列表中移除（由 subscription 自动处理）
- **点击任务项跳转到日志页面（Log）并自动筛选该任务的日志**:
  - 使用 `useNavigate` 跳转到 `/connections/:connectionId/log?taskId=:taskId`
  - Log 页面读取 URL 参数并设置筛选条件

**i18n 新增 keys**:
- `overview.runningJobs`: "Running Jobs" / "进行中的任务"
- 开始时间复用现有翻译 `common.startedAt`

**测试**:
- 前端: 手动测试（启动同步任务，验证卡片显示、实时更新、隐藏逻辑和点击跳转）

---

### 任务 7: 自动删除无活动作业 (P2 - Story 7)

**背景**: 对于定时执行的同步任务，如果源和目标已经完全同步，每次执行都会产生一个"无活动"作业记录。这些无活动的作业记录会随着时间积累，占用数据库空间并降低作业历史的可读性。

**"无活动"判定标准**:
- 传输文件数为 0（filesTransferred = 0）
- 传输字节数为 0（bytesTransferred = 0）
- 删除文件数为 0（filesDeleted = 0）
- 错误数为 0（errorCount = 0）
- 作业状态为成功完成（status = SUCCESS）
- 注：filesChecked 不作为判断条件，即使检查了文件但无传输也视为"无活动"

**后端变更**:

1. **修改 `internal/core/config/config.go`**:
   ```go
   type Config struct {
     // ... 现有字段
     Job struct {
       AutoDeleteEmptyJobs bool `mapstructure:"auto_delete_empty_jobs"` // 新增
     } `mapstructure:"job"`
   }
   ```
   - 在 `setDefaults()` 中添加 `viper.SetDefault("job.auto_delete_empty_jobs", false)`

2. **修改 `internal/rclone/sync.go`**:
   - 在同步完成后检查是否需要删除无活动作业
   - 添加 `shouldDeleteEmptyJob()` 辅助函数判断作业是否为"无活动"
   - 判定逻辑：
     ```go
     func shouldDeleteEmptyJob(job *ent.Job, autoDelete bool) bool {
         // 如果未启用自动删除，返回 false
         if !autoDelete {
             return false
         }
         // 如果作业失败，保留作业记录
         if job.Status != job.StatusSuccess {
             return false
         }
         // 如果有传输活动，保留作业记录
         if job.FilesTransferred > 0 || job.BytesTransferred > 0 {
             return false
         }
         // 满足无活动条件，删除作业
         return true
     }
     ```

3. **修改 `internal/core/services/job_service.go`**:
   - 确保 `DeleteJob(ctx, jobID)` 方法存在且正常工作
   - 利用 ent ORM 的级联删除自动删除关联的 JobLog 记录

**实现流程**:

```
[Sync 完成]
    ↓
[检查 autoDeleteEmptyJobs 配置]
    ↓ (true)
[检查作业状态]
    ↓ (SUCCESS)
[检查传输活动]
    ↓ (filesTransferred = 0 && bytesTransferred = 0)
[调用 JobService.DeleteJob()]
    ↓
[数据库级联删除关联日志]
    ↓
[记录 DEBUG 日志]
```

**实现要点**:
- 利用 Ent ORM 的级联删除（在 schema 中配置为 `entsql.OnDelete(entsql.Cascade)`），删除 Job 时自动删除关联的 JobLog
- 无需显式事务，级联删除在数据库层面原子执行
- 删除过程中的错误应记录警告日志，但不能中断后续流程

**测试**:
- `internal/rclone/sync_test.go` - 测试无活动作业自动删除逻辑:
  - 启用配置 + 成功 + 无活动 → 作业被删除
  - 启用配置 + 成功 + 有活动 → 作业保留
  - 启用配置 + 失败 + 无活动 → 作业保留
  - 禁用配置 + 成功 + 无活动 → 作业保留
  - 删除失败时记录警告日志且不中断流程
- `internal/core/services/job_service_test.go` - 测试 DeleteJob 方法

**无前端变更**: 此功能仅涉及后端配置和同步逻辑。

---

### 任务 8: JOB 记录并展示更多状态信息 (P1 - Story 8)

**背景**: 用户需要在作业执行过程中看到更完整的状态信息，包括删除的文件数和错误数。这些信息对于了解同步进度和排查问题非常重要。

**UI 展示规格** (来自澄清 2025-12-27):
- **删除数、错误数**: 在作业列表页面表格中作为独立列展示
- **零值显示**: 删除数和错误数为 0 时显示 "0"，保持表格列的一致性和可读性
- **实时更新**: 作业进行中时，删除数、错误数通过 Subscription 实时更新，与文件进度/字节进度一致
- **醒目显示**: 错误数 > 0 时以红色徽章形式显示，便于用户快速识别有问题的作业

**数据库变更**:

1. **修改 `internal/core/ent/schema/job.go`**:
   ```go
   // 新增字段
   field.Int("files_deleted").Default(0),    // 删除的文件数
   field.Int("error_count").Default(0),      // 错误数量
   ```
   注意：现有的 `errors` 字段是 `String` 类型用于错误信息文本，新增 `error_count` 是数量

2. **运行 `go generate ./internal/core/ent`** 重新生成 ent 代码

3. **生成数据库迁移**: 添加 `files_deleted` 和 `error_count` 列

**后端变更**:

1. **修改 `internal/api/graphql/schema/job.graphql`**:
   ```graphql
   type Job {
     # ... 现有字段
     filesDeleted: Int!     # 新增 - 删除的文件数
     errorCount: Int!       # 新增 - 错误数量（注意：与 errors: String 区分）
   }
   
   type JobProgressEvent {
     # ... 现有字段
     filesDeleted: Int!     # 新增 - 删除的文件数
     errorCount: Int!       # 新增 - 错误数量
   }
   ```

2. **修改 `internal/rclone/sync.go`**:
   - 在 `processStats()` 中获取额外的统计信息：
     ```go
     // 通过 accounting.StatsInfo 直接获取
     filesDeleted := stats.GetDeletes()    // 删除的文件数
     errorCount := stats.GetErrors()       // 错误数
     ```
   - 更新 `broadcastJobUpdate()` 调用，填充新字段
   - 在作业完成时持久化 `filesDeleted` 和 `errorCount` 到数据库

3. **运行 `go generate ./...`** 重新生成 GraphQL 代码

**前端变更**:

1. **更新 `web/src/api/graphql/queries/subscriptions.ts`**:
   - 在 jobProgress subscription 中添加 filesDeleted、errorCount 字段

2. **更新 `web/src/api/graphql/queries/jobs.ts`**:
   - 在 Job 查询中添加 filesDeleted、errorCount 字段

3. **更新 `web/src/modules/connections/views/History.tsx`**:
   - 在作业列表**表格中新增两列**：删除数、错误数
   - 删除数列：显示数字（0、15 等），值为 0 时显示 "0"
   - 错误数列：显示数字，当值 > 0 时使用**红色徽章**（Badge variant="destructive"）醒目显示

**数据源映射**：

| GraphQL 字段 | 数据源 | 持久化 |
|-------------|--------|--------|
| `Job.filesDeleted` | `accounting.StatsInfo.GetDeletes()` | ✅ 作业完成时写入 DB |
| `Job.errorCount` | `accounting.StatsInfo.GetErrors()` | ✅ 作业完成时写入 DB |
| `JobProgressEvent.filesDeleted` | `accounting.StatsInfo.GetDeletes()` | ❌ 实时推送 |
| `JobProgressEvent.errorCount` | `accounting.StatsInfo.GetErrors()` | ❌ 实时推送 |

**边缘情况处理**：
- `filesDeleted = 0` 时显示 "0"（保持表格列一致性）
- `errorCount = 0` 时显示 "0"（保持表格列一致性）
- `errorCount > 0` 时，错误数以红色徽章形式醒目显示

**测试**:
- 后端: `internal/rclone/sync_test.go` - 测试 StatsInfo 字段获取逻辑
- 后端: `internal/api/graphql/resolver/subscription_test.go` - 测试 subscription 返回新字段
- 后端: `internal/api/graphql/resolver/job_test.go` - 测试 Job 查询返回新字段

**i18n 新增 keys**:
- `job.filesDeleted`: "Deleted" / "已删除"
- `job.errorCount`: "Errors" / "错误数"

---

## Implementation Order

| 优先级 | 任务 | 预估工时 | 依赖 |
|--------|------|----------|------|
| P2 | 任务 1: 扩展 ConnectionQuota | 2h | 无 |
| P1 | 任务 2a: 扩展 JobProgressEvent 总进度 | 2h | 无 |
| P1 | 任务 2b: 新增 transferProgress Subscription | 3h | 无 |
| P2 | 任务 3: 日志数量限制配置 | 2h | 无 |
| P2 | 任务 4: 日志清理定时任务 | 3h | 任务 3 |
| P2 | 任务 5: 层级日志级别配置 | 3h | 无 |
| P1 | 任务 6: 概览页展示进行中的任务列表 | 2h | 无（复用现有 subscription） |
| P2 | 任务 7: 自动删除无活动作业 | 2h | 无 |
| P1 | 任务 8: JOB 记录并展示更多状态信息 | 3h | 无 |

**总预估工时**: 22 小时

---

## File Changes Summary

### Backend (Go)

| 文件 | 变更类型 | 描述 |
|------|----------|------|
| `internal/api/graphql/schema/connection.graphql` | 修改 | 扩展 ConnectionQuota 类型 |
| `internal/api/graphql/schema/job.graphql` | 修改 | 添加 TransferItem, TransferProgressEvent 类型和 transferProgress subscription |
| `internal/api/graphql/resolver/connection.resolvers.go` | 修改 | 更新 Quota resolver |
| `internal/api/graphql/resolver/subscription.resolvers.go` | 修改 | 新增 TransferProgress resolver |
| `internal/api/graphql/subscription/transfer_progress_bus.go` | 新建 | 传输进度事件总线，支持按 connectionId/taskId/jobId 筛选 |
| `internal/rclone/sync.go` | 修改 | 获取 RemoteStats，复用 getStatsInternals 获取传输详情，广播传输进度 |
| `internal/core/config/config.go` | 修改 | 添加日志配置（包括层级日志级别配置） |
| `internal/core/logger/level.go` | 新建 | 层级日志级别匹配算法实现 |
| `internal/core/logger/logger.go` | 修改 | 支持按名称层级设置日志级别 |
| `internal/core/services/log_cleanup_service.go` | 新建 | 日志清理服务 |
| `internal/core/services/job_service.go` | 修改 | 添加日志删除方法 |
| `internal/core/ports/interfaces.go` | 修改 | 添加新接口 |
| `cmd/cloud-sync/serve.go` | 修改 | 初始化新服务，传入日志级别配置 |

### Frontend (TypeScript/SolidJS)

| 文件 | 变更类型 | 描述 |
|------|----------|------|
| `web/src/api/graphql/queries/connections.ts` | 修改 | 更新 quota 查询 |
| `web/src/api/graphql/queries/subscriptions.ts` | 修改 | 更新 jobProgress subscription |
| `web/src/modules/connections/views/Overview.tsx` | 修改 | 显示完整配额、活跃传输和进行中任务卡片 |
| `web/src/modules/connections/views/History.tsx` | 修改 | 显示作业传输详情 |
| `web/src/modules/connections/components/RunningJobsCard.tsx` | 新建 | 进行中任务列表卡片组件 |
| `web/project.inlang/messages/en.json` | 修改 | 添加英文翻译 |
| `web/project.inlang/messages/zh-CN.json` | 修改 | 添加中文翻译 |

### Tests

| 文件 | 变更类型 |
|------|----------|
| `internal/api/graphql/resolver/connection_test.go` | 修改 |
| `internal/api/graphql/resolver/job_test.go` | 修改 |
| `internal/api/graphql/resolver/subscription_test.go` | 修改 |
| `internal/api/graphql/subscription/transfer_progress_bus_test.go` | 新建 |
| `internal/rclone/sync_test.go` | 修改 |
| `internal/core/services/log_cleanup_service_test.go` | 新建 |
| `internal/core/services/job_service_test.go` | 修改 |
| `internal/core/logger/level_test.go` | 新建 |
| `internal/core/logger/logger_test.go` | 修改 |

---

## Generated Artifacts

- [x] plan.md (本文件)
- [x] research.md
- [x] data-model.md
- [x] contracts/schema.graphql
- [x] quickstart.md

# Data Model: UI Detail Improvements

**Feature Branch**: `008-ui-detail-improvements`  
**Created**: 2024-12-24

---

## Entity Changes

### Job 表新增字段 (User Story 8)

| 字段名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `files_deleted` | `int` | `0` | 同步过程中删除的文件数量 |
| `error_count` | `int` | `0` | 同步过程中发生的错误数量 |

**注意**: 
- 现有的 `errors` 字段（类型 `string`，可空）用于存储错误信息文本，新增的 `error_count` 是错误数量
- 这两个字段在作业完成时从 `accounting.StatsInfo` 获取并持久化到数据库

### Ent Schema 变更

**文件**: `internal/core/ent/schema/job.go`

```go
// 现有字段
field.Int64("bytes_transferred").Default(0),
field.Int("files_transferred").Default(0),
field.String("errors").Optional().Nillable(),

// 新增字段
field.Int("files_deleted").Default(0),
field.Int("error_count").Default(0),
```

---

## GraphQL Schema Changes

详细定义请参考 [contracts/schema.graphql](./contracts/schema.graphql)

### 变更概览

| 类型 | 变更类型 | 说明 |
|------|----------|------|
| `Job` | 扩展 | 新增 `filesDeleted`, `errorCount`（数据库持久化） |
| `ConnectionQuota` | 扩展 | 所有字段改为可空；新增 `trashed`, `other`, `objects` |
| `TransferItem` | 新增 | 当前传输文件项（列表项），从 rclone `TransferSnapshot` 映射，包含文件名、文件大小、已传输大小、传输进度百分比 |
| `TransferProgressEvent` | 新增 | 传输进度事件，包含当前传输文件列表 |
| `JobProgressEvent` | 扩展 | 新增 `filesTotal`, `bytesTotal`, `filesDeleted`, `errorCount` |
| `Subscription.transferProgress` | 新增 | 独立的传输进度订阅，支持按 connectionId/taskId/jobId 筛选 |

### 数据来源

| 字段 | rclone API 来源 |
|------|-----------------|
| `ConnectionQuota.*` | `fs.Usage` 直接映射 |
| `TransferItem.*` | `accounting.TransferSnapshot` |
| `Job.filesDeleted` | `accounting.StatsInfo.GetDeletes()` (作业完成时持久化) |
| `Job.errorCount` | `accounting.StatsInfo.GetErrors()` (作业完成时持久化) |
| `JobProgressEvent.filesTotal` | `RemoteStats().totalTransfers` |
| `JobProgressEvent.bytesTotal` | `RemoteStats().totalBytes` |
| `JobProgressEvent.filesDeleted` | `accounting.StatsInfo.GetDeletes()` |
| `JobProgressEvent.errorCount` | `accounting.StatsInfo.GetErrors()` |
| `TransferProgressEvent.transfers` | `Stats.InProgress()` + `Transfer.Snapshot()` |

**注意**: `filesTotal`/`bytesTotal` 会随着扫描进行而动态增加，不是固定值

---

## Configuration Changes

### config.toml 新增配置项

```toml
[log]
level = "info"
# 每个连接保留的最大日志条数，0 表示无限制
max_logs_per_connection = 1000
# 日志清理任务的 cron 表达式
cleanup_schedule = "0 * * * *"
```

### Config struct 变更

**现有**:
```go
type Config struct {
  Log struct {
    Level string `mapstructure:"level"`
  } `mapstructure:"log"`
}
```

**变更后**:
```go
type Config struct {
  Log struct {
    Level                string            `mapstructure:"level"`
    Levels               map[string]string `mapstructure:"levels"`  // 新增: 按名称层级设置日志级别
    MaxLogsPerConnection int               `mapstructure:"max_logs_per_connection"`
    CleanupSchedule      string            `mapstructure:"cleanup_schedule"`
  } `mapstructure:"log"`
  Job struct {
    AutoDeleteEmptyJobs bool `mapstructure:"auto_delete_empty_jobs"` // 新增: 自动删除无活动作业
  } `mapstructure:"job"`
}
```

### 自动删除无活动作业配置 (User Story 7)

#### config.toml 配置示例

```toml
[job]
# 作业完成后，如果没有实际传输活动，是否自动删除该作业记录
# 默认值: false（保留所有作业记录）
auto_delete_empty_jobs = false
```

#### "无活动"判定标准

| 条件 | 说明 |
|------|------|
| `filesTransferred = 0` | 未传输任何文件 |
| `bytesTransferred = 0` | 未传输任何字节 |
| `filesDeleted = 0` | 未删除任何文件 |
| `errorCount = 0` | 未发生任何错误 |
| `status = SUCCESS` | 作业状态为成功完成 |

**注意**:
- `filesChecked` 不作为判断条件（即使检查了文件但无传输也视为"无活动"）
- 失败的作业即使无活动也会保留（便于问题排查）
- 有删除操作或错误的作业会保留（便于用户查看操作记录和问题排查）
- 删除作业时会通过数据库级联删除关联的日志记录

### 层级日志级别配置 (User Story 6)

#### config.toml 配置示例

```toml
[log]
level = "info"                    # 全局日志级别

[log.levels]
"core.db" = "debug"               # core.db 及其子模块使用 debug 级别
"core.scheduler" = "warn"         # core.scheduler 及其子模块使用 warn 级别
"rclone" = "error"                # rclone 及其子模块使用 error 级别
```

#### 层级匹配规则

日志名称按 `.` 拆分后，按以下优先级匹配配置项（**匹配过程区分大小写**）：

| 匹配优先级 | 匹配规则 | 示例 |
|------------|----------|------|
| 1 (最高) | 精确匹配 | `core.db.query` 匹配配置 `core.db.query` |
| 2 | 父级匹配 | `core.db.query` 匹配配置 `core.db` |
| 3 | 更高父级匹配 | `core.db.query` 匹配配置 `core` |
| 4 (最低) | 全局级别 | 未匹配到任何配置时使用 `level` |

**注意**: 配置键必须与代码中定义的 Logger Name 完全一致（区分大小写）。例如，配置 `"Core.DB" = "debug"` 不会匹配日志名称 `core.db`。

#### 实现要点

1. **缓存策略**：
   - 使用 `sync.Map` 实现无锁并发缓存
   - 按需缓存：首次调用 `GetLevelForName()` 时计算并缓存，后续直接查表
   - 配置变更时（应用重启）清空缓存

2. **Logger 工厂函数扩展**：
   - 修改 `Named(name string)` 函数，在创建 Named Logger 时应用层级级别配置
   - 使用 `zap.WrapCore()` 和自定义 `levelFilterCore` 实现级别过滤

3. **匹配算法**：
   ```go
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

   func computeLevelForName(name string) zapcore.Level {
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
   ```

4. **级别过滤核心**：
   ```go
   // levelFilterCore 是一个包装的 zapcore.Core，用于过滤日志级别
   type levelFilterCore struct {
       zapcore.Core
       level zapcore.Level
   }

   func (c *levelFilterCore) Enabled(lvl zapcore.Level) bool {
       return lvl >= c.level && c.Core.Enabled(lvl)
   }
   ```

5. **支持的日志级别**：
   - `debug`, `info`, `warn`, `error`（不区分大小写）
   - 无效的级别值将使用全局级别并记录警告

6. **配置生效时机**：
   - 配置变更后需要重启应用才能生效（无需支持热更新）

---

## State Transitions

### 日志清理流程

```
[定时任务触发]
    ↓
[获取所有连接 ID 列表]
    ↓
[遍历每个连接]
    ↓
[查询该连接的日志数量]
    ↓ (count > maxLogs)
[计算需删除数量: count - maxLogs]
    ↓
[删除最旧的 N 条日志]
    ↓
[继续下一个连接]
```

### 传输进度广播流程

```
[Sync 进行中]
    ↓ (每秒)
[调用 Stats.InProgress()]
    ↓
[获取所有 Transfer]
    ↓
[对每个 Transfer 调用 Snapshot()]
    ↓
[构建 TransferItem 列表]
    ↓
[构建 JobProgressEvent]
    ↓
[通过 JobProgressBus 广播]
    ↓
[WebSocket 推送到前端]
```

---

## Dependencies

| 组件 | 依赖 | 说明 |
|------|------|------|
| ConnectionQuota | rclone fs.Usage | 直接映射 |
| TransferItem | rclone accounting.TransferSnapshot | 提取关键字段 |
| LogCleanupService | ConnectionService | 获取连接列表 |
| LogCleanupService | JobService | 执行日志删除 |
| Scheduler | LogCleanupService | 注册定时任务 |

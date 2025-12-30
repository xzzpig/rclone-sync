# Tasks: UI Detail Improvements

**Input**: Design documents from `/specs/008-ui-detail-improvements/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/schema.graphql ✓

**Tests**: 根据 Constitution 中的 TDD (Backend) 原则，后端变更需要编写测试。

**Organization**: 任务按用户故事分组，以便独立实现和测试每个故事。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 任务所属的用户故事（如 US1, US2, US3, US4）
- 描述中包含精确的文件路径

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: GraphQL Schema 更新和代码生成

- [x] T001 扩展 ConnectionQuota 类型，添加 trashed/other/objects 字段 in internal/api/graphql/schema/connection.graphql
- [x] T002 扩展 JobProgressEvent 类型，添加 filesTotal/bytesTotal 字段 in internal/api/graphql/schema/job.graphql
- [x] T003 新增 TransferItem 和 TransferProgressEvent 类型 in internal/api/graphql/schema/job.graphql
- [x] T004 新增 transferProgress subscription 定义 in internal/api/graphql/schema/job.graphql
- [x] T005 运行 go generate 重新生成 GraphQL 代码

**Checkpoint**: GraphQL Schema 更新完成，可以开始实现各用户故事

---

## Phase 2: User Story 1 - 查看同步作业详细进度 (Priority: P1) 🎯 MVP

**Goal**: 用户在作业执行时能看到总文件数/已传输文件数、总字节数/已传输字节数的详细进度

**Independent Test**: 启动一个包含多个文件的同步任务，在作业执行过程中观察 UI 上的进度信息是否正确显示并实时更新

### Tests for User Story 1

- [x] T006 [P] [US1] 编写 sync.go RemoteStats 获取逻辑的单元测试（含空文件、传输中断场景）in internal/rclone/sync_test.go
- [x] T007 [P] [US1] 编写 jobProgress subscription 返回 filesTotal/bytesTotal 的测试 in internal/api/graphql/resolver/subscription_test.go

### Implementation for User Story 1

- [x] T008 [US1] 修改 processStats() 调用 RemoteStats(false) 获取 totalTransfers/totalBytes in internal/rclone/sync.go
- [x] T009 [US1] 更新 JobProgressEvent 构建逻辑，填充 filesTotal/bytesTotal 字段 in internal/rclone/sync.go
- [x] T010 [US1] 更新 jobProgress subscription 查询，添加 filesTotal/bytesTotal 字段 in web/src/api/graphql/queries/subscriptions.ts
- [x] T011 [US1] 更新 History 视图，在 RUNNING 状态时显示 "45/128" 和 "12 KB/10 MB" 格式的进度 in web/src/modules/connections/views/History.tsx

**Checkpoint**: 用户故事 1 功能完整，可独立测试

---

## Phase 3: User Story 2 - 查看文件传输详情 (Priority: P1)

**Goal**: 用户能看到当前正在传输的具体文件信息和每个文件的传输进度

**Independent Test**: 启动一个包含大文件的同步任务，观察是否能看到当前正在传输的文件名和该文件的传输进度

### Tests for User Story 2

- [x] T012 [P] [US2] 编写 TransferProgressBus 事件总线测试 in internal/api/graphql/subscription/transfer_progress_bus_test.go
- [x] T013 [P] [US2] 编写 transferProgress subscription resolver 测试 in internal/api/graphql/resolver/subscription_test.go

### Implementation for User Story 2

- [x] T014 [US2] 创建 TransferProgressBus 事件总线，支持按 connectionId/taskId/jobId 筛选 in internal/api/graphql/subscription/transfer_progress_bus.go
- [x] T015 [US2] 复用 getStatsInternals() 获取传输列表，实现增量推送机制 in internal/rclone/sync.go
- [x] T016 [US2] 实现 TransferProgress subscription resolver in internal/api/graphql/resolver/subscription.resolvers.go
- [x] T017 [US2] 在 Resolver 结构体中注入 TransferProgressBus in internal/api/graphql/resolver/resolver.go
- [x] T018 [US2] 新增 transferProgress subscription 查询定义 in web/src/api/graphql/queries/subscriptions.ts
- [x] T019 [US2] 更新 Overview 视图，以列表形式展示当前连接下所有活跃传输（每项显示：文件名、文件大小、已传输大小、进度百分比、进度条；空状态显示提示）in web/src/modules/connections/views/Overview.tsx

**Checkpoint**: 用户故事 2 功能完整，可独立测试

---

## Phase 4: User Story 3 - 查看存储配额详情 (Priority: P2)

**Goal**: Storage Usage 卡片显示更详细的配额信息（Trashed、Other、Objects）

**Independent Test**: 查看任意一个连接的 Storage Usage 卡片，确认是否显示完整的配额信息

### Tests for User Story 3

- [x] T020 [P] [US3] 编写 Quota resolver 返回扩展字段的测试 in internal/api/graphql/resolver/connection_test.go

### Implementation for User Story 3

- [x] T021 [US3] 更新 Quota() resolver 返回完整字段（trashed/other/objects）in internal/api/graphql/resolver/connection.resolvers.go
- [x] T022 [US3] 更新 quota 查询，添加 trashed/other/objects 字段 in web/src/api/graphql/queries/connections.ts
- [x] T023 [US3] 更新 Overview 视图 Storage Usage 卡片，显示完整配额信息和优雅降级 in web/src/modules/connections/views/Overview.tsx

**Checkpoint**: 用户故事 3 功能完整，可独立测试

---

## Phase 5: User Story 4 - 日志数量限制管理 (Priority: P2)

**Goal**: 用户能在配置文件中设置日志限制，系统通过定时任务自动清理

**Independent Test**: 在配置文件中设置日志限制数量（如 1000 条），为某个连接生成超过限制的日志，等待定时清理任务执行后验证旧日志被自动清理

### Tests for User Story 4

- [x] T024 [P] [US4] 编写 LogCleanupService 清理逻辑的单元测试 in internal/core/services/log_cleanup_service_test.go
- [x] T025 [P] [US4] 编写 DeleteOldLogsForConnection 方法的测试 in internal/core/services/job_service_test.go

### Implementation for User Story 4

- [x] T026 [US4] 添加 Log 配置结构（MaxLogsPerConnection/CleanupSchedule）in internal/core/config/config.go
- [x] T027 [US4] 添加 LogCleanupService 接口定义 in internal/core/ports/interfaces.go
- [x] T028 [US4] 实现 DeleteOldLogsForConnection 方法（使用 ent API）in internal/core/services/job_service.go
- [x] T029 [US4] 创建 LogCleanupService 实现，使用独立的 cron 实例 in internal/core/services/log_cleanup_service.go
- [x] T030 [US4] 在 serve 命令中初始化 LogCleanupService 并启动定时任务 in cmd/cloud-sync/serve.go
- [x] T031 [US4] 更新配置文件添加日志配置项示例 in config.toml

**Checkpoint**: 用户故事 4 功能完整，可独立测试

---

## Phase 6: User Story 5 - 概览页展示进行中的任务列表 (Priority: P1)

**Goal**: 用户在概览页面能看到一个卡片，展示当前连接下所有正在进行中的同步任务（Job）列表

**Independent Test**: 启动一个或多个同步任务，查看概览页面是否显示进行中的任务卡片，并验证任务列表是否实时更新

### Implementation for User Story 5

- [x] T035 [US5] 创建 RunningJobsCard 组件，展示进行中的作业列表（任务名称、状态、开始时间、文件进度、字节进度、进度条；无任务时隐藏卡片；点击跳转日志页面）in web/src/modules/connections/components/RunningJobsCard.tsx
- [x] T036 [US5] 在 Overview 视图中集成 RunningJobsCard 组件 in web/src/modules/connections/views/Overview.tsx
- [x] T037 [US5] 添加 i18n key（overview.runningJobs）in web/project.inlang/messages/en.json 和 web/project.inlang/messages/zh-CN.json（注：开始时间复用现有翻译 common.startedAt）

**实现说明**:
- 复用现有的 `jobProgress` subscription，按 connectionId 筛选当前连接的作业
- 使用 jobProgressStore 来获取实时进度数据
- 卡片内每个任务项显示：
  - 任务名称
  - 状态徽章
  - 开始时间
  - 文件进度（如 "45/128 files"）
  - 字节进度（如 "256 MB / 1.2 GB"）
  - 以已传输字节数为基准的进度条（显示百分比）
- **无进行中任务时隐藏整个卡片**（而非显示空状态）
- 任务完成后自动从列表中移除
- **点击任务项跳转到日志页面（Log），并自动筛选该任务的日志**

**Checkpoint**: 用户故事 5 功能完整，可独立测试

---

## Phase 7: User Story 6 - 按名称层级设置日志级别 (Priority: P2)

**Goal**: 管理员能在配置文件中按日志名称分别设置日志级别，支持按 `.` 拆分名称后按层级匹配，实现不同模块的精细化日志控制

**Independent Test**: 在配置文件中设置不同模块的日志级别（如 `core.db = debug`），然后观察该模块的日志输出是否符合配置的级别，而其他模块保持全局级别

### Tests for User Story 6

- [x] T038 [P] [US6] 编写层级日志级别匹配算法测试（精确匹配、父级匹配、多级父级匹配、全局级别回退、大小写敏感、空字符串名称、无效级别值、缓存行为）in internal/core/logger/level_test.go
- [x] T039 [P] [US6] 编写 Named Logger 级别控制测试（Named logger 使用正确级别、级别过滤生效）in internal/core/logger/logger_test.go

### Implementation for User Story 6

- [x] T040 [US6] 添加 Levels 配置项（map[string]string 类型的层级日志级别映射）in internal/core/config/config.go
- [x] T041 [US6] 创建 level.go 实现层级匹配算法（levelCache sync.Map、InitLevelConfig、GetLevelForName、computeLevelForName、ParseLevel）in internal/core/logger/level.go
- [x] T042 [US6] 修改 logger.go 支持按名称层级设置日志级别（修改 InitLogger 签名添加 levels 参数、修改 Named 函数应用层级日志级别、添加 levelFilterCore 结构体）in internal/core/logger/logger.go
- [x] T043 [US6] 更新 serve.go 调用 InitLogger 时传入 cfg.Log.Levels 参数 in cmd/cloud-sync/serve.go
- [x] T044 [US6] 更新配置文件添加层级日志级别配置示例（[log.levels] 配置段）in config.toml

**实现说明**:
- **层级匹配规则**（区分大小写）:
  1. 精确匹配：`core.db.query` 匹配配置 `"core.db.query"`
  2. 父级匹配：`core.db.query` 匹配配置 `"core.db"`
  3. 更高父级：`core.db.query` 匹配配置 `"core"`
  4. 全局回退：使用全局 `level` 配置
- **级别值不区分大小写**: `DEBUG`, `Debug`, `debug` 都有效
- **无锁并发缓存**: 使用 `sync.Map` 实现按需缓存
- **仅支持四级**: debug, info, warn, error（无 trace/fatal）
- **无效级别处理**: 使用全局级别并记录警告

**配置示例**:
```toml
[log]
level = "info"                    # 全局日志级别

[log.levels]
"core.db" = "debug"               # core.db 及其子模块使用 debug 级别
"core.scheduler" = "warn"         # core.scheduler 及其子模块使用 warn 级别
"rclone" = "error"                # rclone 及其子模块使用 error 级别
```

**Checkpoint**: 用户故事 6 功能完整，可独立测试

---

## Phase 8: User Story 7 - 自动删除无活动作业 (Priority: P2)

**Goal**: 当作业完成后，如果没有实际传输活动（filesTransferred = 0 且 bytesTransferred = 0），且作业成功完成，自动删除该作业记录

**Independent Test**: 在配置文件中启用该选项，执行一个源和目标完全相同的同步任务，验证作业结束后该作业记录是否被自动删除

### Tests for User Story 7

- [x] T045 [P] [US7] 编写 shouldDeleteEmptyJob 辅助函数的单元测试（含空文件、传输中断场景）in internal/rclone/sync_test.go
- [x] T046 [P] [US7] 编写 DeleteJob 方法的测试（验证级联删除关联日志）in internal/core/services/job_service_test.go

### Implementation for User Story 7

- [x] T047 [US7] 添加 Job 配置结构（AutoDeleteEmptyJobs bool）in internal/core/config/config.go
- [x] T048 [US7] 创建 DeleteJob 方法，通过 ent ORM 级联删除关联日志记录 in internal/core/services/job_service.go
- [x] T049 [US7] 实现 shouldDeleteEmptyJob 辅助函数和同步完成后的自动删除逻辑（注意：删除过程中的错误应记录警告日志，但不能中断后续流程） in internal/rclone/sync.go
- [x] T050 [US7] 更新配置文件添加作业配置项示例 in config.toml

**实现说明**:
- "无活动"判定标准:
  - `filesTransferred = 0`（未传输任何文件）
  - `bytesTransferred = 0`（未传输任何字节）
  - `status = SUCCESS`（作业状态为成功完成）
- `filesChecked` 不作为判断条件（即使检查了文件但无传输也视为"无活动"）
- 失败的作业即使无活动也会保留（便于问题排查）
- 删除作业时通过数据库级联删除关联的 JobLog 记录

**Checkpoint**: 用户故事 7 功能完整，可独立测试

---

## Phase 9: User Story 8 - JOB 记录并展示更多状态信息 (Priority: P1)

**Goal**: 用户在作业执行过程中能看到更完整的状态信息，包括删除的文件数和错误数；已完成的作业也能查看这些持久化的统计信息

**Independent Test**: 启动一个包含删除操作或可能产生错误的同步任务，观察 UI 上是否正确显示删除数和错误数；作业完成后查看历史记录，确认这些信息被持久化

**UI 展示规格** (来自澄清 2025-12-27):
- **删除数、错误数**: 在作业列表页面表格中作为独立列展示，值为 0 时显示 "0"（保持表格一致性）
- **实时更新**: 作业进行中时，删除数、错误数通过 Subscription 实时更新，与文件进度/字节进度一致
- **错误醒目显示**: 错误数 > 0 时以红色徽章形式显示，便于用户快速识别有问题的作业

### Tests for User Story 8

- [ ] T051 [P] [US8] 编写 sync.go StatsInfo 获取 filesDeleted/errorCount 的单元测试 in internal/rclone/sync_test.go
- [ ] T052 [P] [US8] 编写 jobProgress subscription 返回 filesDeleted/errorCount 的测试 in internal/api/graphql/resolver/subscription_test.go
- [ ] T053 [P] [US8] 编写 Job 查询返回 filesDeleted/errorCount 的测试 in internal/api/graphql/resolver/job_test.go

### Implementation for User Story 8

**Schema & Database**:
- [ ] T054 [US8] 修改 Job ent schema，添加 files_deleted/error_count 字段 in internal/core/ent/schema/job.go
- [ ] T055 [US8] 运行 go generate ./internal/core/ent 重新生成 ent 代码
- [ ] T056 [US8] 生成数据库迁移脚本（添加 files_deleted 和 error_count 列）
- [ ] T057 [US8] 扩展 Job 类型，添加 filesDeleted/errorCount 字段 in internal/api/graphql/schema/job.graphql
- [ ] T058 [US8] 扩展 JobProgressEvent 类型，添加 filesDeleted/errorCount 字段 in internal/api/graphql/schema/job.graphql
- [ ] T059 [US8] 运行 go generate ./... 重新生成 GraphQL 代码

**Backend Logic**:
- [ ] T060 [US8] 修改 processStats() 调用 StatsInfo.GetDeletes()/GetErrors() 获取统计信息 in internal/rclone/sync.go
- [ ] T061 [US8] 更新 JobProgressEvent 构建逻辑，填充 filesDeleted/errorCount 字段 in internal/rclone/sync.go
- [ ] T062 [US8] 在作业完成时持久化 filesDeleted 和 errorCount 到数据库 in internal/rclone/sync.go

**Frontend**:
- [ ] T063 [US8] 更新 jobProgress subscription 查询，添加 filesDeleted/errorCount 字段 in web/src/api/graphql/queries/subscriptions.ts
- [ ] T064 [US8] 更新 Job 查询，添加 filesDeleted/errorCount 字段 in web/src/api/graphql/queries/jobs.ts
- [ ] T065 [US8] 更新 History 视图，在表格中添加删除数和错误数列 in web/src/modules/connections/views/History.tsx
- [ ] T066 [US8] 添加 i18n keys（job.filesDeleted, job.errorCount）in web/project.inlang/messages/en.json 和 web/project.inlang/messages/zh-CN.json

**UI 组件实现说明**:
- **删除数列**: 表格新增列，显示数字（0、15 等），使用 `{job.filesDeleted}` 渲染
- **错误数列**: 表格新增列，显示数字；当值 > 0 时使用红色徽章（Badge variant="destructive"）
- **零值处理**: 删除数和错误数为 0 时显示 "0"，保持表格列的一致性

**数据源映射**:
| GraphQL 字段 | 数据源 | 持久化 |
|-------------|--------|--------|
| `Job.filesDeleted` | `accounting.StatsInfo.GetDeletes()` | ✅ 作业完成时写入 DB |
| `Job.errorCount` | `accounting.StatsInfo.GetErrors()` | ✅ 作业完成时写入 DB |
| `JobProgressEvent.filesDeleted` | `accounting.StatsInfo.GetDeletes()` | ❌ 实时推送 |
| `JobProgressEvent.errorCount` | `accounting.StatsInfo.GetErrors()` | ❌ 实时推送 |

**边缘情况处理**:
- `filesDeleted = 0` 时显示 "0"（保持表格列一致性）
- `errorCount = 0` 时显示 "0"（保持表格列一致性）
- `errorCount > 0` 时，错误数以红色徽章形式醒目显示

**Checkpoint**: 用户故事 8 功能完整，可独立测试

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: i18n、文档更新、验证

- [x] T032 [P] 添加英文翻译 keys（overview.trashed, overview.other, overview.objects, overview.quotaUnavailable, overview.activeTransfers, overview.transferProgress, common.noActiveTransfers）in web/project.inlang/messages/en.json
- [x] T033 [P] 添加中文翻译 keys in web/project.inlang/messages/zh-CN.json
- [x] T034 运行 quickstart.md 验证所有用户故事场景

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖，可立即开始
- **User Story 1 (Phase 2)**: 依赖 Phase 1 完成
- **User Story 2 (Phase 3)**: 依赖 Phase 1 完成，可与 US1 并行
- **User Story 3 (Phase 4)**: 依赖 Phase 1 完成，可与 US1/US2 并行
- **User Story 4 (Phase 5)**: 无 GraphQL Schema 依赖，可与其他故事并行
- **User Story 5 (Phase 6)**: 依赖 Phase 1 完成（复用 jobProgress subscription），可与 US1-US4 并行
- **User Story 6 (Phase 7)**: 无 GraphQL Schema 依赖，可与其他故事并行（仅涉及后端配置和日志模块）
- **User Story 7 (Phase 8)**: 无 GraphQL Schema 依赖，可与其他故事并行（仅后端配置和同步逻辑变更）
- **User Story 8 (Phase 9)**: 需要扩展 GraphQL Schema，可与 US1-US7 并行开发
- **Polish (Phase 10)**: 依赖所有用户故事完成

### User Story Dependencies

- **User Story 1 (P1)**: 依赖 Schema 更新 → 可独立测试
- **User Story 2 (P1)**: 依赖 Schema 更新 → 可独立测试
- **User Story 3 (P2)**: 依赖 Schema 更新 → 可独立测试
- **User Story 4 (P2)**: 无 Schema 依赖 → 可独立测试
- **User Story 5 (P1)**: 依赖 Phase 1 完成 → 复用 jobProgress subscription → 可独立测试
- **User Story 6 (P2)**: 无 Schema 依赖 → 可独立测试（仅后端配置变更）
- **User Story 7 (P2)**: 无 Schema 依赖 → 可独立测试（仅后端配置和同步逻辑变更）
- **User Story 8 (P1)**: 需扩展 Schema 和 DB → 可独立测试（扩展 Job 添加 filesDeleted/errorCount；扩展 JobProgressEvent 添加 filesDeleted/errorCount）

### Within Each User Story

- 测试先行（TDD）：测试代码先于实现代码
- 后端先于前端
- Schema/配置 → 服务层 → Resolver → 前端

### Parallel Opportunities

- T001-T004 可并行执行（不同 GraphQL 文件）
- T006-T007 可并行执行（不同测试文件）
- T012-T013 可并行执行（不同测试文件）
- T020, T024-T025 可并行执行（不同测试文件）
- T032-T033 可并行执行（不同语言文件）
- T038-T039 可并行执行（不同测试文件）
- T051-T052 可并行执行（不同测试文件）
- US1-US8 可由不同开发者并行开发（US6/US7 仅后端，无前端变更）

---

## Parallel Example: Setup Phase

```bash
# 并行执行所有 Schema 更新:
Task T001: "扩展 ConnectionQuota 类型 in connection.graphql"
Task T002: "扩展 JobProgressEvent 类型 in job.graphql"
Task T003: "新增 TransferItem/TransferProgressEvent in job.graphql"
Task T004: "新增 transferProgress subscription in job.graphql"

# 然后执行代码生成:
Task T005: "运行 go generate"
```

## Parallel Example: User Story 1

```bash
# 并行执行测试编写:
Task T006: "sync.go RemoteStats 单元测试"
Task T007: "jobProgress subscription 测试"

# 然后按顺序实现:
Task T008 → T009 → T010 → T011
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. 完成 Phase 1: Setup (GraphQL Schema)
2. 完成 Phase 2: User Story 1 (作业详细进度)
3. **验证**: 测试作业进度显示功能
4. 部署/演示

### Incremental Delivery

1. Setup → Schema 就绪
2. Add User Story 1 → 作业进度可用 → 部署 (MVP!)
3. Add User Story 2 → 传输详情可用 → 部署
4. Add User Story 5 → 进行中任务卡片可用 → 部署
5. Add User Story 8 → 作业状态信息（删除数/错误数）可用 → 部署
6. Add User Story 3 → 配额详情可用 → 部署
7. Add User Story 4 → 日志管理可用 → 部署
8. Add User Story 6 → 层级日志级别可用 → 部署
9. Add User Story 7 → 自动删除无活动作业可用 → 部署
10. Polish → i18n 完成 → 最终发布

### Parallel Team Strategy

多开发者并行:
1. 团队一起完成 Setup
2. Setup 完成后:
   - 开发者 A: User Story 1 + User Story 2 + User Story 5 (P1 优先)
   - 开发者 B: User Story 3 + User Story 4 + User Story 6 (P2)
3. 各故事独立完成后集成

---

## Notes

- [P] 任务 = 不同文件，无依赖
- [Story] 标签映射任务到特定用户故事以便追踪
- 每个用户故事应可独立完成和测试
- 验证测试先失败再实现
- 每个任务或逻辑组完成后提交
- 在任意检查点停下来独立验证故事
- 避免：模糊任务、同文件冲突、破坏独立性的跨故事依赖

# Tasks: 同步任务多项优化

**Input**: Design documents from `/specs/013-sync-optimizations/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/schema.graphql

**Tests**: 后端修改需测试覆盖（根据 plan.md 约束）

**Organization**: 任务按用户故事分组，支持独立实现和测试

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行（不同文件、无依赖）
- **[Story]**: 所属用户故事 (US1-US6)
- 包含准确的文件路径

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 本功能在现有项目基础上添加优化，无需新建项目结构

- [x] T001 确认开发环境：Go 1.25+, Node.js, pnpm 已安装
- [x] T002 [P] 创建功能分支 `013-sync-optimizations` 并切换

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 6 个优化项相互独立，无共享的阻塞性前置任务

**⚠️ 注意**: 本功能的各用户故事相互独立，可直接进入用户故事实现阶段

**Checkpoint**: 可立即开始用户故事实现

---

## Phase 3: User Story 1 - 任务完成时 Toast 提醒 (Priority: P1) 🎯 MVP

**Goal**: 同步任务完成后自动显示 Toast 通知，包含成功/失败/取消状态

**Independent Test**: 启动任意同步任务，完成后应在 1 秒内弹出相应状态的 Toast

### Implementation for User Story 1

- [x] T003 [P] [US1] 添加 Toast 通知国际化文案到 `web/project.inlang/messages/en.json`
- [x] T004 [P] [US1] 添加 Toast 通知国际化文案到 `web/project.inlang/messages/zh-CN.json`
- [x] T005 [US1] 在 `web/src/store/tasks.tsx` 中实现任务完成检测和 Toast 触发逻辑（注：实现位置从 jobProgress.tsx 改为 tasks.tsx 以支持 taskName 优化）
- [x] T005A [US1] （人工测试）任务完成到 Toast 显示延迟 ≤1s，确认满足 FR-004/SC-001
- [x] T005B [US1] （人工测试）并发完成 ≥2 任务时 Toast 堆叠显示、不被覆盖（FR-024）

**Checkpoint**: 任务完成时自动显示 Toast，可独立验证

---

## Phase 4: User Story 2 - 传输进度区分上传/下载 (Priority: P1)

**Goal**: 在传输列表中使用不同图标区分上传和下载方向

**Independent Test**: 执行上传/下载任务，传输列表应显示对应方向的箭头图标

### Backend for User Story 2

- [x] T006 [P] [US2] 在 `internal/api/graphql/schema/job.graphql` 添加 TransferDirection 枚举和 TransferItem.direction 字段
- [x] T007 [US2] 运行 `go generate ./internal/api/graphql` 重新生成 GraphQL 代码
- [x] T008 [US2] 在 `internal/rclone/sync.go` 的 `processStats` 中计算并设置 direction 字段（根据 snapshot.SrcFs 与 task.SourcePath 比较判断方向）
- [x] T009 [P] [US2] 为传输方向逻辑添加单元测试到 `internal/rclone/sync_test.go`

### Frontend for User Story 2

- [x] T010 [P] [US2] 更新 `web/src/api/graphql/queries/subscriptions.ts` 订阅类型以包含 direction
- [x] T011 [US2] 在 `web/src/modules/connections/components/ActiveTransfersCard.tsx` 添加上传/下载方向图标显示

**Checkpoint**: 传输列表正确显示上传(↑)和下载(↓)图标，可独立验证

---

## Phase 5: User Story 3 - 同步时展示已检查的文件数 (Priority: P2)

**Goal**: 在任务状态卡片中显示已完成扫描/检查的文件数量

**Independent Test**: 同步大目录时，进度区域应实时显示"已检查 X 个文件"

### Backend for User Story 3

- [x] T012 [US3] 在 `internal/api/graphql/schema/job.graphql` 的 JobProgressEvent 添加 filesChecked 字段
- [x] T013 [US3] 运行 `go generate ./internal/api/graphql` 重新生成 GraphQL 代码
- [x] T014 [US3] 在 `internal/rclone/sync.go` 的 `processStats` 中从 rclone stats 获取 checks 数量（已合并到 getTotalStats 函数中，减少 RemoteStats 调用）

### Frontend for User Story 3

- [x] T015 [P] [US3] 添加检查文件数国际化文案到 `web/project.inlang/messages/en.json`
- [x] T016 [P] [US3] 添加检查文件数国际化文案到 `web/project.inlang/messages/zh-CN.json`
- [x] T017 [P] [US3] 更新 `web/src/api/graphql/queries/subscriptions.ts` 订阅类型以包含 filesChecked
- [x] T018 [US3] 在 `web/src/modules/connections/components/RunningJobsCard.tsx` 添加已检查文件数显示
- [x] T018A [US3] （人工测试）运行大目录同步，验证 filesChecked 前端显示更新延迟 ≤1s（FR-010/SC-004）

**Checkpoint**: 同步时实时显示已检查文件数，可独立验证

---

## Phase 6: User Story 4 - 允许禁用任务 (Priority: P2)

**Goal**: 用户可通过图标按钮禁用/启用任务，禁用后不响应定时和实时触发

**Independent Test**: 禁用带定时触发的任务后，到达触发时间任务不执行

### Database for User Story 4

- [x] T019 [US4] 在 `internal/core/db/schema/task.go` 添加 enabled 布尔字段 (默认 true)
- [x] T020 [US4] 运行 `go generate ./internal/core/ent` 重新生成 Ent 代码
- [x] T021 [US4] 创建数据库迁移文件 `internal/core/db/migrations/20260116020830_add-task-enabled.up.sql`

### Backend GraphQL for User Story 4

- [x] T022 [P] [US4] 在 `internal/api/graphql/schema/task.graphql` 添加 Task.enabled、CreateTaskInput.enabled、UpdateTaskInput.enabled
- [x] T023 [US4] 运行 `go generate ./internal/api/graphql` 重新生成 GraphQL 代码
- [x] T024 [US4] 在 `internal/api/graphql/resolver/task.resolvers.go` 实现 enabled 字段解析

### Backend Logic for User Story 4

- [x] T025 [P] [US4] 在 `internal/core/scheduler/scheduler.go` 添加 enabled 检查，跳过禁用任务
- [x] T026 [P] [US4] 在 `internal/core/watcher/watcher.go` 添加 enabled 检查，跳过禁用任务
- [x] T027 [P] [US4] 为禁用任务跳过逻辑添加单元测试到相应测试文件
- [x] T027A [US4] （测试）禁用任务后手动触发运行仍执行（覆盖 FR-015 后端路径）

### Frontend for User Story 4

- [x] T028 [P] [US4] 添加启用/禁用按钮国际化文案到 `web/project.inlang/messages/en.json`
- [x] T029 [P] [US4] 添加启用/禁用按钮国际化文案到 `web/project.inlang/messages/zh-CN.json`
- [x] T030 [US4] 在 `web/src/modules/connections/views/Tasks.tsx` 添加启用/禁用图标按钮和状态显示（需添加 aria-label 和 aria-pressed 属性）
- [x] T030A [US4] （人工检查）验证启用/禁用按钮的 aria-label、aria-pressed、键盘可操作性
- [x] T031 [US4] 实现调用 updateTask mutation 切换 enabled 状态的逻辑
- [x] T031A [US4] （人工测试）前端禁用后点击运行按钮，任务应能正常进入执行状态（验证 FR-015 前端路径）
- [x] T031B [US4] （测试或人工）禁用任务后重启/重载，enabled 仍为 false，自动触发继续跳过（持久化验证）

**Checkpoint**: 任务可禁用/启用，禁用任务不响应自动触发，可独立验证

---

## Phase 7: User Story 5 - 清理空任务时保留最新记录 (Priority: P3)

**Goal**: 启用 auto_delete_empty_jobs 时，删除上一个空任务而非当前任务，保留最新记录

**Independent Test**: 连续执行两次无变动同步，历史中仅保留最后一次记录

### Implementation for User Story 5

- [x] T032 [US5] 在 `internal/rclone/sync.go` 中添加 isEmptyJob 辅助函数
- [x] T033 [US5] 修改 `internal/rclone/sync.go` 的 RunTask 结束处清理逻辑：查询上一个 Job 并判断是否删除（deletePreviousEmptyJob）
- [x] T034 [P] [US5] 为空任务清理逻辑添加单元测试到 `internal/rclone/sync_test.go`
- [x] T034A [US5] （测试）连续两次无变动任务时删除上一条空记录但保留当前 Job（验证 FR-025）

**Checkpoint**: 空任务滚动替换逻辑正确工作，可独立验证

---

## Phase 8: User Story 6 - 修复缓存数据库大小统计不正确 (Priority: P3)

**Goal**: 缓存数据库大小统计包含 .db + .db-wal + .db-shm 文件总大小

**Independent Test**: 缓存状态卡片显示的大小与 ls -la 计算的三个文件总大小一致

### Implementation for User Story 6

- [x] T035 [US6] 修改 `internal/rclone/backend/metacache/cache_store.go` 的 GetDBSize 方法以包含 WAL 和 SHM 文件
- [x] T036 [P] [US6] 为 GetDBSize 添加单元测试到 `internal/rclone/backend/metacache/cache_store_test.go`
- [x] T036A [US6] （人工测试）构造含 `.db/.db-wal/.db-shm` 场景，对比三文件大小总和与 GetDBSize 输出（允许 WAL/SHM 缺省）

**Checkpoint**: 缓存大小显示准确，可独立验证

---

## Phase 9: User Story 7 - 概览卡片数据分离加载 (Priority: P2)

**Goal**: 将概览页面的存储配额和缓存状态拆分为独立请求，提升感知性能

**Independent Test**: 打开连接概览页面，验证缓存状态卡片立即展示，存储使用卡片异步加载并显示 Skeleton

### Implementation for User Story 7

- [X] T037 [P] [US7] 在 `web/src/api/graphql/queries/connections.ts` 中更新 `ConnectionGetQuotaQuery` (移除 cacheStatus)
- [X] T038 [P] [US7] 在 `web/src/api/graphql/queries/connections.ts` 中新增 `ConnectionGetCacheStatusQuery` (仅含 cacheStatus)
- [X] T039 [US7] 在 `web/src/modules/connections/views/Overview.tsx` 中拆分数据获取逻辑，使用两个独立的 createQuery
- [X] T040 [US7] 在 `web/src/modules/connections/views/Overview.tsx` 中更新 UI 绑定，使 Skeleton 仅受配额查询状态影响

**Checkpoint**: 概览卡片分离加载正常工作，可独立验证

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: 最终验证和优化

- [x] T037 [P] 运行所有后端测试 `go test ./...` 确保通过
- [x] T038 [P] 运行前端构建 `cd web && pnpm build` 确保无错误
- [x] T039 运行 `lsp_diagnostics` 检查所有修改文件无错误
- [x] T040 按 quickstart.md 测试清单验证全部 6 个功能
- [x] T041 [P] （工具）新增/更新文案后运行 `scripts/sort-i18n-keys.js`，确保多语言键排序（宪法 IX）

---

## Phase 10: Additional Optimizations (Post-MVP)

**Purpose**: 实现过程中发现的额外优化

### Optimization 1: Remove taskName from JobProgressEvent

- [x] T042 移除 `internal/api/graphql/schema/job.graphql` 中 JobProgressEvent 的 taskName 字段（冗余，前端可从 tasks store 获取）
- [x] T043 运行 `go generate ./internal/api/graphql` 重新生成
- [x] T044 移除 `internal/rclone/sync.go` 中 4 处 TaskName 设置
- [x] T045 更新 `web/src/api/graphql/queries/subscriptions.ts` 移除 taskName
- [x] T046 将 Toast 逻辑从 `jobProgress.tsx` 移至 `tasks.tsx`（使用 store 中的 task.name）

### Optimization 2: Merge getTotalStats and getCheckingCount

- [x] T047 合并 `internal/rclone/sync.go` 中的 `getTotalStats` 和 `getCheckingCount` 函数，减少 `RemoteStats` 重复调用
- [x] T048 更新 `internal/rclone/sync_test.go` 中相关测试适配新的返回值

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖 - 可立即开始
- **Foundational (Phase 2)**: 无阻塞任务
- **User Stories (Phase 3-8)**: 全部独立，可并行或按优先级顺序执行
- **Polish (Phase 9)**: 依赖所有用户故事完成

### User Story Dependencies

| 用户故事 | 依赖 | 说明 |
|---------|------|------|
| US1 任务完成 Toast | 无 | 纯前端，使用现有 Toast 系统 |
| US2 传输方向区分 | 无 | 后端 + 前端，GraphQL 变更 |
| US3 检查中文件数 | 无 | 后端 + 前端，GraphQL 变更 |
| US4 禁用任务 | 无 | 全栈，数据库迁移 + GraphQL + 前端 |
| US5 空任务清理 | 无 | 纯后端逻辑修改 |
| US6 缓存大小修复 | 无 | 纯后端 bug 修复 |

### Within Each User Story

- GraphQL Schema 变更 → 代码生成 → 后端实现 → 前端实现
- 国际化文案可并行添加
- 测试应与实现同步完成

### Parallel Opportunities

**Backend 并行**:
```
T006 (US2 GraphQL) || T012 (US3 GraphQL) || T019 (US4 Ent Schema)
T009 (US2 测试) || T027 (US4 测试) || T034 (US5 测试) || T036 (US6 测试)
T025 (Scheduler) || T026 (Watcher)
```

**Frontend 并行**:
```
T003 (US1 en.json) || T004 (US1 zh-CN.json)
T015 (US3 en.json) || T016 (US3 zh-CN.json)
T028 (US4 en.json) || T029 (US4 zh-CN.json)
```

**跨用户故事并行** (推荐团队协作):
```
Developer A: US1 + US2 (P1 优先级)
Developer B: US3 + US4 (P2 优先级)
Developer C: US5 + US6 (P3 优先级)
```

---

## Implementation Strategy

### MVP First (User Story 1 + 2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 3: US1 Toast 提醒
3. Complete Phase 4: US2 传输方向区分
4. **STOP and VALIDATE**: 验证 Toast 和传输方向图标
5. Deploy/demo if ready

### Incremental Delivery

1. US1 + US2 (P1) → 核心体验优化 → Deploy/Demo (MVP!)
2. Add US3 + US4 (P2) → 进度可见性 + 任务管理 → Deploy/Demo
3. Add US5 + US6 (P3) → 清理优化 + Bug 修复 → Deploy/Demo

### Recommended Order (Single Developer)

1. T001-T002 (Setup)
2. T003-T005 (US1 - 最简单，纯前端)
3. T035-T036 (US6 - 最简单的后端修改)
4. T032-T034 (US5 - 纯后端逻辑)
5. T006-T011 (US2 - 全栈但范围小)
6. T012-T018 (US3 - 全栈但范围小)
7. T019-T031 (US4 - 最复杂，涉及数据库迁移)
8. T037-T040 (Polish)

---

## Notes

- [P] 任务 = 不同文件，无依赖
- [Story] 标签映射到具体用户故事以便追踪
- 每个用户故事应可独立完成和测试
- 后端修改必须有测试覆盖（plan.md 约束）
- 前端 UI 变更需遵循 Kobalte + Tailwind 组件模式
- 每个任务或逻辑组完成后提交
- 在任何 Checkpoint 停止以独立验证故事
- 避免：模糊任务、同文件冲突、破坏独立性的跨故事依赖

---

*Tasks generated by /speckit.tasks command*

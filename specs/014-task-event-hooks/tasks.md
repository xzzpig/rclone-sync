# Tasks: Task Event Hooks

**Input**: Design documents from `/specs/014-task-event-hooks/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/hook.graphql ✅

**Tests**: Backend 测试按照 Constitution Check 要求包含（III. Test-Driven Development）。

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `internal/` (Go)
- **Frontend**: `web/src/` (SolidJS)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, GraphQL types generation, database schema, and migrations

### GraphQL Schema (must be first - generates types for Ent schema)

- [X] T001 Copy GraphQL contract from `specs/014-task-event-hooks/contracts/hook.graphql` to `internal/api/graphql/schema/hook.graphql`
- [X] T002 Run `go generate ./internal/api/graphql` to generate gqlgen code (generates model.HookEvent, model.HookType, model.HookOnError, model.HookConfig, and extends LogAction with HOOK)

### Database Schema (depends on generated GraphQL types)

- [X] T003 Create Hook entity schema in `internal/core/db/schema/hook.go` with fields (id, enabled, priority, event, type, on_error, config, task_id, connection_id, timestamps), edges (task, connection), and indexes per data-model.md
- [X] T004 Add reverse edge `hooks` to Task entity in `internal/core/db/schema/task.go` with CASCADE delete annotation
- [X] T005 Add reverse edge `hooks` to Connection entity in `internal/core/db/schema/connection.go` with CASCADE delete annotation
- [X] T006 Run `go generate ./internal/core/db/ent` to generate Ent code
- [X] T007 Generate database migration using `scripts/gen-migration.sh add_hooks_table`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Configuration

- [X] T008 Add `[app.hook]` config section to `internal/core/config/config.go` with fields: `Enabled bool`, `DefaultTimeout int` (default 30)
- [X] T009 Add config defaults for hook settings in `internal/core/config/config.go` setDefaults function

### Core Hook Package

- [X] T010 Create HookContext struct in `internal/core/hook/context.go` with TaskInfo, JobInfo, Event, Error, Duration, Stats, Env fields
- [X] T011 [P] Create template FuncMap in `internal/core/hook/template.go` with FormatTime, FormatDuration, FormatSizeBytes, JsonMarshal, Summary functions
- [X] T012 Implement RenderTemplate function in `internal/core/hook/template.go` using Go text/template with hookFuncMap
- [X] T013 Create Executor interface in `internal/core/hook/executor.go` with Execute method signature
- [X] T014 Implement executor struct with client, httpClient, globalConfig dependencies in `internal/core/hook/executor.go`
- [X] T015 Implement NewExecutor constructor in `internal/core/hook/executor.go`
- [X] T016 Create HookCancelError and HookFatalError types in `internal/core/hook/errors.go` for error handling behavior
- [X] T017 Implement buildHookContext helper function in `internal/core/hook/context.go` to construct HookContext from task/job

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Stories 1 & 2 - HTTP Hook & Command Hook (Priority: P1) 🎯 MVP

**Goal**: 实现 HTTP 请求 Hook 和 Shell 命令 Hook 的完整执行机制

**Independent Test**: 
- US1: 配置 HTTP hook，执行同步任务，验证目标 URL 收到正确请求
- US2: 配置命令 hook，触发任务失败，验证命令被正确执行

### Tests for User Stories 1 & 2

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T018 [P] [US1] Unit test for HTTP hook execution in `internal/core/hook/http_test.go` - test request method, headers, body rendering
- [X] T019 [P] [US1] Unit test for HTTP hook timeout in `internal/core/hook/http_test.go`
- [X] T020 [P] [US1] Unit test for HTTP hook error handling in `internal/core/hook/http_test.go`
- [X] T021 [P] [US2] Unit test for command hook execution in `internal/core/hook/command_test.go` - test command, workdir, env vars
- [X] T022 [P] [US2] Unit test for command hook timeout in `internal/core/hook/command_test.go`
- [X] T023 [P] [US2] Unit test for command hook exit code handling in `internal/core/hook/command_test.go`
- [X] T024 [P] [US1] Unit test for template rendering in `internal/core/hook/template_test.go` - test all template functions
- [X] T025 [P] [US1] Unit test for environment variable interpolation in `internal/core/hook/template_test.go`

### Implementation for User Stories 1 & 2

#### HTTP Hook Execution (US1)

- [X] T026 [US1] Create hookHTTPClient with timeout and connection pool in `internal/core/hook/http.go`
- [X] T027 [US1] Implement ExecuteHTTPHook function in `internal/core/hook/http.go` with URL template rendering, header application, body sending
- [X] T028 [US1] Add Content-Type default (application/json) when body is present in `internal/core/hook/http.go`

#### Command Hook Execution (US2)

- [X] T029 [US2] Implement ExecuteCommand function in `internal/core/hook/command.go` using exec.CommandContext with timeout
- [X] T030 [US2] Add environment variable injection (RCLONE_SYNC_*) for command hooks in `internal/core/hook/command.go`
- [X] T031 [US2] Implement working directory and shell execution in `internal/core/hook/command.go`

#### Hook Executor Core

- [X] T032 [US1] Implement getHooksForEvent method in `internal/core/hook/executor.go` to query enabled hooks by task/connection and event type
- [X] T033 [US1] Implement executeOne method in `internal/core/hook/executor.go` to dispatch to HTTP or Command execution based on hook type
- [X] T034 [US1] Implement Execute method in `internal/core/hook/executor.go` with global toggle check (`app.hook.enabled`), priority sorting, sequential execution, and error handling (IGNORE/CANCEL/FATAL)
- [X] T035 [US1] Implement hook execution logging to JobLog in `internal/core/hook/executor.go` (level, path as hook:uuid:event, size as duration/status code)

#### SyncEngine Integration

- [X] T036 [US1] Add hookExecutor dependency to SyncEngine struct in `internal/rclone/sync.go`
- [X] T037 [US1] Inject hook trigger for on_success event in SyncEngine.RunTask in `internal/rclone/sync.go`
- [X] T038 [US1] Inject hook trigger for on_failure event in SyncEngine.RunTask in `internal/rclone/sync.go`
- [X] T039 [US1] Inject hook trigger for on_end event (always, regardless of result) in SyncEngine.RunTask in `internal/rclone/sync.go`
- [X] T040 [US1] Update NewSyncEngine to accept and wire HookExecutor in `internal/rclone/sync.go`

#### GraphQL Resolvers

- [X] T041 [P] [US1] Implement HookQuery resolver (list, get) in `internal/api/graphql/resolver/hook.resolvers.go`
- [X] T042 [P] [US1] Implement HookMutation resolver (create, update, delete) in `internal/api/graphql/resolver/hook.resolvers.go`
- [X] T043 [US1] Implement Task.hooks field resolver in `internal/api/graphql/resolver/hook.resolvers.go`
- [X] T044 [US1] Implement Connection.hooks field resolver in `internal/api/graphql/resolver/hook.resolvers.go`
- [X] T045 [US1] Implement Hook.task and Hook.connection field resolvers in `internal/api/graphql/resolver/hook.resolvers.go`
- [X] T046 [US1] Add validation for taskId/connectionId mutual exclusivity in CreateHookInput resolver

### Integration Tests

- [X] T047 [US1] Integration test: create HTTP hook, run task successfully, verify hook triggered in `internal/api/graphql/resolver/hook_test.go`
- [X] T048 [US2] Integration test: create command hook, trigger task failure, verify command executed in `internal/api/graphql/resolver/hook_test.go`
- [X] T049 [US1] Integration test: verify hook execution logged to JobLog with correct format in `internal/api/graphql/resolver/hook_test.go`
- [X] T050 [US1] Integration test: verify disabled hooks are NOT triggered during task execution in `internal/api/graphql/resolver/hook_test.go`

**Checkpoint**: At this point, HTTP and Command hooks should be fully functional for on_success, on_failure, on_end events

---

## Phase 4: User Story 3 - Web UI 配置 Hooks (Priority: P2)

**Goal**: 通过 Web 界面直观地配置和管理 hooks

**Independent Test**: 通过 Web UI 在任务详情页创建、编辑、删除 hook 配置，验证配置正确保存和生效

### GraphQL Operations

- [X] T051 [P] [US3] Create hook GraphQL operations in `web/src/api/graphql/queries/hooks.ts` (queries: list, get; mutations: create, update, delete)
- [X] T052 [P] [US3] Add hooks field to Task query in existing task operations file
- [X] T053 [P] [US3] Add hooks field to Connection query in existing connection operations file

### UI Components

- [X] T054 [US3] Create HookList component in `web/src/modules/connections/components/HookList.tsx` - display hooks with event type, hook type, enabled status, priority, actions (edit/delete)
- [X] T055 [US3] Create HookForm component in `web/src/modules/connections/components/HookForm.tsx` - form for creating/editing hooks with event type, hook type, enabled toggle, priority, on_error select
- [X] T056 [US3] Add HTTP config fields to HookForm: URL, method select (GET/POST/PUT), headers editor, body textarea with template syntax hint
- [X] T057 [US3] Add Command config fields to HookForm: command input, workDir input, timeout number input
- [X] T058 [US3] Implement form validation in HookForm - require URL for HTTP type (blocking), require command for COMMAND type (blocking), show non-blocking warning for invalid URL format
- [X] T059 [US3] Add HookList to Task settings section (integrate with existing TaskSettingsForm or similar view)
- [X] T060 [US3] Add HookList to Connection settings section (integrate with existing connection detail view)
- [X] T061 [US3] Implement add hook dialog/modal triggered from HookList
- [X] T062 [US3] Implement edit hook dialog/modal with pre-populated form
- [X] T063 [US3] Implement delete hook confirmation dialog
- [X] T064 [US3] Add optimistic UI updates for hook CRUD operations using urql cache

### Global Hook Enable/Disable

- [X] T065 [US3] Implement conditional rendering in frontend: hide hook-related UI when global hook is disabled
- [X] T066 [US3] Add GraphQL query for hook enabled status (extend existing config/system query or add new)

**Checkpoint**: At this point, users can fully configure hooks via Web UI

---

## Phase 5: User Story 4 - on_start 事件支持 (Priority: P3)

**Goal**: 在同步任务开始执行时触发 hook，支持阻止任务执行

**Independent Test**: 配置任务开始时的 hook，启动同步任务，验证 hook 在任务实际执行前被触发

### Tests for User Story 4

- [X] T067 [P] [US4] Unit test for on_start hook execution before sync in `internal/core/hook/executor_test.go`
- [X] T068 [P] [US4] Unit test for on_start CANCEL behavior - hook fails, job marked CANCELLED in `internal/core/hook/executor_test.go`
- [X] T069 [P] [US4] Unit test for on_start FATAL behavior - hook fails, triggers on_failure then on_end in `internal/core/hook/executor_test.go`

### Implementation for User Story 4

- [X] T070 [US4] Inject hook trigger for on_start event at beginning of SyncEngine.RunTask in `internal/rclone/sync.go`
- [X] T071 [US4] Implement CANCEL error handling for on_start: mark job CANCELLED, skip sync, jump to on_end in `internal/rclone/sync.go`
- [X] T072 [US4] Implement FATAL error handling for on_start: mark job FAILED, trigger on_failure then on_end in `internal/rclone/sync.go`

### Integration Tests

- [X] T073 [US4] Integration test: on_start hook with CANCEL on error, verify sync is prevented in `internal/api/graphql/resolver/hook_test.go`
- [X] T074 [US4] Integration test: on_start hook with FATAL on error, verify on_failure and on_end triggered in `internal/api/graphql/resolver/hook_test.go`
- [X] T075 [US4] Integration test: IGNORE on_error mode does not change task status while still triggering on_end in `internal/api/graphql/resolver/hook_test.go`

**Checkpoint**: on_start event now fully functional with error handling behaviors

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T076 [P] Add hook-related i18n translations to `web/project.inlang/messages/` (en.json, zh-CN.json) for all hook-related UI text
- [X] T077 [P] Add backend i18n translations for hook execution errors and logs to `internal/i18n/locales/` (en.toml, zh-CN.toml)
- [X] T078 Implement I18nError pattern for hook execution failures in `internal/core/hook/errors.go`
- [X] T079 Add comment to `JobLog.size` in `internal/api/graphql/schema/job.graphql` to explain its usage (duration/status code) for HOOK action
- [X] T080 [P] Add validation for HookConfig URL format (basic URL validation) and Body template syntax (check for valid Go template syntax)
- [X] T081 [P] Add validation for HookConfig timeout range (1-3600 seconds)
- [X] T082 Code review: ensure no type assertions to `any` or `interface{}` in hook package
- [X] T083 Run quickstart.md validation - test all sample GraphQL operations from quickstart
- [X] T084 人工验证：确认 Hook 执行记录（action=HOOK）在 Job 详情视图的日志列表中正确展示（包含 hook 名称、事件类型、执行结果、耗时）

### 人工测试：User Story 1 & 2 端到端测试 (HTTP Hook & Command Hook)

> **NOTE**: 这些测试需要人工操作完成，验证 hook 机制在真实环境下的完整工作流程

- [X] MT001 [US1] 人工测试：HTTP Hook 端到端 - 配置任务成功时的 HTTP hook（指向 https://webhook.site 或本地 mock server），执行同步任务成功，验证目标 URL 收到正确的 POST 请求（包含任务名称、执行状态、传输统计等信息）
- [X] MT002 [US1] 人工测试：HTTP Hook 自定义 Headers - 配置带有自定义 Headers（如 `Authorization: Bearer xxx`、`X-Custom-Header: value`）的 HTTP hook，触发执行，验证请求中包含所有自定义 Headers
- [X] MT003 [US1] 人工测试：HTTP Hook 失败隔离 - 配置 HTTP hook 指向不存在的 URL（如 `http://invalid-domain-12345.com/hook`），执行同步任务成功，验证：(1) 任务状态仍显示为成功，(2) Hook 执行失败被正确记录到 JobLog
- [X] MT004 [US1] 人工测试：HTTP Hook 模板渲染 - 配置 HTTP hook 的 Body 使用模板语法（如 `{"task": "{{.Task.Name}}", "status": "{{.Job.Status}}", "files": {{.Stats.FilesTransferred}}}`），验证请求 body 被正确渲染为 JSON
- [X] MT005 [US2] 人工测试：命令 Hook 端到端 - 配置任务失败时的命令 hook（执行 `echo "Task $RCLONE_SYNC_TASK_NAME failed" >> /tmp/hook-test.log`），触发任务失败，验证：(1) 日志文件被创建，(2) 文件内容包含正确的任务名称
- [X] MT006 [US2] 人工测试：命令 Hook 环境变量 - 配置命令 hook（执行 `env | grep RCLONE_SYNC >> /tmp/hook-env.log`），触发任务执行，验证所有 RCLONE_SYNC_* 环境变量（TASK_ID, TASK_NAME, JOB_ID, EVENT, STATUS, FILES_TRANSFERRED, BYTES_TRANSFERRED, DURATION_SECONDS 等）正确传递
- [X] MT007 [US2] 人工测试：命令 Hook 超时 - 配置命令 hook（执行 `sleep 60`，超时设置为 2 秒），触发任务执行，验证：(1) 命令在 2 秒后被终止，(2) 超时错误被记录到 JobLog
- [X] MT008 [US2] 人工测试：命令 Hook 失败隔离 - 配置命令 hook（执行 `exit 1` 或不存在的命令），执行同步任务成功，验证：(1) 任务状态仍显示为成功，(2) 命令失败及退出码被记录到 JobLog

### 人工测试：User Story 3 Web UI 交互测试

> **NOTE**: 这些测试验证 Web UI 的交互流程和用户体验

- [X] MT009 [US3] 人工测试：Task Hook 创建流程 - 在任务详情页点击添加 Hook，选择事件类型（on_success）、Hook 类型（HTTP），填写 URL 和 Body 模板，保存后验证：(1) Hook 出现在列表中，(2) 配置信息正确显示
- [X] MT010 [US3] 人工测试：Task Hook 编辑流程 - 编辑已有的 Hook 配置，修改 URL 和优先级，保存后验证修改被正确保存
- [X] MT011 [US3] 人工测试：Task Hook 删除流程 - 删除一个 Hook，确认删除弹窗，验证 Hook 从列表中移除
- [X] MT012 [US3] 人工测试：Connection Hook 创建 - 在连接详情页创建 Hook，验证：(1) Hook 成功创建并显示在列表，(2) 关联该连接的任务执行时，Connection Hook 被触发执行
- [X] MT013 [US3] 人工测试：Hook 启用/禁用切换 - 创建 Hook 后禁用它，执行任务，验证禁用的 Hook 不被触发；重新启用后执行任务，验证 Hook 被触发
- [X] MT014 [US3] 人工测试：全局禁用 Hook - 在 config.toml 中设置 `[app.hook] enabled = false`，重启服务，验证：(1) Web UI 中 Hook 相关界面完全隐藏，(2) 已配置的 Hooks 不会被触发执行
- [X] MT015 [US3] 人工测试：表单验证 - 尝试创建 HTTP Hook 但不填写 URL，验证保存按钮被禁用并显示验证错误；尝试创建 Command Hook 但不填写命令，验证同样的验证行为
- [X] MT016 [US3] 人工测试：URL 格式警告 - 填写无效格式的 URL（如 `not-a-url`），验证显示阻塞警告且不允许保存

### 人工测试：User Story 4 on_start 事件测试

> **NOTE**: 这些测试验证 on_start 事件及其特殊的错误处理行为

- [X] MT017 [US4] 人工测试：on_start Hook 正常执行 - 配置 on_start HTTP hook（创建标记文件或发送请求），启动同步任务，验证：(1) Hook 在任务实际同步开始前被触发，(2) 任务正常完成
- [X] MT018 [US4] 人工测试：on_start CANCEL 模式 - 配置 on_start 命令 hook（执行 `exit 1`），设置 on_error 为 CANCEL，启动任务，验证：(1) 任务被取消，(2) Job 状态为 CANCELLED，(3) 同步操作未执行，(4) on_end hook 被触发
- [X] MT019 [US4] 人工测试：on_start FATAL 模式 - 配置 on_start 命令 hook（执行 `exit 1`），设置 on_error 为 FATAL，启动任务，验证：(1) Job 状态为 FAILED，(2) 同步操作未执行，(3) on_failure 和 on_end hooks 依次被触发

### 人工测试：Edge Cases 边界情况测试

> **NOTE**: 这些测试验证边界情况和特殊场景

- [X] MT020 人工测试：多 Hook 执行顺序 - 为同一事件（on_success）配置 3 个 Hooks，优先级分别设为 30、10、20，每个 Hook 写入不同标记到日志文件，触发任务成功，验证日志文件记录顺序为：priority 10 → 20 → 30
- [X] MT021 人工测试：空同步 Hook 触发 - 配置 on_success hook，执行一个无任何文件变更的同步任务（源和目标完全一致），验证 Hook 仍被正确触发（Stats.FilesTransferred = 0）
- [X] MT022 人工测试：环境变量引用 - 设置环境变量 `export TEST_API_KEY=secret123`，配置 HTTP hook 的 Header 使用 `Authorization: Bearer {{.Env.TEST_API_KEY}}`，触发执行，验证请求 Header 包含 `Authorization: Bearer secret123`
- [X] MT023 人工测试：Task + Connection Hook 混合 - 在 Connection 上配置 priority=20 的 Hook，在 Task 上配置 priority=10 的 Hook，执行任务，验证两个 Hooks 按优先级混合排序后依次执行（Task Hook 先于 Connection Hook）
- [X] MT024 人工测试：on_end 事件 always 触发 - 分别测试任务成功、任务失败、任务取消三种场景，验证 on_end hook 在所有情况下都被触发

### 人工测试：Accessibility 可访问性测试

- [X] MT025 人工测试：键盘导航 - 仅使用键盘（Tab、Enter、Escape 等）完成 Hook 创建、编辑、删除的完整流程，验证所有操作可通过键盘完成
- [X] MT026 人工测试：屏幕阅读器 - 使用屏幕阅读器（如 NVDA、VoiceOver）浏览 Hook 配置界面，验证所有表单元素有正确的 ARIA 标签和提示

- [X] T085 Update AGENTS.md if needed with hook-related development notes
- [X] T086 Verify all new code follows existing project patterns (error handling, logging, etc.)

### Accessibility Tests

- [X] T087 [P] Accessibility test: verify HookForm component keyboard navigation in `web/src/modules/connections/components/HookForm.test.tsx`
- [X] T088 [P] Accessibility test: verify ARIA attributes and focus states for all hook-related UI components

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories 1 & 2 (Phase 3)**: Depends on Foundational phase completion - Core hook execution
- **User Story 3 (Phase 4)**: Depends on Phase 3 completion (needs working hooks) - Web UI
- **User Story 4 (Phase 5)**: Can run in parallel with Phase 4 after Phase 3 - on_start event
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 & 2 (P1)**: Can start after Foundational (Phase 2) - HTTP and Command hooks share infrastructure
- **User Story 3 (P2)**: Depends on US1/US2 completion - UI needs working GraphQL API
- **User Story 4 (P3)**: Can start after US1/US2 - Adds on_start event support to existing executor

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Models/types before services
- Services/logic before API layer
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- All tests for a user story marked [P] can run in parallel
- US4 (on_start) and US3 (Web UI) can run in parallel after US1/US2 complete

---

## Parallel Example: User Stories 1 & 2 Tests

```bash
# Launch all tests for User Stories 1 & 2 together:
Task: "Unit test for HTTP hook execution in internal/core/hook/http_test.go"
Task: "Unit test for HTTP hook timeout in internal/core/hook/http_test.go"
Task: "Unit test for HTTP hook error handling in internal/core/hook/http_test.go"
Task: "Unit test for command hook execution in internal/core/hook/command_test.go"
Task: "Unit test for command hook timeout in internal/core/hook/command_test.go"
Task: "Unit test for command hook exit code handling in internal/core/hook/command_test.go"
Task: "Unit test for template rendering in internal/core/hook/template_test.go"
Task: "Unit test for environment variable interpolation in internal/core/hook/template_test.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 2 Only)

1. Complete Phase 1: Setup (Schema, migrations)
2. Complete Phase 2: Foundational (Config, models, executor framework)
3. Complete Phase 3: User Stories 1 & 2 (HTTP + Command hooks)
4. **STOP and VALIDATE**: Test hooks work via GraphQL API
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add US1 & US2 → Test via GraphQL → Deploy/Demo (MVP!)
3. Add US3 → Test Web UI → Deploy/Demo
4. Add US4 → Test on_start → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: HTTP Hook execution (US1)
   - Developer B: Command Hook execution (US2)
3. After US1 & US2 complete:
   - Developer A: Web UI (US3)
   - Developer B: on_start event (US4)
4. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence

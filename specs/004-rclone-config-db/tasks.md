# Tasks: Rclone 连接配置数据库存储

**Input**: Design documents from `/specs/004-rclone-config-db/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, quickstart.md ✓, contracts/openapi.yaml ✓

**Tests**: 根据 Constitution 要求 (III. Test-Driven Development: ✅ REQUIRED)，所有新功能需先编写测试。

**Organization**: 任务按用户故事分组，以支持独立实现和测试。

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 任务所属用户故事 (如 US1, US2, US3)
- 描述中包含确切的文件路径

## Path Conventions

- **Backend**: `internal/` (Go)
- **Frontend**: `web/src/` (SolidJS)
- **Tests**: `*_test.go` (Go), `*.test.ts` (TypeScript)

---

## Phase 1: Setup（项目初始化）

**Purpose**: 创建项目基础结构和配置

- [x] T001 在 `internal/core/config/config.go` 中添加 Security.EncryptionKey 配置字段
- [x] T002 [P] 创建空的 `internal/core/crypto/` 目录结构

---

## Phase 2: Foundational（基础设施）

**Purpose**: 所有用户故事的核心依赖，必须先完成

**⚠️ CRITICAL**: 在此阶段完成前不能开始任何用户故事工作

### 数据库 Schema

- [x] T003 创建 Connection 实体 schema 在 `internal/core/db/schema/connection.go`
- [x] T004 修改 Task schema，添加 connection_id 外键，移除 remote_name 在 `internal/core/db/schema/task.go`
- [x] T005 运行 `go generate ./internal/core/ent` 生成 Ent 代码
- [x] T006 验证数据库迁移：运行应用确认 schema 变更生效

### 加密模块

- [x] T007 创建加密器接口和测试 在 `internal/core/crypto/crypto_test.go`
- [x] T008 实现 AES-256-GCM 加密器 在 `internal/core/crypto/crypto.go`

### 接口定义

- [x] T009 在 `internal/core/ports/interfaces.go` 中定义 ConnectionService 接口

**Checkpoint**: 基础设施就绪 - 用户故事实现可以开始

---

## Phase 3: User Story 5 - 敏感信息安全存储 (Priority: P1) 🎯 MVP

**Goal**: 确保所有连接配置中的敏感信息以加密形式存储在数据库中

**Independent Test**: 创建连接后直接查询数据库，验证 encrypted_config 字段不可读且解密后正确

### Tests for User Story 5

- [x] T010 [P] [US5] 单元测试：加密/解密配置 map 在 `internal/core/crypto/crypto_test.go`
- [x] T011 [P] [US5] 单元测试：密钥错误时解密失败 在 `internal/core/crypto/crypto_test.go`

### Implementation for User Story 5

- [x] T012 [US5] 实现 NewEncryptor() 构造函数，验证密钥长度 在 `internal/core/crypto/crypto.go`
- [x] T013 [US5] 实现 EncryptConfig() 方法 在 `internal/core/crypto/crypto.go`
- [x] T014 [US5] 实现 DecryptConfig() 方法 在 `internal/core/crypto/crypto.go`

**Checkpoint**: 加密模块完成，可安全存储敏感信息

---

## Phase 4: User Story 1 - 创建新的云存储连接 (Priority: P1)

**Goal**: 用户可以添加新的云存储连接，配置安全存储到数据库

**Independent Test**: 通过 API 创建连接，验证数据库中有记录且配置已加密

### Tests for User Story 1

- [x] T015 [P] [US1] 单元测试：ConnectionService.CreateConnection 在 `internal/core/services/connection_service_test.go`
- [x] T016 [P] [US1] 单元测试：重复名称创建失败 在 `internal/core/services/connection_service_test.go`
- [x] T017 [P] [US1] API 测试：POST /connections 在 `internal/api/handlers/connection_test.go`

### Implementation for User Story 1

- [x] T018 [US1] 实现 ConnectionService 结构体和构造函数 在 `internal/core/services/connection_service.go`
- [x] T019 [US1] 实现 CreateConnection() 方法 在 `internal/core/services/connection_service.go`
- [x] T020 [US1] 实现连接名称验证函数 ValidateConnectionName() 在 `internal/core/services/connection_service.go`
- [x] T021 [US1] 创建 ConnectionHandler 和 Create() 处理函数 在 `internal/api/handlers/connection.go`
- [x] T022 [US1] 在 `internal/api/routes.go` 注册 POST /connections 路由

**Checkpoint**: 可以创建新连接并安全存储 ✅

---

## Phase 4.5: Schema Migration - Legacy Code Update

**Goal**: 修复 Task.remote_name → Task.connection_id 迁移导致的编译错误

**Context**: Phase 2 中的 schema 变更（T004）将 Task.remote_name (string) 改为 Task.connection_id (UUID foreign key)，影响了所有使用 remote_name 的代码。此阶段必须在 Phase 4 完成后立即执行，确保代码可编译和测试。

**Independent Test**: 运行 `go build ./...` 验证无编译错误，运行 `go test ./...` 验证所有测试通过

### Core Service Migration (从 Phase 12 移动)

- [x] T092 更新 TaskService 使用 connection_id 在 `internal/core/services/task_service.go`
  - [x] CreateTask: remote_name string → connectionID uuid.UUID
  - [x] UpdateTask: remote_name string → connectionID uuid.UUID
  - [x] ListTasksByConnection: remote_name string → connectionID uuid.UUID
- [x] T092.1 更新 JobService 使用 connection_id 在 `internal/core/services/job_service.go`
  - [x] ListJobs: remoteName string → connectionID \*uuid.UUID
  - [x] CountJobs: remoteName string → connectionID \*uuid.UUID
  - [x] ListJobLogs: remoteName string → connectionID \*uuid.UUID
  - [x] CountJobLogs: remoteName string → connectionID \*uuid.UUID
- [x] T092.2 更新 JobService 接口定义 在 `internal/core/ports/interfaces.go`

### API Handler Updates

- [x] T093 更新 TaskHandler 使用 connection_id 在 `internal/api/handlers/task.go`
  - [x] CreateTask: 从请求体接收 connection_id (UUID string)
  - [x] UpdateTask: 从请求体接收 connection_id (可选)
  - [x] 添加验证：connection_id 必须存在且有效
- [x] T093.1 更新 JobHandler 使用 connection_id 在 `internal/api/handlers/job.go`
  - [x] ListJobs: 从 query 参数接收 connection_id 替代 remote_name
  - [x] 解析 UUID 并传递给 JobService
- [x] T093.2 更新 LogHandler 使用 connection_id 在 `internal/api/handlers/log.go`
  - [x] ListJobLogs: 从 query 参数接收 connection_id 替代 remote_name
  - [x] 解析 UUID 并传递给 JobService

### Test File Updates - Services

- [x] T094 [P] 更新 task_service_test.go 在 `internal/core/services/task_service_test.go`
  - [x] 所有测试：先创建 Connection，使用 conn.ID 替代 "remote-name"
- [x] T094.1 [P] 更新 job_service_test.go 在 `internal/core/services/job_service_test.go`
  - [x] 所有测试：创建 Connection，使用 conn.ID 参数
- [x] T094.2 [P] 更新 crash_recovery_test.go 在 `internal/core/services/crash_recovery_test.go`
  - [x] 测试设置：创建测试 Connection

### Test File Updates - API Handlers

- [x] T094.3 [P] 更新 task_test.go 在 `internal/api/handlers/task_test.go`
  - [x] 所有测试：请求体使用 connection_id 字段
  - [x] 添加无效 connection_id 测试
- [x] T094.4 [P] 更新 job_test.go 在 `internal/api/handlers/job_test.go`
  - [x] 所有测试：query 参数使用 connection_id
- [x] T094.5 [P] 更新 log_test.go 在 `internal/api/handlers/log_test.go`
  - [x] 所有测试：query 参数使用 connection_id
- [x] T094.6 [P] 更新 setup_test.go 在 `internal/api/handlers/setup_test.go`
  - [x] helper 函数：创建测试 Connection

### Test File Updates - Rclone Integration

- [x] T094.7 [P] 更新 sync_test.go 在 `internal/rclone/sync_test.go`
  - [x] MockTaskService: CreateTask 签名更新
  - [x] 测试用例：使用 uuid.New() 生成 connection_id
- [x] T094.8 [P] 更新 sync_direction_test.go 在 `internal/rclone/sync_direction_test.go`
  - [x] 测试设置：提供有效 connection_id
- [x] T094.9 [P] 更新 sync_integration_test.go 在 `internal/rclone/sync_integration_test.go`
  - [x] 集成测试：创建真实 Connection 或使用 mock

### Implementation Updates (从 Phase 12 移动)

- [x] T095 更新 sync.go 使用 Connection 在 `internal/rclone/sync.go`
  - [x] 从 Task.Edges.Connection 获取配置
  - [x] 移除直接使用 remote_name 的代码
- [x] T095.1 [P] 更新 remote.go 使用 ConnectionService 在 `internal/api/handlers/remote.go`
  - [x] ListProviders: 保持不变（静态数据）
  - [x] GetProviderOptions: 保持不变
  - [x] 注：remote.go 管理旧的 rclone remotes API，与新的 Connection 系统并行存在

### Verification

- [x] T096 验证编译通过 `go build ./...`
- [x] T097 验证所有单元测试通过 `go test ./internal/core/services/... ./internal/api/handlers/...`
- [x] T098 验证 rclone 集成测试通过 `go test ./internal/rclone/...`

**Checkpoint**: 所有代码编译通过，所有测试通过 ✅

---

## Phase 5: User Story 2 - 查看和管理现有连接 (Priority: P1)

**Goal**: 用户可以查看所有已配置的连接列表和详情

**Independent Test**: 创建多个连接后,通过 API 获取列表并验证完整性

### Tests for User Story 2

- [x] T023 [P] [US2] 单元测试：ConnectionService.ListConnections 在 `internal/core/services/connection_service_test.go`
- [x] T024 [P] [US2] 单元测试：ConnectionService.GetConnectionByName 在 `internal/core/services/connection_service_test.go`
- [x] T025 [P] [US2] API 测试：GET /connections 在 `internal/api/handlers/connection_test.go`
- [x] T026 [P] [US2] API 测试：GET /connections/:name 在 `internal/api/handlers/connection_test.go`

### Implementation for User Story 2

- [x] T027 [US2] 实现 ListConnections() 方法 在 `internal/core/services/connection_service.go`
- [x] T028 [US2] 实现 GetConnectionByName() 方法 在 `internal/core/services/connection_service.go`
- [x] T029 [US2] 实现 GetConnectionConfig() 方法（返回解密配置用于编辑）在 `internal/core/services/connection_service.go`
- [x] T030 [US2] 实现 List() 和 Get() 处理函数 在 `internal/api/handlers/connection.go`
- [x] T031 [US2] 实现 GetConfig() 处理函数 在 `internal/api/handlers/connection.go`
- [x] T032 [US2] 注册 GET /connections, GET /connections/:name, GET /connections/:name/config 路由

**Checkpoint**: 可以查看连接列表和详情 ✅

---

## Phase 6: User Story 3 - 更新连接配置 (Priority: P2)

**Goal**: 用户可以修改现有连接的配置信息

**Independent Test**: 更新连接配置后，重新获取验证更改已保存

### Tests for User Story 3

- [x] T033 [P] [US3] 单元测试：ConnectionService.UpdateConnection 在 `internal/core/services/connection_service_test.go`
- [x] T034 [P] [US3] API 测试：PUT /connections/:name 在 `internal/api/handlers/connection_test.go`

### Implementation for User Story 3

- [x] T035 [US3] 实现 UpdateConnection() 方法 在 `internal/core/services/connection_service.go`
- [x] T036 [US3] 实现 Update() 处理函数 在 `internal/api/handlers/connection.go`
- [x] T037 [US3] 注册 PUT /connections/:name 路由

**Checkpoint**: 可以更新连接配置 ✅

---

## Phase 7: User Story 4 - 删除连接 (Priority: P2)

**Goal**: 用户可以删除不再需要的连接，级联删除关联的任务

**Independent Test**: 删除连接后，验证连接和关联任务都已从数据库移除

### Tests for User Story 4

- [x] T038 [P] [US4] 单元测试：ConnectionService.DeleteConnectionByName 在 `internal/core/services/connection_service_test.go`
- [x] T039 [P] [US4] 单元测试：级联删除关联 Task 在 `internal/core/services/connection_service_test.go`
- [x] T040 [P] [US4] API 测试：DELETE /connections/:name 在 `internal/api/handlers/connection_test.go`

### Implementation for User Story 4

- [x] T041 [US4] 实现 DeleteConnectionByName() 方法 在 `internal/core/services/connection_service.go`
- [x] T042 [US4] 实现 HasAssociatedTasks() 方法用于警告检查 在 `internal/core/services/connection_service.go`
- [x] T043 [US4] 实现 Delete() 处理函数（支持 force 参数）在 `internal/api/handlers/connection.go`
- [x] T044 [US4] 注册 DELETE /connections/:name 路由

**Checkpoint**: 可以安全删除连接 ✅

---

## Phase 8: User Story 6 - 令牌自动刷新与连接状态监控 (Priority: P2)

**Goal**: 依赖 rclone 内置令牌刷新机制，提供连接状态监控

**Independent Test**: 使用 OAuth 连接执行操作，验证令牌刷新后数据库配置已更新

### Tests for User Story 6

- [x] T045 [P] [US6] 单元测试：DBStorage.GetValue 在 `internal/rclone/storage_test.go`
- [x] T046 [P] [US6] 单元测试：DBStorage.SetValue 在 `internal/rclone/storage_test.go`
- [x] T047 [P] [US6] 单元测试：DBStorage.HasSection 在 `internal/rclone/storage_test.go`
- [x] T048 [P] [US6] 单元测试：IsConnectionLoaded() 缓存检查 在 `internal/rclone/cache_helper_test.go`
- [x] T049 [P] [US6] API 测试：POST /connections/:name/test 在 `internal/api/handlers/connection_test.go`
- [x] T050 [P] [US6] API 测试：GET /connections/:name/quota 在 `internal/api/handlers/connection_test.go`

### Implementation for User Story 6

- [x] T051 [US6] 创建 DBStorage 结构体和 NewDBStorage() 在 `internal/rclone/storage.go`
- [x] T052 [US6] 实现 DBStorage.GetSectionList() 和 HasSection() 在 `internal/rclone/storage.go`
- [x] T053 [US6] 实现 DBStorage.GetKeyList() 和 GetValue() 在 `internal/rclone/storage.go`
- [x] T054 [US6] 实现 DBStorage.SetValue() 和 DeleteKey() 在 `internal/rclone/storage.go`
- [x] T055 [US6] 实现 DBStorage.DeleteSection() 在 `internal/rclone/storage.go`
- [x] T056 [US6] 实现 DBStorage.Load(), Save(), Serialize() 在 `internal/rclone/storage.go`
- [x] T057 [US6] 实现 DBStorage.Install() 方法 在 `internal/rclone/storage.go`
- [x] T058 [US6] 创建 IsConnectionLoaded() 辅助函数 在 `internal/rclone/cache_helper.go`
- [x] T059 [US6] 实现 Test() 处理函数（测试已保存连接）在 `internal/api/handlers/connection.go`
- [x] T060 [US6] 实现 TestUnsavedConfig() 处理函数 在 `internal/api/handlers/connection.go`
- [x] T061 [US6] 实现 GetQuota() 处理函数 在 `internal/api/handlers/connection.go`
- [x] T062 [US6] 注册 POST /connections/test, POST /connections/:name/test, GET /connections/:name/quota 路由
- [x] T063 [US6] 在应用启动时安装 DBStorage 在 `cmd/cloud-sync/serve.go`

**Checkpoint**: rclone 令牌刷新自动同步到数据库，可获取连接状态 ✅

---

## Phase 9: User Story 7 - 从 rclone.conf 文件导入连接 (Priority: P2)

**Goal**: 用户通过多步向导从 rclone.conf 批量导入连接

**Independent Test**: 准备 rclone.conf 内容，完成导入向导，验证所有连接正确创建

### Tests for User Story 7

- [x] T064 [P] [US7] 单元测试：ParseRcloneConf() 解析 在 `internal/rclone/parser_test.go`
- [x] T065 [P] [US7] 单元测试：解析空/无效内容 在 `internal/rclone/parser_test.go`
- [x] T066 [P] [US7] 单元测试：检测内部名称重复 在 `internal/rclone/parser_test.go`
- [x] T067 [P] [US7] API 测试：POST /import/parse 在 `internal/api/handlers/import_test.go`
- [x] T068 [P] [US7] API 测试：POST /import/execute 在 `internal/api/handlers/import_test.go`

### Implementation for User Story 7

- [x] T069 [US7] 创建 ParsedConnection 结构体 在 `internal/rclone/parser.go`
- [x] T070 [US7] 实现 ParseRcloneConf() 函数（使用 goconfig）在 `internal/rclone/parser.go`
- [x] T071 [US7] 实现 ValidateImport() 函数（检测重复和冲突）在 `internal/rclone/parser.go`
- [x] T072 [US7] 创建 ImportHandler 结构体 在 `internal/api/handlers/import.go`
- [x] T073 [US7] 实现 Parse() 处理函数 在 `internal/api/handlers/import.go`
- [x] T074 [US7] 实现 Execute() 处理函数（批量导入/覆盖）在 `internal/api/handlers/import.go`
- [x] T075 [US7] 注册 POST /import/parse, POST /import/execute 路由

**Checkpoint**: 可以从 rclone.conf 导入连接 ✅

---

## Phase 10: Frontend - 连接管理界面

**Purpose**: 前端用户界面更新，适配新的 /connections API（使用 UUID 标识）

**⚠️ API Breaking Changes**:

- `/remotes` → `/connections`
- 路径参数：`name` (string) → `id` (UUID)
- 前端路由：`/connections/:name` → `/connections/:id`

### 类型定义和 API 客户端

- [x] T076 [P] 更新类型定义 在 `web/src/lib/types.ts`

  - Connection 类型（id, name, type, load_status, load_error, created_at, updated_at）
  - LoadStatus 类型 ('loaded' | 'loading' | 'error')
  - ConnectionConfig, ImportParseResult, ImportPreviewItem, ImportError, ImportResult
  - 更新 Task 类型：remote_name → connection_id

- [x] T077 [P] 重构连接 API 客户端 在 `web/src/api/connections.ts`

  - 迁移到 /connections API
  - 使用 id 替代 name 作为路径参数
  - 新增 getConnection(id), updateConnection(id), getConnectionConfig(id), testConnection(id)

- [x] T078 [P] 添加导入 API 客户端 在 `web/src/api/connections.ts`
  - parseImport(content), executeImport(connections)

### 路由和布局更新

- [x] T079 更新前端路由 在 `web/src/App.tsx`

  - `/connections/:name/*` → `/connections/:id/*`

- [x] T080 更新 ConnectionLayout 在 `web/src/modules/connections/layouts/ConnectionLayout.tsx`

  - 使用 id 参数获取连接详情

- [x] T081 更新 Sidebar 连接列表 在 `web/src/modules/core/components/Sidebar.tsx`
  - 链接地址改为 `/connections/{id}`

### 连接状态显示

- [x] T082 创建 ConnectionStatusBadge 组件 在 `web/src/modules/connections/components/ConnectionStatusBadge.tsx`
- [x] T083 更新连接概览页面 在 `web/src/modules/connections/views/Overview.tsx`
  - 显示连接状态徽章，使用 id 进行操作

### 任务相关更新

- [x] T084 更新 Task 相关组件 在 `web/src/modules/connections/components/`
  - CreateTaskWizard: 使用 connection_id
  - EditTaskDialog: 使用 connection_id
  - Tasks.tsx: 显示关联的 connection

### 导入向导组件

- [x] T085 创建 ImportWizard 容器组件 在 `web/src/modules/connections/components/ImportWizard/ImportWizard.tsx`
- [x] T086 [P] 创建 Step1Input 组件（粘贴配置）在 `web/src/modules/connections/components/ImportWizard/Step1Input.tsx`
- [x] T087 [P] 创建 Step2Preview 组件（预览编辑）在 `web/src/modules/connections/components/ImportWizard/Step2Preview.tsx`
- [x] T088 [P] 创建 Step3Confirm 组件（确认导入）在 `web/src/modules/connections/components/ImportWizard/Step3Confirm.tsx`
- [x] T089 在连接管理页面集成导入向导入口

### 删除和设置更新

- [x] T090 更新删除连接确认对话框（使用 id，显示级联删除警告）
- [x] T091 更新 Settings 页面 在 `web/src/modules/connections/views/Settings.tsx`
  - 使用 id 获取和更新配置

---

## Phase 11: i18n 翻译

**Purpose**: 国际化支持

- [x] T092 [P] 添加后端翻译键 在 `internal/i18n/keys.go` (后端翻译键已完整)
- [x] T093 [P] 添加英文翻译 在 `internal/i18n/locales/en.toml` (后端翻译已完整)
- [x] T094 [P] 添加中文翻译 在 `internal/i18n/locales/zh-CN.toml` (后端翻译已完整)
- [x] T095 [P] 添加前端英文翻译 在 `web/project.inlang/messages/en.json` (新增 36 个翻译键)
- [x] T096 [P] 添加前端中文翻译 在 `web/project.inlang/messages/zh-CN.json` (新增 36 个翻译键)

---

## Phase 12: Polish & Cross-Cutting Concerns

**Purpose**: 完善和优化

**Note**: Phase 4.5 已完成 TaskService/JobService/sync.go 迁移到 connection_id

- [x] T097 运行所有测试验证功能 `go test ./...`
- [x] T098 运行 quickstart.md 验证步骤

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: 无依赖 - 可立即开始
- **Phase 2 (Foundational)**: 依赖 Phase 1 - 阻塞所有用户故事
- **Phase 3 (US5)**: 依赖 Phase 2 - 加密模块是其他故事的基础
- **Phase 4 (US1)**: 依赖 Phase 3 - 创建连接需要加密
- **Phase 5 (US2)**: 依赖 Phase 4 - 查看需要先有连接
- **Phase 6 (US3)**: 依赖 Phase 5 - 更新需要先能查看
- **Phase 7 (US4)**: 依赖 Phase 5 - 删除需要先能查看
- **Phase 8 (US6)**: 依赖 Phase 4 - DBStorage 需要 ConnectionService
- **Phase 9 (US7)**: 依赖 Phase 4, 5 - 导入需要创建和查看功能
- **Phase 10 (Frontend)**: 依赖 Phase 4-9 所有后端 API
- **Phase 11 (i18n)**: 可与 Phase 3-10 并行
- **Phase 12 (Polish)**: 依赖所有功能完成

### User Story Dependencies

| Story          | Dependencies | Can Parallel With |
| -------------- | ------------ | ----------------- |
| US5 (安全存储) | Foundational | -                 |
| US1 (创建连接) | US5          | -                 |
| US2 (查看连接) | US1          | -                 |
| US3 (更新连接) | US2          | US4               |
| US4 (删除连接) | US2          | US3               |
| US6 (令牌刷新) | US1          | US3, US4          |
| US7 (导入向导) | US1, US2     | US3, US4, US6     |

### Parallel Opportunities

- Phase 2 中的 T007, T008 (加密) 和 T003-T006 (Schema) 可并行
- 每个 Phase 内的测试任务 [P] 可并行
- Frontend Phase 中的类型定义和 API 客户端可并行
- i18n Phase 中所有翻译任务可并行

---

## Parallel Example: Phase 8 (US6)

```bash
# 并行启动所有测试任务:
Task: "T045 单元测试：DBStorage.GetValue"
Task: "T046 单元测试：DBStorage.SetValue"
Task: "T047 单元测试：DBStorage.HasSection"
Task: "T048 单元测试：IsConnectionLoaded()"
Task: "T049 API 测试：POST /connections/:name/test"
Task: "T050 API 测试：GET /connections/:name/quota"
```

---

## Implementation Strategy

### MVP First (US1 + US2 + US5)

1. 完成 Phase 1: Setup
2. 完成 Phase 2: Foundational (CRITICAL)
3. 完成 Phase 3: US5 - 安全存储
4. 完成 Phase 4: US1 - 创建连接
5. 完成 Phase 5: US2 - 查看连接
6. **STOP and VALIDATE**: 测试创建和查看连接流程
7. 部署/演示 MVP

### Incremental Delivery

1. Setup + Foundational + US5 → 安全基础设施就绪
2. 添加 US1 → 可以创建连接 (MVP 核心)
3. 添加 US2 → 可以查看连接 (MVP 完整)
4. 添加 US3 + US4 → 完整 CRUD
5. 添加 US6 → rclone 集成完成
6. 添加 US7 → 导入功能
7. 添加 Frontend → 用户界面
8. 添加 i18n + Polish → 生产就绪

---

## Summary

| Metric                     | Value                     |
| -------------------------- | ------------------------- |
| **Total Tasks**            | 99                        |
| **Setup Phase**            | 2 tasks                   |
| **Foundational Phase**     | 7 tasks                   |
| **US5 (安全存储)**         | 5 tasks                   |
| **US1 (创建连接)**         | 8 tasks                   |
| **US2 (查看连接)**         | 10 tasks                  |
| **US3 (更新连接)**         | 5 tasks                   |
| **US4 (删除连接)**         | 7 tasks                   |
| **US6 (令牌刷新)**         | 19 tasks                  |
| **US7 (导入向导)**         | 12 tasks                  |
| **Frontend**               | 16 tasks (+5)             |
| **i18n**                   | 5 tasks                   |
| **Polish**                 | 3 tasks (-3)              |
| **Parallel Opportunities** | 48+ tasks marked [P]      |
| **MVP Scope**              | Phase 1-5 (US1, US2, US5) |

---

## Notes

- [P] 任务 = 不同文件，无依赖，可并行
- [Story] 标签将任务映射到特定用户故事，便于追踪
- 每个用户故事应可独立完成和测试
- 验证测试先失败再实现
- 每个任务或逻辑组完成后提交
- 在任何检查点停下来独立验证故事
- 避免：模糊任务、同文件冲突、破坏独立性的跨故事依赖

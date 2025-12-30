# Tasks: Task 扩展选项配置

**Input**: Design documents from `/specs/009-task-extended-options/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/schema.graphql

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 项目基础配置和 GraphQL Schema 更新

- [x] T001 扩展 GraphQL Schema - TaskSyncOptions 类型添加 filters, noDelete, transfers 字段 in `internal/api/graphql/schema/task.graphql`
- [x] T002 扩展 GraphQL Schema - TaskSyncOptionsInput 输入类型添加 filters, noDelete, transfers 字段 in `internal/api/graphql/schema/task.graphql`
- [x] T003 扩展 GraphQL Schema - file.remote 查询添加 filters, includeFiles 参数 in `internal/api/graphql/schema/file.graphql`
- [x] T004 运行 go generate 重新生成 GraphQL resolver 代码

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 核心基础设施 - 必须在任何 User Story 开始前完成

**⚠️ CRITICAL**: 所有 User Story 依赖此阶段完成

- [x] T005 [P] 扩展 Config 结构体添加 Sync.Transfers 配置项（默认值 4） in `internal/core/config/config.go`
- [x] T006a [P] 创建过滤器验证单元测试 in `internal/rclone/filter_validator_test.go`
- [x] T006b 创建过滤器验证函数 ValidateFilterRules in `internal/rclone/filter_validator.go`
- [x] T006c [P] 创建 rclone filter 语法可用性验证测试（测试常见模式：glob、通配符、目录排除等） in `internal/rclone/filter_syntax_test.go`
- [x] T007 [P] 扩展 SyncOptions 结构体添加 Filters, NoDelete, Transfers 字段 in `internal/rclone/sync.go`
- [x] T008a 创建 TaskService.validateSyncOptions 的测试 in `internal/core/services/task_service_test.go`
- [x] T008b 扩展 TaskService.validateSyncOptions 方法添加 filters 和 transfers 校验 in `internal/core/services/task_service.go`

**Checkpoint**: 基础设施就绪 - User Story 实现可以开始

---

## Phase 3: User Story 1 - 过滤器配置 (Priority: P1) 🎯 MVP

**Goal**: 用户可以为同步任务配置文件过滤规则，通过可视化规则列表界面配置 Include/Exclude 规则，并预览过滤后的文件

**Independent Test**: 创建一个任务并配置过滤器规则（如排除 `node_modules/**`），然后执行同步，验证被排除的文件不会被同步到目标端

### Backend Implementation for User Story 1

- [x] T009a [US1] 添加 Sync 过滤器注入的单元测试 in `internal/rclone/sync_test.go` (TestApplyFilterRules, TestGetSyncOptionsFromTask)
- [x] T009b [US1] 实现 Sync 方法中的过滤器注入逻辑 - 使用 filter.ReplaceConfig 应用规则 in `internal/rclone/sync.go`
- [x] T010a [US1] 添加 ListRemoteDir 过滤器参数的单元测试 in `internal/rclone/connection_test.go` (包含 basePath 测试)
- [x] T010b [US1] 扩展 ListRemoteDir 函数支持 filters 和 includeFiles 参数（过滤器预览功能）in `internal/rclone/connection.go`
- [x] T011a [US1] 添加 file resolver 过滤器预览的集成测试 in `internal/api/graphql/resolver/file_test.go` (TestFileQuery_RemoteWithFilters, TestFileQuery_RemoteFilterPreview)
- [x] T011b [US1] 更新 file.resolvers.go 处理 filters 和 includeFiles 参数 in `internal/api/graphql/resolver/file.resolvers.go`
- [x] T012 [US1] 更新 task.resolvers.go 处理 TaskSyncOptions 中的 filters 字段 in `internal/api/graphql/resolver/task.resolvers.go`

### Frontend Implementation for User Story 1

- [x] T013 [P] [US1] 更新 GraphQL 查询类型定义 - 添加 filters 相关类型 in `web/src/api/graphql/queries/tasks.ts`
- [x] T014 [P] [US1] 更新 GraphQL 文件查询类型定义 - 添加 filters, includeFiles, basePath 参数 in `web/src/api/graphql/queries/files.ts`
- [x] T015 [US1] 创建 FilterRulesEditor 组件 - 可视化规则列表（Include/Exclude 选择 + 模式输入 + 排序/删除）以及 rclone filter 语法文档链接（https://rclone.org/filtering/#filter-add-a-file-filtering-rule） in `web/src/modules/connections/components/FilterRulesEditor.tsx`
- [x] T016 [US1] 创建 FilterPreviewPanel 组件 - 过滤器预览面板（源端/目标端 Tab 切换 + 懒加载 + 500ms 防抖 + 传递 task.remotePath 作为 basePath 以确保过滤器路径正确匹配）in `web/src/modules/connections/components/FilterPreviewPanel.tsx`
- [x] T017 [US1] 扩展 FileBrowser 组件支持根据 isDir 和文件扩展名显示不同图标 in `web/src/components/common/FileBrowser.tsx` (添加 getFileIcon 到 lib/utils.ts)
- [x] T018 [US1] 在任务设置页面添加 "过滤器" Tab 标签页集成 FilterRulesEditor 和 FilterPreviewPanel in `web/src/modules/connections/components/TaskSettingsForm.tsx`

### i18n for User Story 1

- [x] T019 [P] [US1] 添加过滤器相关英文翻译 in `web/project.inlang/messages/en.json`
- [x] T020 [P] [US1] 添加过滤器相关中文翻译 in `web/project.inlang/messages/zh-CN.json`

### Task Detail Display for User Story 1

- [x] T020a [US1] 在任务详情页展示已配置的扩展选项状态（过滤器规则数量、noDelete 状态、transfers 值） in `web/src/modules/connections/views/Tasks.tsx`

**Checkpoint**: 过滤器配置功能完成 - 用户可以配置过滤规则并预览效果

---

## Phase 4: User Story 2 - 保留删除文件 (Priority: P2)

**Goal**: 用户在创建或编辑单向同步任务时，可以选择启用 "保留删除文件" 选项，启用后同步过程中不会删除目标端的多余文件

**Independent Test**: 创建一个单向同步任务并启用 "保留删除文件" 选项，在源端删除一个文件后执行同步，验证目标端对应的文件不会被删除

### Backend Implementation for User Story 2

- [x] T021a [US2] 添加 NoDelete 逻辑的集成测试 in `internal/rclone/sync_integration_test.go` (TestSyncEngine_RunTask_NoDelete)
- [x] T021b [US2] 实现 Sync 方法中的 NoDelete 逻辑 - 使用 CopyDir 替代 Sync in `internal/rclone/sync.go` (已在 T009b 中一并实现)
- [x] T022 [US2] 更新 task.resolvers.go 处理 TaskSyncOptions 中的 noDelete 字段 in `internal/api/graphql/resolver/task.resolvers.go` (已在 T012 中一并实现)

### Frontend Implementation for User Story 2

- [x] T023 [US2] 在任务设置页面添加 "保留删除文件" Checkbox（仅单向同步模式显示）in `web/src/modules/connections/components/TaskSettingsForm.tsx`

### i18n for User Story 2

- [x] T024 [P] [US2] 添加保留删除文件相关英文翻译 in `web/project.inlang/messages/en.json`
- [x] T025 [P] [US2] 添加保留删除文件相关中文翻译 in `web/project.inlang/messages/zh-CN.json`

**Checkpoint**: 保留删除文件功能完成 ✅ - 用户可以选择在单向同步时不删除目标端文件

---

## Phase 5: User Story 3 - 并行传输数量 (Priority: P2)

**Goal**: 用户可以为每个任务配置并行传输数量，控制同步速度和资源占用

**Independent Test**: 创建一个任务并配置并行传输数量为 8，执行同步时观察是否同时传输多个文件

### Backend Implementation for User Story 3

- [x] T026a [US3] 添加 Transfers 配置的单元测试 in `internal/rclone/sync_test.go`
- [x] T026b [US3] 实现 Sync 方法中的 Transfers 配置注入 - 使用 fs.AddConfig 设置并行数 in `internal/rclone/sync.go`
- [x] T027 [US3] 实现 determineTransfers 函数 - 三层回退逻辑（任务级 → 全局配置 → 默认值 4）in `internal/rclone/sync.go`
- [x] T028 [US3] 更新 task.resolvers.go 处理 TaskSyncOptions 中的 transfers 字段 in `internal/api/graphql/resolver/task.resolvers.go`

### Frontend Implementation for User Story 3

- [x] T029 [US3] 在任务设置页面添加 "并行传输数量" 数字输入框（范围 1-64）in `web/src/modules/connections/components/TaskSettingsForm.tsx`

### i18n for User Story 3

- [x] T030 [P] [US3] 添加并行传输数量相关英文翻译 in `web/project.inlang/messages/en.json`
- [x] T031 [P] [US3] 添加并行传输数量相关中文翻译 in `web/project.inlang/messages/zh-CN.json`

**Checkpoint**: 并行传输数量功能完成 ✅ - 用户可以自定义同步时的并发数

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 跨功能优化和文档更新

- [x] T032 [P] 更新 config.toml.example 添加 sync.transfers 配置示例
- [x] T033 [P] 更新 README.md 文档说明新增的扩展选项功能
- [x] T034 代码清理 - 确保所有新增代码遵循项目规范和格式（golangci-lint 通过，使用 i18n 错误代替 fmt.Errorf）
- [x] T035 运行 quickstart.md 中的验证场景确保功能正常工作（所有相关测试通过，TestIntegrationSuite 中有预先存在的问题与本 feature 无关）
- [x] T036 [P] 添加后端错误消息的英文翻译（过滤器验证、transfers 验证、同步错误） in `internal/i18n/locales/en.toml`
- [x] T037 [P] 添加后端错误消息的中文翻译（过滤器验证、transfers 验证、同步错误） in `internal/i18n/locales/zh-CN.toml`
- [x] T038 运行 scripts/sort-i18n-keys.js 对所有 i18n 文件进行字母排序

**Checkpoint**: Phase 6 完成 ✅ - 所有 Task 扩展选项配置功能已完成

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational phase
- **User Story 2 (Phase 4)**: Depends on Foundational phase (可与 US1 并行)
- **User Story 3 (Phase 5)**: Depends on Foundational phase (可与 US1/US2 并行)
- **Polish (Phase 6)**: Depends on all user stories completion

### User Story Dependencies

- **User Story 1 (P1)**: 独立，无依赖其他 User Story
- **User Story 2 (P2)**: 独立，无依赖其他 User Story
- **User Story 3 (P2)**: 独立，无依赖其他 User Story

### Within Each User Story

- Backend implementation before Frontend implementation
- GraphQL resolvers before frontend API integration
- Core components before integration into views
- i18n can run in parallel with other tasks

### Parallel Opportunities

**Phase 1 (Setup)**:
- T001, T002 串行执行（同一文件 task.graphql）
- T003 可与 T001/T002 并行执行（不同文件 file.graphql）

**Phase 2 (Foundational)**:
- T005, T006, T007 可并行执行（不同文件）

**Phase 3 (User Story 1)**:
- T013, T014 可并行（不同 GraphQL 查询文件）
- T019, T020 可并行（不同 i18n 文件）
- T015, T016, T017 可并行（不同组件文件）

**Phase 4 (User Story 2)**:
- T024, T025 可并行（不同 i18n 文件）

**Phase 5 (User Story 3)**:
- T30, T031 可并行（不同 i18n 文件）

**跨 User Story 并行**:
- 完成 Phase 2 后，US1/US2/US3 可由不同开发者并行推进

---

## Parallel Example: User Story 1

```bash
# 第一批并行任务（Backend 独立文件）:
Task T009: 实现 Sync 方法中的过滤器注入逻辑
Task T010: 扩展 ListRemoteDir 函数支持 filters 参数

# 等待 Backend 完成后，第二批并行任务（Frontend）:
Task T013: 更新 GraphQL 查询类型定义 - tasks.ts
Task T014: 更新 GraphQL 文件查询类型定义 - files.ts
Task T015: 创建 FilterRulesEditor 组件
Task T016: 创建 FilterPreviewPanel 组件
Task T017: 扩展 FileBrowser 组件
Task T019: 添加英文翻译
Task T020: 添加中文翻译
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T004)
2. Complete Phase 2: Foundational (T005-T008) - CRITICAL
3. Complete Phase 3: User Story 1 (T009-T020)
4. **STOP and VALIDATE**: 使用 quickstart.md 验证过滤器功能
5. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add User Story 1 → Test → Deploy (MVP - 过滤器配置)
3. Add User Story 2 → Test → Deploy (保留删除文件)
4. Add User Story 3 → Test → Deploy (并行传输数量)
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (过滤器配置)
   - Developer B: User Story 2 (保留删除文件)
   - Developer C: User Story 3 (并行传输数量)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- 所有 User Story 均可独立完成和测试
- 建议先完成 MVP（User Story 1）再推进其他功能
- 测试时可参考 quickstart.md 中的验证场景
- 修改完成后及时提交，每个逻辑单元一次提交

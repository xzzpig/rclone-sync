# Tasks: Rclone Fs Cache Optimization

**Input**: Design documents from `/specs/010-rclone-fs-cache/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅ (N/A), quickstart.md ✅

**Tests**: 根据 Constitution III (Test-Driven Development)，包含测试任务以验证缓存行为。

**Organization**: 任务按 User Story 组织，每个 Story 可独立实现和测试。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 所属 User Story（US1, US2, US3）
- 描述中包含确切的文件路径

## Path Conventions

- **Backend**: `internal/` 目录结构
- **Tests**: 与源文件同目录的 `*_test.go` 文件

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 创建共享的 Fs 缓存辅助函数

- [x] T001 创建 `GetFs(ctx, remote, path)` 和 `ClearFsCache(remoteName)` 辅助函数 in `internal/rclone/cache_helper.go`
- [x] T002 [P] 创建辅助函数的单元测试 in `internal/rclone/cache_helper_test.go`

**说明**: 
- `GetFs` 函数：当 `remote` 为空时使用 `fs.NewFs`（本地路径不缓存），否则使用 `cache.Get`
- `ClearFsCache` 函数：封装 `cache.ClearConfig(remoteName)` 调用
- **FR-003 (新 Fs 加入缓存)**: 由 `cache.Get` 内部自动处理，无需显式 `cache.Put` 调用

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 验证现有缓存失效逻辑，确保基础设施就绪

**⚠️ CRITICAL**: 确认 storage.go 已正确使用 cache.ClearConfig

- [x] T003 验证 `SetValue`、`DeleteSection`、`DeleteKey` 方法中 `cache.ClearConfig` 调用正确 in `internal/rclone/storage.go`

**Checkpoint**: 辅助函数和缓存失效基础设施就绪 - 可以开始实现 User Story

---

## Phase 3: User Story 1 - 重复浏览远程目录时响应更快 (Priority: P1) 🎯 MVP

**Goal**: 用户在文件浏览器中浏览远程目录时，系统复用已缓存的 Fs 实例，减少等待时间

**Independent Test**: 多次快速连续请求同一远程的不同目录，观察第一次请求与后续请求的响应时间差异

### Implementation for User Story 1

- [x] T004 [US1] 修改 `ListRemoteDir` 函数使用 `GetFs` 并实现 `BasePath` 缓存策略 in `internal/rclone/remote.go`
  - 当 `opts.BasePath` 设置时，使用 `remote:BasePath` 作为 Fs 缓存键
  - 这样浏览同一任务下的不同子目录时可以复用同一个 Fs 实例
  - 通过 `f.List(ctx, relativePath)` 访问子目录内容
  - `entry.Remote()` 返回相对于 Fs 根（BasePath）的完整路径，可直接用于过滤匹配
  - 需提取最后路径段作为文件名用于显示
- [x] T005 [US1] 更新 `ListRemoteDir` 相关测试确保缓存行为正确 in `internal/rclone/remote_test.go`
  - 添加 `basePath enables Fs reuse across subdirectories` 测试用例

**Checkpoint**: User Story 1 功能完成，目录浏览可复用缓存 Fs ✅

---

## Phase 4: User Story 2 - 获取存储空间信息时复用连接 (Priority: P2)

**Goal**: 用户查看连接详情页面时，存储空间查询复用已有的 Fs 缓存实例

**Independent Test**: 在已浏览过的远程上请求存储空间信息，验证是否复用 Fs 实例

### Implementation for User Story 2

- [x] T006 [US2] 修改 `GetRemoteQuota` 函数使用 `GetFs(ctx, remoteName, "")` 替换 `fs.NewFs` in `internal/rclone/about.go`
- [x] T007 [US2] 更新 `GetRemoteQuota` 相关测试确保缓存行为正确 in `internal/rclone/about_test.go`

**Checkpoint**: User Stories 1 和 2 都独立可用 ✅

---

## Phase 5: User Story 3 - 同步任务中的 Fs 复用 (Priority: P3)

**Goal**: 同步任务运行时，对于远程端使用缓存策略来优化性能

**Independent Test**: 运行多个指向同一远程的同步任务，观察 Fs 实例创建行为

### Implementation for User Story 3

- [x] T008 [US3] 修改 `RunTask` 函数中源路径使用 `GetFs(ctx, "", task.SourcePath)` in `internal/rclone/sync.go`
- [x] T009 [US3] 修改 `RunTask` 函数中目标路径使用 `GetFs(ctx, connectionName, task.RemotePath)` in `internal/rclone/sync.go`
- [x] T010 [US3] 更新同步相关测试确保缓存行为正确 in `internal/rclone/sync_test.go`
- [x] T011 [US3] 在 `Update` mutation 中添加 `rclone.ClearFsCache(oldName)` 调用 in `internal/api/graphql/resolver/connection.resolvers.go`
- [x] T012 [US3] 在 `Delete` mutation 中添加 `rclone.ClearFsCache(connName)` 调用 in `internal/api/graphql/resolver/connection.resolvers.go`
- [x] T013 [P] [US3] 更新 resolver 测试验证缓存失效逻辑 in `internal/api/graphql/resolver/connection_test.go`

**⚠️ 说明**: T011/T012 是必要的，因为 `ConnectionService.DeleteConnectionByID` 直接使用 Ent 客户端删除，不经过 `storage.go` 的 `DeleteSection`。虽然 `storage.go` 中的 `SetValue`/`DeleteSection`/`DeleteKey` 已调用 `cache.ClearConfig`，但 resolver 层的删除操作走的是不同路径。

**Checkpoint**: 所有 User Stories 功能完成

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 全面测试和验证

- [x] T014 [P] 运行所有 rclone 包测试 `go test ./internal/rclone/...`
- [x] T015 [P] 运行所有 resolver 测试 `go test ./internal/api/graphql/resolver/...`
- [ ] T016 运行 quickstart.md 验证清单，确认所有检查项通过

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖 - 立即开始
- **Foundational (Phase 2)**: 依赖 Setup 完成
- **User Stories (Phase 3-5)**: 依赖 Foundational 完成
  - US1、US2、US3 可并行执行（如果有多人）
  - 或按优先级顺序执行 (P1 → P2 → P3)
- **Polish (Phase 6)**: 依赖所有 User Stories 完成

### User Story Dependencies

- **User Story 1 (P1)**: 仅依赖 Phase 2 完成 - 无其他 Story 依赖
- **User Story 2 (P2)**: 仅依赖 Phase 2 完成 - 可独立测试
- **User Story 3 (P3)**: 仅依赖 Phase 2 完成 - 包含缓存失效逻辑，但可独立测试

### Within Each User Story

- 核心实现优先
- 测试更新随后
- Story 完成后再进入下一优先级

### Parallel Opportunities

- T001 和 T002 可同时进行（不同文件）
- US1、US2、US3 可由不同开发者并行实现
- T008 和 T011/T012 可并行（不同文件）
- T014 和 T015 可并行运行

---

## Parallel Example: User Story 3

```bash
# 以下任务可并行执行（不同文件）：
Task T008: "修改 RunTask 源路径使用 GetFs in internal/rclone/sync.go"
Task T011: "在 Update mutation 中添加 ClearFsCache 调用 in internal/api/graphql/resolver/connection.resolvers.go"

# T009 依赖 T008（同文件，需顺序执行）
# T010 依赖 T008、T009（测试需要实现完成后）
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. 完成 Phase 1: Setup（创建辅助函数）
2. 完成 Phase 2: Foundational（验证现有失效逻辑）
3. 完成 Phase 3: User Story 1（目录浏览缓存）
4. **STOP and VALIDATE**: 独立测试 User Story 1
5. 可部署/演示 MVP

### Incremental Delivery

1. Setup + Foundational → 基础就绪
2. User Story 1 → 目录浏览缓存（MVP!）
3. User Story 2 → 存储空间查询缓存
4. User Story 3 → 同步任务缓存 + 缓存失效
5. 每个 Story 增量交付价值

### Single Developer Strategy

推荐顺序执行：
1. T001-T003（Setup + Foundational）
2. T004-T005（US1 - 最高优先级）
3. T006-T007（US2）
4. T008-T013（US3）
5. T014-T016（验证）

---

## Notes

- 本功能不涉及数据库 schema 变更
- 所有修改集中在 `internal/rclone` 和 `internal/api/graphql/resolver` 目录
- 保持现有错误处理逻辑：`cache.Get` 失败时直接返回错误，不回退
- 对于直接本地路径（`remote` 参数为空），始终使用 `fs.NewFs` 不缓存
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently

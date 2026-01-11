# Tasks: ChangeNotify 缓存加速同步

**Input**: Design documents from `/specs/012-rclone-change-notify-cache/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅

**Tests**: 遵循 Constitution III (TDD)，后端核心组件必须包含单元测试和集成测试。

**Organization**: 任务按用户故事分组，每个故事可独立实现和测试。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 任务所属用户故事（US1, US2, US3, US4, US5）
- 描述中包含精确的文件路径

## Path Conventions

- **Backend**: `internal/` 目录
- **GraphQL**: `internal/api/graphql/`
- **Ent Schema**: `internal/core/db/schema/`
- **Cache Data**: `app_data/cache/`

---

## Phase 1: Setup (项目基础设施)

**Purpose**: 创建新模块目录结构和基础配置

- [X] T001 创建 `internal/rclone/backend/metacache/` 目录结构
- [X] T002 在 `internal/api/graphql/schema/connection.graphql` 中添加 ConnectionOptions 和 ConnectionCacheOptions 类型定义（参考 contracts/schema.graphql）
- [X] T002a 运行 `go generate ./internal/api/graphql` 生成 `model.ConnectionOptions` 等 Go 类型（T004 依赖此类型）
- [X] T003 [P] 确保 `app_data/cache/` 目录在应用启动时自动创建

---

## Phase 2: Foundational (基础组件 - 阻塞所有用户故事)

**Purpose**: 所有用户故事依赖的核心基础设施

**⚠️ CRITICAL**: 此阶段完成前不能开始任何用户故事

### 2.1 数据库 Schema 变更

- [X] T004 修改 `internal/core/ent/schema/connection.go` 添加 `options` JSON 字段（类型 `*model.ConnectionOptions`）
- [X] T005 运行 `go generate ./internal/core/ent` 重新生成 Ent 代码
- [X] T006 运行 `./scripts/gen-migration.sh "add connection options"` 生成数据库迁移

### 2.2 CacheStore 核心实现

- [X] T007 实现 `internal/rclone/backend/metacache/cache_entry.go` 定义 CacheEntry 结构体
- [X] T008 实现 `internal/rclone/backend/metacache/cache_store.go` 核心缓存存储：
  - NewCacheStore() 创建/打开缓存数据库
  - Schema 版本检查和条件重建
  - WAL 模式配置
- [X] T009 实现 `internal/rclone/backend/metacache/cache_store.go` CRUD 方法：
  - Get(path) 获取缓存条目
  - Set(path, entry) 设置缓存条目
  - ListChildren(parent) 列出目录子项
  - MarkStale(path) 标记条目过期
  - Clear() 清空所有缓存
  - Close() 关闭数据库连接
- [X] T010 实现 `internal/rclone/backend/metacache/cache_store.go` 全局缓存存储管理：
  - GetCacheStore(connectionID, dbPath) 单例获取
  - ReleaseCacheStore(connectionID) 释放引用

### 2.3 MetaCache Backend 实现

- [X] T011 实现 `internal/rclone/backend/metacache/metacache.go` 后端注册：
  - fs.Register() 注册 "metacache" 后端
  - Options 结构体定义（remote, info_age, change_notify_poll, db_path, connection_id）
- [X] T012 实现 `internal/rclone/backend/metacache/metacache.go` Fs 结构体和 NewFs：
  - 获取共享 CacheStore
  - 包装远程 Fs
  - 设置 features
- [X] T013 实现 `internal/rclone/backend/metacache/metacache.go` 核心接口：
  - Name(), Root(), Features(), UnWrap()
  - List() 带缓存的目录列举
  - NewObject() 带缓存的对象获取
- [X] T014 实现 `internal/rclone/backend/metacache/metacache.go` ChangeNotify 集成：
  - receiveChangeNotify() 回调处理
  - 订阅底层 Fs 的 ChangeNotify
  - **注意**: 由于 ChangeNotify 仅提供 path 和 EntryType（无完整元数据），采用"标记过期 + 按需获取"策略：
    - 通过 MarkStale() 将路径及其后代标记为过期
    - 清除父目录 dir_loaded 确保下次 List() 时从远程获取最新数据
    - 实际元数据在下次访问时惰性获取（符合 FR-002 和 spec clarifications）
- [X] T015 实现 `internal/rclone/backend/metacache/metacache.go` Shutdown 接口：
  - 关闭 pollIntervalChan
  - 释放 CacheStore 引用

### 2.4 DBStorage 透明注入

- [X] T016 修改 `internal/rclone/storage.go` 添加 CacheSuffix 常量和 dataDir 字段
- [X] T017 修改 `internal/rclone/storage.go` HasSection() 支持 "-cache" 后缀检测
- [X] T018 修改 `internal/rclone/storage.go` GetValue() 支持 "-cache" 后缀配置注入
- [X] T019 实现 `internal/rclone/storage.go` getCacheValueLocked() 方法返回虚拟 metacache 配置
- [X] T020 修改 `internal/rclone/storage.go` GetKeyList() 支持 "-cache" 后缀

### 2.5 Pin Manager 实现

- [X] T021 实现 `internal/rclone/pin_manager.go` 全局 Pin 管理：
  - pinnedFsMap 存储 Pin 住的 Fs
  - PinConnection(conn) 为连接创建并 Pin Fs
  - UnpinConnection(connID) 取消 Pin 并释放资源
- [X] T022 实现 `internal/rclone/pin_manager.go` InitPinnedConnections()：
  - 应用启动时为所有启用缓存的连接初始化 Pin

### 2.6 核心组件单元测试

- [X] T022a [P] 实现 `internal/rclone/backend/metacache/cache_store_test.go` CacheStore 单元测试：
  - 测试 NewCacheStore() 创建/打开/schema 版本检查
  - 测试 DB 版本不匹配时自动删除重建数据库
  - 测试 WAL 模式正确开启（验证 journal_mode=wal）
  - 测试 Get/Set/ListChildren/MarkStale/Clear CRUD 操作
  - 测试 TTL 过期逻辑
- [X] T022b [P] 实现 `internal/rclone/backend/metacache/metacache_test.go` MetaCache 后端单元测试：
  - 测试 NewFs() 初始化和配置解析
  - 测试 List() 缓存命中/未命中逻辑
  - 测试 ChangeNotify 回调处理（验证 MarkStale + 清除父目录 dir_loaded 确保下次访问时刷新）
  - 测试过期目录全量刷新：目录过期后重新列举远程并重置 dir_loaded=true

- [X] T022c [P] 实现 `internal/rclone/pin_manager_test.go` Pin Manager 单元测试：
  - 测试 PinConnection/UnpinConnection 生命周期
  - 测试 InitPinnedConnections 启动逻辑

- [X] T022d [P] 实现 `internal/rclone/storage_test.go` DBStorage 缓存后缀透明注入单元测试：
  - 测试 HasSection() 对 "-cache" 后缀的处理（缓存启用/未启用）
  - 测试 GetValue() 对 "-cache" 后缀返回虚拟 metacache 配置
  - 测试 GetKeyList() 对 "-cache" 后缀返回 metacache 配置键列表
  - 测试连接名本身以 "-cache" 结尾时的正确处理（边界情况）

**Checkpoint**: 基础设施就绪 - 用户故事实现可以开始

---

## Phase 3: User Story 1 - 为连接启用缓存加速大目录同步 (Priority: P1) 🎯 MVP

**Goal**: 对支持 ChangeNotify 的连接启用缓存，使无变更时重复同步耗时显著减少

**Independent Test**: 对 OneDrive 连接启用缓存，执行两次同步，验证第二次无需全量列举

### Implementation for User Story 1

- [X] T023 [US1] 修改 `internal/rclone/sync.go` 添加 GetRemoteName() 辅助函数判断是否使用缓存
- [X] T024 [US1] 修改 `internal/rclone/sync.go` 添加 GetCachedFs() 获取可能带缓存的 Fs
- [X] T025 [US1] 修改同步执行逻辑使用 GetCachedFs() 替代直接获取远程 Fs
- [X] T026 [US1] 修改应用启动逻辑调用 `metacache.InitPinnedConnections()` 初始化缓存服务
- [X] T027 [US1] 实现 TTL 过期检查和自动刷新逻辑：
  - 在 CacheStore.Get() 中检查 cached_at + InfoAge 是否过期
  - 在 Fs.List() 中检测目录过期后自动从远程重新获取并更新缓存
  - **过期目录全量刷新**: 目录过期时删除该目录下所有缓存子项，重新列举远程，重置 dir_loaded=true 和 cached_at
  - 确保符合 FR-009：过期时自动触发完整目录结构重新获取

**Checkpoint**: 用户故事 1 完成 - 可独立测试缓存加速功能

---

## Phase 4: User Story 2 - 不支持变更通知时优雅降级 (Priority: P2)

**Goal**: 当远程不支持 ChangeNotify 或未启用缓存时，同步正常工作

**Independent Test**: 对 S3 连接进行同步，验证使用常规列举流程且结果正确

### Implementation for User Story 2

- [X] T028 [US2] 确保 `internal/rclone/backend/metacache/metacache.go` 在后端不支持 ChangeNotify 时仍正常工作（仅依赖 TTL）
  - 已实现于 metacache.go:198-206: 检测 doChangeNotify == nil 时记录日志并继续使用 TTL
  - 测试覆盖: TestList_ExpiredDirectoryRefresh, TestList_CacheMiss 验证 TTL 刷新逻辑
- [X] T029 [US2] 确保 `internal/rclone/cache_helper.go` GetCachedFs() 在缓存未启用时返回原始 Fs
  - 已实现于 cache_helper.go:97-100: GetRemoteName(name, false) 返回原始名称
  - 测试覆盖: cache_helper_test.go TestGetCachedFs "uses original remote when cache disabled"
- [X] T030 [US2] 在 `internal/rclone/backend/metacache/metacache.go` List() 中实现缓存未命中时回退到远程获取
  - 已实现于 metacache.go:278-311: IsDirLoaded 检查后回退到 wrapped.List()
  - 测试覆盖: metacache_test.go TestList_CacheMiss

**Checkpoint**: 用户故事 2 完成 - 降级逻辑可独立验证 ✅

---

## Phase 5: User Story 3 - 管理连接缓存设置 (Priority: P2)

**Goal**: 用户可为每个连接独立配置缓存参数

**Independent Test**: 在连接设置中启用缓存、修改过期时间，验证配置生效

### 5.1 GraphQL Schema 扩展

> **注意**: ConnectionOptions 和 ConnectionCacheOptions 基础类型已在 Phase 1 T002 中添加，此处仅添加缺失的类型和扩展

- [X] T031 [P] [US3] 修改 `internal/api/graphql/schema/connection.graphql` 添加 ConnectionCacheStatus 类型（运行时状态）
- [X] T032 [P] [US3] 修改 `internal/api/graphql/schema/connection.graphql` 添加 ConnectionOptionsInput 和 ConnectionCacheOptionsInput 输入类型
- [X] T033 [US3] 修改 `internal/api/graphql/schema/connection.graphql` 扩展 Connection 类型添加 options 和 cacheStatus 字段
- [X] T034 [US3] 修改 `internal/api/graphql/schema/connection.graphql` 扩展 CreateConnectionInput 和 UpdateConnectionInput 添加 options 字段

### 5.2 GraphQL Resolver 实现

- [X] T037 [US3] 运行 `go generate ./internal/api/graphql` 重新生成 gqlgen 代码
- [X] T038 [US3] 实现 `internal/api/graphql/resolver/connection.resolvers.go` Options 字段 resolver
- [X] T039 [US3] 实现 `internal/api/graphql/resolver/connection.resolvers.go` CacheStatus 字段 resolver：
  - running: 检查 Fs 是否已 Pin
  - changeNotifySupported: 检查后端是否支持
  - entriesCount: 查询缓存条目数量
  - dbSizeBytes: 获取缓存数据库文件大小
  - lastNotifyTime: 获取最后通知时间
- [X] T040 [US3] 修改 `internal/api/graphql/resolver/connection_mutation.resolvers.go` 处理 Create/Update 时的 options 字段

### 5.3 配置验证

- [X] T041 [US3] 实现 `internal/api/graphql/model/connection_options.go` validateCacheOptions() 验证函数：
  - InfoAge 必须是有效的 Go duration 格式
  - ChangeNotifyPoll 必须 >= 10s

### 5.4 Pin 状态管理

- [X] T042 [US3] 修改连接 Update 逻辑：当 cache.enabled 变更时调用 PinConnection/UnpinConnection
- [X] T043 [US3] 修改连接 Create 逻辑：如果启用缓存则调用 PinConnection

### 5.5 GraphQL 集成测试

- [X] T044 [US3] 实现 CacheStatus resolver 集成测试：
  - 测试 running/changeNotifySupported/entriesCount/dbSizeBytes 字段
  - 测试缓存未启用时返回 null

### 5.6 Frontend 配置 UI

- [X] T045 [US3] 在 `web/src/modules/connections/` 添加连接缓存设置 UI（启用开关、InfoAge、ChangeNotifyPoll）
- [X] T046 [US3] 在 `web/src/modules/connections/` 展示 CacheStatus（running/changeNotifySupported/entriesCount/dbSizeBytes/lastNotifyTime）
- [X] T047 [US3] 在 `web/src/modules/connections/` 集成保存/更新 options 逻辑并处理乐观更新

**Checkpoint**: 用户故事 3 完成 - 缓存配置 UI 可独立验证

---

## Phase 6: User Story 4 - 关闭缓存释放资源 (Priority: P3)

**Goal**: 用户关闭缓存后，后台服务停止、资源释放

**Independent Test**: 关闭已启用缓存的连接的缓存功能，验证后台服务停止

### Implementation for User Story 4

- [X] T048 [US4] 确保 `internal/rclone/pin_manager.go` UnpinConnection() 正确调用 Fs.Shutdown()
- [X] T049 [US4] 确保 `internal/rclone/backend/metacache/metacache.go` Shutdown() 正确关闭 pollIntervalChan 停止 ChangeNotify
- [X] T050 [US4] 实现 Shutdown 单元测试，验证服务在 1 分钟内（SC-004）完全停止并释放资源
- [X] T051 [US4] 在 UnpinConnection() 中添加日志记录服务停止状态

**Checkpoint**: 用户故事 4 完成 - 资源释放可独立验证

---

## Phase 7: User Story 5 - 手动清理连接缓存 (Priority: P3)

**Goal**: 用户可手动清理某个连接的缓存数据

**Independent Test**: 执行清理缓存操作，验证缓存记录被清空

### 7.1 GraphQL Mutation 实现

- [X] T052 [P] [US5] 修改 `internal/api/graphql/schema/connection.graphql` 添加 ClearCacheResult 类型
- [X] T053 [US5] 修改 `internal/api/graphql/schema/connection.graphql` 扩展 ConnectionMutation 添加 clearCache mutation
- [X] T054 [US5] 运行 `go generate ./internal/api/graphql` 重新生成 gqlgen 代码

### 7.2 Resolver 实现

- [X] T055 [US5] 实现 `internal/api/graphql/resolver/connection_mutation.resolvers.go` ClearCache resolver：
  - 获取连接的 CacheStore
  - 调用 Clear() 清空缓存
  - 返回清理结果（成功/条目数/消息）

### 7.3 Service 层支持

- [X] T056 [US5] 在 `internal/rclone/backend/metacache/cache_store.go` Clear() 方法中返回清理的条目数量
- [X] T057 [US5] 添加 GetEntriesCount() 方法用于 CacheStatus 查询

### 7.4 ClearCache 集成测试

- [X] T058 [US5] 实现 ClearCache mutation 集成测试：
  - 测试清理成功返回条目数
  - 测试缓存不存在时的错误处理

### 7.5 Frontend 清理缓存

- [X] T059 [US5] 在 `web/src/modules/connections/` 添加"清理缓存"按钮与确认流程，调用 clearCache mutation
- [X] T060 [US5] 在 `web/src/modules/connections/` 展示清理结果与错误提示（i18n 文案）

**Checkpoint**: 用户故事 5 完成 - 手动清理功能可独立验证

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: 影响多个用户故事的优化和完善

- [X] T061 [P] 添加 i18n 支持：ClearCacheResult.message 等用户可见消息的本地化
- [X] T062 代码审查和重构：确保错误处理一致性
- [x] T063 [P] 更新 README.md 添加缓存功能说明
- [ ] T064 运行 quickstart.md 中的手动测试步骤验证功能完整性
- [ ] T065 [P] 手动验证 添加性能基准测试脚本验证 SC-001（10万+文件目录无变更时 < 2分钟）
- [ ] T066 [P] 手动验证 添加性能回归测试验证 SC-002（未支持 ChangeNotify 时性能无明显退化 < 110%）
- [ ] T067 手动验证 SC-003：制造远程变更并在下一次同步中统计变更反映率（>=95%）
- [ ] T068 手动验证 SC-005：首次启用缓存后同步的额外延迟不超过 10%

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖 - 可立即开始
- **Foundational (Phase 2)**: 依赖 Setup 完成 - **阻塞所有用户故事**
- **User Stories (Phase 3-7)**: 全部依赖 Foundational 完成
  - 用户故事可按优先级顺序（P1 → P2 → P3）执行
  - 或多人并行开发不同用户故事
- **Polish (Phase 8)**: 依赖所有用户故事完成

### User Story Dependencies

- **User Story 1 (P1)**: Foundational 完成后可开始 - 无其他故事依赖
- **User Story 2 (P2)**: Foundational 完成后可开始 - 与 US1 可并行
- **User Story 3 (P2)**: Foundational 完成后可开始 - 与 US1/US2 可并行
- **User Story 4 (P3)**: 依赖 US3（Pin 管理）完成
- **User Story 5 (P3)**: 依赖 US3（GraphQL 基础）完成

### Within Each User Story

- 模型先于服务
- 服务先于 API
- 核心实现先于集成
- 故事完成后再进入下一优先级

### Parallel Opportunities

**Phase 2 内可并行**:
- T007, T011 可同时开始（不同文件）
- T016-T020（DBStorage 修改）需按顺序

**Phase 5 内可并行**:
- T031-T034（GraphQL 类型定义）可同时进行

**跨用户故事可并行**:
- US1, US2, US3 可由不同开发者同时进行
- US4, US5 依赖 US3 的 GraphQL 基础

---

## Parallel Example: Foundational Phase

```bash
# 可并行的任务组 1（数据模型）:
Task T004: 修改 Connection schema 添加 options 字段
Task T007: 实现 CacheEntry 结构体

# 可并行的任务组 2（核心存储，依赖 T007）:
Task T008: 实现 CacheStore 创建/打开
Task T011: 实现 MetaCache 后端注册

# 可并行的任务组 3（依赖 T008, T012）:
Task T021: 实现 Pin Manager
Task T016: 修改 DBStorage 添加 CacheSuffix
```

---

## Implementation Strategy

### MVP First (仅 User Story 1)

1. 完成 Phase 1: Setup
2. 完成 Phase 2: Foundational（关键 - 阻塞所有故事）
3. 完成 Phase 3: User Story 1
4. **STOP and VALIDATE**: 独立测试缓存加速功能
5. 可部署/演示 MVP

### Incremental Delivery

1. Setup + Foundational → 基础就绪
2. 添加 User Story 1 → 独立测试 → 部署（MVP!）
3. 添加 User Story 2 + 3 → 独立测试 → 部署
4. 添加 User Story 4 + 5 → 独立测试 → 部署
5. 每个故事增加价值而不破坏之前的功能

### Single Developer Strategy

按优先级顺序执行：
1. Phase 1 → Phase 2（基础）
2. Phase 3（US1 - MVP）→ 验证
3. Phase 4 + Phase 5（US2 + US3）→ 验证
4. Phase 6 + Phase 7（US4 + US5）→ 验证
5. Phase 8（Polish）

---

## Notes

- [P] 任务 = 不同文件，无依赖关系
- [Story] 标签用于追溯任务归属
- 每个用户故事应可独立完成和测试
- 每个任务或逻辑组完成后提交代码
- 在任何 Checkpoint 处停下来独立验证故事
- 避免：模糊任务、同文件冲突、破坏独立性的跨故事依赖

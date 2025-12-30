# Feature Specification: Rclone Fs Cache Optimization

**Feature Branch**: `010-rclone-fs-cache`  
**Created**: 2025-12-30  
**Status**: Draft  
**Input**: User description: "现在项目中都使用 fs.NewFs 来创建 rclone fs，我希望能在合适的情况下改为 cache.Get 来使用缓存的 rclone fs"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 重复浏览远程目录时响应更快 (Priority: P1)

用户在文件浏览器中浏览远程存储的目录结构时，系统应该复用已缓存的 rclone Fs 实例，而不是每次请求都重新创建新的 Fs 连接，从而减少等待时间。

**Why this priority**: 这是用户最常见的交互场景。用户在创建任务或预览过滤规则时，会频繁切换目录查看文件结构。每次目录切换如果都需要重新初始化 Fs 连接，会导致明显的延迟，影响用户体验。

**Independent Test**: 可以通过多次快速连续请求同一远程的不同目录来验证，观察第一次请求与后续请求的响应时间差异。

**Acceptance Scenarios**:

1. **Given** 用户已连接到远程存储且首次浏览了根目录, **When** 用户继续浏览该远程的子目录, **Then** 系统复用缓存的 Fs 实例，响应时间明显减少
2. **Given** 用户正在浏览远程目录, **When** 缓存中已存在该远程的 Fs 实例, **Then** 系统直接使用缓存实例而不创建新连接
3. **Given** 用户浏览远程目录, **When** 远程连接配置已更新, **Then** 系统创建新的 Fs 实例而不使用旧缓存

---

### User Story 2 - 获取存储空间信息时复用连接 (Priority: P2)

用户查看连接详情页面时，系统获取存储空间使用情况（used/total）应该复用已有的 Fs 缓存实例。

**Why this priority**: 存储空间信息查询是相对轻量的只读操作，适合使用缓存。虽然不如目录浏览频繁，但在连接详情页面是常用功能。

**Independent Test**: 可以在已浏览过的远程上请求存储空间信息，验证是否复用 Fs 实例。

**Acceptance Scenarios**:

1. **Given** 用户已浏览过某远程存储, **When** 用户查看该连接的存储空间信息, **Then** 系统复用缓存的 Fs 实例获取信息

---

### User Story 3 - 同步任务中的 Fs 复用 (Priority: P3)

当同步任务运行时，对于多个使用相同远程的任务，应该考虑缓存策略来平衡性能和数据一致性。

**Why this priority**: 同步操作涉及写入，需要谨慎处理缓存策略以确保数据一致性。这个场景需要更深入的技术调研来确定最佳实践。

**Independent Test**: 运行多个指向同一远程的同步任务，观察 Fs 实例创建行为和数据一致性。

**Acceptance Scenarios**:

1. **Given** 有多个任务指向同一远程存储, **When** 这些任务顺序执行, **Then** 系统根据配置策略决定是否复用 Fs 实例
2. **Given** 同步任务正在运行, **When** 使用缓存的 Fs, **Then** 不会因为缓存而导致数据不一致问题

---

### Edge Cases

- 缓存的 Fs 实例在长时间不活动后连接是否仍然有效？→ 已澄清：直接返回错误，由用户重试
- 用户修改远程配置后，如何确保缓存被正确刷新？→ 已澄清：在 Update 时立即失效
- 并发访问同一缓存 Fs 实例时的线程安全性如何保证？→ 已澄清：依赖 rclone 内置线程安全保证
- 内存使用：大量不同远程的 Fs 实例缓存是否会导致内存问题？→ 已澄清：依赖 rclone 内置缓存管理

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 在适当的只读操作中优先使用 `cache.Get` 获取缓存的 Fs 实例
- **FR-002**: 系统 MUST 在 `cache.Get` 失败时直接返回错误，不进行回退尝试
- **FR-003**: 系统 MUST 将新创建的 Fs 实例加入缓存供后续使用
- **FR-004**: 系统 MUST 确保使用缓存 Fs 的操作在并发场景下是线程安全的
- **FR-005**: 系统 MUST 在远程连接配置更新（Update）或删除（Delete）后，立即通过 `cache.Clear(remoteName)` 使该特定 remote 的旧缓存失效
- **FR-006**: 系统 MUST 对目录列表（ListRemoteDir）操作使用缓存优先策略
- **FR-007**: 系统 MUST 对存储空间查询（GetStorageInfo/About）操作使用缓存优先策略
- **FR-008**: 系统 MUST 对同步操作（Sync/Bisync）的远程端使用缓存策略，与只读操作一致
- **FR-009**: 系统 MUST 对通过直接路径创建的本地 Fs（如 `/aaa/bbb` 形式，非 `remote:path` 格式）始终使用 `fs.NewFs`，不进行缓存（本地文件系统访问无需缓存优化）

### Key Entities

- **Fs Cache**: rclone 内置的 Fs 实例缓存，通过 `cache.Get` 和 `cache.Put` 操作
- **Fs Instance**: rclone 文件系统抽象，代表一个远程或本地存储的连接
- **Remote Configuration**: 用户配置的远程存储连接参数

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 重复浏览同一远程目录时，第二次及后续请求的响应时间比首次请求减少 50% 以上
- **SC-002**: 获取已浏览过的远程存储空间信息时，响应时间比首次访问减少 30% 以上
- **SC-003**: 系统在启用缓存后不产生新的数据一致性问题（通过现有测试套件验证）
- **SC-004**: 所有现有功能测试在启用缓存后仍然通过
- **SC-005**: 用户操作行为与缓存启用前保持一致（无功能性变化）

## Clarifications

### Session 2025-12-30

- Q: 可观测性：是否需要记录缓存命中/未命中？ → A: 不增加额外的可观测性支持 (D)
- Q: 缓存失效范围：修改配置时失效哪些缓存？ → A: 仅失效被修改的特定 remote (A)
- Q: 本地路径缓存策略：如何处理 local 类型？ → A: 仅不缓存直接路径（如 /path），缓存配置中的 local remote (B)
- Q: 缓存回退逻辑：cache.Get 失败时是否回退？ → A: 不回退，直接报错 (C)
- Q: 删除连接时的缓存处理：删除时是否失效缓存？ → A: 是，删除连接时也失效缓存 (A)
- Q: 用户修改远程连接配置后，系统如何确定旧缓存需要失效？ → A: 在 connectionMutationResolver.Update 时立即失效缓存
- Q: 当有大量不同远程存储的 Fs 实例被缓存时，如何控制内存使用？ → A: 依赖 rclone 内置的缓存管理机制，不设置额外限制
- Q: 同步操作的缓存策略？ → A: 同步操作也使用缓存；但通过直接路径（如 `/aaa/bbb`）创建的本地 Fs 保持使用 fs.NewFs，不缓存（注：这里指非 `remote:path` 格式的直接文件系统路径，而非配置中 local 类型的存储）
- Q: 缓存 Fs 实例连接错误时的处理策略？ → A: 直接返回错误给用户，由用户决定是否重试
- Q: 并发访问缓存 Fs 实例时如何保证线程安全？ → A: 依赖 rclone 内置的线程安全保证，不做额外处理

## Assumptions

- rclone 的 `cache.Get` 返回的 Fs 实例支持并发只读操作
- rclone 的缓存机制自动处理连接保活和超时重连
- 同一个 Fs 实例可以在不同的 context 中安全使用（只要操作是只读的）
- 修改远程配置后需要应用层主动刷新缓存（rclone 不会自动感知配置变更）

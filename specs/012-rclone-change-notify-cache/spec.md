 # Feature Specification: ChangeNotify 缓存加速同步

**Feature Branch**: `012-rclone-change-notify-cache`  
**Created**: 2026-01-08  
**Status**: Draft  
**Input**: User description: "按现在逻辑同步目录时，会去递归列出目录下所有目录和文件，这对于文件特别多的目录会特别慢。对于onedrive等部分存储，支持 ChangeNotify 这feature来实时/定时获取远程更新。我们可以使用这一机制对其进行优化。另外，虽然rclone中自带了cache这backend可以对某个backend进行缓存，但1. 它已被标记为废弃 2. cache backend中的doChangeNotify也存在问题，启动后无法关闭。3. 会额外引入bbolt作为依赖。但我们可以参考 cache backend，创建一个项目中自己的backend同时为connection添加启用缓存等相关参数，以允许用户为某个连接启用缓存"

## User Scenarios & Testing *(mandatory)*

<!--
  IMPORTANT: User stories should be PRIORITIZED as user journeys ordered by importance.
  Each user story/journey must be INDEPENDENTLY TESTABLE - meaning if you implement just ONE of them,
  you should still have a viable MVP (Minimum Viable Product) that delivers value.
  
  Assign priorities (P1, P2, P3, etc.) to each story, where P1 is the most critical.
  Think of each story as a standalone slice of functionality that can be:
  - Developed independently
  - Tested independently
  - Deployed independently
  - Demonstrated to users independently
-->

### User Story 1 - 为连接启用缓存加速大目录同步 (Priority: P1)

作为一个拥有大量文件的云存储用户，我希望对特定连接启用缓存功能，利用远程变更通知机制减少全量递归列举，从而让包含数万甚至数十万文件的目录同步更快。

**Why this priority**: 这是用户的核心痛点——大目录同步太慢。直接解决此问题能带来最大价值。

**Independent Test**: 对支持变更通知的 OneDrive 连接启用缓存，在目录无变更时进行两次同步，验证第二次同步无需全量列举且耗时符合 SC-001（10万+文件目录 < 2分钟）。

**Acceptance Scenarios**:

1. **Given** 一个支持变更通知的连接且已启用缓存，**When** 在目录无变更时触发同步，**Then** 系统使用缓存数据完成同步，无需全量递归列举远程目录。
2. **Given** 已启用缓存且远程发生文件变更，**When** 触发同步，**Then** 系统通过变更通知增量更新缓存后完成同步，仅处理变更的文件。
3. **Given** 首次为连接启用缓存，**When** 触发同步，**Then** 同步过程正常进行，缓存在同步过程中按需填充，不阻塞当前同步任务。
4. **Given** 已启用缓存的连接，**When** 用户配置缓存过期时间为 24 小时，**Then** 系统在缓存超过 24 小时后自动重新获取完整目录结构。

---

### User Story 2 - 不支持变更通知时优雅降级 (Priority: P2)

作为用户，当我的远程存储不支持变更通知或我选择不启用缓存时，我希望同步功能仍然正常工作，不会因缺少缓存而失败。

**Why this priority**: 确保向后兼容性，不影响现有用户的工作流程，是功能稳定性的基础。

**Independent Test**: 对 S3 等不支持变更通知的连接进行同步，验证系统自动使用常规列举流程且同步结果正确。

**Acceptance Scenarios**:

1. **Given** 一个不支持变更通知的远程连接，**When** 触发同步，**Then** 系统自动回退到常规递归列举流程并正确完成同步。
2. **Given** 一个支持变更通知的连接但用户未启用缓存，**When** 触发同步，**Then** 系统使用常规列举流程完成同步。

---

### User Story 3 - 管理连接缓存设置 (Priority: P2)

作为用户，我希望能够为每个连接独立配置缓存参数（如启用/关闭、过期时间、轮询间隔），以便根据不同存储的特点进行优化。

**Why this priority**: 灵活的配置能力是满足不同用户场景需求的关键。

**Independent Test**: 在连接设置中启用缓存、修改过期时间和轮询间隔，验证配置生效且仅影响该连接。

**Acceptance Scenarios**:

1. **Given** 用户在连接设置界面，**When** 启用缓存开关，**Then** 该连接开始使用缓存机制，其他连接不受影响。
2. **Given** 用户为某连接配置过期时间为 0（永不过期），**When** 配置保存后，**Then** 该连接的缓存不会因时间过期而自动失效。
3. **Given** 用户修改变更通知轮询间隔为 5 分钟，**When** 配置生效后，**Then** 系统按 5 分钟间隔检查远程变更。

---

### User Story 4 - 关闭缓存释放资源 (Priority: P3)

作为用户，我希望能够随时关闭某个连接的缓存功能，确保后台变更通知服务停止、资源被释放，且同步行为恢复正常。

**Why this priority**: 保障用户对系统行为的可控性，避免不必要的资源占用。

**Independent Test**: 关闭已启用缓存的连接的缓存功能，验证后台服务停止且后续同步使用常规流程。

**Acceptance Scenarios**:

1. **Given** 已启用缓存且后台变更通知正在运行，**When** 用户关闭该连接的缓存，**Then** 后台变更通知服务在 1 分钟内停止。
2. **Given** 用户刚关闭缓存，**When** 触发同步，**Then** 系统使用常规列举流程完成同步。

---

### User Story 5 - 手动清理连接缓存 (Priority: P3)

作为用户，当我怀疑缓存数据不准确时，我希望能够手动清理某个连接的缓存，让系统重新构建。

**Why this priority**: 提供问题恢复手段，增强用户信心。

**Independent Test**: 手动清理缓存后进行同步，验证系统重新构建缓存且同步结果正确。

**Acceptance Scenarios**:

1. **Given** 已有缓存数据的连接，**When** 用户执行手动清理缓存操作，**Then** 该连接的所有缓存记录被清空。
2. **Given** 缓存刚被清理，**When** 触发同步，**Then** 系统重新构建缓存并正确完成同步。

---

### Edge Cases

- **变更通知不完整**：远程支持变更通知但返回的变更信息不完整或有遗漏时，系统通过 TTL 机制保证最终一致性（缓存条目按 InfoAge 自然过期后重新获取）。
- **权限变更**：变更通知权限被撤销或连接中断时，rclone 内部会自动重试，应用层依赖 TTL 作为兜底，缓存过期后自动回退到远程读取。
- **大量重命名/移动**：目录包含大量文件重命名或移动操作时，ChangeNotify 会逐条通知变更，缓存据此更新路径映射；若通知不完整，TTL 过期后自动修复。
- **服务重启后的缓存恢复**：采用惰性校验 + TTL 策略，无需特殊重启处理。每个缓存条目按 InfoAge（默认6小时）独立过期，访问时自动检查并按需从远程刷新，停机期间的变更会在条目过期后被正确同步。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 允许用户为每个连接单独启用或关闭缓存功能。
- **FR-002**: 当远程支持变更通知且缓存已启用时，系统 MUST 尽可能通过变更通知捕获远程变更（包括来自其他设备或 Web 界面的变更），并即时将相关缓存条目标记为过期（无论对应目录是否已标记为完全加载）；实际元数据在下次访问时按需获取，若通知不完整，由 TTL 机制兜底保证最终一致性。
- **FR-003**: 当远程不支持变更通知或缓存未启用时，系统 MUST 自动回退到常规递归列举流程并保证同步正确性。
- **FR-004**: 系统 MUST 支持配置缓存过期时间（InfoAge），默认为 6 小时；设置为空值时使用默认值，设置为 0 或负数时表示永不过期。
- **FR-005**: 系统 MUST 支持配置变更通知的轮询间隔，默认为 1 分钟。
- **FR-006**: 用户 MUST 能够手动清理某个连接的全部缓存数据。
- **FR-007**: 当连接启用缓存后，系统 MUST 在后台持续运行变更通知订阅（即使没有活动的同步任务），直到用户显式关闭缓存。
- **FR-008**: 系统 MUST 支持按需填充缓存：在同步过程中访问到的路径将被缓存，并标记目录是否已完成直接子项的列举。不要求在启用后立即进行后台全量扫描。
- **FR-009**: 缓存过期时，系统 MUST 自动触发完整目录结构重新获取以保证一致性。

### Key Entities

- **连接 (Connection)**：表示一个远程存储连接。关键属性：名称、远程类型、是否启用缓存、缓存配置。
- **缓存配置 (Cache Config)**：表示连接的缓存策略。关键属性：过期时间/InfoAge（默认 6 小时，空值使用默认值，0 或负数表示永不过期）、变更通知轮询间隔（默认 1 分钟）。
- **缓存条目 (Cache Entry)**：映射远程文件/目录的元数据。
  - **存储方式**：独立 SQLite 文件（`app_data/cache/<connection_id>.db`），使用 WAL 模式支持并发读取。
  - **Schema 迁移策略**：版本号 + 条件重建（缓存可丢弃，无需复杂迁移脚本）。
  - **核心字段**：
    - `path`（主键）：文件/目录路径（相对于连接根）
    - `parent`：父目录路径（用于高效查询子项）
    - `mod_time`：修改时间（Unix timestamp，纳秒精度）
    - `is_dir`：是否为目录
    - `size`：文件大小（目录时为 NULL）
    - `hash`：Hash 值（格式：算法:值，如 "md5:abc123"）
    - `dir_loaded`：目录子项是否已完整列举
    - `cached_at`：缓存时间戳（用于 TTL 过期检查）

## Clarifications

### 术语对照

| 用户层面术语 | 技术术语 | 说明 |
|-------------|---------|------|
| 缓存过期时间 | InfoAge | 缓存条目的 TTL，超过此时间后自动从远程刷新（默认 6 小时） |
| 变更通知轮询间隔 | ChangeNotifyPoll | ChangeNotify 检查远程变更的间隔（默认 1 分钟，最小 10 秒） |

### Session 2026-01-08
- Q: 缓存数据应该存储在哪里？ → A: 独立 SQLite 文件（存放于 `app_data/cache/<connection_id>.db`），使用 WAL 模式支持并发读取。研究已确认此方案可行且优于主数据库方案。
- Q: 后台变更通知服务的生命周期是怎样的？ → A: 全局常驻。应用启动时为所有启用缓存的连接启动订阅，直到应用关闭或用户手动禁用缓存。
- Q: 缓存数据的详细程度如何？ → A: 全量元数据。存储路径、大小、修改时间、Hash 等与 rclone Object 兼容的指标，以实现 zero-request 比对。
- Q: 首次启用缓存时的构建策略是什么？ → A: 完全按需填充。仅缓存同步时访问的路径，目录需记录其直接子项是否已完整获取（非递归）。
- Q: 收到变更通知但父目录未完全加载时如何处理？ → A: 即时更新。无论目录是否加载全，都根据变更通知更新或添加缓存中的具体项。
- Q: 变更通知的实现模式是什么？ → A: 由 rclone 底层决定。完全依赖 rclone 的 changeNotify 实现，根据不同远程后端的能力自动选择最佳方式（轮询或事件驱动），应用层无需感知具体机制。
- Q: 服务重启后，系统应如何处理缓存一致性？ → A: 惰性校验 + TTL。无需特殊重启处理，每个缓存条目按 InfoAge（默认6小时）独立过期，访问时自动检查并按需从远程刷新。
- Q: 当多个同步任务同时访问同一连接的缓存时，系统应如何处理并发？ → A: 依赖 SQLite 内置并发控制。使用 WAL 模式支持多读者并发读取，无需应用层额外加锁。
- Q: 系统是否需要对缓存大小或文件数量进行限制以防止资源耗尽？ → A: 无限制。允许缓存无限增长，仅依赖过期时间清理。元数据占用空间小，无需额外限制。
- Q: 当变更通知服务因权限问题或连接中断失败后，系统应如何处理已缓存的旧数据？ → A: 信任 rclone 内部重试机制 + TTL 兜底。rclone 会自动在下次 tick 时重试，即使长期失效，缓存也会按 InfoAge 自然过期，保证最终一致性。
- Q: 系统应如何检测和恢复变更通知不完整或有遗漏的情况？ → A: 依赖 TTL 兜底。由于 ChangeNotify 接口不返回错误，无法主动检测遗漏，但缓存条目按 InfoAge 自然过期后会从远程重新获取，保证最终一致性。
- Q: MetaCache 后端如何与现有系统集成？ → A: 标准 rclone 后端注册 + DBStorage 透明注入。通过 `-cache` 后缀命名约定（如 `myonedrive-cache:`）显式控制是否使用缓存。
- Q: 多个任务使用同一连接不同路径时，缓存如何处理？ → A: 基于连接共享。同一连接的不同路径共享同一个 CacheStore，ChangeNotify 订阅也共享，最大化缓存利用率。
- Q: Connection 如何存储缓存配置？ → A: 使用 JSON 类型的 `options` 字段（参考 Task.options 设计），包含 `cache.enabled`、`cache.infoAge`、`cache.changeNotifyPoll` 等配置。
- Q: ChangeNotify 回调的能力限制是什么？ → A: rclone 的 ChangeNotify 回调仅提供变更路径 (path) 和条目类型 (EntryType)，不包含完整元数据（如 size、modTime、hash 等）。因此采用"标记过期 + 按需获取"策略，避免每次通知都触发远程请求。
- Q: 为什么不在收到 ChangeNotify 时立即获取元数据？ → A: 批量变更（如大量文件重命名）会产生多个通知，立即获取元数据会导致大量并发远程请求，影响性能。采用惰性获取策略，仅在下次访问时按需刷新。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 对支持变更通知的连接，包含 10 万+ 文件的目录在无变更时，重复同步可在 2 分钟内完成（相比首次全量扫描显著缩短）。
- **SC-002**: 对不支持变更通知的连接，启用缓存后同步总耗时不超过未启用缓存时的 110%（无明显性能退化）。
- **SC-003**: 远程发生变更后，95% 的变更可在下一次同步中体现（受轮询间隔影响，默认 1 分钟内）。
- **SC-004**: 用户关闭连接缓存后，后台变更通知服务在 1 分钟内完全停止。
- **SC-005**: 首次启用缓存后触发的同步，不应因缓存构建而产生超过 10% 的额外延迟。

# Feature Specification: Task Event Hooks

**Feature Branch**: `014-task-event-hooks`  
**Created**: 2026-01-17  
**Status**: Draft  
**Input**: User description: "我希望在任务执行完成、任务执行失败等时间点，允许用户通过hook机制，执行自定义命令或者发出HTTP请求，以便与外部系统更好的交互。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 任务完成时发送 HTTP 通知 (Priority: P1)

作为用户，我希望在同步任务成功完成后，系统能自动向我指定的 URL 发送 HTTP 请求，以便我的监控系统能够及时收到通知。

**Why this priority**: 这是 hook 机制的核心功能，HTTP 通知是与外部系统集成最常见的方式，能够实现与 Slack、钉钉、企业微信等通知系统的对接。

**Independent Test**: 可以通过配置一个任务完成时的 HTTP hook，执行同步任务，验证目标 URL 是否收到正确的请求来独立测试。

**Acceptance Scenarios**:

1. **Given** 用户已配置任务完成时触发的 HTTP hook（POST 请求到指定 URL），**When** 同步任务成功完成，**Then** 系统自动发送 HTTP POST 请求到配置的 URL，请求包含任务执行信息（任务名称、执行状态、执行时间、传输统计等）。

2. **Given** 用户配置了带有自定义 Headers 的 HTTP hook，**When** hook 被触发，**Then** 请求中包含用户配置的所有自定义 Headers。

3. **Given** HTTP hook 配置的目标服务不可用，**When** hook 被触发，**Then** 系统记录 hook 执行失败的日志，但不影响主任务的状态（任务仍显示为成功完成）。

---

### User Story 2 - 任务失败时执行本地命令 (Priority: P1)

作为系统管理员，我希望在同步任务失败时能自动执行本地脚本，以便进行告警、重试或其他自动化处理。

**Why this priority**: 命令执行与 HTTP 通知同等重要，为用户提供了更灵活的本地自动化能力，可以执行备份脚本、发送邮件、触发其他进程等。

**Independent Test**: 可以通过配置一个任务失败时执行的命令 hook，手动触发任务失败场景，验证命令是否被正确执行来独立测试。

**Acceptance Scenarios**:

1. **Given** 用户已配置任务失败时触发的命令 hook（执行指定 shell 命令），**When** 同步任务失败，**Then** 系统自动执行配置的命令，并将任务上下文信息（任务名称、错误信息等）通过环境变量传递给命令。

2. **Given** 配置的命令执行超时，**When** hook 被触发，**Then** 系统在超时后终止命令执行，记录超时日志，但不影响主任务的状态记录。

3. **Given** 配置的命令执行失败（非零退出码），**When** hook 被触发，**Then** 系统记录命令执行失败的日志及退出码，但不影响主任务的状态。

---

### User Story 3 - 通过 Web UI 配置 Hooks (Priority: P2)

作为用户，我希望通过 Web 界面直观地配置和管理 hooks，无需手动编辑配置文件。

**Why this priority**: 用户友好的配置界面是易用性的关键，但核心功能（hook 执行机制）优先级更高。

**Independent Test**: 可以通过 Web UI 在任务或连接详情页创建、编辑、删除 hook 配置，验证配置能够正确保存和生效来独立测试。

**Acceptance Scenarios**:

1. **Given** 用户在任务详情页面或连接详情页面，**When** 用户点击添加 hook，**Then** 显示 hook 配置表单，允许选择触发事件类型（任务完成/任务失败/任务开始）和 hook 类型（HTTP/命令）。

2. **Given** 用户正在配置 HTTP 类型的 hook，**When** 填写配置，**Then** 可以配置：启用状态、优先级（数字）、URL、HTTP 方法（GET/POST/PUT）、自定义 Headers、请求 Body 模板。

3. **Given** 用户正在配置命令类型的 hook，**When** 填写配置，**Then** 可以配置：启用状态、优先级（数字）、命令内容、工作目录、执行超时时间。

4. **Given** 用户已保存 hook 配置，**When** 查看任务或连接详情，**Then** 可以看到关联的所有 hooks 列表，并可以编辑或删除。

5. **Given** 用户在连接上配置了 Hook，**When** 关联该连接的任务执行完成，**Then** 连接上的 Hook 会被触发，如同配置在任务上一样。

---

### User Story 4 - 任务开始时触发 Hook (Priority: P3)

作为用户，我希望在同步任务开始执行时也能触发 hook，以便在任务开始时进行资源准备或状态更新。

**Why this priority**: 任务开始事件相对于完成/失败事件使用频率较低，但能提供完整的任务生命周期覆盖。

**Independent Test**: 可以通过配置任务开始时的 hook，启动同步任务，验证 hook 是否在任务实际执行前被触发来独立测试。

**Acceptance Scenarios**:

1. **Given** 用户已配置任务开始时触发的 hook，**When** 同步任务开始执行，**Then** 系统在任务实际执行前先触发配置的 hook。

---

### Edge Cases

- 当用户配置了多个同类型事件的 hooks 时，系统应按配置顺序依次执行所有 hooks
- 当 hook 执行时间过长时（超过配置的超时时间，默认 30 秒），应有超时机制防止阻塞后续操作
- 当配置的 HTTP URL 无效或命令不存在时，应在配置保存时给予警告，但允许保存
- 当任务执行速度很快时（如空同步），hook 仍应被正确触发
- 当 hook 配置包含敏感信息（如 API Key）时，应支持通过环境变量引用（如 `{{.Env.API_KEY}}`），避免敏感值直接存入数据库
- 当 hook 功能被全局禁用时，Web UI 应完全隐藏 hook 相关界面，不允许用户查看或配置 hooks

## Requirements *(mandatory)*

### Functional Requirements

#### 1. 事件类型

- **FR-001**: 系统支持以下任务事件触发 hooks：
  - `on_start` - 任务开始执行前
  - `on_success` - 任务成功完成后
  - `on_failure` - 任务失败后
  - `on_end` - 任务结束后（无论成功或失败，最后触发）

#### 2. Hook 类型与配置

- **FR-002 (HTTP Hook)**: 支持配置 URL、HTTP 方法（GET/POST/PUT）、自定义请求头、请求体模板、执行超时时间（默认 30 秒）。URL、请求头（Key 和 Value）及请求体均支持 Go text/template 语法。
- **FR-003 (命令 Hook)**: 支持配置 shell 命令、工作目录、执行超时时间（超时后自动终止）。命令内容支持 Go text/template 语法，同时任务上下文信息也通过环境变量传递给命令。

#### 3. 模板系统

- **FR-004**: 模板引擎使用 Go text/template 语法，上下文包含：
  - **预定义变量**：`Task`（任务信息）、`Job`（执行信息）、`Event`（事件类型）、`Error`（错误信息）、`Duration`（执行耗时）、`Stats`（传输统计）、`Env`（环境变量映射，如 `{{.Env.API_KEY}}`）
  - **辅助函数**：`FormatTime`（格式化时间）、`FormatDuration`（格式化耗时）、`FormatSizeBytes`（格式化字节大小）、`JsonMarshal`（转换为 JSON）、`Summary`（生成默认摘要）

#### 4. 执行顺序与优先级

- **FR-005**: 同一事件的多个 hooks 按优先级从小到大顺序同步执行（0 为最高优先级，null/未设置排最后，相同优先级按创建时间顺序）。任务级和连接级 Hooks 混合排序后统一执行。

#### 5. 错误处理

- **FR-006**: 每个 Hook 支持配置错误处理行为：
  - `ON_ERROR_IGNORE`（默认）- 忽略错误，继续执行后续 hooks
  - `ON_ERROR_CANCEL` - 停止任务，Job 标记为 `CANCELLED`，仅触发 on_end hook
  - `ON_ERROR_FATAL` - 停止任务，Job 标记为 `FAILED`，依次触发 on_failure 和 on_end hook
- **FR-006a**: 对于 on_start 事件，当配置为 CANCEL 或 FATAL 时，hook 失败将阻止任务开始执行。

#### 6. 执行记录

- **FR-007**: 系统记录每次 hook 执行结果（成功/失败、响应状态码/退出码、执行耗时、错误信息），作为 Job 历史详情的一部分在 Web UI 中展示。Hook 执行记录随 Job 记录一同保留/清理。

#### 7. 配置管理

- **FR-008 (Web UI 配置)**: 用户可通过 Web UI 为每个任务和每个连接独立配置 hooks。连接上的 Hooks 作为共享 Hooks，关联任务触发事件时会被执行。
- **FR-009 (单个 Hook 状态)**: 每个 Hook 支持独立的启用/禁用状态，禁用的 Hook 不会被触发。
- **FR-010 (全局开关)**: 管理员可通过配置文件（config.toml）全局启用或禁用 hook 功能，变更在服务重启后生效。全局禁用时，所有 hooks 不触发执行，Web UI 隐藏 hook 相关界面。

### Key Entities

- **Hook**: 表示一个 hook 配置，包含启用状态、优先级、触发事件类型、hook 类型（HTTP/命令）、具体配置参数、关联的任务或连接
- **HookExecution**: 表示一次 hook 执行记录，包含执行时间、执行结果、响应/退出码、错误信息、关联的 job
- **TaskContext**: 表示传递给 hook 的任务上下文数据，包含任务元信息、执行统计、错误详情，无论 Hook 是配置在任务还是连接上，均传递任务上下文内容

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 用户可以在 2 分钟内通过 Web UI 完成一个 hook 的配置
- **SC-002**: Hook 在任务状态变更后 5 秒内被触发执行
- **SC-003**: 95% 以上的 hook 执行结果能被正确记录到日志中
- **SC-004**: Hook 执行失败不会导致任务状态误报（100% 隔离）
- **SC-005**: 用户可以通过 hook 与至少 3 种主流通知系统（Slack、钉钉、企业微信等）实现集成

## Clarifications

### Session 2026-01-17

- Q: 如果任务涉及多个连接（源/目标），Hook 如何触发？ → A: 目前源均为本地目录，仅触发任务关联的远端连接级 Hook。
- Q: 是否需要内置通知服务支持（Discord、Slack 等）？ → A: 保持通用 HTTP + 命令设计，提供常见服务的配置示例文档。
- Q: 命令 Hook 是否需要安全限制？ → A: 无限制，完全信任用户配置的命令。
- Q: HTTP Hook 请求失败是否需要自动重试？ → A: 不重试，失败后仅记录日志。
- Q: HTTP Hook 响应体如何处理？ → A: 忽略响应体，仅记录状态码。
- Q: 单个任务可配置的最大 Hook 数量限制？ → A: 无限制（允许任意数量 Hook）。
- Q: Hook 执行的指标追踪？ → A: 仅记录日志，不提供指标追踪。

## Assumptions

- 命令 hook 在服务器端执行，使用服务器的 shell 环境
- HTTP 请求支持 HTTPS，信任系统级证书
- Hook 执行默认超时时间为 30 秒
- 同一事件的多个 hooks 串行执行，不并行
- Hook 配置存储在现有数据库中，与任务关联
- Hook 功能默认启用，管理员可通过配置文件禁用
- 配置文件格式与现有 config.toml 保持一致，新增 `[app.hook]` 配置节

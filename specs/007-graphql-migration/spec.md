# Feature Specification: GraphQL Migration

**Feature Branch**: `007-graphql-migration`  
**Created**: 2024-12-20  
**Status**: Draft  
**Input**: User description: "现在 前后端是通过 restful接口 来进行请求的。但 restful接口 对于 接口定义、类型强校验不够友好。我希望切换成 schema first 的 graphql。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 前端开发者获得强类型接口支持 (Priority: P1)

作为前端开发者，我希望通过 GraphQL schema 自动生成 TypeScript 类型定义，这样在调用后端接口时能够获得完整的类型检查和智能提示，从而减少因类型不匹配导致的运行时错误。

**Why this priority**: 这是切换到 GraphQL 的核心价值——解决 RESTful 接口类型校验不友好的问题。类型安全能显著提升开发效率和代码质量。

**Independent Test**: 可以通过创建一个简单的 GraphQL query 并验证前端能否获得正确的 TypeScript 类型提示来独立测试。

**Acceptance Scenarios**:

1. **Given** 后端定义了 GraphQL schema，**When** 前端运行代码生成命令，**Then** 前端项目中生成对应的 TypeScript 类型定义文件
2. **Given** 前端使用生成的类型调用 API，**When** 传入错误类型的参数，**Then** TypeScript 编译器报告类型错误
3. **Given** 前端编辑器打开 API 调用代码，**When** 开发者输入查询参数，**Then** 编辑器提供字段名自动补全和类型提示

---

### User Story 2 - 后端开发者使用 Schema-First 方式定义接口 (Priority: P1)

作为后端开发者，我希望通过 GraphQL schema 文件先定义接口规范，然后生成对应的代码骨架，这样能确保接口定义清晰、文档化，并与实现保持一致。

**Why this priority**: Schema-first 是用户明确要求的开发模式，确保接口定义与实现分离，便于团队协作和接口治理。

**Independent Test**: 可以通过定义一个新的 GraphQL 类型和 resolver 来独立测试 schema-first 工作流程。

**Acceptance Scenarios**:

1. **Given** 开发者编写了新的 GraphQL schema 定义，**When** 运行代码生成命令，**Then** 后端生成对应的接口骨架代码
2. **Given** schema 定义了必填字段，**When** resolver 返回缺少该字段的数据，**Then** 系统在运行时产生明确的验证错误
3. **Given** 开发者修改了 schema 中的字段类型，**When** 运行代码生成，**Then** 生成的代码反映新的类型定义

---

### User Story 3 - 现有功能平滑迁移 (Priority: P2)

作为系统用户，我希望现有的所有功能在迁移到 GraphQL 后仍然正常工作，这样迁移过程不会影响现有功能的使用。

**Why this priority**: 功能回归是迁移项目最大的风险，必须确保迁移不破坏现有功能。

**Independent Test**: 可以通过对比迁移前后的 API 响应数据是否一致来独立验证。

**Acceptance Scenarios**:

1. **Given** 用户使用任务管理功能，**When** 通过新的 GraphQL API 调用，**Then** 创建/查询/更新/删除任务的操作与原 REST API 结果一致
2. **Given** 用户使用连接管理功能，**When** 通过新的 GraphQL API 调用，**Then** 连接的 CRUD 操作与原 REST API 结果一致
3. **Given** 用户使用文件浏览功能，**When** 通过新的 GraphQL API 查询，**Then** 返回的文件列表与原 REST API 结果一致

---

### User Story 4 - 按需获取数据减少传输 (Priority: P3)

作为前端开发者，我希望能够只请求需要的字段，避免获取不必要的数据，从而减少网络传输量并提升页面加载速度。

**Why this priority**: GraphQL 的按需查询是其重要优势，但在功能迁移完成后才能充分体现价值。

**Independent Test**: 可以通过对比请求相同资源时 REST 和 GraphQL 的响应数据量来验证。

**Acceptance Scenarios**:

1. **Given** 前端只需要任务的名称和状态，**When** 使用 GraphQL 查询并只选择这两个字段，**Then** 响应数据只包含请求的字段
2. **Given** 前端需要任务及其关联的连接信息，**When** 使用单个 GraphQL 查询，**Then** 一次请求返回任务和关联连接数据

---

### Edge Cases

- 当客户端发送格式错误的 GraphQL 查询时，系统返回清晰的本地化错误信息
- 当查询嵌套层级超过最大深度限制时，系统拒绝请求并返回错误
- 当 schema 和 resolver 实现不一致时，系统提供明确的错误提示

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须提供 GraphQL 端点，支持 query、mutation 操作
- **FR-013**: Mutation 设计必须采用粗粒度（面向任务/业务动作）风格，确保复杂操作（如导入并执行）的原子性，并返回具体的业务错误信息
- **FR-012**: 列表查询（任务、作业、日志等）必须支持偏移量分页（Limit/Offset，默认 limit=20, offset=0），允许前端跳转到特定页码
- **FR-002**: 系统必须支持 schema-first 开发模式，先定义 schema 文件再生成代码
- **FR-003**: 后端必须能够根据 GraphQL schema 自动生成 Go resolver 接口代码
- **FR-004**: 前端必须能够根据 GraphQL schema 自动生成 TypeScript 类型定义和查询 hooks
- **FR-005**: 系统必须实现现有 RESTful API 的全部功能对应的 GraphQL 操作：
  - 任务管理 (Task CRUD + Run)
  - 连接管理 (Connection CRUD + Test + Quota)
  - 作业查询 (Job 列表/详情/进度)
  - 日志查询 (Log 列表)
  - 提供者查询 (Provider 列表/配置项)
  - 文件浏览 (本地/远程文件列表，支持单层按需加载)
  - 导入功能 (解析/执行)
- **FR-006**: 系统必须支持 GraphQL 实时订阅 (Subscription) 以替代现有的 SSE 事件推送
  - Job 必须支持细粒度订阅以获取实时进度和状态变更
- **FR-007**: GraphQL schema 必须包含完整的字段描述和文档注释
- **FR-008**: 系统必须提供 GraphQL Playground 或类似工具供开发调试使用
- **FR-009**: 系统必须限制 GraphQL 查询的最大嵌套深度（默认 8 层）以防止性能问题，同时在 Schema 中尽可能暴露实体间的关联关系以支持灵活查询
- **FR-010**: GraphQL 错误消息必须使用现有 i18n 系统实现本地化
- **FR-011**: 迁移完成后必须删除所有 REST API 端点
- **FR-014**: 后端必须实现数据加载优化机制以解决 GraphQL N+1 查询问题
  - 当查询包含关联实体时，系统必须批量加载数据而非逐条查询
  - 必须支持在单个数据库往返中加载相同类型的多个关联实体

### Key Entities

- **Task**: 同步任务，包含源/目标连接、调度规则、同步选项等
- **Connection**: 远程存储连接配置，包含类型、凭证等敏感信息
- **Job**: 任务执行记录，包含执行状态、进度、错误信息等
- **Log**: 系统日志记录
- **Provider**: 支持的存储提供者类型及其配置选项
- **File**: 本地或远程文件/目录信息

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 前端开发者在编写 API 调用时能获得 100% 的字段类型提示覆盖
- **SC-002**: 后端新增或修改 API 字段后，前端能在 1 分钟内通过代码生成获得更新的类型定义
- **SC-003**: 现有全部 API 功能通过 GraphQL 实现后，功能验收测试通过率达到 100%
- **SC-004**: 类型不匹配错误能在编译期被发现，而非运行时
- **SC-005**: 开发者查阅 API 文档时，通过 GraphQL schema 能获得所有字段的类型和描述信息
- **SC-006**: 查询包含关联实体时（如任务列表含连接信息），数据库查询次数不随列表项数量线性增长

## Clarifications

### Session 2024-12-20

- Q: 列表数据的分页策略 → A: 偏移量分页 (Offset-based)，支持跳转到特定页码
- Q: 实时数据更新的交互模式 → A: 混合模式，关键状态（如任务/作业进度）用细粒度订阅，通用通知用全局流
- Q: 复杂操作的 Mutation 设计风格 → A: 粗粒度 Mutation (Task-oriented)，确保操作原子性并简化前端逻辑
- Q: 文件浏览的加载行为 → A: 单层按需加载 (Shallow/Lazy loading)，与现有 REST 设计保持一致
- Q: 数据关联的深度限制 → A: 全图可见，尽可能暴露实体间关系以支持灵活查询，配合深度限制防止性能问题
- Q: Frontend Framework → A: SolidJS (Correction from React)
- Q: REST API 废弃策略 → A: 迁移后废弃 REST，GraphQL 完全取代 REST，迁移完成后删除旧接口
- Q: GraphQL Subscription 实现范围 → A: 必须同步实现，Subscription 是本次迁移的必要部分
- Q: 查询复杂度保护策略 → A: 深度限制，限制查询最大嵌套深度
- Q: 敏感数据字段处理策略 → A: 不需要，单机单用户应用无需特殊权限控制
- Q: GraphQL 错误消息本地化 → A: 需要本地化，使用现有 i18n 系统返回用户语言的错误信息
- Q: N+1 查询优化 → A: 必须实现，后端需要支持批量数据加载以避免关联查询时产生 N+1 数据库查询问题

## Assumptions

- 项目团队熟悉 GraphQL 基本概念和使用方式
- 后端使用 Go 语言，将采用成熟的 Go GraphQL 库实现
- 前端使用 TypeScript + SolidJS，将采用成熟的 GraphQL 客户端库
- 现有的数据模型 (ent) 可以与 GraphQL schema 集成
- SSE 事件推送可以使用 GraphQL Subscription 替代实现相同功能
- 本项目为单机单用户应用，无需实现用户认证和字段级权限控制
- 迁移期间可以同时保留 REST 和 GraphQL 两套接口，便于渐进式迁移；迁移完成后删除 REST 接口

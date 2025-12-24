# Research: GraphQL Migration

**Feature Branch**: `007-graphql-migration`  
**Date**: 2024-12-20

## Research Tasks

### 1. 后端 GraphQL 框架选型

**Decision**: gqlgen (99designs/gqlgen)

**Rationale**:
- 用户明确指定使用 gqlgen
- Schema-first 开发模式，符合需求 FR-002
- Go 语言原生支持，与现有后端技术栈一致
- 自动代码生成 resolver 接口骨架，满足 FR-003
- 成熟的 Subscription 支持（基于 WebSocket），可替代现有 SSE
- 与 Ent ORM 有良好的集成方案
- 高声誉（Context7 评级: High）

**Alternatives Considered**:
- `graph-gophers/graphql-go`: 反射式，非 schema-first，不符合用户要求
- `genqlient`: 客户端库，非服务端框架

### 2. 前端 GraphQL 客户端选型

**Decision**: urql + @urql/exchange-graphcache + gql.tada

**Rationale**:
- 用户明确指定使用此组合
- urql: 轻量级、可扩展的 GraphQL 客户端，支持 SolidJS
- @urql/exchange-graphcache: 规范化缓存，支持乐观更新，符合 Constitution VIII 的 Optimistic UI 要求
- gql.tada: 无需代码生成的 TypeScript 类型安全，直接从 schema 推断类型
  - 编译时类型检查，满足 SC-001, SC-004
  - 开发体验优于传统 codegen 方案

**Alternatives Considered**:
- Apollo Client: 功能全面但较重，SolidJS 支持需额外封装
- graphql-codegen: 需要额外代码生成步骤，gql.tada 更轻量

### 3. N+1 查询优化方案

**Decision**: vikstrous/dataloadgen

**Rationale**:
- gqlgen 官方推荐的 dataloader 实现
- Go 泛型支持，类型安全
- 批量加载模式，满足 FR-014
- 与 Ent 查询模式兼容

**Implementation Pattern**:
```go
// Loaders 结构通过中间件注入到 context
type Loaders struct {
    ConnectionLoader *dataloadgen.Loader[uuid.UUID, *ent.Connection]
    TaskLoader       *dataloadgen.Loader[uuid.UUID, *ent.Task]
}

// 在 resolver 中使用
func (r *taskResolver) Connection(ctx context.Context, obj *model.Task) (*model.Connection, error) {
    return loaders.GetConnection(ctx, obj.ConnectionID)
}
```

### 4. GraphQL Subscription 实现

**Decision**: gqlgen 内置 WebSocket Transport + 简化 Subscription 设计

**Rationale**:
- gqlgen 原生支持 graphql-ws 协议
- 可完全替代现有 SSE 广播器
- 基于前端实际使用情况简化设计：前端使用全局订阅监听所有作业进度

**Subscription Design** (简化后):
```graphql
type Subscription {
  """
  订阅作业进度事件
  
  - 无参数：订阅所有作业进度（全局订阅，匹配现有 SSE 用法）
  - taskId：仅订阅指定任务的作业
  - connectionId：仅订阅指定连接相关的作业
  """
  jobProgress(taskId: ID, connectionId: ID): JobProgressEvent!
}

type JobProgressEvent {
  jobId: ID!
  taskId: ID!           # 方便前端按任务更新本地状态
  connectionId: ID!     # 方便前端按连接过滤
  status: JobStatus!
  filesTransferred: Int!
  bytesTransferred: BigInt!
  startTime: DateTime!
  endTime: DateTime
  currentFile: String
  percentage: Float
}
```

**优点**:
- ✅ 无参数 = 全局订阅（符合前端当前 SSE 用法）
- ✅ 可选参数支持按需过滤
- ✅ 强类型，无需 JSON payload
- ✅ 单一 Subscription 类型，简化实现

### 5. Ent 与 gqlgen 集成

**Decision**: 手动映射 + 自定义 Model

**Rationale**:
- 避免直接暴露 Ent 实体（包含敏感字段如 EncryptedConfig）
- GraphQL model 与 Ent entity 解耦，便于 API 演进
- 使用 resolver 层转换数据

**Integration Pattern**:
```go
// GraphQL model (generated from schema)
type Task struct {
    ID           uuid.UUID
    Name         string
    // ... 公开字段
}

// Resolver 转换
func entTaskToModel(t *ent.Task) *model.Task {
    return &model.Task{
        ID:   t.ID,
        Name: t.Name,
        // ...
    }
}
```

### 6. 错误本地化

**Decision**: 复用现有 go-i18n + I18nError 模式

**Rationale**:
- Constitution IX 要求使用现有 i18n 系统
- 现有 `internal/core/errs/i18n_error.go` 已实现 I18nError
- GraphQL 错误扩展中携带本地化消息

**Implementation**:
```go
// 在 gqlgen 错误处理器中转换 I18nError
func errorPresenter(ctx context.Context, e error) *gqlerror.Error {
    if i18nErr, ok := e.(errs.I18nError); ok {
        locale := apicontext.GetLocale(ctx)
        return &gqlerror.Error{
            Message: i18n.Localize(locale, i18nErr.Key(), i18nErr.Args()),
            Extensions: map[string]any{"code": i18nErr.Code()},
        }
    }
    return graphql.DefaultErrorPresenter(ctx, e)
}
```

### 7. 查询深度限制

**Decision**: gqlgen 内置 ComplexityLimit + DepthLimit

**Rationale**:
- 满足 FR-009 防止性能问题
- gqlgen 提供开箱即用的复杂度计算

**Configuration**:
```go
srv := handler.NewDefaultServer(generated.NewExecutableSchema(cfg))
srv.Use(extension.FixedComplexityLimit(200))
// 自定义深度限制中间件
```

### 8. 分页策略

**Decision**: Offset-based 分页

**Rationale**:
- FR-012 明确要求支持跳转到特定页码
- 与前端 UI 设计一致

**Schema Pattern**:
```graphql
type TaskConnection {
  items: [Task!]!
  totalCount: Int!
  pageInfo: OffsetPageInfo!
}

type OffsetPageInfo {
  limit: Int!
  offset: Int!
  hasNextPage: Boolean!
}
```

### 9. Mutation 设计风格

**Decision**: 粗粒度 (Task-oriented) Mutations + 命名空间模式

**Rationale**:
- FR-013 要求确保复杂操作的原子性
- 按实体拆分 Mutation 命名空间（如 `task`, `connection`, `import`）
- 错误处理策略：
  - **意外错误**（数据库、内部异常）→ 使用 GraphQL errors 抛出
  - **预期失败**（连接测试不通、配置解析错误）→ 使用 Result union 类型

**Examples**:
```graphql
type Mutation {
  task: TaskMutation!
  connection: ConnectionMutation!
  import: ImportMutation!
}

type TaskMutation {
  # 失败时直接抛出 GraphQL error
  create(input: CreateTaskInput!): Task!
  update(id: ID!, input: UpdateTaskInput!): Task!
  delete(id: ID!): Task!
  run(taskId: ID!): Job!
}

type ConnectionMutation {
  create(input: CreateConnectionInput!): Connection!
  update(id: ID!, input: UpdateConnectionInput!): Connection!
  delete(id: ID!): Connection!
  # 测试失败是预期的业务结果，用 union 表示
  test(id: ID!): TestConnectionResult!
  testUnsaved(input: TestConnectionInput!): TestConnectionResult!
}

# 仅保留预期失败场景的 union
union TestConnectionResult = ConnectionTestSuccess | ConnectionTestFailure
union ImportParseResult = ImportParseSuccess | ImportParseError
```

### 10. GraphQL Playground

**Decision**: GraphiQL (内置于 gqlgen)

**Rationale**:
- 满足 FR-008
- gqlgen 默认支持
- 开发环境自动启用，生产环境可禁用

## Technology Integration Summary

| Layer | Technology | Version |
|-------|------------|---------|
| Backend GraphQL | gqlgen | latest |
| Backend Dataloader | dataloadgen | latest |
| Frontend Client | urql | latest |
| Frontend Cache | @urql/exchange-graphcache | latest |
| Frontend Types | gql.tada | latest |
| Transport (Query/Mutation) | HTTP POST | - |
| Transport (Subscription) | WebSocket (graphql-ws) | - |

## Dependencies to Add

### Backend (go.mod)
```
github.com/99designs/gqlgen
github.com/vektah/gqlparser/v2
github.com/vikstrous/dataloadgen
github.com/gorilla/websocket
```

### Frontend (package.json)
```
urql
@urql/core
@urql/exchange-graphcache
gql.tada
graphql
graphql-ws
```

### 11. gqlgen Directives 使用

**Decision**: 使用 `@goField(forceResolver: true)` 控制 resolver 生成

**Rationale**:
- 关系字段（如 Task.connection, Job.logs）需要独立 resolver 以支持 DataLoader
- 计算字段（如 Connection.loadStatus, Task.options）需要自定义逻辑
- 命名空间字段（Query.task, Mutation.connection）需要返回空对象实例

**Directives 定义**:
```graphql
directive @goField(
  forceResolver: Boolean
  name: String
  omittable: Boolean
) on INPUT_FIELD_DEFINITION | FIELD_DEFINITION

directive @goModel(
  model: String
  models: [String!]
) on OBJECT | INPUT_OBJECT | SCALAR | ENUM | INTERFACE | UNION

directive @goEnum(value: String) on ENUM_VALUE

directive @goTag(
  key: String!
  value: String!
) on INPUT_FIELD_DEFINITION | FIELD_DEFINITION
```

**应用场景**:
| 类型 | 字段 | 原因 |
|------|------|------|
| Task | options, connection, jobs, latestJob | ent edge / 类型转换 |
| Connection | config, loadStatus, loadError, tasks, quota | 解密 / 运行时状态 / 分页 |
| Job | task, logs | ent edge / 分页 |
| JobLog | job | ent edge |
| Query/Mutation | 所有命名空间字段 | 返回空对象实例 |

### 12. Task Options 类型化

**Decision**: 将 Task.options 从 JSON 改为强类型 `TaskSyncOptions`

**Rationale**:
- 提供编译时类型安全
- 与后端 Go 代码一致（`getConflictResolutionFromOptions` 函数）
- 更好的 API 文档和自动补全

**Type Definition**:
```graphql
"""冲突解决策略（仅用于双向同步）"""
enum ConflictResolution {
  NEWER   # 保留较新文件，重命名较旧文件
  LOCAL   # 保留本地文件，删除远程
  REMOTE  # 保留远程文件，删除本地
  BOTH    # 保留两者，添加冲突后缀
}

type TaskSyncOptions {
  conflictResolution: ConflictResolution
}

input TaskSyncOptionsInput {
  conflictResolution: ConflictResolution
}
```

### 13. Connection 状态字段

**Decision**: 添加 `loadStatus` 和 `loadError` 运行时状态字段

**Rationale**:
- 与前端现有逻辑一致（`load_status`, `load_error`）
- 追踪连接配置的加载状态
- 非持久化字段，通过 resolver 计算

**Type Definition**:
```graphql
enum ConnectionLoadStatus {
  LOADED   # 已加载
  LOADING  # 加载中
  ERROR    # 加载失败
}

type Connection {
  # ... 其他字段 ...
  loadStatus: ConnectionLoadStatus! @goField(forceResolver: true)
  loadError: String @goField(forceResolver: true)
}
```

## Open Questions (Resolved)

所有澄清问题均已在 spec.md 的 Clarifications 部分解决：
- ✅ 分页策略: Offset-based
- ✅ Subscription 范围: 细粒度 + 全局流
- ✅ Mutation 风格: 粗粒度
- ✅ 文件浏览: 单层按需加载
- ✅ 关联深度: 全图可见 + 深度限制
- ✅ REST 废弃: 迁移后删除
- ✅ N+1 优化: 必须实现
- ✅ 错误本地化: 必须实现

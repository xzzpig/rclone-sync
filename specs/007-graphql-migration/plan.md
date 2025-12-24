# Implementation Plan: GraphQL Migration

**Branch**: `007-graphql-migration` | **Date**: 2024-12-20 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/007-graphql-migration/spec.md`

## Summary

将现有的 RESTful API 迁移到 Schema-First GraphQL，以获得更好的类型安全和接口定义。后端使用 gqlgen（用户指定），前端使用 urql + @urql/exchange-graphcache + gql.tada（用户指定）。主要工作包括：定义 GraphQL schema、实现 resolver、设置 dataloader 解决 N+1 问题、实现 WebSocket subscription 替代 SSE、前端迁移到 GraphQL 客户端。

## Technical Context

**Language/Version**: Go 1.21+ (Backend), TypeScript 5.x (Frontend)  
**Primary Dependencies**: 
- Backend: gqlgen, dataloadgen, gorilla/websocket
- Frontend: urql, @urql/exchange-graphcache, gql.tada, graphql-ws
**Storage**: SQLite with Ent ORM (现有)  
**Testing**: go test (后端), vitest (前端)  
**Target Platform**: Linux server, Web browser
**Project Type**: Web (frontend + backend)
**Performance Goals**: 
- GraphQL endpoint 响应 < 100ms (p95)
- Subscription 延迟 < 500ms
**Constraints**: 
- 必须支持 Offset-based 分页
- 查询深度限制防止性能问题
- 错误消息本地化
**Scale/Scope**: 
- 单机单用户应用
- 迁移现有 20+ REST endpoints

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Rclone-First Architecture | ✅ Pass | 不涉及，GraphQL 是 API 层变更 |
| II. Web-First Interface | ✅ Pass | 继续通过 Web UI 交互 |
| III. Test-Driven Development | ⚠️ Required | 所有 resolver 必须有测试 |
| IV. Independent User Stories | ✅ Pass | 各功能模块独立迁移 |
| V. Observability and Reliability | ✅ Pass | GraphQL 使用现有日志系统 |
| VI. Modern Component Architecture | ✅ Pass | 前端继续使用 SolidJS |
| VII. Accessibility and UX Standards | ✅ Pass | 不涉及 UI 变更 |
| VIII. Performance and Optimistic UI | ✅ Pass | urql graphcache 支持乐观更新 |
| IX. Internationalization Standards | ⚠️ Required | 错误消息必须本地化 |

**Gate Status**: ✅ PASS - 无阻塞性违规

## Schema Sharing Strategy

前后端共用同一个 GraphQL schema 文件作为单一数据源 (Single Source of Truth):

1. **契约 Schema**: `specs/007-graphql-migration/contracts/schema.graphql` - 完整的单文件 schema 作为 API 契约
2. **实现 Schema**: `internal/api/graphql/schema/*.graphql` - 按实体拆分的多文件实现
3. **前端访问**: gql.tada 配置指向后端 schema 目录，自动合并多文件

**Schema 拆分策略**:

契约文件保持单文件完整性，实现时按实体拆分并使用 `extend` 语法：

```text
internal/api/graphql/schema/
├── schema.graphql      # 空根类型 + 指令 + 标量 + 分页
├── task.graphql        # Task + extend Query/Mutation
├── connection.graphql  # Connection + extend Query/Mutation
├── job.graphql         # Job/JobLog + extend Query/Subscription
├── provider.graphql    # Provider + extend Query
├── file.graphql        # FileEntry + extend Query
└── import.graphql      # Import + extend Mutation
```

**拆分原则**:
- 根类型 (Query/Mutation/Subscription) 保持空实现
- 各模块使用 `extend type Query { ... }` 扩展根类型
- 相关枚举随实体放入同一文件

**工作流程**:
1. 开发者修改 `internal/api/graphql/schema/*.graphql` 中的相应文件
2. 后端运行 `gqlgen generate` 生成 resolver 接口
3. 前端 TypeScript 编译时自动获得新类型（无需手动步骤）
4. 重大变更时同步更新 `contracts/schema.graphql` 契约文件

## Project Structure

### Documentation (this feature)

```text
specs/007-graphql-migration/
├── plan.md              # This file
├── research.md          # Phase 0 output ✅
├── data-model.md        # Phase 1 output ✅
├── quickstart.md        # Phase 1 output ✅
├── contracts/
│   └── schema.graphql   # Phase 1 output ✅
├── checklists/
│   └── requirements.md  # Requirements checklist
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Backend - GraphQL Layer (新增)
internal/api/graphql/
├── schema/
│   └── schema.graphql       # GraphQL schema 定义
├── generated/
│   └── generated.go         # gqlgen 生成的代码
├── model/
│   └── models_gen.go        # gqlgen 生成的 model
├── resolver/
│   ├── resolver.go          # Resolver 根结构
│   ├── query.resolver.go    # Query resolvers
│   ├── mutation.resolver.go # Mutation resolvers
│   ├── subscription.resolver.go # Subscription resolvers
│   ├── task.resolver.go     # Task 类型 resolvers
│   ├── connection.resolver.go # Connection 类型 resolvers
│   └── job.resolver.go      # Job 类型 resolvers
└── dataloader/
    ├── loaders.go           # Loader 集合和中间件
    ├── connection_loader.go # Connection dataloader
    └── task_loader.go       # Task dataloader

# Backend - 现有代码修改
internal/api/
├── routes.go                # 添加 GraphQL 端点
└── server.go                # 集成 GraphQL handler

# 根目录配置
gqlgen.yml                   # gqlgen 配置文件

# Frontend - GraphQL Client (新增)
web/src/api/graphql/
├── client.ts                # urql client 配置
├── provider.tsx             # GraphQL Provider
└── queries/
    ├── tasks.ts             # Task 查询/变更
    ├── connections.ts       # Connection 查询/变更
    ├── jobs.ts              # Job 查询
    ├── providers.ts         # Provider 查询
    └── files.ts             # File 查询

# Frontend - 现有代码修改
web/src/api/                 # 删除 REST API 调用
web/src/modules/             # 迁移到 GraphQL hooks
```

**Structure Decision**: Web 应用架构，后端新增 `internal/api/graphql/` 目录，前端新增 `web/src/api/graphql/` 目录。现有 REST handlers 在迁移完成后删除。

## Complexity Tracking

> 本次迁移无 Constitution 违规需要说明。

| Aspect | Complexity | Justification |
|--------|------------|---------------|
| gqlgen + Ent 集成 | Medium | 需要手动映射，但避免直接暴露 Ent 实体 |
| Dataloader 实现 | Medium | 需要为每个关联关系实现 loader |
| Subscription 迁移 | Medium | 替换 SSE 为 WebSocket，需重构事件分发 |
| 前端迁移 | High | 所有 API 调用需重写，但类型安全收益大 |

## Phase Outputs

### Phase 0: Research
- ✅ `research.md` - 技术选型和设计决策

### Phase 1: Design & Contracts
- ✅ `data-model.md` - 实体定义和关系
- ✅ `contracts/schema.graphql` - 完整 GraphQL schema
- ✅ `quickstart.md` - 开发环境设置指南

### Phase 2: Tasks (待生成)
- `tasks.md` - 由 `/speckit.tasks` 命令生成

## Migration Strategy

### 阶段 1: 并行运行
1. 实现 GraphQL 端点 (`/api/graphql`)
2. 保留现有 REST 端点
3. 前端逐模块迁移

### 阶段 2: 功能迁移顺序
1. Provider 查询（只读，低风险）
2. Connection 管理（CRUD）
3. Task 管理（CRUD + Run）
4. Job/Log 查询
5. File 浏览
6. Import 功能
7. Subscription（替换 SSE）

### 阶段 3: 清理
1. 删除 REST handlers
2. 删除 REST 路由
3. 更新文档

## Dependencies

### 后端新增依赖

```go
// go.mod
require (
    github.com/99designs/gqlgen v0.17.x
    github.com/vektah/gqlparser/v2 v2.5.x
    github.com/vikstrous/dataloadgen v0.1.x
    github.com/gorilla/websocket v1.5.x
)
```

### 前端新增依赖

```json
// package.json
{
  "dependencies": {
    "urql": "^4.x",
    "@urql/core": "^5.x",
    "@urql/exchange-graphcache": "^7.x",
    "graphql": "^16.x",
    "graphql-ws": "^5.x"
  },
  "devDependencies": {
    "gql.tada": "^1.x",
    "@0no-co/graphqlsp": "^1.x"
  }
}
```

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| 迁移期间功能回归 | Medium | High | 并行运行，逐模块迁移 |
| N+1 性能问题 | High | Medium | Dataloader 必须实现 |
| Subscription 不稳定 | Low | Medium | WebSocket 重连机制 |
| 类型映射错误 | Medium | Low | 编译时类型检查 |

## Next Steps

1. 运行 `/speckit.tasks` 生成任务分解
2. 按 `tasks.md` 中的优先级实现
3. 每个任务完成后更新 `checklists/requirements.md`

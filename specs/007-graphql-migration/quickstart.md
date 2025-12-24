# Quickstart: GraphQL Migration

**Feature Branch**: `007-graphql-migration`  
**Date**: 2024-12-20

## Overview

本指南帮助开发者快速设置 GraphQL 开发环境并理解核心开发工作流程。

## Prerequisites

### 后端
- Go 1.21+
- 现有项目依赖已安装 (`go mod download`)

### 前端
- Node.js 18+
- pnpm 8+
- 现有前端依赖已安装 (`pnpm install`)

## Quick Setup

### 1. 安装后端依赖

```bash
# 进入项目根目录
cd /home/xzzpig/workspaces/golang/rclone-sync

# 安装 gqlgen CLI
go install github.com/99designs/gqlgen@latest

# 添加 GraphQL 依赖
go get github.com/99designs/gqlgen
go get github.com/vektah/gqlparser/v2
go get github.com/vikstrous/dataloadgen
go get github.com/gorilla/websocket
```

### 2. 初始化 gqlgen 配置

```bash
# 创建 gqlgen 配置文件
cat > gqlgen.yml << 'EOF'
schema:
  - internal/api/graphql/schema/*.graphql

exec:
  filename: internal/api/graphql/generated/generated.go
  package: generated

model:
  filename: internal/api/graphql/model/models_gen.go
  package: model

resolver:
  layout: follow-schema
  dir: internal/api/graphql/resolver
  package: resolver

models:
  ID:
    model:
      - github.com/99designs/gqlgen/graphql.UUID
  DateTime:
    model:
      - github.com/99designs/gqlgen/graphql.Time
  JSON:
    model:
      - github.com/99designs/gqlgen/graphql.Map
  BigInt:
    model:
      - github.com/99designs/gqlgen/graphql.Int64
EOF
```

> **注意**: Schema 中使用 `@goField(forceResolver: true)` 的字段会生成独立的 resolver 方法，需要手动实现。

### 3. 创建目录结构

```bash
# 创建 GraphQL 相关目录
mkdir -p internal/api/graphql/{schema,model,resolver,generated,dataloader}
```

### 3.1 Schema 文件拆分

contracts/schema.graphql 作为单一契约文件保存完整 schema。实现时按实体拆分成多个文件，使用 `extend` 语法扩展根类型：

```bash
# 实现目录结构
internal/api/graphql/schema/
├── schema.graphql      # 根类型 (空 Query/Mutation/Subscription) + 指令 + 标量 + 分页通用类型
├── task.graphql        # Task 相关类型 + extend type Query/Mutation
├── connection.graphql  # Connection 相关类型 + extend type Query/Mutation
├── job.graphql         # Job/JobLog 相关类型 + extend type Query/Subscription
├── provider.graphql    # Provider 相关类型 + extend type Query
├── file.graphql        # FileEntry 相关类型 + extend type Query
└── import.graphql      # Import 相关类型 + extend type Mutation
```

**拆分原则：**
- `schema.graphql` 定义空的根类型，各模块使用 `extend` 扩展
- 每个实体文件包含该实体的类型、枚举、输入类型和命名空间类型
- 相关枚举放在对应实体文件中（如 `JobStatus` 放在 `job.graphql`）

**示例 - schema.graphql：**
```graphql
# 根类型（空实现，由各模块 extend 扩展）
type Query
type Mutation
type Subscription

schema {
  query: Query
  mutation: Mutation
  subscription: Subscription
}
```

**示例 - task.graphql：**
```graphql
# Task 相关枚举
enum SyncDirection { UPLOAD DOWNLOAD BIDIRECT }

# Task 类型定义
type Task { ... }
type TaskQuery { ... }
type TaskMutation { ... }

# 扩展根类型
extend type Query {
  task: TaskQuery! @goField(forceResolver: true)
}

extend type Mutation {
  task: TaskMutation! @goField(forceResolver: true)
}
```

gqlgen 配置使用通配符加载所有 schema 文件：
```yaml
# gqlgen.yml
schema:
  - internal/api/graphql/schema/*.graphql
```

### 4. 生成代码

```bash
# 生成 GraphQL 代码
go run github.com/99designs/gqlgen generate
```

### 5. 整合到现有 Gin 服务器

项目使用 Gin 作为 HTTP 框架，需要将 gqlgen 生成的 handler 整合到现有路由中。

#### 5.1 创建 GraphQL Handler

```go
// internal/api/graphql/handler.go
package graphql

import (
	"context"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	
	"github.com/xzzpig/rclone-sync/internal/api/graphql/dataloader"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/generated"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/resolver"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
)

// NewHandler creates a new GraphQL handler with all transports configured.
func NewHandler(deps *resolver.Dependencies) *handler.Server {
	srv := handler.New(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver.New(deps),
	}))

	// 配置缓存
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	// WebSocket 配置（用于 Subscription）
	srv.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 生产环境应限制 origin
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		KeepAlivePingInterval: 10 * time.Second,
	})

	// 启用查询缓存和自动持久化查询
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	// 启用 introspection（开发环境）
	srv.Use(extension.Introspection{})

	return srv
}

// GinHandler wraps the GraphQL handler for Gin compatibility.
func GinHandler(srv *handler.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		srv.ServeHTTP(c.Writer, c.Request)
	}
}

// PlaygroundHandler returns a handler for GraphiQL playground.
func PlaygroundHandler(endpoint string) gin.HandlerFunc {
	h := playground.Handler("GraphQL Playground", endpoint)
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
```

#### 5.2 创建 Dataloader 中间件

```go
// internal/api/graphql/dataloader/middleware.go
package dataloader

import (
	"context"
	"net/http"
	
	"github.com/gin-gonic/gin"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
)

type ctxKey string

const loadersKey ctxKey = "dataloaders"

// Loaders holds all dataloaders for a request.
type Loaders struct {
	ConnectionLoader *ConnectionLoader
	TaskLoader       *TaskLoader
	// 添加更多 loader...
}

// NewLoaders creates a new Loaders instance for the request.
func NewLoaders(client *ent.Client) *Loaders {
	return &Loaders{
		ConnectionLoader: NewConnectionLoader(client),
		TaskLoader:       NewTaskLoader(client),
	}
}

// Middleware injects dataloaders into the request context.
func Middleware(client *ent.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		loaders := NewLoaders(client)
		ctx := context.WithValue(c.Request.Context(), loadersKey, loaders)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// For retrieves the dataloaders from context.
func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}
```

#### 5.3 创建 Resolver 依赖

```go
// internal/api/graphql/resolver/resolver.go
package resolver

import (
	"github.com/xzzpig/rclone-sync/internal/api/graphql/generated"
	"github.com/xzzpig/rclone-sync/internal/api/sse"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"github.com/xzzpig/rclone-sync/internal/core/runner"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// Dependencies holds all dependencies required by resolvers.
type Dependencies struct {
	EntClient   *ent.Client
	SyncEngine  *rclone.SyncEngine
	TaskRunner  *runner.Runner
	JobService  ports.JobService
	Watcher     ports.Watcher
	Scheduler   ports.Scheduler
	Broadcaster *sse.Broadcaster
}

// Resolver is the root resolver.
type Resolver struct {
	deps *Dependencies
}

// New creates a new Resolver with dependencies.
func New(deps *Dependencies) *Resolver {
	return &Resolver{deps: deps}
}

// Query returns the query resolver.
func (r *Resolver) Query() generated.QueryResolver {
	return &queryResolver{r}
}

// Mutation returns the mutation resolver.
func (r *Resolver) Mutation() generated.MutationResolver {
	return &mutationResolver{r}
}

// Subscription returns the subscription resolver.
func (r *Resolver) Subscription() generated.SubscriptionResolver {
	return &subscriptionResolver{r}
}

type queryResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
```

#### 5.4 注册 GraphQL 路由

修改 `internal/api/routes.go`，添加 GraphQL 端点：

```go
// internal/api/routes.go
package api

import (
	// ... existing imports ...
	
	"github.com/xzzpig/rclone-sync/internal/api/graphql"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/dataloader"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/resolver"
)

// RegisterAPIRoutes registers all API routes to the given router group.
func RegisterAPIRoutes(router *gin.RouterGroup, deps RouterDeps) error {
	// ... existing initialization code ...

	// Initialize GraphQL dependencies
	gqlDeps := &resolver.Dependencies{
		EntClient:   deps.Client,
		SyncEngine:  deps.SyncEngine,
		TaskRunner:  deps.TaskRunner,
		JobService:  deps.JobService,
		Watcher:     deps.Watcher,
		Scheduler:   deps.Scheduler,
		Broadcaster: deps.Broadcaster,
	}
	
	// Create GraphQL handler
	gqlHandler := graphql.NewHandler(gqlDeps)
	
	// GraphQL endpoints (with dataloader middleware)
	gqlGroup := router.Group("/graphql")
	gqlGroup.Use(dataloader.Middleware(deps.Client))
	{
		// GraphQL query/mutation endpoint
		gqlGroup.POST("", graphql.GinHandler(gqlHandler))
		gqlGroup.GET("", graphql.GinHandler(gqlHandler)) // For subscriptions upgrade
		
		// GraphiQL Playground (development only)
		if deps.Config.App.Environment == "development" {
			gqlGroup.GET("/playground", graphql.PlaygroundHandler("/api/graphql"))
		}
	}

	// ... existing REST routes (keep for backward compatibility during migration) ...
}
```

#### 5.5 更新 RouterDeps 结构体

确保 `RouterDeps` 包含所有必要的依赖：

```go
// internal/api/routes.go

// RouterDeps contains all dependencies required for setting up API routes.
type RouterDeps struct {
	Client      *ent.Client
	Config      *config.Config
	SyncEngine  *rclone.SyncEngine
	TaskRunner  *runner.Runner
	JobService  ports.JobService
	Watcher     ports.Watcher
	Scheduler   ports.Scheduler
	Broadcaster *sse.Broadcaster
}
```

#### 5.6 更新 Server Setup

确保在 `server.go` 中正确传递依赖：

```go
// internal/api/server.go

func SetupRouter(deps RouterDeps) *gin.Engine {
	// ... existing middleware setup ...

	// API Group
	apiGroup := r.Group("/api")
	{
		if err := RegisterAPIRoutes(apiGroup, deps); err != nil {
			srvLog().Fatal("Failed to register API routes", zap.Error(err))
		}
	}

	// ... rest of setup ...
}
```

### 6. 安装前端依赖

```bash
cd web

# 安装 GraphQL 相关依赖
pnpm add urql @urql/core @urql/exchange-graphcache @urql/exchange-persisted graphql graphql-ws @urql/solid
pnpm add -D gql.tada @0no-co/graphqlsp
```

### 7. 配置 gql.tada

为了支持自定义标量（Scalar）并获得完整的类型推断支持，我们需要创建一个自定义的 `graphql` 函数。

#### 1. 添加 TypeScript 插件配置

```bash
# 添加 TypeScript 插件配置到 tsconfig.json
cat > tsconfig.graphql.json << 'EOF'
{
  "compilerOptions": {
    "plugins": [
      {
        "name": "gql.tada/ts-plugin",
        "schema": "../internal/api/graphql/schema/schema.graphql",
        "tadaOutputLocation": "./src/graphql-env.d.ts"
      }
    ]
  }
}
EOF
```

#### 2. 创建自定义 graphql 函数

创建 `src/api/graphql/graphql.ts` 文件，用于初始化 `gql.tada` 并配置标量映射。

```bash
# 确保目录存在
mkdir -p src/api/graphql

cat > src/api/graphql/graphql.ts << 'EOF'
import { initGraphQLTada } from 'gql.tada';
import type { introspection } from '@/graphql-env.d.ts';

export const graphql = initGraphQLTada<{
  introspection: introspection;
  scalars: {
    // 将 GraphQL Scalar 映射到 TypeScript 类型
    DateTime: string;
    JSON: any;
    BigInt: number;
    ID: string;
  };
}>();

export type { FragmentOf, ResultOf, VariablesOf } from 'gql.tada';
export { readFragment } from 'gql.tada';
EOF
```

**关键点说明：**

1.  **`initGraphQLTada`**: 初始化函数，传入泛型配置。
2.  **`introspection`**: 引用 `gql.tada` CLI 自动生成的类型定义（在 `src/graphql-env.d.ts`）。
3.  **`scalars`**: 定义 GraphQL 自定义标量到 TypeScript 类型的映射。
    *   `DateTime` -> `string`: 时间通常以 ISO 字符串传输。
    *   `BigInt` -> `number`: JS 中通常用 number 表示（注意精度）或使用 bigint。
    *   `JSON` -> `any`: 灵活的 JSON 对象。
4.  **导出辅助类型**: 重新导出 `ResultOf`, `FragmentOf` 等，方便在组件中定义 Props 类型。

**使用示例：**

```typescript
import { graphql, type ResultOf } from '@/api/graphql/graphql';

// 定义 Fragment
const TaskFragment = graphql(`
  fragment TaskItem on Task {
    id
    name
    createdAt # 自动推断为 string (DateTime)
  }
`);

// 定义 Query
const TasksQuery = graphql(`
  query GetTasks {
    tasks {
      ...TaskItem
    }
  }
`, [TaskFragment]); // 注册 Fragment

// 推断类型
type Task = ResultOf<typeof TaskFragment>;
```

## Development Workflow

### 后端开发流程

```
1. 修改 Schema
   └── internal/api/graphql/schema/*.graphql

2. 生成代码
   └── go run github.com/99designs/gqlgen generate

3. 实现 Resolver
   └── internal/api/graphql/resolver/*.go
   
4. 添加 Dataloader（如需要）
   └── internal/api/graphql/dataloader/*.go

5. 运行测试
   └── go test ./internal/api/graphql/...

6. 启动开发服务器
   └── go run ./cmd/cloud-sync serve
```

### 前端开发流程

```
1. 编写 GraphQL 查询
   └── web/src/api/graphql/*.ts

2. 类型自动推断（gql.tada）
   └── TypeScript 语言服务器自动工作

3. 使用 urql hooks
   └── web/src/modules/*/*.tsx

4. 运行开发服务器
   └── pnpm dev
```

## Key Code Patterns

### 后端: Resolver 实现

```go
// internal/api/graphql/resolver/task.resolver.go
package resolver

import (
    "context"
    
    "github.com/xzzpig/rclone-sync/internal/api/graphql/model"
    "github.com/xzzpig/rclone-sync/internal/api/graphql/dataloader"
)

// Tasks is the resolver for the tasks field.
func (r *queryResolver) Tasks(ctx context.Context, pagination *model.PaginationInput) (*model.TaskConnection, error) {
    limit, offset := 20, 0
    if pagination != nil {
        if pagination.Limit != nil {
            limit = *pagination.Limit
        }
        if pagination.Offset != nil {
            offset = *pagination.Offset
        }
    }
    
    // 使用 Ent 查询
    tasks, err := r.entClient.Task.Query().
        Limit(limit).
        Offset(offset).
        All(ctx)
    if err != nil {
        return nil, err
    }
    
    total, err := r.entClient.Task.Query().Count(ctx)
    if err != nil {
        return nil, err
    }
    
    return &model.TaskConnection{
        Items:      convertTasks(tasks),
        TotalCount: total,
        PageInfo: &model.OffsetPageInfo{
            Limit:           limit,
            Offset:          offset,
            HasNextPage:     offset+len(tasks) < total,
            HasPreviousPage: offset > 0,
        },
    }, nil
}

// Connection is the resolver for the connection field on Task.
func (r *taskResolver) Connection(ctx context.Context, obj *model.Task) (*model.Connection, error) {
    // 使用 dataloader 避免 N+1
    return dataloader.For(ctx).ConnectionLoader.Load(ctx, obj.ConnectionID)
}
```

### 后端: Dataloader 实现

```go
// internal/api/graphql/dataloader/connection_loader.go
package dataloader

import (
    "context"
    "time"
    
    "github.com/google/uuid"
    "github.com/vikstrous/dataloadgen"
    "github.com/xzzpig/rclone-sync/internal/api/graphql/model"
    "github.com/xzzpig/rclone-sync/internal/core/ent"
)

type connectionReader struct {
    client *ent.Client
}

func (r *connectionReader) getConnections(ctx context.Context, ids []uuid.UUID) ([]*model.Connection, []error) {
    connections, err := r.client.Connection.Query().
        Where(connection.IDIn(ids...)).
        All(ctx)
    if err != nil {
        errs := make([]error, len(ids))
        for i := range errs {
            errs[i] = err
        }
        return nil, errs
    }
    
    // 按 ID 顺序返回结果
    result := make([]*model.Connection, len(ids))
    connMap := make(map[uuid.UUID]*ent.Connection)
    for _, c := range connections {
        connMap[c.ID] = c
    }
    for i, id := range ids {
        if c, ok := connMap[id]; ok {
            result[i] = convertConnection(c)
        }
    }
    return result, nil
}

func NewConnectionLoader(client *ent.Client) *dataloadgen.Loader[uuid.UUID, *model.Connection] {
    reader := &connectionReader{client: client}
    return dataloadgen.NewLoader(reader.getConnections, dataloadgen.WithWait(2*time.Millisecond))
}
```

### 后端: Subscription 实现

```go
// internal/api/graphql/resolver/subscription.resolver.go
package resolver

import (
    "context"
    
    "github.com/google/uuid"
    "github.com/xzzpig/rclone-sync/internal/api/graphql/model"
)

// JobProgress is the resolver for the jobProgress subscription.
// 支持可选的 taskId 和 connectionId 过滤
func (r *subscriptionResolver) JobProgress(
    ctx context.Context,
    taskID *uuid.UUID,
    connectionID *uuid.UUID,
) (<-chan *model.JobProgressEvent, error) {
    ch := make(chan *model.JobProgressEvent, 10)
    
    // 订阅全局 job 进度更新
    go func() {
        defer close(ch)
        
        sub := r.eventBus.Subscribe("job.progress")
        defer sub.Unsubscribe()
        
        for {
            select {
            case <-ctx.Done():
                return
            case event := <-sub.Channel():
                progress, ok := event.(*model.JobProgressEvent)
                if !ok {
                    continue
                }
                
                // 应用可选过滤
                if taskID != nil && progress.TaskID != *taskID {
                    continue
                }
                if connectionID != nil && progress.ConnectionID != *connectionID {
                    continue
                }
                
                ch <- progress
            }
        }
    }()
    
    return ch, nil
}
```

### 前端: urql Client 设置

```typescript
// web/src/api/graphql/client.ts
import { Client, fetchExchange, subscriptionExchange } from '@urql/core';
import { cacheExchange } from '@urql/exchange-graphcache';
import { persistedExchange } from '@urql/exchange-persisted';
import { createClient as createWSClient } from 'graphql-ws';

const wsClient = createWSClient({
  url: 'ws://localhost:8080/api/graphql',
});

export const client = new Client({
  url: '/api/graphql',
  exchanges: [
    cacheExchange({
      keys: {
        // Pagination & Connections
        TaskConnection: () => null,
        ConnectionConnection: () => null,
        JobConnection: () => null,
        JobLogConnection: () => null,
        OffsetPageInfo: () => null,
        
        // Namespaces (Singletons)
        TaskQuery: () => null,
        ConnectionQuery: () => null,
        JobQuery: () => null,
        LogQuery: () => null,
        ProviderQuery: () => null,
        FileQuery: () => null,
        
        // Value Objects / Results
        TaskSyncOptions: () => null,
        ConnectionQuota: () => null,
        ConnectionTestSuccess: () => null,
        ConnectionTestFailure: () => null,
        ImportParseSuccess: () => null,
        ImportParseError: () => null,
        ImportExecuteResult: () => null,
        ParsedConnection: () => null,
        FileEntry: () => null,
        ProviderOption: () => null,
        OptionExample: () => null,

        // Custom Keys
        Provider: (data) => data.name,
      },
      resolvers: {
        TaskQuery: {
          get: (_, args) => ({ __typename: 'Task', id: args.id }),
        },
        ConnectionQuery: {
          get: (_, args) => ({ __typename: 'Connection', id: args.id }),
        },
        JobQuery: {
          get: (_, args) => ({ __typename: 'Job', id: args.id }),
        },
        ProviderQuery: {
          get: (_, args) => ({ __typename: 'Provider', name: args.name }),
        },
      },
    }),
    // Automatic Persisted Queries - 减少网络传输
    persistedExchange({
      // 使用 GET 请求以利用 CDN 缓存（可选）
      preferGetForPersistedQueries: true,
      // 对 mutation 也启用 APQ（可选）
      enableForMutation: true,
    }),
    fetchExchange,
    subscriptionExchange({
      forwardSubscription: (request) => ({
        subscribe: (sink) => ({
          unsubscribe: wsClient.subscribe(request, sink),
        }),
      }),
    }),
  ],
});
```

**APQ 工作原理:**
1. 客户端首次请求只发送查询的 SHA256 hash
2. 如果服务器缓存中没有该 hash，返回 `PersistedQueryNotFound` 错误
3. urql 自动重试，发送完整查询 + hash
4. 后续请求服务器已缓存，只需发送 hash 即可

**优势:**
- 减少网络传输体积（只发送短 hash 而非完整查询）
- 配合 `preferGetForPersistedQueries: true` 可使用 GET 请求，更好地利用 CDN 缓存
- 服务器缓存失效时自动回退发送完整查询

### 前端: gql.tada 查询

```typescript
// web/src/api/graphql/queries/tasks.ts
import { graphql } from '@/api/graphql/graphql';

// 使用命名空间模式：query { task { list(...) } }
export const TasksQuery = graphql(`
  query Tasks($pagination: PaginationInput) {
    task {
      list(pagination: $pagination) {
        items {
          id
          name
          sourcePath
          remotePath
          direction
          schedule
          realtime
          connection {
            id
            name
            type
          }
          latestJob {
            id
            status
            startTime
          }
        }
        totalCount
        pageInfo {
          limit
          offset
          hasNextPage
        }
      }
    }
  }
`);

// 运行任务：失败时抛出 GraphQL error，不使用 union
export const RunTaskMutation = graphql(`
  mutation RunTask($taskId: ID!) {
    task {
      run(taskId: $taskId) {
        id
        status
        startTime
      }
    }
  }
`);

// 测试连接：预期失败用 union 表示
export const TestConnectionMutation = graphql(`
  mutation TestConnection($id: ID!) {
    connection {
      test(id: $id) {
        ... on ConnectionTestSuccess {
          message
        }
        ... on ConnectionTestFailure {
          error
        }
      }
    }
  }
`);

// 全局订阅作业进度
export const JobProgressSubscription = graphql(`
  subscription JobProgress($taskId: ID, $connectionId: ID) {
    jobProgress(taskId: $taskId, connectionId: $connectionId) {
      jobId
      taskId
      connectionId
      status
      filesTransferred
      bytesTransferred
      startTime
      endTime
      currentFile
      percentage
    }
  }
`);
```

### 前端: SolidJS 组件使用

```typescript
// web/src/modules/tasks/TaskList.tsx
import { createQuery, createMutation } from '@urql/solid';
import { TasksQuery, RunTaskMutation } from '@/api/graphql/queries/tasks';

export function TaskList() {
  const [tasksResult] = createQuery({
    query: TasksQuery,
    variables: { pagination: { limit: 20, offset: 0 } },
  });
  
  const [, runTask] = createMutation(RunTaskMutation);
  
  const handleRunTask = async (taskId: string) => {
    const result = await runTask({ taskId });
    if (result.data?.runTask.__typename === 'Job') {
      // 任务开始成功
    }
  };
  
  return (
    <For each={tasksResult.data?.tasks.items}>
      {(task) => (
        <div>
          <span>{task.name}</span>
          <button onClick={() => handleRunTask(task.id)}>Run</button>
        </div>
      )}
    </For>
  );
}
```

## Testing

### 后端测试

```bash
# 运行 GraphQL resolver 测试
go test ./internal/api/graphql/resolver/... -v

# 运行集成测试
go test ./internal/api/graphql/... -tags=integration -v
```

### 前端测试

```bash
cd web

# 类型检查
pnpm typecheck

# 运行测试
pnpm test
```

## GraphQL Playground

启动开发服务器后，访问 `http://localhost:8080/graphql` 可打开 GraphiQL Playground。

示例查询：

```graphql
# 查询任务列表（命名空间模式）
query {
  task {
    list(pagination: { limit: 10, offset: 0 }) {
      items {
        id
        name
        connection {
          name
          type
        }
      }
      totalCount
    }
  }
}

# 运行任务（命名空间模式，失败时抛出 GraphQL error）
mutation {
  task {
    run(taskId: "uuid-here") {
      id
      status
      startTime
    }
  }
}

# 测试连接（预期失败用 union 表示）
mutation {
  connection {
    test(id: "uuid-here") {
      ... on ConnectionTestSuccess {
        message
      }
      ... on ConnectionTestFailure {
        error
      }
    }
  }
}

# 订阅所有作业进度（全局订阅）
subscription {
  jobProgress {
    jobId
    taskId
    connectionId
    status
    filesTransferred
    bytesTransferred
    percentage
    currentFile
  }
}

# 订阅指定任务的作业进度
subscription {
  jobProgress(taskId: "task-uuid-here") {
    jobId
    status
    filesTransferred
    percentage
  }
}
```

## Troubleshooting

### 常见问题

1. **gqlgen 生成失败**
   - 检查 schema 语法: `go run github.com/99designs/gqlgen validate`
   - 确保所有类型都有对应的 model

2. **gql.tada 类型不更新**
   - 重启 TypeScript 语言服务器
   - 检查 schema 路径配置

3. **Subscription 连接失败**
   - 检查 WebSocket 端点配置
   - 确认后端支持 graphql-ws 协议

4. **N+1 查询问题**
   - 使用 dataloader 批量加载
   - 检查 resolver 是否正确使用 loader

## Next Steps

1. 按照 `tasks.md` 中的任务顺序实现功能
2. 参考 `data-model.md` 了解实体关系
3. 使用 `contracts/schema.graphql` 作为 API 契约

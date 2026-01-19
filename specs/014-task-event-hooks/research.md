# Research: Task Event Hooks

**Feature Branch**: `014-task-event-hooks`  
**Date**: 2026-01-17  
**Status**: Complete

## Executive Summary

本研究解决了实现任务事件钩子功能所需的所有技术决策点。主要研究领域包括：数据模型设计（多态关联）、模板系统实现、命令/HTTP 执行机制、任务执行流程集成点、以及配置系统扩展。

---

## Research Topics

### 1. Hook 实体数据模型设计（多态关联）

**问题**: Hook 需要能够关联到 Task 或 Connection，如何在 Ent ORM 中实现？

**Decision**: 采用「多可选 Edge 模式」（Multiple Optional Edges）

**Rationale**: 
- Ent ORM 不支持原生多态关联（如 Rails 的 `polymorphic`）
- 多可选 Edge 模式是 Ent 社区推荐的标准模式（参见 [ent/ent#1048](https://github.com/ent/ent/issues/1048)）
- 完全类型安全，支持 Ent 的图遍历（Graph Traversal）
- 仅两种关联类型（Task/Connection），不会造成 Schema 臃肿

**Implementation Pattern**:
```go
// Hook Schema
func (Hook) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("task", Task.Type).Ref("hooks").Unique().Field("task_id"),
        edge.From("connection", Connection.Type).Ref("hooks").Unique().Field("connection_id"),
    }
}

// Task Schema - 添加反向 Edge
edge.To("hooks", Hook.Type).Annotations(entsql.OnDelete(entsql.Cascade))

// Connection Schema - 添加反向 Edge  
edge.To("hooks", Hook.Type).Annotations(entsql.OnDelete(entsql.Cascade))
```

**Validation**: 使用 Ent Hooks 确保 `task_id` 和 `connection_id` 互斥（只能设置其一）。

**Alternatives Considered**:
- 接口/联合类型模式：过于复杂，不符合 Ent 设计哲学
- 单独的中间表：增加不必要的复杂性

---

### 2. Hook 执行记录设计

**问题**: 如何存储 Hook 执行记录，以及与 Job 的关系？

**Decision**: 复用现有 `JobLog` 实体，新增 `HOOK` 类型的 `LogAction`

**Rationale**:
- Hook 执行是 Job 生命周期的一部分，复用 JobLog 更自然
- 避免不必要的实体膨胀
- 自动继承现有的 JobLog 清理机制
- UI 中 Job 详情页已有 JobLog 展示逻辑

**现有 JobLog 结构**:
```go
func (JobLog) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("job_id", uuid.UUID{}),
        field.Enum("level").GoType(model.LogLevel("")),     // INFO/WARNING/ERROR
        field.Time("time").Default(time.Now),
        field.String("path").Optional(),                     // 复用: Hook 名称
        field.Enum("what").GoType(model.LogAction("")),      // 新增: HOOK
        field.Int64("size").Optional(),                      // 复用: 执行耗时(ms) 或状态码
    }
}
```

**Implementation**:
1. 新增 `LogAction` 枚举值：`HOOK`
2. 记录格式：
   - `level`: `INFO`（成功）/ `ERROR`（失败）
   - `path`: Hook 标识，格式 `hook:<hook_id>:<event>` 如 `hook:abc123:on_success`
   - `what`: `HOOK`
   - `size`: 执行耗时（毫秒），负值表示 HTTP 状态码或退出码（如 `-200` 表示 HTTP 200, `-1` 表示命令退出码 1）

**示例日志记录**:
```go
// Hook 执行成功
jobQuery.AddJobLog(ctx, jobID, "INFO", "HOOK", "hook:abc123:on_success", 150)  // 150ms

// Hook 执行失败（HTTP 500）
jobQuery.AddJobLog(ctx, jobID, "ERROR", "HOOK", "hook:abc123:on_success HTTP 500: Internal Error", -500)

// 命令执行失败（退出码 1）
jobQuery.AddJobLog(ctx, jobID, "ERROR", "HOOK", "hook:def456:on_failure Exit code 1: command not found", -1)
```

**Alternatives Considered**:
- 创建独立的 HookExecution 实体：增加复杂性，Hook 执行本质上就是 Job 的事件日志
- 嵌入到 Job 的 JSON 字段：不利于查询和过滤

---

### 3. 模板系统实现（Go text/template）

**问题**: 如何安全高效地实现 Hook 的模板渲染？

**Decision**: 使用 Go `text/template`，预编译模板，自定义 FuncMap

**Rationale**:
- `text/template` 适用于非 HTML 输出（JSON、命令行）
- 预编译避免重复解析开销
- FuncMap 提供辅助函数，增强模板能力

**Implementation Pattern**:
```go
// 预定义 FuncMap
var hookFuncMap = template.FuncMap{
    "FormatTime":      func(t time.Time) string { return t.Format(time.RFC3339) },
    "FormatDuration":  func(d time.Duration) string { return d.Round(time.Second).String() },
    "FormatSizeBytes": func(b int64) string { return humanize.Bytes(uint64(b)) },
    "JsonMarshal":     func(v interface{}) string { b, _ := json.Marshal(v); return string(b) },
    "Summary":         generateDefaultSummary,
}

// 模板上下文结构
type HookContext struct {
    Task     *TaskInfo         // 任务信息
    Job      *JobInfo          // 执行信息
    Event    string            // 事件类型: on_start, on_success, on_failure, on_end
    Error    string            // 错误信息（仅失败时有值）
    Duration time.Duration     // 执行耗时
    Stats    *TransferStats    // 传输统计
    Env      map[string]string // 环境变量映射
}

// 渲染模板（带错误处理）
func RenderTemplate(tmplStr string, ctx *HookContext) (string, error) {
    tmpl, err := template.New("hook").Funcs(hookFuncMap).Parse(tmplStr)
    if err != nil {
        return "", fmt.Errorf("template parse error: %w", err)
    }
    
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, ctx); err != nil {
        return "", fmt.Errorf("template execute error: %w", err)
    }
    return buf.String(), nil
}
```

**Best Practices Applied**:
- 先 `Funcs()` 再 `Parse()`
- 使用 `bytes.Buffer` 捕获输出，便于错误处理
- FuncMap 仅用于格式化，避免业务逻辑

**Alternatives Considered**:
- `html/template`：自动转义会破坏 JSON 输出
- 第三方模板引擎（如 pongo2）：增加依赖，Go 标准库足够

---

### 4. 命令执行机制（os/exec）

**问题**: 如何安全地执行用户配置的命令，支持超时和环境变量？

**Decision**: 使用 `exec.CommandContext` 实现超时控制，继承并扩展环境变量

**Rationale**:
- `CommandContext` 是 Go 标准的超时控制方式
- 继承 `os.Environ()` 确保系统环境可用
- 参数化命令天然防御注入攻击

**Implementation Pattern**:
```go
func ExecuteCommand(ctx context.Context, command string, workDir string, timeout time.Duration, envVars map[string]string) (exitCode int, output string, err error) {
    // 创建带超时的 Context
    execCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    // 使用 shell 执行命令（支持管道、重定向等）
    cmd := exec.CommandContext(execCtx, "sh", "-c", command)
    
    // 设置工作目录
    if workDir != "" {
        cmd.Dir = workDir
    }
    
    // 继承环境变量并添加自定义变量
    cmd.Env = os.Environ()
    for k, v := range envVars {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }
    
    // 捕获输出
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    err = cmd.Run()
    
    // 处理超时
    if execCtx.Err() == context.DeadlineExceeded {
        return -1, "", fmt.Errorf("command timed out after %v", timeout)
    }
    
    // 提取退出码
    if err != nil {
        var exitErr *exec.ExitError
        if errors.As(err, &exitErr) {
            return exitErr.ExitCode(), stderr.String(), err
        }
        return -1, "", err
    }
    
    return 0, stdout.String(), nil
}
```

**Environment Variables for Command Hooks**:
```
RCLONE_SYNC_TASK_ID=<uuid>
RCLONE_SYNC_TASK_NAME=<name>
RCLONE_SYNC_JOB_ID=<uuid>
RCLONE_SYNC_EVENT=on_success|on_failure|on_start|on_end
RCLONE_SYNC_STATUS=SUCCESS|FAILED|CANCELLED
RCLONE_SYNC_ERROR=<error message>
RCLONE_SYNC_FILES_TRANSFERRED=<count>
RCLONE_SYNC_BYTES_TRANSFERRED=<bytes>
RCLONE_SYNC_DURATION_SECONDS=<seconds>
```

**Alternatives Considered**:
- 直接执行命令（不通过 shell）：限制用户使用管道等 shell 特性
- 使用第三方库：标准库功能足够

---

### 5. HTTP 请求机制（net/http）

**问题**: 如何可靠地发送 HTTP 请求，支持自定义头部和超时？

**Decision**: 使用自定义 `http.Client`，配置合理的超时和连接池

**Rationale**:
- 避免使用无超时的 `http.DefaultClient`
- 连接池复用提高性能
- 支持 Context 取消

**Implementation Pattern**:
```go
// 全局复用的 HTTP 客户端
var hookHTTPClient = &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}

func ExecuteHTTPHook(ctx context.Context, method, url string, headers map[string]string, body string) (statusCode int, err error) {
    var bodyReader io.Reader
    if body != "" {
        bodyReader = strings.NewReader(body)
    }
    
    req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
    if err != nil {
        return 0, fmt.Errorf("failed to create request: %w", err)
    }
    
    // 设置默认 Content-Type
    if body != "" && req.Header.Get("Content-Type") == "" {
        req.Header.Set("Content-Type", "application/json")
    }
    
    // 应用自定义头部
    for k, v := range headers {
        req.Header.Set(k, v)
    }
    
    resp, err := hookHTTPClient.Do(req)
    if err != nil {
        return 0, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()
    
    // 忽略响应体，仅记录状态码
    io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
    
    return resp.StatusCode, nil
}
```

**Alternatives Considered**:
- 使用第三方 HTTP 库（如 resty）：增加依赖，标准库足够
- 每次请求创建新客户端：性能差

---

### 6. 任务执行流程集成点

**问题**: 在哪里注入 Hook 触发逻辑？

**Decision**: 在 `SyncEngine.RunTask` 方法中注入 Hook 执行逻辑

**Rationale**:
- `SyncEngine.RunTask` 是任务执行的核心方法
- 已有清晰的状态变更点（Running/Success/Failed/Cancelled）
- 现有的 `broadcastJobUpdate` 机制可作为参考

**Integration Points** (in `internal/rclone/sync.go`):

| Event | Location | Line (approx) |
|-------|----------|---------------|
| `on_start` | Job 创建后，状态变更为 Running 后 | ~157 |
| `on_success` | 状态变更为 Success 后 | ~308 |
| `on_failure` | 状态变更为 Failed 后 | ~276 |
| `on_cancelled` | 状态变更为 Cancelled 后 | ~251 |
| `on_end` | 所有状态变更后，方法返回前 | ~325 |

**Implementation Approach**:
```go
// 在 SyncEngine 中添加 HookExecutor 依赖
type SyncEngine struct {
    // ... existing fields
    hookExecutor ports.HookExecutor
}

// 在 RunTask 中的触发点
func (e *SyncEngine) RunTask(ctx context.Context, task *ent.Task, trigger model.JobTrigger) error {
    // ... job creation ...
    
    // Hook: on_start
    e.hookExecutor.Execute(ctx, task, jobEntity, model.HookEventOnStart, nil)
    
    // ... sync execution ...
    
    // Hook: on_success / on_failure / on_cancelled
    if syncErr != nil {
        if errors.Is(ctx.Err(), context.Canceled) {
            e.hookExecutor.Execute(ctx, task, jobEntity, model.HookEventOnCancelled, syncErr)
        } else {
            e.hookExecutor.Execute(ctx, task, jobEntity, model.HookEventOnFailure, syncErr)
        }
    } else {
        e.hookExecutor.Execute(ctx, task, jobEntity, model.HookEventOnSuccess, nil)
    }
    
    // Hook: on_end (always, regardless of result)
    e.hookExecutor.Execute(ctx, task, jobEntity, model.HookEventOnEnd, syncErr)
    
    return syncErr
}
```

**Alternatives Considered**:
- 使用 Ent Hook 监听 Job 状态变更：无法获取完整的任务上下文
- 使用 EventBus 订阅：增加复杂性，且需要处理异步问题

---

### 7. 全局开关与配置系统

**问题**: 如何在配置中添加 Hook 功能的全局开关？

**Decision**: 在 `config.toml` 中添加 `[app.hook]` 配置节

**Rationale**:
- 符合现有配置结构（如 `[app.job]`, `[app.sync]`）
- 使用 Viper 自动支持环境变量覆盖
- 重启生效符合预期（配置类变更）

**Configuration Schema**:
```toml
[app.hook]
# 是否启用 Hook 功能（全局开关）
# 禁用时：所有 hooks 不触发，Web UI 隐藏相关界面
# Default: true
enabled = true

# Hook 执行的默认超时时间（秒）
# 适用于 HTTP 和命令类型的 Hook
# Default: 30
default_timeout = 30
```

**Go Config Struct**:
```go
type Config struct {
    // ... existing fields
    App struct {
        // ... existing fields
        Hook struct {
            Enabled        bool `mapstructure:"enabled"`
            DefaultTimeout int  `mapstructure:"default_timeout"`
        } `mapstructure:"hook"`
    } `mapstructure:"app"`
}
```

**Environment Variables**:
- `RCLONESYNC_APP_HOOK_ENABLED=false`
- `RCLONESYNC_APP_HOOK_DEFAULT_TIMEOUT=60`

**Alternatives Considered**:
- 数据库存储配置：需要额外的管理界面，不符合 "重启生效" 的需求
- 单独的配置文件：增加复杂性

---

### 8. GraphQL Schema 扩展

**问题**: 如何定义 Hook 相关的 GraphQL 类型？

**Decision**: 创建新的 `hook.graphql` schema 文件，遵循现有命名空间模式

**Rationale**:
- 符合现有的模块化 Schema 组织（task.graphql, job.graphql, connection.graphql）
- 命名空间查询/变更模式（如 `query.hook`, `mutation.hook`）保持一致性

**Schema Design** (`internal/api/graphql/schema/hook.graphql`):
```graphql
# ENUMS
enum HookEvent {
    ON_START
    ON_SUCCESS
    ON_FAILURE
    ON_END
}

enum HookType {
    HTTP
    COMMAND
}

enum HookOnError {
    IGNORE
    CANCEL
    FATAL
}

# TYPES
type Hook {
    id: ID!
    enabled: Boolean!
    priority: Int
    event: HookEvent!
    type: HookType!
    onError: HookOnError!
    httpConfig: HookHTTPConfig
    commandConfig: HookCommandConfig
    task: Task
    connection: Connection
    createdAt: DateTime!
    updatedAt: DateTime!
}

type HookHTTPConfig {
    url: String!
    method: String!
    headers: [KeyValuePair!]
    body: String
}

type HookCommandConfig {
    command: String!
    workDir: String
    timeout: Int!
}

# EXTEND EXISTING TYPES
extend type Task {
    hooks: [Hook!]! @goField(forceResolver: true)
}

extend type Connection {
    hooks: [Hook!]! @goField(forceResolver: true)
}

# Hook 执行记录：复用现有 JobLog，新增 LogAction 枚举值
extend enum LogAction {
    HOOK  # Hook 执行记录
}

# NAMESPACED OPERATIONS
type HookQuery {
    list(taskId: ID, connectionId: ID): [Hook!]! @goField(forceResolver: true)
    get(id: ID!): Hook @goField(forceResolver: true)
}

type HookMutation {
    create(input: CreateHookInput!): Hook! @goField(forceResolver: true)
    update(id: ID!, input: UpdateHookInput!): Hook! @goField(forceResolver: true)
    delete(id: ID!): Hook! @goField(forceResolver: true)
}

extend type Query {
    hook: HookQuery! @goField(forceResolver: true)
}

extend type Mutation {
    hook: HookMutation! @goField(forceResolver: true)
}
```

---

### 9. 错误处理行为实现

**问题**: 如何实现 `ON_ERROR_IGNORE`, `ON_ERROR_CANCEL`, `ON_ERROR_FATAL` 三种错误处理行为？

**Decision**: 使用自定义 error 类型区分不同行为，通过 `errors.As` 判断

**Implementation Pattern**:
```go
// 自定义错误类型
type HookCancelError struct {
    HookID   uuid.UUID
    HookName string
    Cause    error
}

func (e *HookCancelError) Error() string {
    return fmt.Sprintf("hook %s requested cancel: %v", e.HookName, e.Cause)
}

func (e *HookCancelError) Unwrap() error { return e.Cause }

type HookFatalError struct {
    HookID   uuid.UUID
    HookName string
    Cause    error
}

func (e *HookFatalError) Error() string {
    return fmt.Sprintf("hook %s failed fatally: %v", e.HookName, e.Cause)
}

func (e *HookFatalError) Unwrap() error { return e.Cause }

// 执行逻辑
func (e *HookExecutor) ExecuteHooks(ctx context.Context, hooks []*ent.Hook, hookCtx *HookContext) error {
    for _, hook := range hooks {
        if !hook.Enabled {
            continue
        }
        
        err := e.executeOne(ctx, hook, hookCtx)
        if err != nil {
            e.logHookError(hook, err)
            
            switch hook.OnError {
            case model.HookOnErrorCancel:
                return &HookCancelError{HookID: hook.ID, HookName: hook.Name, Cause: err}
            case model.HookOnErrorFatal:
                return &HookFatalError{HookID: hook.ID, HookName: hook.Name, Cause: err}
            // ON_ERROR_IGNORE: continue to next hook
            }
        }
    }
    return nil
}
```

**on_start Event Special Handling**:
对于 `on_start` 事件，当 Hook 失败且配置为 `CANCEL` 或 `FATAL` 时，需要阻止任务开始执行：
```go
// 在 SyncEngine.RunTask 中
err := e.hookExecutor.ExecuteHooks(ctx, hooks, hookCtx)
if err != nil {
    var cancelErr *hook.HookCancelError
    var fatalErr *hook.HookFatalError
    
    if errors.As(err, &cancelErr) {
        // 标记 Job 为 CANCELLED，不执行同步
        e.jobQuery.UpdateJobStatus(ctx, jobEntity.ID, model.JobStatusCancelled, err.Error())
        return nil
    }
    if errors.As(err, &fatalErr) {
        // 标记 Job 为 FAILED，触发 on_failure 和 on_end
        return err
    }
}
```

---

### 10. 环境变量引用支持

**问题**: 如何支持在 Hook 配置中引用环境变量（如 `${API_KEY}`），避免敏感值直接存入数据库？

**Decision**: 在模板渲染时通过 `.Env` 变量提供环境变量访问

**Rationale**:
- 符合 spec 中的 FR-004 要求
- 模板语法天然支持：`{{.Env.API_KEY}}`
- 敏感值保留在环境变量中，不存入数据库

**Implementation**:
```go
// HookContext 包含环境变量映射
type HookContext struct {
    // ... other fields
    Env map[string]string
}

// 构建上下文时填充 Env
func buildHookContext(task *ent.Task, job *ent.Job, event string, err error) *HookContext {
    env := make(map[string]string)
    for _, e := range os.Environ() {
        parts := strings.SplitN(e, "=", 2)
        if len(parts) == 2 {
            env[parts[0]] = parts[1]
        }
    }
    
    return &HookContext{
        // ... other fields
        Env: env,
    }
}
```

**Usage Example** (in hook body template):
```json
{
  "token": "{{.Env.SLACK_TOKEN}}",
  "message": "Task {{.Task.Name}} completed"
}
```

---

## Summary of Technical Decisions

| Topic | Decision | Key Rationale |
|-------|----------|---------------|
| 多态关联 | 多可选 Edge 模式 | Ent 推荐，类型安全 |
| 执行记录 | 复用 JobLog，新增 HOOK action | 避免实体膨胀，自动继承清理机制 |
| 模板系统 | Go text/template + FuncMap | 标准库足够，预编译优化 |
| 命令执行 | exec.CommandContext + shell | 超时控制，支持管道 |
| HTTP 请求 | 自定义 http.Client | 连接池复用，合理超时 |
| 集成点 | SyncEngine.RunTask 注入 | 清晰的状态变更点 |
| 全局配置 | app.hook 配置节 | 符合现有结构 |
| GraphQL | hook.graphql 命名空间 | 模块化，一致性 |
| 错误处理 | 执行时根据 OnError 决策 | 灵活的错误响应 |
| 环境变量 | 模板 .Env 变量 | 敏感值不入库 |

---

## Next Steps

Phase 0 研究完成。所有 "NEEDS CLARIFICATION" 项已解决。可以进入 Phase 1：设计与合约阶段。

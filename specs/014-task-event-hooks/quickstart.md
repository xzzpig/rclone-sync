# Quickstart: Task Event Hooks

**Feature Branch**: `014-task-event-hooks`  
**Date**: 2026-01-17

本文档提供 Task Event Hooks 功能的快速实现指南，帮助开发者快速理解和实现核心功能。

---

## 1. 功能概述

Task Event Hooks 允许用户在任务执行的关键时间点（开始、成功、失败、结束）自动执行 HTTP 请求或 Shell 命令，实现与外部系统的集成。

### 核心特性

- **4 种事件类型**: `on_start`, `on_success`, `on_failure`, `on_end`
- **2 种 Hook 类型**: HTTP 请求、Shell 命令
- **模板支持**: Go text/template 语法，动态渲染内容
- **错误处理**: IGNORE（默认）、CANCEL、FATAL 三种策略
- **多态关联**: Hook 可关联到 Task 或 Connection

---

## 2. 数据库设置

### 2.1 生成迁移文件

```bash
# 生成 Ent schema
go generate ./internal/core/db/ent

# 使用 Atlas 生成迁移差异
atlas migrate diff add_hooks_table \
  --dir "file://internal/core/db/migrations" \
  --to "ent://internal/core/db/schema" \
  --dev-url "sqlite://file?mode=memory"
```

### 2.2 关键 Schema 文件

创建 `internal/core/db/schema/hook.go`（参见 [data-model.md](./data-model.md)）。

---

## 3. 核心实现

### 3.1 Hook 执行器

```go
// internal/core/hook/executor.go
package hook

import (
    "context"
    "github.com/xzzpig/rclone-sync/internal/core/db/ent"
    "github.com/xzzpig/rclone-sync/internal/api/graphql/model"
)

type Executor interface {
    // Execute 执行指定事件的所有 hooks
    Execute(ctx context.Context, task *ent.Task, job *ent.Job, event model.HookEvent, syncErr error) error
}

type executor struct {
    client       *ent.Client
    httpClient   *http.Client
    globalConfig *config.HookConfig
}

func NewExecutor(client *ent.Client, cfg *config.HookConfig) Executor {
    return &executor{
        client:       client,
        httpClient:   newHTTPClient(cfg.DefaultTimeout),
        globalConfig: cfg,
    }
}
```

### 3.2 模板渲染

```go
// internal/core/hook/template.go
package hook

import (
    "bytes"
    "text/template"
    "time"
    
    "github.com/dustin/go-humanize"
)

var hookFuncMap = template.FuncMap{
    "FormatTime":      func(t time.Time) string { return t.Format(time.RFC3339) },
    "FormatDuration":  func(d time.Duration) string { return d.Round(time.Second).String() },
    "FormatSizeBytes": func(b int64) string { return humanize.Bytes(uint64(b)) },
    "JsonMarshal":     func(v interface{}) string { 
        b, _ := json.Marshal(v)
        return string(b) 
    },
    "Summary":         generateSummary,
}

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

### 3.3 集成到 SyncEngine

```go
// internal/rclone/sync.go
func (e *SyncEngine) RunTask(ctx context.Context, task *ent.Task, trigger model.JobTrigger) error {
    // 创建 Job
    job, err := e.jobQuery.CreateJob(ctx, task.ID, trigger)
    if err != nil {
        return err
    }
    
    // Hook: on_start
    hookCtx := e.buildHookContext(task, job, model.HookEventOnStart, nil)
    if err := e.hookExecutor.Execute(ctx, task, job, model.HookEventOnStart, nil); err != nil {
        // 处理 CANCEL/FATAL 错误
        if isCancelError(err) {
            e.jobQuery.UpdateJobStatus(ctx, job.ID, model.JobStatusCancelled, err.Error())
            goto OnEnd
        }
        if isFatalError(err) {
            e.jobQuery.UpdateJobStatus(ctx, job.ID, model.JobStatusFailed, err.Error())
            goto OnFailure
        }
    }
    
    // 执行同步...
    syncErr := e.doSync(ctx, task, job)
    
    if syncErr != nil {
        if errors.Is(ctx.Err(), context.Canceled) {
            e.jobQuery.UpdateJobStatus(ctx, job.ID, model.JobStatusCancelled, syncErr.Error())
        } else {
            e.jobQuery.UpdateJobStatus(ctx, job.ID, model.JobStatusFailed, syncErr.Error())
            goto OnFailure
        }
    } else {
        e.jobQuery.UpdateJobStatus(ctx, job.ID, model.JobStatusSuccess, "")
        // Hook: on_success
        e.hookExecutor.Execute(ctx, task, job, model.HookEventOnSuccess, nil)
    }
    goto OnEnd

OnFailure:
    // Hook: on_failure
    e.hookExecutor.Execute(ctx, task, job, model.HookEventOnFailure, syncErr)

OnEnd:
    // Hook: on_end (always)
    e.hookExecutor.Execute(ctx, task, job, model.HookEventOnEnd, syncErr)
    
    return syncErr
}
```

---

## 4. GraphQL API

### 4.1 创建 Hook

```graphql
mutation CreateHook {
  hook {
    create(input: {
      taskId: "task-uuid-here"
      event: ON_SUCCESS
      type: HTTP
      enabled: true
      priority: 10
      onError: IGNORE
      config: {
        url: "https://hooks.slack.com/services/xxx"
        method: "POST"
        headers: { "Content-Type": "application/json" }
        body: "{\"text\": \"Task {{.Task.Name}} completed!\"}"
      }
    }) {
      id
      enabled
      event
      type
      config {
        url
        method
      }
    }
  }
}
```

### 4.2 查询 Task 的 Hooks

```graphql
query GetTaskHooks {
  task {
    get(id: "task-uuid-here") {
      id
      name
      hooks {
        id
        enabled
        event
        type
        priority
        onError
        config {
          # HTTP Hook 字段
          url
          method
          headers
          body
          # Command Hook 字段
          command
          workDir
          timeout
        }
      }
    }
  }
}
```

### 4.3 查询 Hook 列表

```graphql
query ListHooks {
  hook {
    list(taskId: "task-uuid-here", event: ON_SUCCESS) {
      id
      enabled
      event
      type
      priority
    }
  }
}
```

---

## 5. 配置示例

### 5.1 全局配置 (config.toml)

```toml
[app.hook]
# 是否启用 Hook 功能（全局开关）
enabled = true

# Hook 执行的默认超时时间（秒）
default_timeout = 30
```

### 5.2 常见 Hook 模板

#### Slack 通知

```json
{
  "channel": "#sync-alerts",
  "username": "RcloneSync Bot",
  "icon_emoji": "{{if eq .Event \"on_success\"}}:white_check_mark:{{else}}:x:{{end}}",
  "text": "Task *{{.Task.Name}}* {{.Event}}"
}
```

#### 钉钉通知

```json
{
  "msgtype": "markdown",
  "markdown": {
    "title": "同步{{if eq .Event \"on_success\"}}成功{{else}}失败{{end}}",
    "text": "### {{.Task.Name}}\n- 状态: {{.Job.Status}}\n- 耗时: {{FormatDuration .Duration}}\n- 文件: {{.Stats.FilesTransferred}}个\n{{if .Error}}- 错误: {{.Error}}{{end}}"
  }
}
```

#### 企业微信通知

```json
{
  "msgtype": "text",
  "text": {
    "content": "任务 {{.Task.Name}} {{if eq .Event \"on_success\"}}完成{{else}}失败{{end}}\n耗时: {{FormatDuration .Duration}}"
  }
}
```

#### 命令 Hook - 发送邮件

```bash
echo "Task {{.Task.Name}} {{.Event}}" | mail -s "Sync Notification" admin@example.com
```

#### 命令 Hook - 触发下游任务

```bash
curl -X POST http://localhost:8080/api/trigger-backup \
  -H "Content-Type: application/json" \
  -d '{"taskId": "{{.Task.ID}}", "status": "{{.Job.Status}}"}'
```

---

## 6. 环境变量

### 命令 Hook 可用的环境变量

| Variable | Description |
|----------|-------------|
| `RCLONE_SYNC_TASK_ID` | 任务 UUID |
| `RCLONE_SYNC_TASK_NAME` | 任务名称 |
| `RCLONE_SYNC_JOB_ID` | Job UUID |
| `RCLONE_SYNC_EVENT` | 事件类型 |
| `RCLONE_SYNC_STATUS` | Job 状态 |
| `RCLONE_SYNC_ERROR` | 错误信息 |
| `RCLONE_SYNC_FILES_TRANSFERRED` | 传输文件数 |
| `RCLONE_SYNC_BYTES_TRANSFERRED` | 传输字节数 |
| `RCLONE_SYNC_DURATION_SECONDS` | 执行耗时（秒） |

### 模板中引用环境变量

```
Authorization: Bearer {{.Env.API_TOKEN}}
```

---

## 7. 测试要点

### 单元测试

```go
// internal/core/hook/executor_test.go
func TestExecutor_HTTPHook(t *testing.T) {
    // 1. 设置 mock HTTP server
    // 2. 创建 Hook 配置
    // 3. 执行 Hook
    // 4. 验证请求内容
}

func TestExecutor_CommandHook(t *testing.T) {
    // 1. 创建临时脚本
    // 2. 配置命令 Hook
    // 3. 执行 Hook
    // 4. 验证环境变量和退出码
}

func TestExecutor_TemplateRendering(t *testing.T) {
    // 1. 准备 HookContext
    // 2. 渲染各种模板
    // 3. 验证输出
}
```

### 集成测试

```go
// internal/rclone/sync_test.go
func TestSyncEngine_HookExecution(t *testing.T) {
    // 1. 创建 Task 和 Hook
    // 2. 执行任务
    // 3. 验证 Hook 被触发
    // 4. 验证 JobLog 记录
}
```

---

## 8. 实现检查清单

- [ ] 创建 `internal/core/db/schema/hook.go`
- [ ] 生成数据库迁移文件
- [ ] 添加 `internal/api/graphql/schema/hook.graphql`
- [ ] 运行 `go generate ./...` 生成代码
- [ ] 实现 `internal/core/hook/executor.go`
- [ ] 实现 `internal/core/hook/template.go`
- [ ] 修改 `internal/core/config/config.go` 添加 Hook 配置
- [ ] 修改 `internal/rclone/sync.go` 注入 Hook 触发点
- [ ] 实现 GraphQL resolvers
- [ ] 添加 i18n 翻译文件
- [ ] 编写单元测试
- [ ] 编写集成测试
- [ ] 实现前端 UI 组件

---

## 9. 相关文档

- [Feature Spec](./spec.md) - 功能规格说明
- [Research](./research.md) - 技术研究
- [Data Model](./data-model.md) - 数据模型详细定义
- [GraphQL Contract](./contracts/hook.graphql) - API 契约

# Quickstart: UI Detail Improvements

**Feature Branch**: `008-ui-detail-improvements`  
**Created**: 2024-12-24

---

## Prerequisites

- Go 1.21+
- Node.js 18+ with pnpm
- 已有可用的开发环境

---

## Quick Verification

### 1. 验证配额信息扩展

```bash
# 启动后端
go run ./cmd/cloud-sync serve

# 在另一个终端，使用 GraphQL Playground 或 curl 查询
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query { connection { list { items { id name quota { total used free trashed other objects } } } } }"
  }'
```

**预期结果**: 
- `quota.trashed`, `quota.other`, `quota.objects` 字段存在
- 不支持的存储后端这些字段返回 `null`

### 2. 验证传输进度详情（列表形式）

```bash
# 在 GraphQL Playground 订阅 transferProgress
subscription {
  transferProgress(connectionId: "YOUR_CONNECTION_ID") {
    jobId
    taskId
    connectionId
    transfers {
      name       # 文件名称（含路径）
      size       # 文件总大小（字节）
      bytes      # 已传输大小（字节）
    }
  }
}

# 触发一个同步任务（包含大文件效果更明显）
```

**预期结果**:
- 同步进行中时，`transfers` 数组以**列表形式**返回所有正在传输的文件
- 每个列表项包含:
  - `name`: 文件名称（含路径），如 `documents/report.pdf`
  - `size`: 文件总大小（字节）
  - `bytes`: 已传输大小（字节）
- 前端计算百分比: `percentage = bytes / size * 100`
- 前端以人类可读格式显示大小，如 `45 MB / 128 MB (35%)`

### 3. 验证日志清理配置

```toml
# config.toml
[log]
level = "info"
max_logs_per_connection = 100  # 设置较小值便于测试
cleanup_schedule = "* * * * *" # 每分钟执行一次（仅测试用）
```

```bash
# 启动后端
go run ./cmd/cloud-sync serve

# 查看日志输出，应该看到清理任务执行
# 2024-12-24T10:00:00 INFO Running log cleanup task...
```

---

### 4. 验证自动删除无活动作业

```toml
# config.toml
[job]
auto_delete_empty_jobs = true  # 启用自动删除
```

```bash
# 启动后端
go run ./cmd/cloud-sync serve

# 创建一个测试任务，源和目标目录内容完全相同
# 执行同步任务（由于内容相同，不会产生任何传输）

# 查看数据库，验证无活动作业是否被自动删除
sqlite3 app_data/cloud-sync.db "SELECT id, status, files_transferred, bytes_transferred FROM jobs ORDER BY created_at DESC LIMIT 5;"

# 预期结果：
# - 无活动的成功作业（files_transferred=0, bytes_transferred=0）不在数据库中
# - 有活动的作业或失败的作业仍保留
```

**预期结果**:
- 当 `auto_delete_empty_jobs = true` 时，无活动的成功作业自动删除
- 当 `auto_delete_empty_jobs = false`（默认）时，所有作业保留
- 失败的作业即使无活动也会保留

---

### 5. 验证作业状态信息展示（User Story 8）

```bash
# 在 GraphQL Playground 订阅 jobProgress
subscription {
  jobProgress(connectionId: "YOUR_CONNECTION_ID") {
    jobId
    taskId
    connectionId
    status
    filesTransferred
    bytesTransferred
    filesTotal
    bytesTotal
    filesDeleted    # 新增：删除的文件数
    errorCount      # 新增：错误数
    startTime
  }
}

# 触发一个同步任务（包含删除操作或可能产生错误的场景）
```

**预期结果**:
- `filesDeleted`: 实时显示同步过程中删除的文件数
- `errorCount`: 实时显示同步过程中发生的错误数

```bash
# 验证已完成作业的持久化数据
sqlite3 app_data/cloud-sync.db "SELECT id, status, files_deleted, error_count FROM jobs ORDER BY created_at DESC LIMIT 5;"
```

**预期结果**:
- `files_deleted` 和 `error_count` 列有正确的值
- 这些值在作业完成时从 rclone 的 `accounting.StatsInfo` 获取并持久化

---

## Development Workflow

### Step 1: 修改 GraphQL Schema

```bash
# 编辑 schema 文件
vim internal/api/graphql/schema/connection.graphql
vim internal/api/graphql/schema/job.graphql

# 重新生成代码
go generate ./...
```

### Step 2: 实现 Resolver

```bash
# 编辑 resolver 文件
vim internal/api/graphql/resolver/connection.resolvers.go
vim internal/api/graphql/resolver/job.resolvers.go

# 编写测试
vim internal/api/graphql/resolver/connection_test.go
vim internal/api/graphql/resolver/job_test.go

# 运行测试
go test ./internal/api/graphql/resolver/... -v
```

### Step 3: 修改 Sync Engine

```bash
# 编辑 sync.go 获取传输详情
vim internal/rclone/sync.go

# 运行测试
go test ./internal/rclone/... -v
```

### Step 4: 添加日志清理服务

```bash
# 创建新服务
vim internal/core/services/log_cleanup_service.go
vim internal/core/services/log_cleanup_service_test.go

# 修改配置
vim internal/core/config/config.go

# 修改调度器
vim internal/core/scheduler/scheduler.go

# 运行测试
go test ./internal/core/services/... -v
go test ./internal/core/scheduler/... -v
```

### Step 5: 前端开发

```bash
cd web

# 更新 GraphQL 查询
vim src/api/graphql/queries/connections.ts
vim src/api/graphql/queries/subscriptions.ts

# 重新生成类型
pnpm graphql-codegen

# 更新组件
vim src/modules/connections/views/Overview.tsx
vim src/modules/connections/views/History.tsx

# 添加翻译
vim project.inlang/messages/en.json
vim project.inlang/messages/zh-CN.json

# 启动开发服务器
pnpm dev
```

---

## Testing Checklist

### Backend Tests

```bash
# 全量测试
go test ./... -v

# 特定模块
go test ./internal/api/graphql/resolver/... -v
go test ./internal/rclone/... -v
go test ./internal/core/services/... -v
go test ./internal/core/scheduler/... -v
```

### Frontend Manual Tests

1. **配额信息展示**
   - [ ] 打开连接的 Overview 页面
   - [ ] 验证 Storage Usage 卡片显示完整信息
   - [ ] 验证不支持的字段显示 N/A 或隐藏

2. **传输进度展示**
   - [ ] 启动一个同步任务
   - [ ] 验证 Overview 页面显示活跃传输
   - [ ] 验证 History 页面显示正在传输的文件详情
   - [ ] 验证进度实时更新

3. **日志清理**
   - [ ] 配置 `max_logs_per_connection = 50`
   - [ ] 生成超过 50 条日志
   - [ ] 等待清理任务执行
   - [ ] 验证日志数量降到 50

4. **作业状态信息展示 (User Story 8)**
   - [ ] 启动一个同步任务（包含删除操作或可能产生错误的场景）
   - [ ] 验证 History 页面表格新增了"删除数"和"错误数"两列
   - [ ] 验证删除数为 0 时显示 "0"（而非隐藏）
   - [ ] 验证错误数为 0 时显示 "0"（而非隐藏）
   - [ ] 验证错误数 > 0 时以红色徽章形式醒目显示
   - [ ] 验证作业进行中时删除数和错误数实时更新

---

## Configuration Reference

```toml
# config.toml

[log]
# 日志级别: debug, info, warn, error
level = "info"

# 每个连接保留的最大日志条数
# 0 = 无限制（不清理）
# 默认: 1000
max_logs_per_connection = 1000

# 日志清理任务的 cron 表达式
# 格式: 分 时 日 月 周
# 默认: "0 * * * *" (每小时整点)
cleanup_schedule = "0 * * * *"

[job]
# 作业完成后，如果没有实际传输活动，是否自动删除该作业记录
# "无活动"定义: filesTransferred = 0 且 bytesTransferred = 0 且 filesDeleted = 0 且 errorCount = 0 且 status = SUCCESS
# 失败的作业或有删除/错误的作业即使无传输也会保留
# 默认: false（保留所有作业记录）
auto_delete_empty_jobs = false
```

---

## Common Issues

### Q: 配额信息返回 null

**原因**: 存储后端不支持 About 接口  
**解决**: 这是正常行为，前端应优雅处理

### Q: currentTransfers 总是空数组

**原因**: 传输太快，polling 间隔内已完成  
**解决**: 使用大文件测试，或降低网络速度

### Q: 日志清理不生效

**原因**: `max_logs_per_connection = 0` 表示无限制  
**解决**: 设置大于 0 的值

---

## Related Files

| 类型 | 文件 |
|------|------|
| Spec | `specs/008-ui-detail-improvements/spec.md` |
| Plan | `specs/008-ui-detail-improvements/plan.md` |
| Research | `specs/008-ui-detail-improvements/research.md` |
| Data Model | `specs/008-ui-detail-improvements/data-model.md` |
| Contract | `specs/008-ui-detail-improvements/contracts/schema.graphql` |

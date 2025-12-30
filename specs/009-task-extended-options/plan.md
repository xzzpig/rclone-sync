# Implementation Plan: Task 扩展选项配置

**Branch**: `009-task-extended-options` | **Date**: 2025-12-28 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/009-task-extended-options/spec.md`

## Summary

为同步任务添加扩展选项配置功能，包括：
1. **文件过滤器** - 使用 rclone filter 语法配置包含/排除规则
2. **保留删除文件** - 单向同步时不删除目标端多余文件
3. **并行传输数量** - 配置同步时的并发传输数量（1-64）

技术实现：使用 rclone 原生的 `filter` 包进行规则解析和应用，通过 `fs.AddConfig` 注入配置到 context。

## Technical Context

**Language/Version**: Go 1.21+, TypeScript 5.x
**Primary Dependencies**: rclone (fs/filter, fs/config), ent (ORM), gqlgen (GraphQL), SolidJS, urql, ShadcnUI
**Storage**: SQLite (Task.options JSON 字段)
**Testing**: go test, vitest (前端)
**Target Platform**: Linux/Windows/macOS 服务器, 现代浏览器
**Project Type**: Web application (Go backend + SolidJS frontend)
**Performance Goals**: 过滤器验证 < 10ms
**Constraints**: 规则数量建议 < 100 条, transfers 范围 1-64
**Scale/Scope**: 单用户/小团队使用

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| 无新数据库表 | ✅ PASS | 使用现有 Task.options JSON 字段 |
| 无新第三方依赖 | ✅ PASS | 使用 rclone 现有包 |
| 复用现有 UI 组件 | ✅ PASS | FileBrowser, Tab, Switch 等 |
| GraphQL Schema 扩展 | ✅ PASS | 扩展现有类型，无破坏性变更 |

## Project Structure

### Documentation (this feature)

```text
specs/009-task-extended-options/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── schema.graphql
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output (to be created)
```

### Source Code (repository root)

```text
# Backend (Go)
internal/
├── api/graphql/
│   ├── schema/
│   │   ├── task.graphql      # 扩展 TaskSyncOptions
│   │   └── file.graphql      # 扩展 file.remote 查询参数
│   └── resolver/
│       ├── task.resolvers.go # 任务解析器
│       └── file.resolvers.go # 文件解析器（扩展支持过滤器预览）
├── core/
│   ├── config/
│   │   └── config.go         # 添加 Sync.Transfers 配置
│   └── services/
│       └── task_service.go   # 任务服务扩展
└── rclone/
    ├── filter_validator.go   # 新增：过滤器验证
    ├── connection.go         # 扩展：ListRemoteDir 支持过滤器预览
    └── sync.go               # 扩展：应用过滤器/noDelete/transfers

# Frontend (SolidJS/TypeScript)
web/src/
├── api/graphql/queries/
│   ├── tasks.ts              # 任务查询扩展
│   └── files.ts              # 文件查询扩展
├── components/common/
│   └── FileBrowser.tsx       # 修改：支持显示文件图标（根据 isDir 和扩展名区分）
├── modules/connections/
│   ├── components/
│   │   ├── FilterRulesEditor.tsx   # 新增：过滤器规则编辑器
│   │   └── FilterPreviewPanel.tsx  # 新增：过滤器预览面板
│   └── views/
│       └── Tasks.tsx         # 任务设置页面扩展
└── project.inlang/messages/
    ├── en.json               # 英文翻译
    └── zh-CN.json            # 中文翻译
```

**Structure Decision**: Web application 结构，后端 Go + 前端 SolidJS，遵循现有项目架构。

## Implementation Phases

### Phase 1: Backend - Configuration & Validation

1. **配置扩展** (`internal/core/config/config.go`)
   - 添加 `Sync.Transfers` 配置项（默认 4）

2. **过滤器验证** (`internal/rclone/filter_validator.go`)
   - 创建 `ValidateFilterRules(rules []string) error`
   - 使用 rclone filter 包验证规则语法

3. **GraphQL Schema** (`internal/api/graphql/schema/`)
   - 扩展 `TaskSyncOptions` 类型
   - 扩展 `TaskSyncOptionsInput` 输入
   - 扩展 `file.remote` 查询参数

### Phase 2: Backend - Sync Engine

1. **Storage Service** (`internal/rclone/connection.go`)
   - 扩展 `ListRemoteDir` 添加 Filters/IncludeFiles 参数
   - 使用 `filter.ReplaceConfig` 应用过滤器

2. **Sync Engine** (`internal/rclone/sync.go`)
   - 扩展 `SyncOptions` 添加 Filters/NoDelete/Transfers 字段
   - **transfers 三层回退逻辑**: 任务级 → 全局配置 → 默认值 4
   - 使用 `filter.ReplaceConfig` 注入过滤器
   - 使用 `fs.AddConfig` 设置 transfers
   - 使用 `CopyDir` 实现 noDelete 功能

### Phase 3: Backend - Services & Resolvers

1. **Task Service** (`internal/core/services/task_service.go`)
   - 添加 `validateSyncOptions` 验证方法
   - 更新 CreateTask/UpdateTask 调用验证

2. **Resolvers** (`internal/api/graphql/resolver/`)
   - 更新 task.resolvers.go
   - 更新 file.resolvers.go

### Phase 4: Frontend

1. **Components**
   - 创建 `FilterRulesEditor` 组件
   - 创建 `FilterPreviewPanel` 组件

2. **Task Settings**
   - 添加 "过滤器" Tab
   - 添加 "保留删除文件" 开关
   - 添加 "并行传输数量" 输入

3. **i18n**
   - 添加英文/中文翻译

## Complexity Tracking

无 Constitution Check 违规，无需记录。

## Test Files

| 文件 | 变更类型 |
|------|----------|
| `internal/rclone/filter_validator_test.go` | 新建 |
| `internal/rclone/sync_test.go` | 修改 |
| `internal/rclone/connection_test.go` | 修改 |
| `internal/api/graphql/resolver/file_test.go` | 修改 |
| `internal/core/services/task_service_test.go` | 修改 |

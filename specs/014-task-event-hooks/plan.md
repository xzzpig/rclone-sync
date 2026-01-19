# Implementation Plan: Task Event Hooks

**Branch**: `014-task-event-hooks` | **Date**: 2026-01-17 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/014-task-event-hooks/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

实现任务事件钩子机制，允许用户在任务执行的关键时间点（开始、成功、失败、结束）通过 HTTP 请求或本地命令与外部系统集成。技术方案采用多可选 Edge 模式实现 Hook 与 Task/Connection 的多态关联，使用 Go text/template 渲染动态内容，通过 exec.CommandContext 和自定义 http.Client 实现可靠的执行机制。

## Technical Context

**Language/Version**: Go 1.25+ (Backend), TypeScript 5.x (Frontend)  
**Primary Dependencies**: Gin, gqlgen, Ent ORM, SolidJS, urql, Tailwind CSS  
**Storage**: SQLite with Ent ORM, golang-migrate for migrations  
**Testing**: Go testing (unit + integration), vitest (frontend)  
**Target Platform**: Linux/macOS/Windows server, modern browsers  
**Project Type**: Web application (Go backend + SolidJS frontend)  
**Performance Goals**: Hook 在任务状态变更后 5 秒内触发，95%+ 执行记录正确存储  
**Constraints**: Hook 执行不阻塞主任务流程（除 on_start 的 CANCEL/FATAL），默认超时 30 秒  
**Scale/Scope**: 单任务支持任意数量 Hook，执行记录随 Job 保留/清理

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Rclone-First Architecture | ✅ PASS | Hook 机制不涉及 rclone 同步逻辑，仅在同步完成后触发 |
| II. Web-First Interface | ✅ PASS | Hook 配置通过 Web UI 完成 (FR-008) |
| III. Test-Driven Development | ⚠️ REQUIRED | Backend 必须先写测试，前端测试可选 |
| IV. Independent User Stories | ✅ PASS | 各 User Story 可独立实现和测试 |
| V. Observability and Reliability | ✅ PASS | Hook 执行有日志记录 (FR-007)，失败不影响主任务 |
| VI. Modern Component Architecture | ✅ PASS | 前端使用 SolidJS + Tailwind CSS |
| VII. Accessibility and UX Standards | ⚠️ REQUIRED | Hook 配置 UI 需遵循 WCAG 2.1 AA |
| VIII. Performance and Optimistic UI | ✅ PASS | Hook 配置变更使用乐观更新 |
| IX. Internationalization (i18n) | ⚠️ REQUIRED | 新增 UI 文本需外部化到翻译文件 |
| X. Schema-First API Contract | ✅ PASS | 新增 hook.graphql schema 定义 |
| XI. Database Index Strategy | ⚠️ REQUIRED | Hook 和 HookExecution 表需添加适当索引 |

## Project Structure

### Documentation (this feature)

```text
specs/014-task-event-hooks/
├── plan.md              # This file
├── research.md          # Phase 0 output ✅
├── data-model.md        # Phase 1 output ✅
├── quickstart.md        # Phase 1 output ✅
├── contracts/           # Phase 1 output ✅
│   └── hook.graphql     # GraphQL API contract
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
# Backend (Go)
internal/
├── core/
│   ├── db/
│   │   ├── schema/
│   │   │   └── hook.go              # NEW: Hook entity schema
│   │   └── migrations/              # NEW: Migration files
│   ├── config/
│   │   └── config.go                # MODIFY: Add app.hook config
│   └── hook/                        # NEW: Hook execution package
│       ├── executor.go              # Hook execution logic
│       ├── template.go              # Template rendering
│       └── executor_test.go         # Unit tests
├── rclone/
│   └── sync.go                      # MODIFY: Inject hook triggers
├── api/graphql/
│   ├── schema/
│   │   └── hook.graphql             # NEW: GraphQL schema
│   │   └── job.graphql              # MODIFY: Add HOOK to LogAction enum
│   └── resolver/
│       └── hook.resolvers.go        # NEW: GraphQL resolvers

# Frontend (SolidJS)
web/src/
├── modules/connections/
│   ├── components/
│   │   ├── HookForm.tsx             # NEW: Hook configuration form
│   │   └── HookList.tsx             # NEW: Hook list component
│   └── views/
│       └── Settings.tsx             # MODIFY: Add hook config section
├── api/graphql/
│   └── operations/
│       └── hook.ts                  # NEW: GraphQL operations
└── locales/
    ├── en/
    │   └── hook.json                # NEW: English translations
    └── zh-CN/
        └── hook.json                # NEW: Chinese translations
```

**Structure Decision**: Web application 结构，遵循现有的 internal/ (backend) + web/src/ (frontend) 分离模式。新增 `internal/core/hook/` 包封装 Hook 执行逻辑，与现有的 `internal/core/runner/` 和 `internal/rclone/` 集成。

## Constitution Check (Post-Design)

*Re-evaluated after Phase 1 design completion.*

| Principle | Status | Implementation Notes |
|-----------|--------|----------------------|
| I. Rclone-First Architecture | ✅ PASS | Hook 在 SyncEngine 层触发，不干扰 rclone 同步逻辑 |
| II. Web-First Interface | ✅ PASS | Hook 配置通过 GraphQL API + Web UI 完成 |
| III. Test-Driven Development | ✅ ADDRESSED | `quickstart.md` 包含测试要点，实现时需先写测试 |
| IV. Independent User Stories | ✅ PASS | 各 User Story 可独立实现（HTTP Hook, Command Hook, UI）|
| V. Observability and Reliability | ✅ PASS | Hook 执行记录复用 JobLog，失败不影响主任务状态 |
| VI. Modern Component Architecture | ✅ PASS | 前端组件遵循 Atomic Design |
| VII. Accessibility and UX Standards | ✅ ADDRESSED | UI 实现时需遵循 WCAG 2.1 AA |
| VIII. Performance and Optimistic UI | ✅ PASS | Hook CRUD 操作使用乐观更新 |
| IX. Internationalization (i18n) | ✅ ADDRESSED | `quickstart.md` 标注需创建 locales 文件 |
| X. Schema-First API Contract | ✅ PASS | `contracts/hook.graphql` 已定义完整 API |
| XI. Database Index Strategy | ✅ PASS | `data-model.md` 定义了 5 个必要索引 |

**Post-Design Conclusion**: 所有 Constitution 原则均已满足或在设计中明确标注实现要求。

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

无违规需要说明。所有设计决策均符合 Constitution 原则。

## Phase 1 Artifacts Summary

| Artifact | Path | Status |
|----------|------|--------|
| Data Model | `specs/014-task-event-hooks/data-model.md` | ✅ Complete |
| GraphQL Contract | `specs/014-task-event-hooks/contracts/hook.graphql` | ✅ Complete |
| Quickstart Guide | `specs/014-task-event-hooks/quickstart.md` | ✅ Complete |
| Agent Context | `AGENTS.md` | ✅ Updated |

## Next Steps

Phase 1 完成。可执行 `/speckit.tasks` 生成 `tasks.md` 进入 Phase 2 任务分解。

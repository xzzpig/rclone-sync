# Implementation Plan: Rclone 连接配置数据库存储

**Branch**: `004-rclone-config-db` | **Date**: 2025-12-15 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-rclone-config-db/spec.md`

## Summary

将 rclone 连接配置从传统的 rclone.conf 文件迁移到 SQLite 数据库存储，提供更好的安全性（敏感信息加密）、可管理性（CRUD 操作）和一致性。采用直接从数据库配置创建 fs.Fs 的简化方案（复用现有 TestRemote 模式），通过自定义 DBConfigMapper 实现令牌刷新自动同步。

**包装类型兼容方案**: 通过实现 config.Storage 接口，利用 rclone 的按需加载机制自动处理包装类型（alias、crypt、compress、combine 等）的依赖关系，无需预加载。

## Technical Context

**Language/Version**: Go 1.25  
**Primary Dependencies**: Gin v1.11, rclone v1.72.0 (as library), Ent v0.14.5, go-i18n v2.6.0  
**Storage**: SQLite with Ent ORM (existing infrastructure)  
**Testing**: Go standard testing + testify v1.11  
**Target Platform**: Linux server (cross-platform support)  
**Project Type**: web - Go backend + SolidJS frontend  
**Performance Goals**: 配置读取延迟增加不超过 500ms, 连接创建 < 30s  
**Constraints**: 敏感信息 100% 加密存储, 加密密钥从配置/环境变量获取  
**Scale/Scope**: 支持 50+ 种 rclone 提供商, 单用户应用

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle                                 | Status      | Notes                                                |
| ----------------------------------------- | ----------- | ---------------------------------------------------- |
| I. Rclone-First Architecture              | ✅ PASS     | 继续使用 rclone 库进行所有同步操作，仅改变配置存储层 |
| II. Web-First Interface                   | ✅ PASS     | 所有连接管理操作通过 Web UI 进行                     |
| III. Test-Driven Development              | ✅ REQUIRED | 所有新功能需先编写测试：ConfigProvider、加密、API    |
| IV. Independent User Stories              | ✅ PASS     | 7 个用户故事可独立实现和测试                         |
| V. Observability and Reliability          | ✅ PASS     | 连接状态监控、错误日志记录                           |
| VI. Modern Component Architecture         | ✅ PASS     | 导入向导使用 SolidJS 组件实现                        |
| VII. Accessibility and UX Standards       | ✅ REQUIRED | 多步导入向导需符合 WCAG 2.1 AA                       |
| VIII. Performance and Optimistic UI       | ✅ PASS     | 配置读取性能符合要求                                 |
| IX. Internationalization (i18n) Standards | ✅ REQUIRED | 所有新增文本使用 go-i18n/Paraglide                   |

## Project Structure

### Documentation (this feature)

```text
specs/004-rclone-config-db/
├── plan.md              # This file
├── research.md          # Phase 0 output - rclone ConfigProvider research
├── data-model.md        # Phase 1 output - Connection entity design
├── quickstart.md        # Phase 1 output - Implementation quickstart
├── contracts/           # Phase 1 output - OpenAPI spec for connection API
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Backend (Go)
internal/
├── core/
│   ├── config/
│   │   └── config.go           # Add encryption key config
│   ├── db/
│   │   └── schema/
│   │       └── connection.go   # NEW: Connection entity schema
│   ├── ent/                    # Generated Ent code (auto-generated)
│   ├── services/
│   │   └── connection_service.go  # NEW: Connection CRUD + status service
│   └── crypto/                 # NEW: Encryption utilities
│       ├── crypto.go           # AES-256-GCM encryption
│       └── crypto_test.go
├── rclone/
│   ├── connection.go           # MODIFY: Add NewFsFromConnection, DBConfigMapper
│   ├── sync.go                 # MODIFY: Use NewFsFromConnection
│   └── parser.go               # NEW: rclone.conf parser for import
└── api/
    └── handlers/
        ├── remote.go           # MODIFY: Use ConnectionService
        └── import.go           # NEW: Import wizard endpoints

# Frontend (SolidJS)
web/src/
├── api/
│   └── connections.ts          # MODIFY: Add import API calls
├── modules/
│   └── connections/
│       ├── components/
│       │   ├── ImportWizard/           # NEW: Multi-step import wizard
│       │   │   ├── ImportWizard.tsx
│       │   │   ├── Step1Input.tsx
│       │   │   ├── Step2Preview.tsx
│       │   │   └── Step3Confirm.tsx
│       │   └── ConnectionStatusBadge.tsx  # NEW: Status indicator
│       └── views/
│           └── Overview.tsx    # MODIFY: Add status display

# Tests
internal/core/services/connection_service_test.go  # NEW
internal/rclone/connection_test.go                 # MODIFY: Add fs.Fs creation tests
internal/rclone/parser_test.go                     # NEW
internal/core/crypto/crypto_test.go                # NEW
internal/api/handlers/remote_test.go               # MODIFY: Expand tests
internal/api/handlers/import_test.go               # NEW
```

**Structure Decision**: 使用现有的 web 项目结构（Go backend + SolidJS frontend）。采用简化方案：直接从数据库配置创建 fs.Fs（复用 TestRemote 模式），通过 DBConfigMapper 实现令牌刷新自动同步，无需复杂的配置文件同步机制。

## Complexity Tracking

> 无 Constitution 违规需要说明

| Violation | Why Needed | Simpler Alternative Rejected Because |
| --------- | ---------- | ------------------------------------ |
| N/A       | N/A        | N/A                                  |

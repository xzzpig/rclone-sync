# Implementation Plan: Rclone Fs Cache Optimization

**Branch**: `010-rclone-fs-cache` | **Date**: 2025-12-30 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/010-rclone-fs-cache/spec.md`

## Summary

将项目中使用 `fs.NewFs` 创建 rclone Fs 的调用改为使用 `cache.Get` 来复用缓存的 Fs 实例，减少重复创建连接的开销，提升目录浏览、存储空间查询和同步操作的响应速度。主要修改集中在 `internal/rclone` 包的 `remote.go`、`about.go`、`sync.go` 文件，同时需要在连接更新和删除时通过 `cache.ClearConfig` 主动失效缓存。

## Technical Context

**Language/Version**: Go 1.24+  
**Primary Dependencies**: rclone v1.72.1 (as Go library), Gin, gqlgen  
**Storage**: SQLite with Ent ORM  
**Testing**: Go testing (`go test`), 项目已有单元测试和集成测试  
**Target Platform**: Linux server (Docker container)  
**Project Type**: web (backend Go + frontend SolidJS)  
**Performance Goals**: 重复目录浏览响应时间减少 50%+，存储空间查询响应时间减少 30%+  
**Constraints**: 不增加新的可观测性需求，依赖 rclone 内置缓存管理机制  
**Scale/Scope**: 内部重构，影响 3 个源文件的 fs.NewFs 调用

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Rclone-First Architecture | ✅ PASS | 使用 rclone 内置的 cache 包，符合 rclone 优先原则 |
| II. Web-First Interface | ✅ N/A | 本功能为后端内部优化，不涉及 UI 变更 |
| III. Test-Driven Development | ✅ PASS | 需为修改的函数添加/更新单元测试和集成测试 |
| IV. Independent User Stories | ✅ PASS | 缓存优化独立于其他功能，可单独实现和测试 |
| V. Observability and Reliability | ✅ PASS | 保持现有日志级别，不增加额外可观测性（按规范要求） |
| VI-VIII. Frontend Principles | ✅ N/A | 本功能不涉及前端变更 |
| IX. Internationalization | ✅ N/A | 本功能不涉及新的用户可见文本 |
| X. Schema-First API Contract | ✅ N/A | 本功能不涉及 API 变更 |

**Gate Result**: ✅ PASS - 所有相关原则均满足

## Project Structure

### Documentation (this feature)

```text
specs/010-rclone-fs-cache/
├── plan.md              # This file
├── research.md          # Phase 0 output - rclone cache API 研究
├── data-model.md        # Phase 1 output - 不涉及数据模型变更
├── quickstart.md        # Phase 1 output - 实现指南
└── checklists/
    └── requirements.md  # 需求检查清单
```

### Source Code (repository root)

```text
internal/rclone/
├── remote.go            # ListRemoteDir - 需改用 cache.Get
├── about.go             # GetRemoteQuota - 需改用 cache.Get
├── sync.go              # RunTask - 需改用 cache.Get (同步操作)
├── cache_helper.go      # 已有 IsConnectionLoaded 使用 cache.Get
├── storage.go           # 已有 cache.ClearConfig 调用
├── remote_test.go       # 需更新测试
├── about_test.go        # 需更新测试
└── sync_test.go         # 需更新测试

internal/api/graphql/resolver/
└── connection.resolvers.go  # Update/Delete 时需添加 cache.ClearConfig 调用
```

**Structure Decision**: 使用现有的 web application 结构，修改集中在 `internal/rclone` 包

## Complexity Tracking

> 无 Constitution 违规需要记录

本功能为简单的 API 替换重构：
- 将 `fs.NewFs` 替换为 `cache.Get`
- 在连接更新/删除时添加 `cache.ClearConfig` 调用
- 保持原有错误处理逻辑（不回退，直接返回错误）

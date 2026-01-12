# Implementation Plan: ChangeNotify 缓存加速同步

**Branch**: `012-rclone-change-notify-cache` | **Date**: 2026-01-08 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/012-rclone-change-notify-cache/spec.md`

## Summary

通过 rclone 的 ChangeNotify 机制和独立 SQLite 元数据缓存，优化大目录同步性能。对于支持变更通知的远程存储（如 OneDrive、Google Drive），可以将 10 万+ 文件目录的同步时间从 5 分钟降至 2 分钟以内。

核心技术方案：
- 实现自定义 `metacache` 后端，包装远程 Fs 提供缓存层
- 通过 DBStorage 透明注入虚拟连接配置（`-cache` 后缀命名约定）
- 使用 rclone fs/cache 的 Pin 机制确保 Fs 常驻内存
- 采用惰性校验 + TTL 策略保证最终一致性

命名与注入规则：当连接 `cache.enabled=true` 时，内部使用 `<connection>-cache:` 形式注入 metacache 配置；关闭时使用原始 remote。

## Technical Context

**Language/Version**: Go 1.21+  
**Primary Dependencies**: github.com/rclone/rclone v1.72.1, entgo.io/ent, 99designs/gqlgen  
**Storage**: SQLite (github.com/mattn/go-sqlite3) - 主数据库 + 独立缓存数据库  
**Testing**: go test, testify  
**Target Platform**: Linux server (Docker), 跨平台支持  
**Project Type**: web (Go backend + SolidJS frontend)  
**Performance Goals**: 10万+ 文件目录无变更时重复同步 < 2分钟  
**Constraints**: ChangeNotify 轮询间隔 >= 10s  
**Scale/Scope**: 单连接支持 100万+ 文件缓存

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| 新增外部依赖 | ✅ PASS | 无新增依赖，使用现有 github.com/mattn/go-sqlite3 |
| 数据库 Schema 变更 | ✅ PASS | 仅新增 Connection.options JSON 字段 |
| API 兼容性 | ✅ PASS | 新增 GraphQL 字段，向后兼容 |
| 性能影响 | ✅ PASS | 仅优化，无性能退化风险 |

## Project Structure

### Documentation (this feature)

```text
specs/012-rclone-change-notify-cache/
├── plan.md              # 本文件
├── research.md          # Phase 0 研究输出 ✅
├── data-model.md        # Phase 1 数据模型 ✅
├── quickstart.md        # Phase 1 开发指南 ✅
├── contracts/           # Phase 1 API 合约
│   └── schema.graphql   # GraphQL schema 扩展 ✅
└── tasks.md             # Phase 2 任务分解（待创建）
```

### Source Code (repository root)

```text
internal/
├── rclone/
│   ├── backend/
│   │   └── metacache/       # [新增] MetaCache 后端
│   │       ├── metacache.go     # 后端注册和 Fs 实现
│   │       ├── cache_store.go   # SQLite 缓存存储
│   │       └── cache_entry.go   # 缓存条目类型
│   ├── pin_manager.go       # [新增] Fs Pin 管理
│   └── storage.go           # [修改] DBStorage 透明注入
├── api/graphql/
│   ├── model/
│   │   └── connection_options.go  # [新增] ConnectionOptions 类型
│   └── schema/
│       └── connection.graphql     # [修改] Connection schema 扩展
├── core/
│   ├── db/schema/
│   │   └── connection.go    # [修改] 添加 options 字段
│   └── ports/
│       └── connection.go    # [修改] ConnectionService 接口扩展
app_data/
└── cache/                   # 缓存数据库存储目录
    └── <connection_id>.db   # 每个连接的缓存文件
```


**Structure Decision**: 采用 web 应用结构（Go backend + SolidJS frontend）。后端实现位于 `internal/rclone/backend/metacache/`，生命周期管理位于 `internal/rclone/pin_manager.go`，遵循项目现有的包组织方式。

## Complexity Tracking

无 Constitution 违规，无需额外说明。

---

## Phase 1 设计产物

| 产物 | 状态 | 路径 |
|------|------|------|
| data-model.md | ✅ 完成 | [data-model.md](./data-model.md) |
| contracts/schema.graphql | ✅ 完成 | [contracts/schema.graphql](./contracts/schema.graphql) |
| quickstart.md | ✅ 完成 | [quickstart.md](./quickstart.md) |

## 下一步

运行 `/speckit.tasks` 命令将本计划分解为具体的实现任务。


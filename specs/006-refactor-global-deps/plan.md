# Implementation Plan: Refactoring Global Variable Dependencies

**Branch**: `006-refactor-global-deps` | **Date**: 2025-12-19 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-refactor-global-deps/spec.md`

## Summary

Refactor the three global variables in the system (`db.Client`, `config.Cfg`, `logger.L`) to improve code testability and maintainability. Adopt a manual dependency injection approach: pass `db` and `config` via constructor parameters; for the `logger`, retain the singleton pattern but provide `Get()`/`Named()` getter methods instead of direct access to the global variable. The refactoring will be carried out in stages, prioritizing `db.Client` with the least usage (2 instances), followed by `config.Cfg` (5 instances), and finally `logger.L` (19+ instances).

## Technical Context

**Language/Version**: Go (latest stable)
**Primary Dependencies**: 
- Gin (Web framework)
- Ent (ORM)
- Zap (Logging library)
- Viper (Configuration management)
**Storage**: SQLite with Ent ORM
**Testing**: Go testing + testify
**Target Platform**: Linux server / Docker
**Project Type**: web (Go backend + SolidJS frontend)
**Performance Goals**: Maintain existing performance with no additional overhead
**Constraints**: 
- Do not introduce external DI frameworks
- Refactoring does not change any business logic
- Maintain all existing tests passing
**Scale/Scope**: 
- db.Client: 2 usages
- config.Cfg: 5 usages
- logger.L: 19+ usages (distributed across api, core/services, rclone, scheduler, runner, watcher, db modules)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Check (Phase 0)

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Rclone-First Architecture | ✅ PASS | Refactoring does not involve rclone core logic |
| II. Web-First Interface | ✅ PASS | Refactoring does not involve the user interface |
| III. Test-Driven Development | ✅ PASS | Refactoring will improve testability, keeping tests passing at each commit |
| IV. Independent User Stories | ✅ PASS | Refactoring does not affect functional independence |
| V. Observability and Reliability | ✅ PASS | Logger refactoring will maintain structured logging capabilities |
| VI. Modern Component Architecture | N/A | Backend refactoring, does not involve the frontend |
| VII. Accessibility and UX Standards | N/A | Backend refactoring, does not involve the frontend |
| VIII. Performance and Optimistic UI | N/A | Backend refactoring, does not involve the frontend |
| IX. Internationalization Standards | ✅ PASS | Refactoring does not affect i18n functionality |

**Pre-Design Gate Status**: ✅ PASS - No violations

### Post-Design Check (Phase 1)

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Rclone-First Architecture | ✅ PASS | Design maintains rclone as the sync engine architecture |
| II. Web-First Interface | ✅ PASS | API interfaces remain unchanged, only internal refactoring |
| III. Test-Driven Development | ✅ PASS | Design includes test helper functions to improve testability |
| IV. Independent User Stories | ✅ PASS | Dependency injection enhances module independence |
| V. Observability and Reliability | ✅ PASS | Named Logger design enhances log observability |
| VI. Modern Component Architecture | N/A | Backend refactoring, does not involve the frontend |
| VII. Accessibility and UX Standards | N/A | Backend refactoring, does not involve the frontend |
| VIII. Performance and Optimistic UI | N/A | Backend refactoring, does not involve the frontend |
| IX. Internationalization Standards | ✅ PASS | Refactoring does not affect i18n functionality |

**Post-Design Gate Status**: ✅ PASS - Design complies with all applicable principles

## Project Structure

### Documentation (this feature)

```text
specs/006-refactor-global-deps/
├── plan.md              # This file
├── research.md          # Phase 0 output - Dependency injection best practices research
├── data-model.md        # Phase 1 output - Refactored module structure
├── quickstart.md        # Phase 1 output - Quick start guide
├── contracts/           # Phase 1 output - N/A (Internal refactoring, no API changes)
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
# Affected Core Modules
internal/
├── core/
│   ├── db/
│   │   ├── db.go           # Remove Client global variable, change to return *ent.Client
│   │   └── migrate.go      # Remove config.Cfg dependency, accept parameters
│   ├── config/
│   │   └── config.go       # Remove Cfg global variable, InitConfig returns Config
│   └── logger/
│       └── logger.go       # Retain singleton, add Get()/Named() methods
├── api/
│   ├── routes.go           # Accept db client and config as parameters
│   └── server.go           # Accept config as parameter
├── core/services/
│   └── job_service.go      # Use logger.Named()
├── core/scheduler/
│   └── scheduler.go        # Use logger.Named()
├── core/runner/
│   └── runner.go           # Use logger.Named()
├── core/watcher/
│   ├── watcher.go          # Use logger.Named()
│   └── recursive.go        # Use logger.Named()
├── rclone/
│   └── sync.go             # Use logger.Named()
└── api/sse/
    └── broadcaster.go      # Use logger.Named()

cmd/
└── cloud-sync/
    ├── main.go             # Application entry point, responsible for creating and assembling all dependencies
    ├── root.go             # Configuration initialization
    └── serve.go            # Service startup, dependency assembly
```

**Structure Decision**: Maintain the existing directory structure, only modifying the internal implementation of each module. Major changes are concentrated in the `cmd/cloud-sync/` entry point for dependency assembly, and each consuming module is changed to accept dependency injection.

## Complexity Tracking

> No entry required - No violations in constitution check.

# Implementation Plan: Rclone Cloud Sync Manager

**Branch**: `001-cloud-sync` | **Date**: 2025-12-04 | **Spec**: [Link](../spec.md)
**Input**: Feature specification from `/specs/001-cloud-sync/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

The goal is to build a "Synology Cloud Sync"-like application on top of `rclone`.
The system will provide a Web UI to manage rclone remotes and define sync tasks.
It must support Real-time sync (via file watching), Scheduled sync (cron-like), and Bidirectional sync (using `rclone bisync` logic).
Technical approach involves a Go backend directly importing `rclone` packages (as a library) and a Web frontend.

## Technical Context

**Language/Version**: Go (latest stable) for backend, HTML/JS (SolidJS) for frontend.
**Primary Dependencies**: 
- `rclone` (as Go library) - Core sync engine.
- Web Framework: `gin` (`github.com/gin-gonic/gin`).
- UI Framework: SolidJS + SolidUI.
- File Watcher (e.g., `fsnotify`) - For real-time sync.
- Scheduler (e.g., `robfig/cron`) - For scheduled tasks.
- CLI/Config: `cobra` + `viper`.
- ORM: `ent` (SQLite).
- Logging: `zap` (Integrated with Ent & Rclone).
**Tool Management**: Go tool dependencies (tools.go pattern) for `ent`, `golangci-lint`, etc.
**Storage**:
- `rclone.conf` - For remote configs.
- SQLite - For task definitions and job history.
**Testing**: Go standard `testing` package.
**Target Platform**: Linux (primary), cross-platform compatible.
**Project Type**: Single binary web application (backend + embedded frontend).
**Performance Goals**: Minimal overhead over rclone; responsiveness UI.
**Constraints**: Must handle long-running sync jobs reliable; restart resilience.
**Scale/Scope**: Personal/SOHO use; dozens of tasks, thousands of files.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

1. **Principle 1**: ✅ Rclone-First Architecture - All sync operations through rclone library
2. **Principle 2**: ✅ Web-First Interface - Complete functionality through Web UI
3. **Principle 3**: ✅ Test-Driven Development - Tests written before implementation
4. **Principle 4**: ✅ Independent User Stories - Each story independently implementable
5. **Principle 5**: ✅ Observability and Reliability - Structured logging and real-time updates

## Project Structure

### Documentation (this feature)

```text
specs/001-cloud-sync/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/
└── cloud-sync/          # Main entry point

internal/
├── api/                 # REST API handlers
├── core/                # Core logic (Task manager, Scheduler)
├── rclone/              # Rclone wrapper/integration
├── ui/                  # Embedded frontend assets
└── utils/               # Shared utilities

pkg/                     # Public libraries (if any)

web/                     # Frontend source code
├── src/
├── public/
└── package.json
```

**Structure Decision**: Standard Go project layout with a separate `web` directory for the frontend, which will be built and embedded into the Go binary.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |

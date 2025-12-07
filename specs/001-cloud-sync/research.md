# Research: Rclone Cloud Sync Manager

**Feature**: `001-cloud-sync`
**Date**: 2025-12-04

## 1. Rclone Integration Strategy

**Decision**: Use `github.com/rclone/rclone` as a direct Go library dependency.
**Rationale**: User explicitly requested using rclone as a Go dependency. This allows for a single self-contained binary without external runtime dependencies on the `rclone` executable.
**Implementation Detail**: We will use `github.com/rclone/rclone/fs` and related packages. We may need to initialize the rclone config and backend system programmatically (similar to how `cmd/rclone` does it).
**Refinement**: We will avoid `os/exec` entirely.

## 2. File Watching Mechanism

**Decision**: Use `github.com/fsnotify/fsnotify`.
**Rationale**: Standard cross-platform file system notification library for Go.
**Alternatives**:
- Polling: Inefficient for large directory trees.
- `rclone mount`: Could rely on kernel VFS, but complex to manage.

## 3. Web UI Framework

**Decision**: SolidJS + SolidUI + Go embedding.
**Rationale**: User preference for SolidJS. SolidJS offers high performance and a small bundle size. SolidUI provides a set of accessible and customizable components.
**Alternatives**:
- React: Larger ecosystem but heavier.
- Server-side templates: Less interactive.

## 4. Bidirectional Sync Logic

**Decision**: Reuse `rclone bisync` logic.
**Rationale**: User suggestion. Rclone's `bisync` module already implements robust bidirectional synchronization with state tracking (listing files, checking differences).
**Implementation Detail**: Investigate `github.com/rclone/rclone/cmd/bisync` or `github.com/rclone/rclone/librclone/bisync` (if available) to see if we can invoke the logic programmatically without executing a subprocess. If the internal API is too coupled to the CLI, we might need to adapt specific functions or use the `operations` package which `bisync` relies on.
**Refinement**: `bisync` maintains a state file (listing) to detect deletions. We should ensure we map this state storage to our app's data location.

## 5. Scheduler

**Decision**: `github.com/robfig/cron/v3`.
**Rationale**: Robust, standard cron parser and runner for Go.

## 6. Data Persistence

**Decision**: SQLite (`github.com/mattn/go-sqlite3` or pure Go `modernc.org/sqlite`) for relational data (Tasks, Jobs) + TOML for configuration.
**Rationale**: Structured storage needed for Task definitions and Job history. Application-level configuration (port, paths, log levels) should be in a text-based config file (TOML) handled by `viper` for ease of manual editing and deployment.

## 7. CLI and Configuration

**Decision**: `spf13/cobra` for CLI and `spf13/viper` for configuration.
**Rationale**: User request. These are the standard libraries for building Go applications. `cobra` provides a robust structure for commands (start server, manage tasks via CLI), and `viper` handles configuration loading from files/env vars seamlessly.

## 8. Database ORM Framework

**Decision**: `ent` (`entgo.io/ent`) with SQLite.
**Rationale**: User choice. Ent provides strong type safety and robust schema definitions.
**Migration Strategy**:
- **Development**: Auto-migration (Schema Create) on startup.
- **Production**: Versioned migration (Atlas/golang-migrate).
- **Configuration**: Allow forcing migration mode via config/env.

## 9. Web Framework

**Decision**: `gin` (`github.com/gin-gonic/gin`).
**Rationale**: User choice. Gin is a mature, high-performance web framework with a massive ecosystem. It integrates well with standard `net/http` (via wrappers if needed) and provides easy routing, middleware, and JSON validation.

## 10. Logging Library

**Decision**: `zap` (`go.uber.org/zap`).
**Rationale**: User choice. High performance, structured logging.
**Integration Strategy**:
- **Configuration**: Log level and output paths configurable via `config.toml`.
- **Ent Integration**: Use `zap` as the logger for Ent's debug mode (implementing Ent's logger interface).
- **Rclone Integration**: Redirect rclone's global logging to `zap`. Since rclone logs are global, we will capture them and wrap them in a zap logger (e.g., `zap.Named("rclone")`).
- **Job Logs**: For per-job business logic logs (JobLog entity), we will use specific Zap loggers with fields `zap.String("jobID", ...)` AND persist critical events to the SQLite `job_logs` table.


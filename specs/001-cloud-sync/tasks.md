---
description: "Task list for Rclone Cloud Sync Manager implementation"
---

# Tasks: Rclone Cloud Sync Manager

**Input**: Design documents from `/specs/001-cloud-sync/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/api.yaml

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create project structure (cmd, internal, web, etc.) per plan.md
- [X] T002 Initialize Go module and install dependencies (gin, ent, cobra, viper, zap, rclone)
- [X] T003 Setup Go tool dependencies for Ent (`entgo.io/ent/cmd/ent`) and others
- [X] T004 [P] Initialize SolidJS frontend project in `web/`
- [X] T005 [P] Configure `golangci-lint` and `.gitignore`

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

- [x] T006 Initialize Cobra application (using `cobra-cli init`) and create `serve` command in `cmd/cloud-sync/`
- [X] T007 Setup Viper configuration loading (config.toml) in `internal/core/config/config.go` and integrate with Cobra `rootCmd`
    - Add `Environment` config to control log format and server mode
- [X] T008 Setup Zap logging with Rclone redirection in `internal/core/logger/logger.go`
    - Redirect `log/slog` to Zap for Rclone v1.72+ support
    - Configure Rclone log level in `internal/rclone/config.go`
- [X] T009 Initialize Ent schema and generate code in `internal/core/ent/`
- [X] T010 Implement database connection and migration logic (Auto/Versioned) in `internal/core/db/db.go`
- [X] T011 Setup Gin router and middleware (Logger, Recovery, CORS) in `internal/api/server.go`
- [X] T012 [P] Create basic SolidJS layout (Sidebar, Header) in `web/src/App.tsx`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Manage Cloud Connections (Priority: P1) 🎯 MVP

**Goal**: Configure and manage rclone remotes.

**Independent Test**: Add a remote via API/UI and verify it appears in `rclone.conf`.

### Implementation for User Story 1

- [X] T013 [US1] Research & Prototype: Verify `fs/config` and `fs.Registry` usage for programmatic config management (create a small proof-of-concept script `cmd/poc-config/main.go`)
- [X] T014 [US1] Implement Rclone Config wrapper in `internal/rclone/config.go`:
    - List remotes: `configfile.LoadConfig` + `config.GetSectionList`
    - Get remote info: `config.FileGet`
    - Create/Update remote: `config.FileSet` + `config.SaveConfig`
    - Delete remote: `config.FileDelete`
- [X] T015 [US1] Implement Provider Schema Service in `internal/rclone/providers.go`:
    - List providers: Iterate `fs.Registry`
    - Get provider options: Retrieve `fs.Reg.Options` for dynamic UI forms
- [X] T016 [US1] Implement Remote API handlers (GET, POST, DELETE, Schema) in `internal/api/handlers/remote.go`
- [X] T017 [US1] Register Remote routes in `internal/api/routes.go`
- [X] T018 [P] [US1] Create Remote management UI (List, Add Form) in `web/src/pages/Remotes.tsx`
- [X] T019 [P] [US1] Integrate Remote API with frontend in `web/src/api/remotes.ts`

**Checkpoint**: User Story 1 fully functional.

---

## Phase 4: User Story 2 - Create and Manage Sync Tasks (Priority: P1)

**Goal**: Define sync tasks between local and remote.

**Independent Test**: Create a task and verify it is saved to DB.

### Implementation for User Story 2

- [X] T020 [US2] Define `Task` schema in `internal/core/ent/schema/task.go` and regenerate Ent
- [X] T021 [US2] Implement Task CRUD service logic in `internal/core/services/task_service.go`
- [X] T022 [US2] Implement Task API handlers in `internal/api/handlers/task.go`
- [X] T023 [US2] Register Task routes in `internal/api/routes.go`
- [X] T024 [P] [US2] Create Task management UI (List, Create Wizard) in `web/src/pages/Tasks.tsx`
- [X] T025 [P] [US2] Integrate Task API with frontend in `web/src/api/tasks.ts`

**Checkpoint**: User Story 2 fully functional.

---

## Phase 5: User Story 3 - Real-time and Scheduled Sync (Priority: P2)

**Goal**: Automate sync tasks via schedule or file watching.

**Independent Test**: Trigger a sync manually or via schedule and verify execution.

### Implementation for User Story 3

- [X] T026 [US3] Define `Job` and `JobLog` schema in `internal/core/ent/schema/job.go` and regenerate Ent
- [X] T027 [US3] Implement Job service (Create, Update Status, Log) in `internal/core/services/job_service.go`
- [X] T028 [US3] Research & Prototype: Verify `bisync` programmatic invocation and stats tracking (create `cmd/poc-bisync/main.go`)
- [X] T029 [US3] Implement Sync Engine (Rclone wrapper) in `internal/rclone/sync.go`:
    - Invoke `bisync.Bisync(ctx, fs1, fs2, opts)` from `github.com/rclone/rclone/cmd/bisync`
    - Isolate jobs using `accounting.WithStatsGroup(ctx, taskID)`
    - **Implement a reliable polling loop that calls `stats.Transferred()` to get events and then iterates through the list, calling `stats.RemoveTransfer(tr)` on each to manually prune the log.**
    - **Convert polled transfer events into `JobLog` entities and persist them to the database via `JobService`.**
    - Configure `opt.Workdir` to `app_data/bisync_state` for state management
- [X] T030 [US3] Set `accounting.MaxCompletedTransfers` to `-1` (unlimited) on startup in `internal/rclone/sync.go`
- [X] T031 [US3] Implement Scheduler (Cron) in `internal/core/scheduler/scheduler.go`
- [X] T032 [US3] Implement File Watcher (fsnotify) in `internal/core/watcher/watcher.go`
- [X] T033 [US3] Implement Task Runner (Orchestrator) in `internal/core/runner/runner.go`
- [X] T034 [US3] Add "Run Now" API endpoint in `internal/api/handlers/task.go`
- [X] T035 [P] [US3] Add "Run Now" button and Schedule display in UI

**Checkpoint**: User Story 3 fully functional.

---

## Phase 6: User Story 4 - Dashboard and Monitoring (Priority: P2)

**Goal**: Visualize sync status and history.

**Independent Test**: View dashboard and verify real-time updates.

### Implementation for User Story 4

- [X] T036 [US4] Implement Job History API handlers in `internal/api/handlers/job.go`
- [X] T037 [US4] Implement Real-time status broadcaster (SSE/WebSocket) in `internal/api/sse/broadcaster.go`
- [X] T038 [US4] Hook Sync Engine progress to Broadcaster in `internal/rclone/sync.go`
- [X] T039 [P] [US4] Create Dashboard UI (Active Jobs, Recent History) in `web/src/pages/Dashboard.tsx`
- [X] T040 [P] [US4] Create Job Details/Log Viewer UI in `web/src/pages/JobDetails.tsx`

**Checkpoint**: User Story 4 fully functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T041 Embed frontend assets into Go binary in `internal/ui/embed.go`
- [X] T042 Implement graceful shutdown in `cmd/cloud-sync/serve.go`
- [X] T043 Add comprehensive error handling and validation
- [X] T044 Finalize documentation (README.md)

## Phase 8: Code Review Follow-ups

**Purpose**: Address issues identified in code review

- [X] T045 Add unit tests for `pollStats` method in `internal/rclone/sync.go` to ensure reflection logic works correctly with rclone library
- [X] T046 Add comprehensive documentation comments to `pollStats` method explaining why reflection is used and the risks involved
- [X] T047 Add integration tests for the entire sync flow to ensure stats collection works end-to-end
- [X] T048 Refine and standardize error handling architecture:
    - Create `internal/core/errs/errors.go` for domain sentinel errors (`ErrNotFound`, `ErrInvalid`, `ErrSystem`) to decouple layers.
    - Update `internal/core/services` to translate DB/Ent errors into domain errors using `errors.Join(ErrDomain, err)`.
    - Enhance `internal/api/handlers/error.go` to map domain errors to `AppError` (HTTP codes) automatically.
    - Apply consistent error wrapping in `internal/rclone` to preserve stack/context.
    - Ensure background runners (`scheduler`, `watcher`) log errors with full context (Zap fields) instead of just error strings.
- [X] T049 Increase test coverage for core components:
    - Create `internal/core/runner/runner_test.go`: Test `Start`, `Stop`, job queuing, and concurrency limits.
    - Create `internal/core/scheduler/scheduler_test.go`: Verify cron expression parsing and job triggering.
    - Create `internal/core/watcher/watcher_test.go`: Verify file event detection, debounce logic, and path filtering.
    - Update `internal/rclone/sync_test.go`: Cover edge cases in `Sync` (e.g., context cancellation, error propagation).
    - Create `internal/api/handlers/handler_test.go`: Setup shared test infrastructure (TestMain, in-memory DB).
    - Create `internal/api/handlers/task_test.go` & `remote_test.go`: Integration tests for CRUD operations.
- [X] T050 Configure frontend API base URL and Vite Proxy:
    - Create `web/src/api/config.ts` to centralize `API_BASE` (set to `/api`)
    - Configure Vite proxy in `web/vite.config.ts` to forward `/api` to `http://localhost:8080/api`
    - Refactor `web/src/api/remotes.ts` to use centralized `API_BASE`
    - Refactor `web/src/api/tasks.ts` to use centralized `API_BASE`
    - Refactor `web/src/api/jobs.ts` to use centralized `API_BASE`
- [X] T051 Enhance file watcher functionality:
    - **Refactor recursive watching**: Encapsulate recursive directory monitoring logic into a dedicated utility/struct (e.g., `RecursiveWatcher`) to decouple low-level fsnotify management from `Watcher` business logic.
    - **Implement full lifecycle recursive events**:
        - **Init**: Use `filepath.Walk` to add all existing subdirectories.
        - **Add**: Listen for `Create` events to dynamically add new subdirectories to the watcher.
        - **Remove**: Listen for `Remove`/`Rename` events to stop watching removed directories and clean up internal state.
    - Optimize path matching logic: Replace O(N) linear search with a more efficient lookup structure (e.g., Trie/Radix Tree or specific parent-directory map) to find the responsible task(s) for a file event.
    - Fix matching accuracy: Ensure prefix matching strictly respects directory boundaries (using `filepath.Rel` or path separator checks) to avoid false positives (e.g., `/data` matching `/database`).
    - Implement file filtering: Add logic to ignore events for files matching exclusion patterns (e.g., `.git`, temporary files) to reduce unnecessary sync triggers.
    - **Architecture Decision - Shared Watcher**: Use a single `fsnotify` instance for all tasks to avoid hitting OS limits (e.g., Linux `max_user_instances` default is 128). Implement reference counting or a many-to-one mapping (path -> []taskIDs) to safely manage shared watches.
- [X] T052 Implement graceful shutdown and crash recovery:
    - **Update `Runner`**: Modify `internal/core/runner/runner.go` to track active tasks with `sync.WaitGroup`. Implement `Stop()` to cancel all contexts and wait for completion.
    - **Handle Context Cancellation**: Update `internal/rclone/sync.go` to detect `context.Canceled` and set Job status to `cancelled` instead of `failed`.
    - **Startup Recovery**: Add logic in `internal/core/runner` (or called from `serve.go`) to find jobs stuck in `running` state on startup and mark them `failed` (msg: "System crash or unexpected shutdown").
    - **Wire Shutdown**: Update `cmd/cloud-sync/serve.go` to call `taskRunner.Stop()` during the graceful shutdown sequence.

## Phase 9: Feature Completeness - One-way Sync

**Goal**: Support one-way sync (Upload/Download) in addition to Bidirectional sync.

- [X] T053 [US3] Refactor `SyncEngine` in `internal/rclone/sync.go` to support `task.Direction`:
    - Import `github.com/rclone/rclone/fs/sync` package.
    - In `RunTask`, switch on `task.Direction` (enum: "upload", "download", "bidirectional").
    - Implement `upload` (Source -> Remote) using `sync.Sync(statsCtx, f2, f1, false)`.
    - Implement `download` (Remote -> Source) using `sync.Sync(statsCtx, f1, f2, false)`.
    - Ensure `bidirectional` continues to use `bisync.Bisync`.
    - Ensure unified stats collection/logging works for all directions.
- [X] T054 [US3] Add integration tests for one-way sync modes in `internal/rclone/sync_test.go` to verify `upload` and `download` directions work as expected.

## Dependencies

1. **Setup** -> **Foundational**
2. **Foundational** -> **US1** (Remotes needed for Tasks)
3. **US1** -> **US2** (Tasks need Remotes)
4. **US2** -> **US3** (Automation needs Tasks)
5. **US3** -> **US4** (Monitoring needs Execution)

## Parallel Execution Examples

- **Frontend/Backend**: T018 (UI) and T016 (API) can be done in parallel once T011 is ready.
- **Services**: T031 (Scheduler) and T032 (Watcher) are independent components.

# Tasks: Refactor Global Variable Dependencies

**Input**: Design documents from `/specs/006-refactor-global-deps/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, quickstart.md ✓

**Tests**: This refactoring does not include new testing tasks. The goal is to improve testability and ensure existing tests pass.

**Organization**: Tasks are organized by refactoring modules, with each module corresponding to a user story phase.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1=db refactor, US2=config refactor, US3=logger+entry point)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `internal/` at repository root
- **Commands**: `cmd/cloud-sync/`

---

## Phase 1: Setup

**Purpose**: Prepare refactoring infrastructure

- [x] T001 Create logger Getter methods `Get()` and `Named()` in internal/core/logger/logger.go
- [x] T002 [P] Create default logger (Info level) for uninitialized scenarios in internal/core/logger/logger.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Prepare dependency injection infrastructure

**⚠️ CRITICAL**: All user story phases depend on this phase completion

- [x] T003 Define `RouterDeps` struct for router dependency injection in internal/api/routes.go

**Checkpoint**: Infrastructure ready - can start user story implementation ✅

---

## Phase 3: User Story 1 - Developer writes unit tests (Priority: P1) 🎯 MVP

**Goal**: Refactor db.Client global variable so developers can easily inject mock database dependencies

**Independent Test**: Can write unit tests using mock db client for any service; tests can run without depending on a real database

### Implementation for User Story 1

- [x] T004 [US1] Modify `InitDB` function to return `(*ent.Client, error)` instead of setting a global variable in internal/core/db/db.go
- [x] T005 [US1] Modify `CloseDB` function to accept `*ent.Client` parameter in internal/core/db/db.go
- [x] T006 [US1] Remove `db.Client` global variable declaration in internal/core/db/db.go
- [x] T007 [US1] Update `Migrate` function to accept expanded parameters (environment string) instead of `config.Cfg` in internal/core/db/migrate.go
- [x] T008 [US1] Update logger.L calls in internal/core/db/migrate.go to logger.Named("core.db")
- [x] T009 [US1] Update logger.L calls in internal/core/db/db.go to logger.Named("core.db")
- [x] T010 [US1] Update internal/core/db/migrate_test.go to adapt to the new Migrate signature

**Checkpoint**: db.Client refactoring complete; isolated tests can be written for database-related services ✅

---

## Phase 4: User Story 2 - Developer understands module dependencies (Priority: P2)

**Goal**: Refactor config.Cfg global variable so module dependencies are explicitly expressed through function signatures

**Independent Test**: View the constructor of any service; config dependency should be clearly seen as a parameter

### Implementation for User Story 2

- [x] T011 [US2] Modify `InitConfig` function to `Load` returning `(*Config, error)` in internal/core/config/config.go
- [x] T012 [US2] Remove `config.Cfg` global variable declaration in internal/core/config/config.go
- [x] T013 [US2] Update `NewServer` function to accept `*config.Config` parameter in internal/api/server.go
- [x] T014 [US2] Update logger.L calls in internal/api/server.go to logger.Named("api.server")
- [x] T015 [US2] Update `SetupRouter` function to accept `RouterDeps` parameter in internal/api/routes.go

**Checkpoint**: config.Cfg refactoring complete; module dependencies are clearly visible ✅

---

## Phase 5: User Story 3 - System runs in multiple environments (Priority: P3)

**Goal**: Update application entry points, unify dependency assembly management, and complete logger module refactoring

**Independent Test**: Can start services in different environments with different configurations; all dependencies are unifiedly created at the entry point

### Implementation for User Story 3 - Entry Point Updates

- [x] T016 [US3] Update cmd/cloud-sync/root.go to use config.Load() instead of config.InitConfig()
- [x] T017 [US3] Update cmd/cloud-sync/serve.go to unify creation and assembly of all dependencies
- [x] T018 [US3] Update cmd/cloud-sync/main.go to ensure correct startup flow

### Implementation for User Story 3 - Logger Module Migration

- [x] T019 [P] [US3] Remove logger.L global variable declaration in internal/core/logger/logger.go
- [x] T020 [P] [US3] Update internal/api/sse/broadcaster.go to use logger.Named("api.sse")
- [x] T021 [P] [US3] Update internal/core/services/job_service.go to use logger.Named("service.job")
- [x] T022 [P] [US3] Update internal/core/scheduler/scheduler.go to use logger.Named("core.scheduler")
- [x] T023 [P] [US3] Update internal/core/runner/runner.go to use logger.Named("core.runner")
- [x] T024 [P] [US3] Update internal/core/watcher/watcher.go to use logger.Named("core.watcher")
- [x] T025 [P] [US3] Update internal/core/watcher/recursive.go to use logger.Named("core.watcher")
- [x] T026 [P] [US3] Update internal/rclone/sync.go to use logger.Named("sync.engine")

**Checkpoint**: All global variables refactored; application can be flexibly configured and started in any environment ✅

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and cleanup

- [x] T027 [P] Update all test files using db.Client to adapt to the new dependency injection method
- [x] T028 [P] Update all test files using config.Cfg to adapt to the new dependency injection method
- [x] T029 Run all existing tests to ensure they pass using `go test ./...`
- [x] T030 [P] Use grep to verify that direct access to db.Client, config.Cfg, and logger.L no longer exists in the codebase
- [ ] T031 Run validation scenarios in quickstart.md to confirm functionality is normal

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately ✅
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories ✅
- **User Story 1 (Phase 3)**: Depends on Foundational (Phase 2) ✅
- **User Story 2 (Phase 4)**: Depends on User Story 1 (Phase 3) - routes.go needs to simultaneously accept db and config ✅
- **User Story 3 (Phase 5)**: Depends on User Story 2 (Phase 4) - entry points need to assemble all dependencies ✅
- **Polish (Phase 6)**: Depends on all user stories complete ✅

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational is complete - db.Client refactor is the minimum scope of change ✅
- **User Story 2 (P2)**: Depends on US1 completion - config refactor requires the db parameter passing pattern to be established ✅
- **User Story 3 (P3)**: Depends on US2 completion - entry point needs to handle both db and config passing ✅

### Within Each User Story

- Module core function modifications take priority over consumer updates
- Remove global variables after functional verification
- Logger migration tasks can be executed in parallel (different files)

### Parallel Opportunities

- Setup phase: T002 can be parallelized
- US3 Logger migration: T020-T026 can all be parallelized (different files)
- Polish phase: T027, T028, T030 can be parallelized

---

## Parallel Example: User Story 3 Logger Migration

```bash
# All logger migration tasks can be executed in parallel (different files):
Task T020: "Update internal/api/sse/broadcaster.go to use logger.Named()"
Task T021: "Update internal/core/services/job_service.go to use logger.Named()"
Task T022: "Update internal/core/scheduler/scheduler.go to use logger.Named()"
Task T023: "Update internal/core/runner/runner.go to use logger.Named()"
Task T024: "Update internal/core/watcher/watcher.go to use logger.Named()"
Task T025: "Update internal/core/watcher/recursive.go to use logger.Named()"
Task T026: "Update internal/rclone/sync.go to use logger.Named()"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup - Create Logger Getter methods ✅
2. Complete Phase 2: Foundational - Define dependency injection structs ✅
3. Complete Phase 3: User Story 1 - db.Client refactor ✅
4. **STOP and VALIDATE**: Verify that isolated tests can be written for db-related services ✅
5. If satisfied, can pause; main testing benefits already achieved.

### Incremental Delivery

1. Setup + Foundational → Infrastructure ready ✅
2. Add User Story 1 (db.Client) → Test validation → **MVP complete** ✅
3. Add User Story 2 (config.Cfg) → Test validation → Dependencies clarified ✅
4. Add User Story 3 (logger + entry point) → Test validation → Full refactoring complete ✅
5. Each phase is independently verifiable without breaking previous functionality.

### Risk Mitigation

- Refactor from least to most used: db (2 places) → config (5 places) → logger (31 places)
- Run all tests after each phase completion
- Logger changes are many but follow a unified pattern, allowing batch processing.

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story phase should be independently testable upon completion
- Commit code after each task or logical group is completed
- Validation can be paused at any Checkpoint
- Avoid: Vague tasks, same-file conflicts, cross-story dependencies

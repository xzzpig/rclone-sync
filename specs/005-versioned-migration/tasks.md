# Tasks: Versioned Database Migration

**Input**: Design documents from `/specs/005-versioned-migration/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, quickstart.md ✓

**Tests**: Not explicitly required; basic verification tests are only included in Phase 6.

**Organization**: Tasks are grouped by User Story, supporting independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can be executed in parallel (different files, no dependencies)
- **[Story]**: The User Story the task belongs to (US1, US2, US3)
- Descriptions include precise file paths

## Path Conventions

- **Go Backend**: `internal/core/db/`
- **Migration Scripts**: `internal/core/db/migrations/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialization of project dependencies and infrastructure

- [x] T001 Add golang-migrate dependency: Execute `go get github.com/golang-migrate/migrate/v4 github.com/golang-migrate/migrate/v4/database/sqlite3 github.com/golang-migrate/migrate/v4/source/iofs`
- [x] T002 Create migration scripts directory: `mkdir -p internal/core/db/migrations`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure depended upon by all User Stories

**⚠️ Critical**: No User Story can start before this phase is completed

- [x] T003 Create embed.go file, embedding only up.sql scripts in `internal/core/db/embed.go`
- [x] T030 Refine detection logic description: No pre-check needed, rely on golang-migrate's table conflict error reporting (FR-010)
- [x] T031 Add logic to automatically clean up down.sql in gen-migration.sh (Optional, as a helper utility)

**Checkpoint**: Infrastructure ready - User Story implementation can begin

---

## Phase 3: User Story 1 - Secure Database Migration During App Version Upgrades (Priority: P1) 🎯 MVP

**Goal**: Automatically detect and execute all pending migrations upon application startup, preserving all existing data.

**Independent Test**: Create an old version database containing test data, run the application upgrade, and verify that data is fully preserved and the structure is correctly updated.

### Implementation for User Story 1

- [x] T019 [P] Write unit tests for the migration module (Migrate function, error handling) in `internal/core/db/migrate_test.go`
- [x] T008 [US1] Generate initial baseline migration script (containing DDL for all existing ent schemas) in `internal/core/db/migrations/20251219123754_initial.up.sql` (Note: Atlas automatically generates `down.sql` files but the application does not use them; only forward migrations are supported)
- [x] T009 [US1] Create migration log adapter migrateLogger implementing the migrate.Logger interface in `internal/core/db/migrate.go`
- [x] T010 [US1] Implement Migrate function: Create migration source from embed.FS, create SQLite driver, execute migrations in `internal/core/db/migrate.go`
- [x] T011 [US1] Modify InitDB function: Replace original Client.Schema.Create automatic migration with versioned migrations in `internal/core/db/db.go`
- [x] T012 [US1] Implement logic to prevent app startup and output detailed English error logs when migration fails in `internal/core/db/db.go`

**Checkpoint**: User Story 1 should be fully functional, supporting both fresh databases and databases with versioned migrations.

---

## Phase 4: User Story 2 - Developers Create New Migration Scripts (Priority: P2)

**Goal**: Developers can conveniently use the Atlas CLI to generate new migration scripts from the ent schema.

**Independent Test**: Run the migration generation command after modifying the ent schema, verifying that golang-migrate migration files are generated in the correct format.

### Implementation for User Story 2

- [x] T013 [US2] Update developer workflow documentation in quickstart.md, ensuring atlas migrate diff command examples are correct in `specs/005-versioned-migration/quickstart.md`
- [x] T014 [US2] Verify flake.nix includes the atlas CLI tool in `flake.nix`
- [x] T015 [US2] Create migration generation script `scripts/gen-migration.sh` wrapping Atlas commands to simplify the development process in the project root
- [x] T015b [US2] Add data migration example documentation in quickstart.md, explaining how to write data transformation SQL in .up.sql files (satisfies FR-005) in `specs/005-versioned-migration/quickstart.md`

**Checkpoint**: Developers can use the `atlas migrate diff` command to generate migration files from the ent schema and understand how to write data migrations.

---

## Phase 5: User Story 3 - View Migration Status (Priority: P3)

**Goal**: Administrators can view the current database migration status, including the database version and pending migrations.

**Independent Test**: Run the status query command and verify it displays the correct migration history and current version.

### Implementation for User Story 3

- [x] T020 [P] Write unit tests for migration status queries in `internal/core/db/migrate_test.go`
- [x] T016 [US3] Implement GetMigrationStatus function: Query schema_migrations table and return current version and dirty state in `internal/core/db/migrate.go`
- [x] T017 [US3] Implement GetPendingMigrations function: Compare executed versions with embedded migration files and return the list of pending migrations in `internal/core/db/migrate.go`
- [x] T018 [US3] Output current migration status (version, number of pending migrations) in application startup logs in `internal/core/db/db.go`

**Checkpoint**: Administrators can view migration status via logs or functions.

---

## Phase 6: User Story 4 - Rapid Iterative Development for Developers (Priority: P2)

**Goal**: Developers can use automatic migration mode during local development and unit testing to speed up development.

**Independent Test**: Configure and switch migration modes, verifying the system correctly uses automatic or versioned migrations.

### Implementation for User Story 4

- [x] T023 [US4] Add Database.MigrationMode configuration item in config.go (default is versioned) in `internal/core/config/config.go`
- [x] T024 [US4] Define MigrationMode type and constants (Versioned/Auto) in `internal/core/db/migrate.go`
- [x] T025 [US4] Modify InitDB function signature, adding mode parameter to support explicit migration mode specification in `internal/core/db/db.go`
- [x] T026 [US4] Modify serve.go to read migration mode from configuration and pass it to InitDB in `cmd/cloud-sync/serve.go`
- [x] T027 [US4] Write unit tests for switching migration modes in `internal/core/db/migrate_test.go`
- [x] T028 [US4] Update quickstart.md to add migration mode configuration documentation in `specs/005-versioned-migration/quickstart.md` (Completed)

**Checkpoint**: Developers can switch between automatic and versioned migration modes via configuration or programming interface.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements affecting multiple User Stories

- [x] T021 Run verification steps in quickstart.md: New database test, no-change test, dirty state test
- [x] T029 Verify migration mode switching functionality: Both versioned and automatic migration modes work correctly

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - blocks all User Stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User Stories can proceed in parallel (if multiple people are available)
  - Or execute in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on completion of all desired User Stories

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) completion - no dependency on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) completion - no dependency on other stories (documentation/tooling workflow)
- **User Story 3 (P3)**: Depends on US1 migration infrastructure - requires T010 completion before starting
- **User Story 4 (P2)**: Depends on US1 migration infrastructure - requires T010/T011 completion before starting

### Within Each User Story

- Models/Types take priority over Services
- Core implementation takes priority over integration
- Complete one story before moving to the next priority

### Parallel Opportunities

- T019, T020 can have tests written in parallel
- Documentation tasks for US2 (T013-T015) can proceed in parallel with US1 implementation

---

## Parallel Example: User Story Implementation

```bash
# Test tasks can be started in parallel:
Task T019: "Write unit tests for migration module in internal/core/db/migrate_test.go"
Task T020: "Write unit tests for migration status query in internal/core/db/migrate_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (Critical - blocks all stories)
3. Complete Phase 3: User Story 1
4. **Stop and Verify**: Independently test User Story 1
5. Deploy/Demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Independent testing → Deploy/Demo (MVP!)
3. Add User Story 2 → Independent testing → Documentation refinement
4. Add User Story 3 → Independent testing → Status query functionality
5. Each story adds value without breaking previous ones

---

## Notes

- [P] Task = Different files, no dependencies
- [Story] labels map tasks to specific User Stories for tracking
- Each User Story should be independently completable and testable
- Commit after each task or logical group is completed
- Stop at any checkpoint to independently verify stories
- Avoid: Vague tasks, conflicts in the same file, cross-story dependencies that break independence
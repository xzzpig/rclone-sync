# Tasks: GraphQL Migration

**Input**: Design documents from `/specs/007-graphql-migration/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/schema.graphql ✅, quickstart.md ✅

**Tests**: Included per Constitution III (TDD is NON-NEGOTIABLE)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `internal/api/graphql/` (new), `internal/api/` (existing)
- **Frontend**: `web/src/api/graphql/` (new), `web/src/` (existing)

## Terminology

- **Namespace resolver**: gqlgen pattern where Query/Mutation root fields return intermediate types (e.g., `Query.task` returns `TaskQuery`, `Mutation.connection` returns `ConnectionMutation`) to group related operations by domain

---

## Phase 1: Setup (Shared Infrastructure) ✅ COMPLETED

**Purpose**: Project initialization, dependencies installation, and basic structure

- [x] T001 Install backend GraphQL dependencies (gqlgen, gqlparser, dataloadgen, gorilla/websocket) via `go get`
- [x] T002 [P] Install frontend GraphQL dependencies (urql, @urql/exchange-graphcache, gql.tada, graphql-ws) via `pnpm add`
- [x] T003 Create backend GraphQL directory structure in `internal/api/graphql/{schema,model,resolver,generated,dataloader}/`
- [x] T004 [P] Create frontend GraphQL directory structure in `web/src/api/graphql/`
- [x] T005 Create gqlgen configuration file in `gqlgen.yml`
- [x] T006 [P] Configure gql.tada TypeScript plugin in `web/tsconfig.json`
- [x] T007 Split schema from `specs/007-graphql-migration/contracts/schema.graphql` into multiple files in `internal/api/graphql/schema/` using `extend` syntax:
  - `schema.graphql` - 空根类型 + 指令 + 标量 + 分页
  - `task.graphql` - Task + extend Query/Mutation
  - `connection.graphql` - Connection + extend Query/Mutation  
  - `job.graphql` - Job/JobLog + extend Query/Subscription
  - `provider.graphql` - Provider + extend Query
  - `file.graphql` - FileEntry + extend Query
  - `import.graphql` - Import + extend Mutation

---

## Phase 2: Foundational (Blocking Prerequisites) ✅ COMPLETED

**Purpose**: Core GraphQL infrastructure that MUST be complete before ANY user story can be implemented

- [x] T008 Run gqlgen code generation to create `internal/api/graphql/generated/generated.go` and `internal/api/graphql/model/models_gen.go`
- [x] T009 Extend generated `resolver.go` to add Dependencies injection in `internal/api/graphql/resolver/resolver.go`
- [x] T010 Implement Dataloader infrastructure (Loaders struct, middleware) in `internal/api/graphql/dataloader/loaders.go`
- [x] T011 [P] Implement ConnectionLoader in `internal/api/graphql/dataloader/connection_loader.go`
- [x] T012 [P] Implement TaskLoader in `internal/api/graphql/dataloader/task_loader.go`
- [x] T013 [P] Implement JobLoader in `internal/api/graphql/dataloader/job_loader.go`
- [x] T014 Create GraphQL handler with transports (HTTP, WebSocket) in `internal/api/graphql/handler.go`
- [x] T015 Implement i18n error presenter for GraphQL errors in `internal/api/graphql/errors.go`
- [ ] T016 Configure query depth limit middleware in `internal/api/graphql/complexity.go`
- [x] T017 Register GraphQL routes (`/api/graphql`, `/api/graphql/playground`) in `internal/api/routes.go`
- [x] T018 Create urql client with cacheExchange and subscriptionExchange in `web/src/api/graphql/client.ts`
- [x] T019 Create GraphQL Provider component for SolidJS in `web/src/api/graphql/provider.tsx`
- [x] T020 Integrate GraphQL Provider into app root in `web/src/App.tsx`

**Checkpoint**: ✅ Foundation ready - GraphQL endpoint accessible, code generation working

---

## Phase 3: User Story 1 & 2 - Namespace Resolvers and Basic Queries (Priority: P1) ✅ COMPLETED

**Goal**: Establish schema-first workflow (US2) and verify frontend type inference (US1)

**Independent Test**: 
- Backend: Run `gqlgen generate` after schema change, verify resolver interface updates
- Frontend: Write a query, verify TypeScript provides field autocompletion

### 3.1 Namespace Resolver Entry Points

实现命名空间 resolver 入口点，使 Query.task、Query.connection 等返回对应的命名空间对象。

- [x] T021 [US1/US2] Implement Query namespace resolvers (Task, Connection, Job, Log, Provider, File) in `internal/api/graphql/resolver/schema.resolvers.go`
- [x] T022 [US1/US2] Implement Mutation namespace resolvers (Task, Connection, Import) in `internal/api/graphql/resolver/schema.resolvers.go`

### 3.2 Basic Query Resolvers (Read-Only)

实现基础查询 resolver，验证端到端类型安全。

- [x] T023 [P] [US1/US2] Implement TaskQuery.list resolver in `internal/api/graphql/resolver/task.resolvers.go`
- [x] T024 [P] [US1/US2] Implement TaskQuery.get resolver in `internal/api/graphql/resolver/task.resolvers.go`
- [x] T025 [P] [US1/US2] Implement ConnectionQuery.list resolver in `internal/api/graphql/resolver/connection.resolvers.go`
- [x] T026 [P] [US1/US2] Implement ConnectionQuery.get resolver in `internal/api/graphql/resolver/connection.resolvers.go`

### 3.3 Frontend Type Safety Verification

- [x] T027 [US1/US2] Create sample GraphQL queries with gql.tada in `web/src/api/graphql/queries/tasks.ts`
- [x] T028 [US1/US2] Create sample GraphQL queries with gql.tada in `web/src/api/graphql/queries/connections.ts`
- [x] T029 [US1/US2] Verify TypeScript type inference works in IDE for GraphQL queries (manual validation)

**Checkpoint**: ✅ Schema-first workflow validated - backend generates code from schema, frontend gets type hints from schema

---

## Phase 4: User Story 3 - Full Feature Migration (Priority: P2) ✅ COMPLETED

**Goal**: Migrate all existing REST API functionality to GraphQL while maintaining feature parity

**Independent Test**: Compare GraphQL responses with existing REST API responses for same operations

### Phase 4.1: Provider Queries (Read-only, Low Risk)

- [x] T030 [P] [US3] Implement ProviderQuery.list resolver in `internal/api/graphql/resolver/provider.resolvers.go`
- [x] T031 [P] [US3] Implement ProviderQuery.get resolver in `internal/api/graphql/resolver/provider.resolvers.go`
- [x] T032 [US3] Create provider queries in `web/src/api/graphql/queries/providers.ts`

### Phase 4.2: Connection Type Field Resolvers

实现 Connection 类型的字段 resolver（使用 Dataloader 避免 N+1 问题）。

- [x] T033 [P] [US3] Implement Connection.config resolver in `internal/api/graphql/resolver/connection.resolvers.go`
- [x] T034 [P] [US3] Implement Connection.loadStatus resolver in `internal/api/graphql/resolver/connection.resolvers.go`
- [x] T035 [P] [US3] Implement Connection.loadError resolver in `internal/api/graphql/resolver/connection.resolvers.go`
- [x] T036 [P] [US3] Implement Connection.tasks resolver (with Dataloader) in `internal/api/graphql/resolver/connection.resolvers.go`
- [x] T037 [P] [US3] Implement Connection.quota resolver in `internal/api/graphql/resolver/connection.resolvers.go`

### Phase 4.3: Connection Mutations

- [x] T038 [US3] Implement ConnectionMutation.create resolver in `internal/api/graphql/resolver/connection.resolvers.go`
- [x] T039 [US3] Implement ConnectionMutation.update resolver in `internal/api/graphql/resolver/connection.resolvers.go`
- [x] T040 [US3] Implement ConnectionMutation.delete resolver in `internal/api/graphql/resolver/connection.resolvers.go`
- [x] T041 [US3] Implement ConnectionMutation.test resolver (returns TestConnectionResult union) in `internal/api/graphql/resolver/connection.resolvers.go`
- [x] T042 [US3] Implement ConnectionMutation.testUnsaved resolver in `internal/api/graphql/resolver/connection.resolvers.go`
- [x] T043 [US3] Create connection mutations in `web/src/api/graphql/queries/connections.ts`

### Phase 4.4: Task Type Field Resolvers

- [x] T044 [P] [US3] Implement Task.options resolver in `internal/api/graphql/resolver/task.resolvers.go`
- [x] T045 [P] [US3] Implement Task.connection resolver (with Dataloader) in `internal/api/graphql/resolver/task.resolvers.go`
- [x] T046 [P] [US3] Implement Task.jobs resolver in `internal/api/graphql/resolver/task.resolvers.go`
- [x] T047 [P] [US3] Implement Task.latestJob resolver in `internal/api/graphql/resolver/task.resolvers.go`

### Phase 4.5: Task Mutations

- [x] T048 [US3] Implement TaskMutation.create resolver in `internal/api/graphql/resolver/task.resolvers.go`
- [x] T049 [US3] Implement TaskMutation.update resolver in `internal/api/graphql/resolver/task.resolvers.go`
- [x] T050 [US3] Implement TaskMutation.delete resolver in `internal/api/graphql/resolver/task.resolvers.go`
- [x] T051 [US3] Implement TaskMutation.run resolver (creates and starts Job) in `internal/api/graphql/resolver/task.resolvers.go`
- [x] T052 [US3] Create task mutations in `web/src/api/graphql/queries/tasks.ts`

### Phase 4.6: Job & Log Queries

- [x] T053 [P] [US3] Implement Job.task resolver (with Dataloader) in `internal/api/graphql/resolver/job.resolvers.go`
- [x] T054 [P] [US3] Implement Job.logs resolver in `internal/api/graphql/resolver/job.resolvers.go`
- [x] T055 [P] [US3] Implement JobLog.job resolver (with Dataloader) in `internal/api/graphql/resolver/job.resolvers.go`
- [x] T056 [US3] Implement JobQuery.list resolver (with optional taskId filter) in `internal/api/graphql/resolver/job.resolvers.go`
- [x] T057 [US3] Implement JobQuery.progress resolver in `internal/api/graphql/resolver/job.resolvers.go`
- [x] T058 [US3] Implement LogQuery.list resolver in `internal/api/graphql/resolver/job.resolvers.go`
- [x] T059 [US3] Create job queries in `web/src/api/graphql/queries/jobs.ts`
- [x] T060 [P] [US3] Create log queries in `web/src/api/graphql/queries/logs.ts`

### Phase 4.7: File Browsing

- [x] T061 [US3] Implement FileQuery.local resolver in `internal/api/graphql/resolver/file.resolvers.go`
- [x] T062 [US3] Implement FileQuery.remote resolver in `internal/api/graphql/resolver/file.resolvers.go`
- [x] T063 [US3] Create file queries in `web/src/api/graphql/queries/files.ts`

### Phase 4.8: Import Functionality

- [x] T064 [US3] Implement ImportMutation.parse resolver (returns ImportParseResult union) in `internal/api/graphql/resolver/import.resolvers.go`
- [x] T065 [US3] Implement ImportMutation.execute resolver in `internal/api/graphql/resolver/import.resolvers.go`
- [x] T066 [US3] Create import mutations in `web/src/api/graphql/queries/import.ts`

### Phase 4.9: Subscription (Replace SSE)

- [x] T067 [US3] Create event bus for job progress events in `internal/api/graphql/subscription/eventbus.go`
- [x] T068 [US3] Integrate event bus with existing runner/broadcaster in `internal/api/graphql/resolver/resolver.go`
- [x] T069 [US3] Implement Subscription.jobProgress resolver in `internal/api/graphql/resolver/job.resolvers.go`
- [x] T070 [US3] Create jobProgress subscription in `web/src/api/graphql/queries/subscriptions.ts`

**Checkpoint**: ✅ All REST API functionality now available via GraphQL, SSE replaced with WebSocket subscription

---

## Phase 5: User Story 4 - Frontend Migration to GraphQL (Priority: P3) ✅ COMPLETED

**Goal**: Migrate frontend modules to use GraphQL, leveraging field selection for reduced data transfer

**Independent Test**: Compare network payload size between REST and GraphQL for same data requests

### Implementation for User Story 4

- [x] T071 [US4] Migrate connection module to GraphQL in `web/src/modules/connections/`
  - Settings.tsx, AddConnectionDialog.tsx, DynamicConfigForm.tsx migrated to GraphQL
  - ConnectionSidebarItem.tsx uses GraphQL types
- [x] T072 [US4] Migrate task module to GraphQL in `web/src/modules/connections/`
  - Tasks.tsx, TaskSettingsForm.tsx, EditTaskDialog.tsx, CreateTaskWizard.tsx migrated
  - store/tasks.tsx uses GraphQL types with camelCase properties
- [x] T073 [US4] Migrate job module to GraphQL in `web/src/modules/connections/`
  - History.tsx, Overview.tsx migrated to GraphQL with camelCase properties
  - store/history.tsx uses GraphQL types
- [x] T074 [US4] Migrate file browser module to GraphQL in `web/src/components/common/`
  - FileBrowser.tsx uses GraphQL FileEntry type with isDir (camelCase)
- [x] T075 [US4] Migrate import module to GraphQL in `web/src/modules/connections/components/ImportWizard/`
  - ImportWizard.tsx, Step2Preview.tsx, Step3Confirm.tsx, EditImportConfigDialog.tsx migrated
- [x] T076 [US4] Replace SSE event handling with GraphQL subscription in `web/src/store/`
  - Subscription implemented in web/src/api/graphql/queries/subscriptions.ts
  - AppShell.tsx uses GraphQL subscription for job progress
- [x] T077 [US4] Implement cache updates for mutations using graphcache in `web/src/api/graphql/client.ts`
  - Cache invalidation configured for Task and Connection create/delete mutations
  - Note: True optimistic updates not supported for namespace pattern mutations
- [x] T078 [US4] Verify field selection reduces payload (only request needed fields)
  - All GraphQL queries request only needed fields
  - Example: TasksListQuery only fetches id, name, status fields for list view
  - Example: ConnectionsListQuery only fetches id, name, type, loadStatus for sidebar

**Checkpoint**: ✅ Frontend fully migrated to GraphQL, using GraphQL types throughout

---

## Phase 6: Testing Infrastructure ✅ COMPLETED

**Purpose**: Setup test infrastructure and write tests

### Backend Test Infrastructure

- [x] T079 Create GraphQL resolver test setup with mock dependencies in `internal/api/graphql/resolver/resolver_test.go`
- [x] T080 [P] Write Dataloader unit tests in `internal/api/graphql/dataloader/loaders_test.go`
- [x] T081 [P] Write ConnectionLoader tests in `internal/api/graphql/dataloader/connection_loader_test.go`
- [x] T082 [P] Write TaskLoader tests in `internal/api/graphql/dataloader/task_loader_test.go`
- [x] T083 [P] Write JobLoader tests in `internal/api/graphql/dataloader/job_loader_test.go`
- [x] T084 Write i18n error presenter tests in `internal/api/graphql/errors_test.go`
- [x] T085 Write GraphQL handler integration tests in `internal/api/graphql/handler_test.go`

### Resolver Tests

- [x] T086 [P] Write TaskQuery/TaskMutation resolver tests in `internal/api/graphql/resolver/task_test.go`
- [x] T087 [P] Write ConnectionQuery/ConnectionMutation resolver tests in `internal/api/graphql/resolver/connection_test.go`
- [x] T088 [P] Write ProviderQuery resolver tests in `internal/api/graphql/resolver/provider_test.go`
- [x] T089 [P] Write JobQuery/LogQuery resolver tests in `internal/api/graphql/resolver/job_test.go`
- [x] T090 [P] Write FileQuery resolver tests in `internal/api/graphql/resolver/file_test.go`
- [x] T091 [P] Write ImportMutation resolver tests in `internal/api/graphql/resolver/import_test.go`
- [x] T092 Write Subscription resolver tests in `internal/api/graphql/resolver/subscription_test.go`
- [x] T093 Write mutation atomicity integration tests (verify transaction rollback on partial failure) in `internal/api/graphql/resolver/atomicity_test.go`

---

## Phase 7: Polish & Cross-Cutting Concerns ✅ COMPLETED

**Purpose**: Cleanup, documentation, and final validation

- [x] T094 Remove deprecated REST API handlers in `internal/api/handlers/`
- [x] T095 Remove deprecated REST API routes in `internal/api/routes.go`
- [x] T096 [P] Remove deprecated SSE broadcaster (if fully replaced) in `internal/api/sse/`
- [x] T097 [P] Remove frontend REST API client code in `web/src/api/` (non-GraphQL)
- [x] T098 Update README.md with GraphQL endpoint documentation
- [x] T099 Run quickstart.md validation scenarios
- [x] T100 Verify all acceptance scenarios from spec.md

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1 (Setup) ✅
    ↓
Phase 2 (Foundational) ✅
    ↓
Phase 3 (US1/US2 - MVP) ← Current Focus
    ↓
Phase 4 (US3 - Full Migration)
    ↓
Phase 5 (US4 - Frontend Migration)
    ↓
Phase 6 (Testing) ← Can run in parallel with Phase 4/5
    ↓
Phase 7 (Polish)
```

### User Story Dependencies

- **US1 + US2 (P1)**: Can start now - Core schema-first workflow
- **US3 (P2)**: Can start after Phase 3 - Feature parity migration
- **US4 (P3)**: Depends on US3 completion - Frontend needs backend resolvers

### Within Each Phase

- Namespace resolvers (T021-T022) BEFORE entity query resolvers
- Query resolvers BEFORE mutation resolvers (same domain)
- Type field resolvers can be parallel within same file
- Backend resolvers BEFORE frontend query files

### Parallel Opportunities

**Phase 3 (US1/US2)**:
```
T021 + T022 (sequential - same file)
Then: T023 || T024 || T025 || T026 (parallel - different resolvers)
Then: T027 || T028 (parallel - different frontend files)
```

**Phase 4 (US3)** - Sub-phases can overlap where no dependencies:
```
4.1 Provider (T030-T032) - Independent, can run anytime
4.2 Connection Type Fields (T033-T037) || 4.4 Task Type Fields (T044-T047) - Parallel
4.3 Connection Mutations (T038-T043) || 4.5 Task Mutations (T048-T052) - Parallel
4.6 Job/Log (T053-T060) - After Task resolvers for relationship
4.7 File (T061-T063) || 4.8 Import (T064-T066) - Parallel
4.9 Subscription (T067-T070) - After Job resolver
```

**Phase 6 (Testing)**:
```
T080 || T081 || T082 || T083 (Dataloader tests - parallel)
T086 || T087 || T088 || T089 || T090 || T091 (Resolver tests - parallel)
```

**Phase 7 (Polish)**:
```
T094 + T095 (sequential - REST removal)
T096 || T097 (parallel - SSE and frontend cleanup)
```

---

## Implementation Strategy

### MVP First (Phase 3: User Stories 1 & 2)

1. ✅ Phase 1: Setup - COMPLETED
2. ✅ Phase 2: Foundational - COMPLETED  
3. **Phase 3: User Stories 1 & 2** ← START HERE
   - Implement namespace resolvers
   - Implement basic list/get queries
   - Verify frontend type safety
4. **STOP and VALIDATE**: 
   - Backend: Run `gqlgen generate`, verify resolver interfaces update
   - Frontend: Write query, verify TypeScript autocompletion
5. Demo GraphQL Playground with basic queries

### Feature Parity (Phase 4: User Story 3)

1. Complete MVP above
2. Complete Phase 4: User Story 3 (full migration)
3. **VALIDATE**: Compare REST and GraphQL responses for each feature
4. Both REST and GraphQL available - parallel operation

### Full Migration (Phase 5 + 6 + 7)

1. Complete Feature Parity above
2. Complete Phase 5: User Story 4 (frontend migration)
3. Complete Phase 6: Testing
4. Complete Phase 7: Polish (cleanup)
5. **VALIDATE**: REST endpoints removed, only GraphQL remains

### Migration Sequence per plan.md

1. Provider queries (read-only, low risk)
2. Connection management (CRUD)
3. Task management (CRUD + Run)
4. Job/Log queries
5. File browsing
6. Import functionality
7. Subscription (replace SSE)

---

## Summary Statistics

| Phase | Tasks | Status |
|-------|-------|--------|
| Phase 1: Setup | T001-T007 (7) | ✅ Completed |
| Phase 2: Foundational | T008-T020 (13) | ✅ Completed |
| Phase 3: US1/US2 (MVP) | T021-T029 (9) | ✅ Completed |
| Phase 4: US3 (Migration) | T030-T070 (41) | ✅ Completed |
| Phase 5: US4 (Frontend) | T071-T078 (8) | ✅ Completed |
| Phase 6: Testing | T079-T093 (15) | ✅ Completed |
| Phase 7: Polish | T094-T100 (7) | ✅ Completed |
| **Total** | **100 tasks** | **100 completed, 0 remaining** |

---

## Notes

- All resolvers must use Dataloader for relationship fields to avoid N+1
- Error messages must use existing i18n system (I18nError)
- Subscription replaces existing SSE broadcaster
- REST endpoints remain until Phase 7 cleanup
- GraphQL Playground available at `/api/graphql/playground` in development
- Query depth limited to prevent performance issues (configurable in complexity.go)
- gqlgen generates resolver files by domain (task.resolvers.go, connection.resolvers.go, etc.)
- Resolver implementations go in the gqlgen-generated files, not new files

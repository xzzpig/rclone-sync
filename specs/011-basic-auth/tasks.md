# Tasks: HTTP Basic Auth 认证

**Input**: Design documents from `/specs/011-basic-auth/`
**Prerequisites**: spec.md (user stories), research.md (implementation decisions), data-model.md (config structure), contracts/http-auth.md (API behavior)

**Tests**: Tests are NOT explicitly requested - implementation tasks only.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Path Conventions

Based on existing project structure:
- **Backend**: `internal/` at repository root (Go)
- **Configuration**: `internal/core/config/`
- **API/Middleware**: `internal/api/` and `internal/api/context/`
- **i18n**: `internal/i18n/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: No project structure changes needed - this feature adds to existing codebase

*No setup tasks required - project structure already exists.*

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Configuration infrastructure that MUST be complete before authentication can be implemented

**⚠️ CRITICAL**: User Story implementation cannot begin until this phase is complete

**注意**: 服务启动阶段无多语言环境，配置验证错误直接使用 ConstError 报错

- [X] T001 Add Auth struct to Config in `internal/core/config/config.go`
  - Add `Auth` field with `Username` and `Password` subfields
  - Add `mapstructure` tags for TOML binding
- [X] T002 [P] Add `IsAuthEnabled()` method to Config in `internal/core/config/config.go`
  - Returns true if both Username and Password are non-empty
- [X] T003 [P] Add `ValidateAuth()` method to Config in `internal/core/config/config.go`
  - Returns error (ConstError) if only one of Username/Password is set
  - 使用 `errs.ErrValidation` 或在 config 包内定义本地 ConstError
- [X] T004 Add auth configuration validation on startup in `cmd/cloud-sync/serve.go`
  - Call `ValidateAuth()` after loading config
  - Fatal error with ConstError message if validation fails (无需 i18n)

**Checkpoint**: Foundation ready - authentication middleware can now be implemented

---

## Phase 3: User Story 2 - 配置凭据 (Priority: P1)

**Goal**: Enable administrators to configure authentication credentials via config file or environment variables

**Independent Test**: 
1. Set `[auth]` block in config.toml with username and password → service starts successfully
2. Set only username without password → service refuses to start with error message
3. Set environment variables `CLOUDSYNC_AUTH_USERNAME` and `CLOUDSYNC_AUTH_PASSWORD` → environment variables override config file

**Note**: User Story 2 is implemented first because it provides the configuration that User Story 1 depends on.

### Implementation for User Story 2

- [X] T005 [US2] Add example auth configuration to `config.toml` (commented out)
  - Add `[auth]` section with `username` and `password` fields as comments
  - Document environment variable overrides
- [X] T006 [US2] Update README.md and README_CN.md documentation for auth configuration
  - Add section on enabling HTTP Basic Auth in both files
  - Document environment variable options
  - Add security recommendations (HTTPS, file permissions)

**Checkpoint**: Configuration is complete and validated - authentication middleware can use these values

---

## Phase 4: User Story 1 - 访问受保护资源 (Priority: P1)

**Goal**: Protect all resources except `/health` with HTTP Basic Auth when credentials are configured

**Independent Test**:
1. Access any page without credentials → browser shows HTTP Basic Auth dialog
2. Enter correct credentials → access granted to all pages
3. Enter incorrect credentials → 401 returned, dialog reappears
4. Access `/health` → always returns 200 without auth

### Implementation for User Story 1

- [X] T007 [US1] Create Basic Auth middleware in `internal/api/context/auth.go`
  - Implement `BasicAuthMiddleware(username, password string) gin.HandlerFunc`
  - Use `crypto/subtle.ConstantTimeCompare` for password comparison
  - Set `WWW-Authenticate: Basic realm="Login Required"` header on 401
  - Log authentication failures at Warn level with IP, username, and path using zap (never log passwords)
  - Set `gin.AuthUserKey` in context on success
- [X] T008 [US1] Create optional auth wrapper function in `internal/api/context/auth.go`
  - Implement `OptionalAuthMiddleware(cfg *config.Config) gin.HandlerFunc`
  - Returns no-op middleware if `IsAuthEnabled()` returns false
  - Returns `BasicAuthMiddleware` if auth is enabled
- [X] T009 [US1] Integrate auth middleware into router in `internal/api/server.go`
  - Register `/health` endpoint BEFORE auth middleware
  - Apply `OptionalAuthMiddleware(cfg *config.Config)` after health check registration
  - Ensure all other routes (API, static files) are protected
  - Config is passed by pointer; credentials are read on each request (stateless validation)

**Checkpoint**: User Story 1 is complete - all resources except /health require authentication when enabled

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, testing validation, and cleanup

- [X] T010 [P] Add auth middleware tests in `internal/api/context/auth_test.go`
  - Test: no auth header returns 401 with WWW-Authenticate
  - Test: invalid credentials return 401
  - Test: valid credentials pass through
  - Test: uses constant-time comparison
- [X] T011 [P] Add auth config validation tests in `internal/core/config/config_test.go`
  - Test: both empty is valid (disabled)
  - Test: both set is valid (enabled)
  - Test: only username set returns error
  - Test: only password set returns error
  - Test: `IsAuthEnabled()` returns correct values
  - Test: environment variables override config file values
- [X] T012 Run quickstart.md validation scenarios manually
  - Test all curl commands from quickstart.md
  - Verify browser authentication dialog works
  - Test health endpoint remains open

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: N/A - no setup required
- **Foundational (Phase 2)**: No dependencies - can start immediately
- **User Story 2 (Phase 3)**: Depends on Foundational (T001-T004)
- **User Story 1 (Phase 4)**: Depends on Foundational (T001-T004)
- **Polish (Phase 5)**: Depends on User Stories 1 and 2

### User Story Dependencies

- **User Story 2 (P1 - Config)**: Depends on Foundational phase only
- **User Story 1 (P1 - Auth)**: Depends on Foundational phase only (reads config directly)

Both user stories can be implemented in parallel after Foundational phase, but logically US2 (config) should be complete before testing US1 (auth).

### Within Each Phase

- Foundational tasks T002-T003 marked [P] can run in parallel
- T004 depends on T001-T003 (config methods must exist)
- US1 tasks T007→T008→T009 must be sequential
- Polish tasks T010-T011 marked [P] can run in parallel

### Parallel Opportunities

```
Phase 2 Parallel Group:
  T002, T003 (all [P])

Phase 5 Parallel Group:
  T010, T011 (all [P])
```

---

## Implementation Strategy

### MVP Scope (Recommended First Delivery)

1. Complete Phase 2 (Foundational) - ~4 tasks
2. Complete Phase 3 (User Story 2 - Config) - ~2 tasks
3. Complete Phase 4 (User Story 1 - Auth) - ~3 tasks

Total MVP: **9 tasks**

### Incremental Delivery

1. **Increment 1**: Foundational + Configuration (T001-T006)
   - Deliverable: Auth can be configured but not yet enforced
   - Verification: Service starts with valid config, rejects invalid config

2. **Increment 2**: Authentication Middleware (T007-T009)
   - Deliverable: Full authentication working
   - Verification: All quickstart.md scenarios pass

3. **Increment 3**: Polish (T010-T012)
   - Deliverable: Production-ready with tests
   - Verification: All tests pass, documentation complete

---

## File Summary

| File | Operation | Tasks |
|------|-----------|-------|
| `internal/core/config/config.go` | Modify | T001, T002, T003 |
| `cmd/cloud-sync/serve.go` | Modify | T004 |
| `config.toml` | Modify | T005 |
| `README.md` or `README_CN.md` | Modify | T006 |
| `internal/api/context/auth.go` | Create | T007, T008 |
| `internal/api/server.go` | Modify | T009 |
| `internal/api/context/auth_test.go` | Create | T010 |
| `internal/core/config/config_test.go` | Modify | T011 |

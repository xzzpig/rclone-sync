# Feature Specification: Refactoring Global Variable Dependencies

**Feature Branch**: `006-refactor-global-deps`  
**Created**: 2025-12-19  
**Status**: Draft  
**Input**: User description: "Currently, some modules in the system use public global variables for other modules, such as db, config, and logger. Please identify all such instances and plan modifications to avoid this usage pattern."

## Clarifications

### Session 2025-12-19

- Q: Strategy for handling dependency initialization failure (e.g., database connection failure)? → A: Fail-fast - Any core dependency (db/config) initialization failure will immediately terminate startup and return a clear error.
- Q: How to detect and handle circular dependencies? → A: Compile-time detection - Rely on the Go compiler to detect package circular imports, and break dependency loops through interface abstraction.
- Q: How to smoothly migrate existing test code? → A: Progressive migration - Update relevant tests immediately after refactoring each module, ensuring all tests pass with every commit.
- Q: What is the default log level when the Logger is not initialized? → A: Info level - Defaults to outputting Info and above logs (Info, Warn, Error).
- Q: Dependency injection implementation method? → A: Manual dependency injection - Manually create and assemble dependencies in main or startup code, without external frameworks.

## Overview

Currently, three public global variables in the system are directly accessed by multiple modules:

| Global Variable | Location | Usage Count | Using Modules |
| -------- | ---- | -------- | -------- |
| `db.Client` | `internal/core/db/db.go` | 2 | routes.go, setup_test.go |
| `config.Cfg` | `internal/core/config/config.go` | 5 | routes.go, server.go, migrate.go, migrate_test.go |
| `logger.L` | `internal/core/logger/logger.go` | 31 | Multiple modules (api, core/services, rclone, scheduler, runner, watcher, db) |

This usage of global variables leads to:
- High coupling between modules, making independent testing difficult.
- Unclear dependency relationships, resulting in poor code maintainability.
- Potential race conditions in concurrent scenarios.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Developer writing unit tests (Priority: P1)

As a developer, I want to easily inject mock dependencies when writing unit tests without modifying global state, so that I can write isolated and reliable unit tests.

**Why this priority**: This is the core value of the refactoring—improving code testability. If unit testing is not easy, code quality will be difficult to guarantee.

**Independent Test**: Can be verified by writing a unit test for any service using mock dependencies; the test should be able to run without depending on a real database or configuration.

**Acceptance Scenarios**:

1. **Given** a service function requiring database access, **When** a developer writes a unit test, **Then** a mock database client can be injected without modifying global variables.
2. **Given** a service function requiring configuration information, **When** a developer writes a unit test, **Then** custom configuration can be injected without depending on a config file.
3. **Given** a service function requiring logging, **When** a developer writes a unit test, **Then** a mock logger can be injected to verify log output.

---

### User Story 2 - Developer understanding module dependencies (Priority: P2)

As a developer, I want to clearly understand the dependencies of a function or struct by looking at its signature, so that I can quickly understand code dependencies and responsibility boundaries.

**Why this priority**: Clear dependency relationships aid in code maintenance and new feature development, forming the foundation of good code architecture.

**Independent Test**: Reviewing the constructor of any service should clearly show all external dependencies being passed in as parameters.

**Acceptance Scenarios**:

1. **Given** any service struct, **When** a developer reviews its constructor, **Then** db and config dependencies are explicitly passed as parameters.
2. **Given** a module's public interface, **When** a developer reads the interface definition, **Then** no implicit dependencies on global variables are found.

---

### User Story 3 - System running in various environments (Priority: P3)

As an operator, I want the system to be able to flexibly configure dependencies in different environments (development, testing, production), so that different implementations can be used based on environmental needs.

**Why this priority**: Although there may currently be only a single deployment environment, a good dependency injection architecture provides the foundation for future expansion.

**Independent Test**: Can be verified by starting the service with different configurations in different environments.

**Acceptance Scenarios**:

1. **Given** the application startup process, **When** starting in different environments, **Then** dependency implementations suitable for that environment can be flexibly injected.
2. **Given** a testing environment, **When** running integration tests, **Then** an in-memory database can be used instead of a real database.

---

### Edge Cases

- **Dependency Initialization Failure Handling**: Adopt a fail-fast strategy; any core dependency (db/config) initialization failure immediately terminates startup and returns a clear error message.
- **Circular Dependency Detection and Handling**: Rely on the Go compiler to detect package circular imports, and break dependency loops through interface abstraction.
- **Test Code Migration Strategy**: Adopt progressive migration; update relevant tests immediately after refactoring each module, ensuring all tests pass with every commit.

## Requirements *(mandatory)*

### Functional Requirements

**db Module Refactoring**:
- **FR-001**: The system MUST remove the `db.Client` global variable.
- **FR-002**: The system MUST pass the database client to required modules via constructors or parameters.
- **FR-003**: All modules using `db.Client` MUST be updated to accept dependency injection.

**config Module Refactoring**:
- **FR-004**: The system MUST remove the `config.Cfg` global variable.
- **FR-005**: The system MUST pass configuration objects to required modules via constructors or parameters.
- **FR-006**: All modules using `config.Cfg` MUST be updated to accept dependency injection.

**logger Module Refactoring**:
- **FR-007**: The system MUST remove the `logger.L` global variable.
- **FR-008**: The logger module MUST provide an `Init` method for initializing the Logger (configuring log levels, output formats, etc.).
- **FR-009**: The logger module MUST provide a public Getter method for other modules to obtain a logger instance.
- **FR-010**: The Getter method MUST return a default Logger (Info level, outputting Info, Warn, Error logs) if the Logger is not initialized, rather than panicking or returning nil.
- **FR-011**: The logger module MUST provide a method to get a named Logger (e.g., `Named(name string)`).
- **FR-012**: Logger names MUST support hierarchical naming separated by "." (e.g., `core.runner`, `service.task`, `api.file`).

**General Requirements**:
- **FR-013**: The application entry point MUST be responsible for creating and assembling all dependencies.
- **FR-014**: Existing test code MUST be updated to adapt to the new dependency acquisition methods.
- **FR-015**: The system MUST maintain full compatibility with existing functionality; refactoring should not change any business logic.

### Affected Modules

The following modules need to be updated to accept dependency injection:

**Modules using db.Client**:
- `internal/api/routes.go`
- `internal/api/handlers/setup_test.go`

**Modules using config.Cfg**:
- `internal/api/routes.go`
- `internal/api/server.go`
- `internal/core/db/migrate.go`
- `internal/core/db/migrate_test.go`

**Modules using logger.L** (31 instances, major modules):
- `internal/api/server.go`
- `internal/api/sse/broadcaster.go`
- `internal/core/services/job_service.go`
- `internal/core/scheduler/scheduler.go`
- `internal/core/runner/runner.go`
- `internal/core/watcher/watcher.go`
- `internal/core/watcher/recursive.go`
- `internal/core/db/db.go`
- `internal/core/db/migrate.go`
- `internal/rclone/sync.go`
- Multiple test files

### Key Entities

- **DatabaseClient**: Database client providing data access capabilities, currently using ent.Client.
- **Config**: Application configuration containing items for server, database, logging, etc.
- **Logger**: Logger providing structured logging capabilities.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All public global variables (`db.Client`, `config.Cfg`, `logger.L`) are removed from the codebase.
- **SC-002**: db and config dependencies are obtained through explicit dependency injection; logger is obtained through a public Getter method.
- **SC-003**: All existing test cases still pass after refactoring.
- **SC-004**: Developers can write unit tests for any service without depending on global state.

## Assumptions

- The current codebase uses the Go language and will adopt manual dependency injection (constructor injection or method parameter injection) without introducing external DI frameworks.
- Logger usage is the most widespread (31 instances) and will be the part with the largest refactoring workload.
- Refactoring can be performed in stages, prioritizing global variables with fewer usages (`db.Client`), then `config.Cfg`, and finally `logger.L`.
- Direct assignments to global variables in test code will be replaced by using test helper functions to create dependencies.

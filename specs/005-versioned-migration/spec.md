# Feature Specification: Versioned Database Migration

**Feature Branch**: `005-versioned-migration`  
**Created**: 2025-12-18  
**Status**: Draft  
**Input**: User description: "Currently, the database only supports auto-migration, which is very unfriendly for adjusting database structure and content during application version updates after public release. Therefore, support for versioned migration needs to be added."

## Background

The current system uses the auto-migration feature of the ent ORM (`Client.Schema.Create`), which has the following issues:

1. **Irreversible Operations**: Auto-migration can only move "forward" and cannot roll back to previous versions.
2. **Data Loss Risk**: Auto-migration might directly delete columns or tables, leading to data loss.
3. **No Version Tracking**: It is impossible to know which migration version the current database is on.
4. **Unable to Execute Data Migration**: It only handles schema changes and cannot migrate data synchronously during schema changes.

## Clarifications

### Session 2025-12-19

- Q: Does versioned migration need to support rollback (downgrade) to previous database versions? → A: No rollback needed; only forward migration is supported. Rely on backups for recovery in case of failure.
- Q: Should the system automatically back up the database before executing migrations? → A: No automatic backup; backups are the responsibility of the user or external tools.
- Q: How should the application handle a database migration failure? → A: Prevent startup and report an error, displaying migration failure information and requiring user intervention.
- Q: How should the application access migration scripts at runtime? → A: Embedded in the binary file; use Go embed to compile migration scripts into the application.
- Q: How does the system handle the upgrade process to versioned migration for existing users already using auto-migration? → A: Automatic upgrades are not supported. If business tables are detected but no migration records exist, an error is reported directly, requiring the user to delete the old database and start over.
- Q: Does the system need to handle locking mechanisms for concurrent migrations from multiple instances? → A: No; it is assumed to be a single-machine application, and the user ensures only a single instance is running.
- Q: Should the system output detailed migration execution logs? → A: Yes; output detailed execution processes and error information to the application's standard log stream.
- Q: Should the system support using auto-migration in specific scenarios? → A: Yes; unit tests specify the mode via InitDB parameters, and normal application startup sets it via specific configurations.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Safe Database Migration During App Version Upgrade (Priority: P1)

As an application administrator, I want the database to migrate safely to the new structure when the application is upgraded to a new version, while retaining all existing data, so that users do not lose any historical records.

**Why this priority**: This is the core requirement of versioned migration, directly affecting user data security and the upgrade experience.

**Independent Test**: Can be verified by creating an old version database containing test data, then running the application upgrade to ensure data is fully retained and the structure is correctly updated.

**Acceptance Scenarios**:

1. **Given** a user is running an old version of the application with stored data, **When** the user upgrades to a new version and starts the application, **Then** the application automatically detects and executes all pending migration scripts, and data is fully retained.
2. **Given** the database is already at the latest version, **When** the user starts the application, **Then** the application detects no migration is needed and starts normally without additional operations.
3. **Given** an error occurs during the migration process, **When** a migration script fails to execute, **Then** the application prevents startup and displays detailed error information, requiring the user to restore from backup or contact support.

---

### User Story 2 - Developer Creates New Migration Scripts (Priority: P2)

As a developer, I want to be able to easily create new migration scripts to modify the database structure or migrate data, so that these changes can be included in new application versions.

**Why this priority**: This is a key part of the development workflow but is transparent to end-users.

**Independent Test**: Can be verified by running the migration generation command to ensure migration files are generated in the correct format.

**Acceptance Scenarios**:

1. **Given** a developer modifies the ent schema, **When** the developer runs the migration generation command, **Then** the system automatically generates migration files containing the schema differences.
2. **Given** a developer needs to perform data migration, **When** the developer creates a migration file and adds data migration logic, **Then** data is correctly transformed when the migration is executed.

---

### User Story 3 - View Migration Status (Priority: P3)

As an application administrator, I want to be able to view the current migration status of the database to understand the database version and pending migrations.

**Why this priority**: This is an auxiliary function that helps diagnose problems but does not affect core migration functionality.

**Independent Test**: Can be verified by running a status query command to ensure the correct migration history and current version are displayed.

**Acceptance Scenarios**:

1. **Given** the database has executed several migrations, **When** the administrator views the migration status, **Then** a list of executed migration versions and the current version are displayed.
2. **Given** there are pending migrations, **When** the administrator views the migration status, **Then** the number and names of pending migrations are displayed.

---

### User Story 4 - Developer Rapid Iteration Development (Priority: P2)

As a developer, I want to be able to use auto-migration mode during local development and while running unit tests, so that I can iterate quickly without generating migration scripts every time.

**Why this priority**: This is a key requirement for development efficiency, equal in importance to User Story 2, but does not affect production environment users.

**Independent Test**: Can be verified by switching the migration mode via configuration to ensure the system correctly uses either auto-migration or versioned migration.

**Acceptance Scenarios**:

1. **Given** a developer is in a local development environment, **When** the developer configures the system to use auto-migration mode and starts the application, **Then** the system uses ent auto-migration to synchronize the database structure.
2. **Given** a developer runs unit tests, **When** the test code initializes the database and specifies auto-migration mode, **Then** the system uses ent auto-migration to create the test database structure.
3. **Given** the production environment is configured for versioned migration mode (default), **When** the application starts, **Then** the system uses versioned migration to execute database updates.

---

### Edge Cases

- How to handle migration script execution failure?
  - The system prevents the application from starting, displays detailed error information, and requires the user to restore from backup or contact support.
- How to handle when the migration sequence is disrupted (e.g., manual modification of the migration record table)?
  - The system should detect inconsistency and report an error.
- How to handle when business tables are detected but no migration records exist?
  - The system prevents startup and reports an error (automatically detected by golang-migrate as a table conflict), requiring the user to delete the old database and start over.
- How to handle when the database structure produced by auto-migration in a development environment is incompatible with versioned migration scripts?
  - Developers are responsible for generating migration scripts immediately after schema changes to keep both methods synchronized; databases under auto-migration mode are not guaranteed to be compatible with production environments.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system must automatically detect and execute all pending migrations upon application startup.
- **FR-002**: The system must record the current migration version status of the database (version number, dirty state).
- **FR-003**: The system must execute migrations in version order; skipping is not allowed.
- **FR-004**: The system must support schema migration (adding/modifying/deleting tables and columns).
- **FR-005**: The system must support data migration (transforming existing data during schema changes).
- **FR-006**: The system must prevent application startup and display detailed error information if a single migration fails.
- **FR-007**: Developers must be able to use the atlas CLI to generate new migration scripts.
- **FR-008**: Developers must be able to use the atlas CLI to view migration status.
- **FR-009**: The system must maintain compatibility with existing ent schema definitions.
- **FR-010**: The system must prevent startup and report an error when business tables are detected but no migration records exist (implemented via natural conflicts when golang-migrate attempts to create existing tables).
- **FR-011**: The system must embed migration scripts (only `.up.sql`) into the application binary.
- **FR-012**: The system only supports forward migration and does not support rollback operations (implemented by excluding `.down.sql` files via embed pattern matching).
- **FR-013**: The system must output detailed migration execution logs (including SQL statements and error details) to the standard log stream.
- **FR-014**: The system must support two migration modes: versioned migration (`versioned`) and auto-migration (`auto`).
- **FR-015**: The system must select the migration mode based on configuration items at application startup, defaulting to versioned migration.
- **FR-016**: The system must provide a programming interface to allow explicit specification of the migration mode during database initialization (for unit tests).

### Key Entities

- **Migration Record**: Represents the execution record of a migration, including version number, name, execution time, and execution status.
- **Migration Script**: Represents database changes for a version, containing upgrade operations (only contains `up` operations, no `down` operations).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: When a user upgrades the application, 100% of historical data is retained and accessible.
- **SC-002**: When an error is encountered during migration execution, the application prevents startup, and the database remains in the state it was in before the failure.
- **SC-003**: Developers can complete the creation of a new migration script within 5 minutes.
- **SC-004**: The migration check process at application startup is transparent to the user, requiring no additional configuration or operations.
- **SC-005**: Clear error reporting when an incompatible database is detected, prompting the user that the old database needs to be deleted.
- **SC-006**: The application is distributed as a single executable file containing all migration scripts.
- **SC-007**: Developers can switch between auto-migration and versioned migration modes via configuration or programming interface.

## Assumptions

- Use Atlas CLI to generate versioned migration scripts from ent schema, and use golang-migrate to execute migrations at application startup.
- Migration scripts are stored as SQL files in the code repository and embedded into the binary during build.
- SQLite database supports basic atomic operations.
- Only supports new databases or databases already under versioned migration; upgrading from auto-migration is not supported.
- Database backup is the responsibility of the user or deployment scripts; the application does not provide automatic backup functions.
- The application deployment mode is single-machine single-instance; concurrent migration execution by multiple instances is not considered.
- Auto-migration mode in development/test environments is only for rapid iteration and does not guarantee data compatibility with the production environment.
- Developers should generate versioned migration scripts immediately after schema changes to ensure structural consistency between the two migration modes.

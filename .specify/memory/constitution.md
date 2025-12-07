<!--
SYNC IMPACT REPORT
Version change: 1.0.1 -> 1.0.2
Modified principles: Governance (Compliance formatting), Typo fixes
Added sections: None
Removed sections: None
Templates requiring updates: ✅ None
Follow-up TODOs: None
-->
# Rclone Cloud Sync Manager Constitution

## Core Principles

### I. Rclone-First Architecture
All sync operations MUST be implemented through rclone as a library. Direct filesystem operations are prohibited except for configuration and state management. Rclone's configuration, remotes, and sync commands are the single source of truth for all cloud operations.

### II. Web-First Interface
All user interactions MUST be through the Web UI. CLI is for development and debugging only. The Web UI provides complete functionality for managing remotes, tasks, and monitoring sync operations.

### III. Test-Driven Development (NON-NEGOTIABLE)
All features MUST be implemented with tests first. Unit tests for internal logic, integration tests for API endpoints, and end-to-end tests for user workflows. Red-Green-Refactor cycle is strictly enforced.

### IV. Independent User Stories
Each user story (Manage Cloud Connections, Create Sync Tasks, Real-time/Scheduled Sync, Dashboard) MUST be independently implementable and testable. No story should depend on another for core functionality.

### V. Observability and Reliability
All operations MUST be logged with structured logging. Sync operations MUST be resumable and handle network interruptions gracefully. The system MUST provide real-time status updates for all active operations.

## Technical Constraints

### Technology Stack
- Backend: Go (latest stable) with Gin web framework
- Frontend: SolidJS with TypeScript
- Database: SQLite with Ent ORM
- Sync Engine: rclone as Go library
- Real-time Updates: Server-Sent Events (SSE)
- File Watching: fsnotify
- Configuration: TOML files with Viper

### Performance Requirements
- Real-time sync MUST trigger within 30 seconds of file changes
- System MUST handle thousands of files without memory leaks
- Web UI MUST remain responsive during long-running sync operations

### Security Requirements
- All cloud credentials MUST be encrypted at rest
- No credentials in logs or error messages
- Secure communication between frontend and backend

## Development Workflow

### Code Quality
- All code MUST pass golangci-lint checks
- All PRs require review and passing tests
- Documentation updates required for API changes

### Testing Strategy
- Unit tests for all internal packages
- Integration tests for all API endpoints
- End-to-end tests for critical user workflows
- Performance tests for sync operations

## Governance

This constitution supersedes all other development practices. Amendments REQUIRE documentation, team approval, and migration plan. Versioning follows Semantic Versioning (MAJOR.MINOR.PATCH). All code reviews MUST verify compliance with these principles. Complexity MUST be justified with clear business value.

**Version**: 1.0.2 | **Ratified**: 2025-12-04 | **Last Amended**: 2025-12-07

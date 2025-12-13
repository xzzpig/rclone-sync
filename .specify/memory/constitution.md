<!--
SYNC IMPACT REPORT
Version change: 1.0.2 -> 1.1.0
Modified principles: Technical Constraints, Development Workflow
Added sections: VI. Modern Component Architecture, VII. Accessibility & UX Standards, VIII. Performance & Optimistic UI
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

### VI. Modern Component Architecture
Frontend MUST use SolidJS with a component-based architecture (Atomic Design). Components MUST be small, composable, and use Tailwind CSS for styling. Logic MUST be decoupled from presentation using SolidJS primitives (Signals, Memos). Global state MUST be managed via granular stores, avoiding monolithic context providers where possible.

### VII. Accessibility and UX Standards
All UI components MUST be WCAG 2.1 AA compliant. Interfaces MUST be fully responsive (Mobile-First) and support keyboard navigation. Semantic HTML MUST be used over `div` soups. Interactive elements MUST provide clear focus states and ARIA attributes where semantic elements are insufficient.

### VIII. Performance and Optimistic UI
The interface MUST implement Optimistic UI patterns for mutations to ensure perceived instant latency. Network waterfalls MUST be minimized by leveraging parallel data fetching. Reactive computations MUST be efficient to avoid unnecessary re-renders. Lazy loading MUST be used for clean code splitting on routes and heavy assets.

## Technical Constraints

### Technology Stack
- Backend: Go (latest stable) with Gin web framework
- Frontend: SolidJS with TypeScript
- Styling: Tailwind CSS
- Database: SQLite with Ent ORM
- Sync Engine: rclone as Go library
- Real-time Updates: Server-Sent Events (SSE)
- File Watching: fsnotify
- Configuration: TOML files with Viper

### Performance Requirements
- Real-time sync MUST trigger within 30 seconds of file changes
- System MUST handle thousands of files without memory leaks
- Web UI MUST remain responsive during long-running sync operations
- First Contentful Paint (FCP) MUST be under 1.5s

### Security Requirements
- All cloud credentials MUST be encrypted at rest
- No credentials in logs or error messages
- Secure communication between frontend and backend
- Content Security Policy (CSP) MUST be strictly enforced

## Development Workflow

### Code Quality
- All code MUST pass golangci-lint checks
- Frontend code MUST pass ESLint and Prettier checks
- All PRs require review and passing tests
- Documentation updates required for API changes

### Testing Strategy
- Unit tests for all internal packages
- Integration tests for all API endpoints
- End-to-end tests for critical user workflows
- Performance tests for sync operations
- Accessibility audits for UI components

## Governance

This constitution supersedes all other development practices. Amendments REQUIRE documentation, team approval, and migration plan. Versioning follows Semantic Versioning (MAJOR.MINOR.PATCH). All code reviews MUST verify compliance with these principles. Complexity MUST be justified with clear business value.

**Version**: 1.1.0 | **Ratified**: 2025-12-04 | **Last Amended**: 2025-12-07

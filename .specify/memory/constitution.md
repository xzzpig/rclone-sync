<!--
SYNC IMPACT REPORT
Version change: 1.5.0 -> 1.6.0
Modified principles: None
Added sections: Principle XI - Database Index Strategy
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

### III. Test-Driven Development (Backend Mandatory)

Backend features MUST be implemented with tests first. Unit tests for internal logic and integration tests for API endpoints are NON-NEGOTIABLE. Frontend unit tests are currently NOT REQUIRED. End-to-end tests for critical user workflows are maintained. Red-Green-Refactor cycle is strictly enforced for backend.

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

### IX. Internationalization (i18n) Standards

All user-visible text MUST be externalized into translation resource files—hardcoded strings are prohibited. The system MUST support Chinese (zh-CN) and English (en) with an extensible architecture for future languages. Frontend MUST use Paraglide for compile-time type-safe translations; Backend MUST use go-i18n with embedded TOML locale files. Language detection MUST follow priority: user preference (localStorage) > Accept-Language header > English fallback. Translation keys MUST be organized hierarchically by feature/module namespace. Missing translations MUST fall back to English gracefully—never display raw keys to users. All API error messages MUST be translatable via I18nError pattern. Date, time, and number formatting MUST respect the user's locale settings.

### X. Schema-First API Contract

All API interactions MUST be defined via GraphQL Schema first. Code generation MUST be used to ensure type safety across Backend (Go) and Frontend (TypeScript). Ad-hoc API endpoints are prohibited. Resolvers MUST implement the generated interfaces strictly.

### XI. Database Index Strategy

All database CRUD operations MUST be reviewed for appropriate indexing needs. When adding new queries, mutations, or entity relationships, developers MUST evaluate whether database indexes are required to maintain query performance. Indexes MUST be added for:
- Fields used in WHERE clauses for frequent queries
- Foreign key columns used in JOIN operations
- Fields used in ORDER BY for paginated results
- Unique constraint columns

Index additions MUST be implemented via migration files using golang-migrate. Over-indexing MUST be avoided—each index adds write overhead. Index decisions MUST be documented in the corresponding migration file comments.

## Technical Constraints

### Technology Stack

- Backend: Go (latest stable) with Gin web framework and gqlgen
- Frontend: SolidJS with TypeScript and urql
- Styling: Tailwind CSS
- Database: SQLite with Ent ORM
- Database Migration: golang-migrate (runtime execution) + Atlas CLI (schema diff generation)
- Sync Engine: rclone as Go library
- Real-time Updates: GraphQL Subscriptions (WebSocket)
- File Watching: fsnotify
- Configuration: TOML files with Viper
- Frontend i18n: Paraglide (inlang)
- Backend i18n: go-i18n with TOML locales
- Frontend GraphQL: gql.tada (type-safe queries) + urql (client)

### Performance Requirements

- Real-time sync MUST trigger within 30 seconds of file changes
- System MUST handle thousands of files without memory leaks
- Web UI MUST remain responsive during long-running sync operations
- First Contentful Paint (FCP) MUST be under 1.5s
- Language switch MUST complete within 1 second without page reload
- Database migration check MUST complete in under 1 second at application startup

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
- Translation keys MUST be sorted alphabetically (use scripts/sort-i18n-keys.js)

### Testing Strategy

- Unit tests for all internal backend packages
- Integration tests for all API endpoints
- End-to-end tests for critical user workflows
- Performance tests for sync operations
- Accessibility audits for UI components
- i18n coverage tests to ensure all UI strings are translated

## Governance

This constitution supersedes all other development practices. Amendments REQUIRE documentation, team approval, and migration plan. Versioning follows Semantic Versioning (MAJOR.MINOR.PATCH). All code reviews MUST verify compliance with these principles. Complexity MUST be justified with clear business value.

**Version**: 1.6.0 | **Ratified**: 2025-12-04 | **Last Amended**: 2025-12-30

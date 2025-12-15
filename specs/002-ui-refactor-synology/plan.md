# Implementation Plan: UI Refactor to Synology Cloud Sync UX

**Branch**: `002-ui-refactor-synology` | **Date**: 2025-12-07 | **Spec**: [specs/002-ui-refactor-synology/spec.md](./spec.md)
**Input**: Feature specification from `specs/002-ui-refactor-synology/spec.md`

## Summary

The goal is to completely refactor the UI to mimic the Synology Cloud Sync user experience, shifting from a generic layout to a connection-centric sidebar navigation. The frontend will be rebuilt using SolidJS and specific components from Solid-UI (installed via `solidui-cli`) to achieve an "Enterprise" look and feel. The backend will remain largely consistent but may require API adjustments to support the new data loading patterns (lazy loading overview, specific history filtering).

## Technical Context

**Language/Version**: Go (Latest Stable), TypeScript 5.9
**Primary Dependencies**: Gin (Go), SolidJS (Frontend), Tailwind CSS, Solid-UI (Component Library), Rclone (Sync Engine)
**Storage**: SQLite with Ent ORM
**Testing**: Go standard testing (Backend)
**Target Platform**: Web Application (Linux/Docker host)
**Project Type**: Web application
**Performance Goals**: < 1.5s FCP, Instant navigation (< 100ms perceived), Optimistic UI updates
**Constraints**: Must use Rclone as a defined library; Mobile-responsive (stack navigation); WCAG 2.1 AA Accessibility
**Scale/Scope**: ~10-15 high-fidelity screens, supports manageable number of connections (1-20) and tasks (1-100)

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

- [x] **I. Rclone-First Architecture**: Feature relies entirely on rclone for syncing.
- [x] **II. Web-First Interface**: This IS the web interface refactor.
- [x] **III. Test-Driven Development**: Plan includes TDD steps for new API endpoints and component logic.
- [x] **IV. Independent User Stories**: User stories are split by priority and function (Connection vs Task).
- [x] **V. Observability**: Layout includes dedicated "Log" and "History" views.
- [x] **VI. Modern Component Architecture**: Uses SolidJS + Solid-UI (Atomic Design).
- [x] **VII. Accessibility**: Solid-UI components (based on Radix/Headless concepts) generally support a11y; will verify.
- [x] **VIII. Performance**: Optimistic UI and Skeleton loading explicitly requested.

## Project Structure

### Documentation (this feature)

```text
specs/002-ui-refactor-synology/
├── plan.md              # This file
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Backend
cmd/cloud-sync/         # Entry point
internal/
├── api/                # Gin Routes (Updated)
├── core/               # Core Logic
└── rclone/             # Rclone Integration

# Frontend
web/
├── src/
│   ├── components/
│   │   ├── ui/         # Solid-UI installed components (Button, Card, etc.)
│   │   └── common/     # Shared app components (Layout, Wrappers)
│   ├── modules/        # Business Modules (View + Logic)
│   │   ├── connections/
│   │   ├── tasks/
│   │   └── history/
│   ├── layouts/        # AppShell (Sidebar layout)
│   └── lib/            # Utilities
├── tailwind.config.js
└── components.json     # Solid-UI config
```

**Structure Decision**: Option 2: Web application (Separate Go Backend / SolidJS Frontend folders)

## Complexity Tracking

| Violation             | Why Needed                 | Simpler Alternative Rejected Because                                                    |
| --------------------- | -------------------------- | --------------------------------------------------------------------------------------- |
| Solid-UI CLI          | User Request / Consistency | Manual copying is error-prone and harder to update                                      |
| Modules-based folders | Scalability                | Flat structure becomes unmanageable with many distinct domains (Connection, Task, Logs) |

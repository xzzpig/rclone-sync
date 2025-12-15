# Implementation Plan: Multi-Language Support (i18n)

**Branch**: `003-i18n-support` | **Date**: 2025-12-14 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-i18n-support/spec.md`

## Summary

Add internationalization (i18n) support to the Cloud Sync application, allowing users to switch interface languages between Chinese (Simplified) and English. Features include:

- Frontend uses **paraglide-js** compile-time i18n library (zero runtime overhead, type-safe)
- Backend uses **go-i18n** library (supports plural rules, message templates)
- Backend Go API supports localized responses based on `Accept-Language` header
- Automatic browser language detection and localStorage persistence
- Dynamic date/time formatting support for Chinese and English locales

## Technical Context

**Language/Version**:

- Backend: Go 1.25 with Gin web framework
- Frontend: SolidJS 1.9.10 with TypeScript 5.9.3

**Primary Dependencies**:

- Frontend: `@inlang/paraglide-js`, `@inlang/paraglide-vite`, SolidJS, @solidjs/router, @tanstack/solid-query, date-fns, Vite
- Backend: `github.com/nicksnyder/go-i18n/v2`, Gin, Ent ORM, SQLite, Viper

**Storage**: SQLite (no schema changes needed for i18n - language preference stored in localStorage)

**Testing**:

- Backend: Go test with testify

**Target Platform**: Web (Linux server backend, modern browsers frontend)

**Project Type**: Web application (frontend + backend)

**Performance Goals**:

- Language switch must complete within 1 second
- First Contentful Paint (FCP) under 1.5s (per constitution)

**Constraints**:

- No database schema changes for i18n
- Translation files bundled with frontend build
- Backend translations embedded in binary

**Scale/Scope**:

- 2 languages (zh-CN, en)
- 24 UI components to localize _(see [tasks.md](./tasks.md) T030-T053)_
- ~20 API endpoints with user-facing messages

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle                         | Requirement                    | Compliance  | Notes                                    |
| :-------------------------------- | :----------------------------- | :---------- | :--------------------------------------- |
| I. Rclone-First                   | Sync via rclone library        | ✅ N/A      | i18n doesn't affect sync operations      |
| II. Web-First Interface           | UI through Web only            | ✅ PASS     | Language switcher in Web UI sidebar      |
| III. Test-Driven Development      | Tests first for all features   | ✅ REQUIRED | Must add i18n tests for frontend/backend |
| IV. Independent User Stories      | Stories independently testable | ✅ PASS     | Each i18n story can be tested separately |
| V. Observability                  | Structured logging             | ✅ PASS     | No changes to logging required           |
| VI. Modern Component Architecture | SolidJS + Atomic Design        | ✅ PASS     | LanguageSwitcher as composable component |
| VII. Accessibility & UX           | WCAG 2.1 AA, keyboard nav      | ✅ REQUIRED | Language switcher must be accessible     |
| VIII. Performance & Optimistic UI | Instant perceived latency      | ✅ REQUIRED | Lazy-load translations, instant switch   |

**Security Requirements**:

- No credentials in translations ✅
- CSP compliance ✅ (translations bundled, no external loading)

**Gate Result**: ✅ PASS - May proceed to Phase 0

## Project Structure

### Documentation (this feature)

```text
specs/003-i18n-support/
├── plan.md              # This file
├── research.md          # Phase 0 output - i18n library research
├── data-model.md        # Phase 1 output - translation resource structure
├── quickstart.md        # Phase 1 output - developer setup guide
├── contracts/           # Phase 1 output - API localization contracts
│   └── openapi.yaml     # Updated API spec with Accept-Language
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
# Option 2: Web application (frontend + backend)
internal/                    # Backend Go code
├── api/
│   ├── context/
│   │   └── middleware.go   # Add Accept-Language parsing middleware
│   └── handlers/
│       └── error.go        # Localized error messages
├── i18n/                   # NEW: Backend i18n package (go-i18n)
│   ├── i18n.go             # Bundle init, T/TWithData/TPlural functions
│   ├── keys.go             # Message ID constants
│   ├── locales/            # Translation files (embedded via go:embed)
│   │   ├── en.toml         # English translations (TOML format)
│   │   └── zh-CN.toml      # Chinese translations (TOML format)
│   └── i18n_test.go        # Translation tests

web/                        # Frontend SolidJS code
├── project.inlang/         # NEW: Inlang project config (paraglide-js)
│   ├── settings.json       # Language settings
│   └── messages/
│       ├── en.json         # English translations
│       └── zh-CN.json      # Chinese translations
├── src/
│   ├── paraglide/          # AUTO-GENERATED by paraglide-js
│   │   ├── messages.js     # Compiled translation functions
│   │   └── runtime.js      # Runtime utilities
│   ├── store/
│   │   └── locale.tsx      # NEW: Locale store (Context/Provider pattern)
│   ├── components/
│   │   └── common/
│   │       └── LanguageSwitcher.tsx  # NEW: Language toggle component
│   └── lib/
│       └── api.ts          # Add Accept-Language header to requests
└── vite.config.ts          # Add paraglide plugin
```

**Structure Decision**: Web application structure with paraglide-js for frontend (compile-time i18n with zero runtime overhead) and go-i18n for backend (supports plurals, message templates, embedded TOML files). Frontend translations are JSON files compiled to typed functions. **Frontend uses Locale Store (Context/Provider pattern) for language state management**, consistent with existing TaskProvider/HistoryProvider patterns. Backend translations are TOML files embedded via go:embed and loaded by go-i18n Bundle. Development mode supports hot-reload from local files.

## Complexity Tracking

No violations requiring justification. Design follows existing architecture patterns.

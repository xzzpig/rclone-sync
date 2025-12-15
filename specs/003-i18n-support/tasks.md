# Tasks: Multi-Language Support (i18n)

**Input**: Design documents from `/specs/003-i18n-support/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓, quickstart.md ✓

**Tests**: Tests are included as this project follows TDD approach per the project constitution.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## User Stories Summary

| ID  | Title                             | Priority | Description                                                                  |
| --- | --------------------------------- | -------- | ---------------------------------------------------------------------------- |
| US1 | Automatic Language Detection      | P1       | Automatically select the display language based on browser language settings |
| US2 | Manual Language Switching         | P1       | Users can manually change the language and persist it to localStorage        |
| US3 | Full UI Localization              | P2       | All user-visible text supports multi-language display                        |
| US4 | Dynamic Content Localization      | P3       | Date and time formats are localized according to language region settings    |
| US5 | Backend API Response Localization | P2       | User-visible information returned by the API supports multi-languages        |

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize frontend and backend i18n infrastructure

- [x] T001 [P] Install paraglide-js dependencies in web/package.json (`@inlang/paraglide-js`, `@inlang/paraglide-vite`)
- [x] T002 [P] Install go-i18n dependencies (`go get github.com/nicksnyder/go-i18n/v2/i18n github.com/BurntSushi/toml golang.org/x/text/language`)
- [x] T003 Create Inlang project configuration in web/project.inlang/settings.json
- [x] T004 [P] Configure Vite plugin for paraglide-js in web/vite.config.ts
- [x] T005 [P] Create backend i18n directory structure (internal/i18n/, internal/i18n/locales/)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core i18n infrastructure, must be completed before all user stories

**⚠️ CRITICAL**: User stories cannot begin before this phase is complete

### Frontend Foundation

- [x] T006 Create English translation messages file in web/project.inlang/messages/en.json
- [x] T007 Create Chinese translation messages file in web/project.inlang/messages/zh-CN.json
- [x] T008 Run paraglide-js initial compilation to generate web/src/paraglide/

### Backend Foundation

- [x] T009 [P] Create i18n bundle initialization in internal/i18n/i18n.go
- [x] T010 [P] Create message key constants in internal/i18n/keys.go
- [x] T011 [P] Create English translation file in internal/i18n/locales/en.toml
- [x] T012 [P] Create Chinese translation file in internal/i18n/locales/zh-CN.toml
- [x] T013 Create I18nError type and helper functions in internal/i18n/i18n.go
- [x] T014 Create context helper functions (WithLocalizer, LocalizerFromContext, Ctx) in internal/i18n/i18n.go
- [x] T015 Create LocaleMiddleware in internal/api/context/middleware.go
- [x] T016 Create I18nErrorMiddleware in internal/api/context/middleware.go
- [x] T017 Register i18n middlewares in internal/api/server.go
- [x] T018 Call i18n.Init() in application startup (cmd/cloud-sync/serve.go)

### Tests for Foundation

- [x] T019 [P] Write i18n package tests in internal/i18n/i18n_test.go
- [x] T019b [P] Write translation fallback test to verify fallback behavior for missing translation keys in internal/i18n/i18n_test.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Automatic Language Detection (Priority: P1) 🎯 MVP

**Goal**: When a user visits for the first time, the system automatically selects the appropriate display language based on browser language settings.

**Independent Test**: Change browser language settings and refresh the page to verify that the UI text is displayed in the corresponding language.

### Implementation for User Story 1

- [x] T021 [US1] Create Locale Store (Context/Provider) in web/src/store/locale.tsx
- [x] T022 [US1] Implement detectLanguage() function with browser language detection
- [x] T023 [US1] Add LocaleProvider to App root in web/src/App.tsx
- [x] T024 [US1] Update API client to use locale callback for Accept-Language header in web/src/lib/api.ts

**Checkpoint**: User Story 1 should now detect and apply browser language automatically

---

## Phase 4: User Story 2 - Manual Language Switching (Priority: P1)

**Goal**: Users can manually change the language via a language switcher, with the selection persisted to localStorage.

**Independent Test**: Click the language switcher, verify the interface immediately switches language, and the setting persists after refreshing the page.

### Implementation for User Story 2

- [x] T026 [US2] Create LanguageSwitcher component in web/src/components/common/LanguageSwitcher.tsx
- [x] T027 [US2] Add LanguageSwitcher to Sidebar (next to ModeToggle) in web/src/modules/core/components/Sidebar.tsx
- [x] T028 [US2] Implement localStorage persistence in Locale Store setLocale action (Note: Must ensure state updates do not trigger full page reload to satisfy FR-009)
- [x] T029 [US2] Add accessibility attributes (ARIA labels) to LanguageSwitcher

**Checkpoint**: User Story 2 should allow manual language switching with persistence

---

## Phase 5: User Story 3 - Full UI Localization (Priority: P2)

**Goal**: All user-visible text in the application supports multi-language display.

**Independent Test**: After switching languages, navigate through all pages and verify that all text is correctly displayed in the selected language.

### Implementation for User Story 3

> **Note**: This phase involves updating many existing components. Tasks are organized by module.

#### Core Module

- [x] T030 [P] [US3] Localize Sidebar navigation labels in web/src/modules/core/components/Sidebar.tsx
- [x] T031 [P] [US3] Localize MobileHeader in web/src/modules/core/components/MobileHeader.tsx
- [x] T032 [P] [US3] Localize WelcomeView in web/src/modules/core/views/WelcomeView.tsx
- [x] T033 [P] [US3] Localize RecentActivity in web/src/modules/core/components/RecentActivity.tsx
- [x] T034 [P] [US3] Localize StatCard in web/src/modules/core/components/StatCard.tsx

#### Connections Module

- [x] T035 [P] [US3] Localize Overview view in web/src/modules/connections/views/Overview.tsx
- [x] T036 [P] [US3] Localize Tasks view in web/src/modules/connections/views/Tasks.tsx
- [x] T037 [P] [US3] Localize History view in web/src/modules/connections/views/History.tsx
- [x] T038 [P] [US3] Localize Log view in web/src/modules/connections/views/Log.tsx
- [x] T039 [P] [US3] Localize Settings view in web/src/modules/connections/views/Settings.tsx
- [x] T040 [P] [US3] Localize AddConnectionDialog in web/src/modules/connections/components/AddConnectionDialog.tsx
- [x] T041 [P] [US3] Localize CreateTaskWizard in web/src/modules/connections/components/CreateTaskWizard.tsx
- [x] T042 [P] [US3] Localize EditTaskDialog in web/src/modules/connections/components/EditTaskDialog.tsx
- [x] T043 [P] [US3] Localize TaskSettingsForm in web/src/modules/connections/components/TaskSettingsForm.tsx
- [x] T044 [P] [US3] Localize DynamicConfigForm in web/src/modules/connections/components/DynamicConfigForm.tsx
- [x] T045 [P] [US3] Localize FileBrowser in web/src/components/common/FileBrowser.tsx
- [x] T046 [P] [US3] Localize ConnectionSidebarItem in web/src/modules/connections/components/ConnectionSidebarItem.tsx
- [x] T047 [P] [US3] Localize ProviderSelection in web/src/modules/connections/components/ProviderSelection.tsx

#### Common Components

- [x] T048 [P] [US3] Localize HelpTooltip in web/src/components/common/HelpTooltip.tsx
- [x] T049 [P] [US3] Localize StatusIcon in web/src/components/common/StatusIcon.tsx
- [x] T050 [P] [US3] Localize TableSkeleton in web/src/components/common/TableSkeleton.tsx

#### Layouts

- [x] T051 [P] [US3] Localize AppShell in web/src/layouts/AppShell.tsx
- [x] T052 [P] [US3] Localize ConnectionLayout in web/src/modules/connections/layouts/ConnectionLayout.tsx
- [x] T053 [P] [US3] Localize ConnectionViewLayout in web/src/modules/connections/layouts/ConnectionViewLayout.tsx

**Checkpoint**: All UI text should now be localized in both Chinese and English

---

## Phase 6: User Story 5 - Backend API Response Localization (Priority: P2)

**Goal**: User-visible information returned by the backend API corresponds to the language specified in the request's Accept-Language header.

**Independent Test**: Send API requests with different language preference headers and verify that the response messages are in the corresponding language.

### Tests for User Story 5

- [x] T054 [P] [US5] Write LocaleMiddleware tests in internal/api/context/locale_test.go
- [x] T055 [P] [US5] Write I18nErrorMiddleware tests in internal/api/context/locale_test.go

### Implementation for User Story 5

- [x] T056 [US5] Update error responses in internal/api/handlers/error.go to use i18n.T()
- [x] T057 [P] [US5] Update task handlers to use I18nError in internal/api/handlers/task.go
- [x] T058 [P] [US5] Update remote handlers to use I18nError in internal/api/handlers/remote.go
- [x] T059 [P] [US5] Update job handlers to use I18nError in internal/api/handlers/job.go
- [x] T060 [P] [US5] Update file handlers to use I18nError in internal/api/handlers/files.go
- [x] T061 [P] [US5] Update log handlers to use I18nError in internal/api/handlers/log.go
- [x] T062 [US5] Pass context.Context with localizer to service layer calls in all handlers

**Checkpoint**: API responses should now be localized based on Accept-Language header

---

## Phase 7: User Story 4 - Dynamic Content Localization (Priority: P3)

**Goal**: Date, time, and relative time are localized and formatted according to the selected language.

**Independent Test**: After switching languages, view pages containing dates/times and verify that the format conforms to the locale settings.

### Implementation for User Story 4

- [x] T063 [US4] Create date formatting utilities with locale support in web/src/lib/date.ts
- [x] T064 [US4] Update IntervalUpdated component to use localized relative time in web/src/components/common/IntervalUpdated.tsx
- [x] T065 [P] [US4] Update History view to use localized date formats
- [x] T066 [P] [US4] Update task last sync/next sync displays to use localized formats
- [x] T067 [US4] Add relative time translations to messages files (time_justNow, time_minutesAgo, etc.)

**Checkpoint**: All date/time displays should now be formatted according to selected locale

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Refinement, testing, and documentation

- [x] T068 [P] Add missing translations identified during testing:
  - Run app and check console for missing translation warnings
  - Use paraglide TypeScript type checking to identify unused/missing keys
  - Review T073 edge case verification results
- [x] T069 [P] Install Inlang VS Code extension recommendation in web/.vscode/extensions.json
- [x] T070 Run quickstart.md validation - verify all steps work
- [x] T071 [P] Add translation contribution guide to README.md
- [x] T072 Final code review and cleanup
- [x] T073 Verify all edge cases from spec.md:
  - Translation fallback when text missing
  - Language switch during page operation
  - Form data preservation during language switch
  - Long translation text layout handling:
    - Check buttons, labels, and navigation items with longer Chinese text
    - Verify text truncation with ellipsis works correctly
    - Test responsive layouts at different viewport widths
  - Translation resource load failure fallback

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1: Setup
    ↓
Phase 2: Foundational (BLOCKS all user stories)
    ↓
┌──────────────────────────────────────────────────────┐
│  User Stories can proceed in parallel after Phase 2  │
├──────────────────────────────────────────────────────┤
│  Phase 3: US1 (P1) ─┬→ Phase 4: US2 (P1)            │
│                     │                                │
│  Phase 5: US3 (P2) ─┼→ depends on US1/US2 complete  │
│  Phase 6: US5 (P2) ─┘                                │
│                                                      │
│  Phase 7: US4 (P3) → depends on US3 for components  │
└──────────────────────────────────────────────────────┘
    ↓
Phase 8: Polish (Final)
```

### User Story Dependencies

- **US1 (Automatic Language Detection)**: Foundational only - No dependencies on other stories
- **US2 (Manual Language Switching)**: Depends on US1 (Locale Store must exist)
- **US3 (Full UI Localization)**: Depends on US1/US2 (LocaleProvider and paraglide setup)
- **US4 (Dynamic Content Localization)**: Depends on US3 (components must be localized first)
- **US5 (Backend API Localization)**: Foundational only - Can run parallel to frontend stories

### Parallel Opportunities

**Within Phase 2 (Foundational)**:

- T009, T010, T011, T012 can run in parallel (different files)
- T019 can run after T009-T014 complete

**Within Phase 5 (US3 - Complete UI Localization)**:

- All T030-T053 can run in parallel (different component files)

**Within Phase 6 (US5 - Backend API)**:

- T054, T055 can run in parallel (different test files)
- T057, T058, T059, T060, T061 can run in parallel (different handler files)

---

## Parallel Example: Phase 5 (US3)

```bash
# All component localizations can run in parallel:
Task: T030 - Localize Sidebar navigation
Task: T031 - Localize MobileHeader
Task: T032 - Localize WelcomeView
Task: T033 - Localize RecentActivity
# ... (all T030-T053 can run simultaneously)
```

---

## Implementation Strategy

### MVP First (US1 + US2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: US1 (Auto-detection)
4. Complete Phase 4: US2 (Manual switching)
5. **STOP and VALIDATE**: Test language detection and switching
6. Deploy/demo basic i18n with limited translations

### Incremental Delivery

1. **MVP**: Setup + Foundation + US1 + US2 → Basic language switching works
2. **+US3**: Full UI localization → All text translated
3. **+US5**: API localization → Backend messages translated
4. **+US4**: Date/time formatting → Complete localization
5. **+Polish**: Edge cases and documentation

### Parallel Team Strategy

With 2+ developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: Frontend (US1 → US2 → US3 → US4)
   - Developer B: Backend (US5)
3. Integration testing after each story

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Run `pnpm dev` in web/ to see paraglide compilation results
- Run `go test ./internal/i18n/...` to verify backend i18n
- Avoid: vague tasks, same file conflicts, cross-story dependencies

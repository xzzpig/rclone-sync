# Tasks: UI Refactor to Synology Cloud Sync UX

**Feature**: UI Refactor to Synology Cloud Sync UX
**Status**: Planning
**Spec**: [specs/002-ui-refactor-synology/spec.md](./spec.md)

## Phase 1: Setup & Infrastructure
**Goal**: Initialize the new frontend project structure and install dependencies following [Solid-UI Vite Guide](https://www.solid-ui.com/docs/installation/vite).

- [x] T001 Clean up existing web directory (remove all contents to ensure fresh start)
- [x] T002 Initialize new Vite + SolidJS + TypeScript project: `pnpm create vite web --template solid-ts`
- [x] T003 Install TailwindCSS dependencies: `pnpm add -D tailwindcss@3 postcss autoprefixer @types/node`
- [x] T004 Initialize Tailwind CSS: `pnpx tailwindcss init -p` and configure `web/tailwind.config.js` content
- [x] T005 Configure Path Aliases: Update `web/tsconfig.json` (add `"paths": {"@/*": ["./src/*"]}`) and `web/vite.config.ts` (resolve alias)
- [x] T006 Initialize Solid-UI: Run `pnpx solidui-cli@latest init`
- [x] T007 Install Core App Dependencies: `pnpm add @solidjs/router @tanstack/solid-query lucide-solid axios class-variance-authority clsx tailwind-merge`
- [x] T008 Setup Project Structure: Create `components`, `modules`, `layouts`, `lib`, `api` dirs and `src/lib/api.ts` client
- [x] T009 Implement SSE (Server-Sent Events) client utility in `web/src/lib/sse.ts` for real-time event subscription

## Phase 2: Foundational Components & Mobile Layout
**Goal**: Build the core layout and shared components required for all user stories, ensuring responsiveness from the start.

- [x] T010 [P] Test & Implement AppShell layout with Sidebar and Main Content area in `web/src/layouts/AppShell.tsx`. **Critical**: Implement "Stack Navigation" for mobile (Sidebar is the root view; selecting an item pushes content view). Desktop uses persistent sidebar.
- [x] T011 [P] Test & Implement Sidebar component with navigation logic in `web/src/components/common/Sidebar.tsx`
- [x] T012 [P] Test & Implement "Welcome/Empty State" screen for when no connection is selected in `web/src/modules/core/views/WelcomeView.tsx` (Desktop only)
- [x] T013 [P] Test & Implement "Loading" and "Error" boundary components in `web/src/components/common/`
- [x] T014 [P] Test & Implement Skeleton loading components (CardSkeleton, ListSkeleton) in `web/src/components/ui/skeleton.tsx`
- [x] T015 [P] Test & Implement global store/context for Connection state in `web/src/store/connections.ts`
- [x] T016 [P] Test & Implement "Mobile Header" with "Back" navigation logic (visible only when viewing details on mobile) in `web/src/components/common/MobileHeader.tsx`
- [x] T017 [P] Configure simple Responsive utility (useMediaQuery) to switch between Split View (Desktop) and Stack View (Mobile)

## Phase 3: User Story 1 - Connection-Centric Sidebar (P1)
**Goal**: Users can see and manage their cloud connections from the sidebar.

- [x] T018 [US1] [Backend] Test & Implement update to GET /remotes to return list of objects with type/provider
- [x] T019 [US1] Test & Implement API service for fetching connections in `web/src/api/connections.ts`
- [x] T020 [US1] Integrate connection list into Sidebar component
- [x] T021 [US1] Implement error status indicators (badges/icons) in Sidebar list items
    - **Status**: Completed
    - [x] [Backend] Update `TaskService.ListAllTasks` in `internal/core/services/task_service.go` to eager load latest Job (using `WithJobs` and order desc)
    - [x] [Frontend] Update `web/src/lib/types.ts` with `Job` interface (Status enum, Trigger enum) and `Task` edges
    - [x] [Frontend] Create `web/src/store/tasks.tsx` store with `getTaskStatus` selector to aggregate task statuses
    - [x] [Frontend] Integrate Task Store into `web/src/layouts/AppShell.tsx` (fetch on mount)
    - [x] [Frontend] Update `web/src/components/common/Sidebar.tsx` to use `getTaskStatus` and render badges (Spinner/Green/Red)
- [x] T022 [US1] Create "Add Connection" wizard dialog in `web/src/modules/connections/components/AddConnectionDialog.tsx`
- [x] T023 [US1] Test & Implement Step 1: Provider Selection (Grid of icons for S3, OneDrive, etc.) with search/filter capability
- [x] T024 [US1] Test & Implement Step 2: Dynamic Configuration Form based on selected provider (Rclone fields)
- [x] T025 [US1] [Backend] Test & Implement POST /remotes/test for connection verification
- [x] T026 [US1] Test & Implement Step 3: Verification & Connection Test (Calls dry-run or list endpoint)
- [x] T027 [US1] Add "App Settings" placeholder link/modal to Sidebar bottom (FR-014)

## Phase 4: User Story 2 - Connection Overview (P2)
**Goal**: Display real-time status and usage info when a connection is selected.
**Architecture**: This phase uses a dynamic subscription model. The `ConnectionLayout` subscribes to a connection-specific SSE endpoint (`/api/connections/:name/events`) on mount and unsubscribes on unmount.

- [x] T028 [US2] **[Layout]** Create `web/src/modules/connections/layouts/ConnectionLayout.tsx`.
    - [x] Implement dynamic SSE subscription to `/api/connections/:name/events` on component mount.
    - [x] Ensure SSE connection is closed on component unmount.
    - [x] Add `<Tabs>` component for navigation (Overview, Tasks, etc.).
- [x] T029 [US2] **[View]** Test & Implement "Overview" tab view in `web/src/modules/connections/views/Overview.tsx`.
- [x] T030 [US2] **[Backend]** Test & Implement `GET /api/remotes/:name/quota` (Rclone About).
    - [x] Add `About` method to `internal/rclone` wrapper.
    - [x] Add `GetRemoteQuota` handler in `internal/api`.
    - [x] Register route in `cmd/cloud-sync/serve.go`.
- [x] T031 [US2] **[Data]** Implement async data fetching for Quota with Skeleton loading state in `Overview.tsx`.
- [x] T032 [US2] **[Component]** Create 'Current Status' summary component for Overview tab.
    - [x] **[Ref T021]** Create `web/src/store/tasks.tsx` to manage and aggregate task statuses from API and SSE.
    - [x] Implement status aggregation logic: Any Running > Any Failed > All Success > Idle.
    - [x] Component should reactively display status from the `tasks.tsx` store.
- [x] T033 [US2] **[UI]** Display connection details (Type, Status from T032, Quota Bar) in `Overview.tsx`.
- [x] T033a [US2] **[Backend]** Implement `GET /api/connections/:name/events` SSE endpoint. **(Ref T051, T052)**
- [x] T033b [US2] **[Backend]** Update `GET /api/tasks?remote_name=:name` to eager load latest job for initial status. **(Ref T021)**
- [x] T033c [US2] **[Component]** Install Tabs component using `pnpx solidui-cli@latest add tabs`.

## Phase 5: User Story 3 - Task Management (P1)
**Goal**: Manage sync tasks for the selected connection.

- [X] T034 [US3] [Backend] Test & Implement update to GET /tasks to support filtering by remote_name
- [X] T035 [US3] Test & Implement API service for fetching tasks by connection in `web/src/api/tasks.ts`
- [X] T036 [US3] Create "Task List" tab view with Toolbar Actions (Create, Run Now, Edit, Delete, History) in `web/src/modules/connections/views/TaskList.tsx`
- [X] T037 [US3] Test & Implement Task Table component: Support Row Selection (Single select) and enable/disable toolbar buttons based on selection state.
- [X] T038 [US3] Test & Implement "Run Now" toolbar action: Triggers manual sync for the *selected* task
- [X] T039 [US3] Test & Implement "Create" & "Edit" toolbar action: Opens Modal to modify Schedule/Direction for the *selected* task
- [X] T040 [US3] Test & Implement "Delete" toolbar action: Shows confirmation for the *selected* task, then deletes.
- [x] T041 [US3] Test & Implement "History" toolbar action: Navigates to History tab using the *selected* task ID

## Phase 6: User Story 4 - Create a New Sync Task (P3)
**Goal**: Wizard for creating new sync tasks.

- [X] T042 [US4] Create "Create Task Wizard" component (Step 1: Paths, Step 2: Settings) in `web/src/modules/tasks/components/CreateTaskWizard.tsx`
- [X] T043 [US4] [Backend] Test & Implement GET /files/local for local file browsing
- [X] T044 [US4] [Backend] Test & Implement GET /files/remote/:name for remote file browsing
- [X] T045 [US4] Test & Implement File Browser component for selecting Local/Remote paths in `web/src/components/common/FileBrowser.tsx`
- [X] T046 [US4] Integrate File Browser into Wizard Step 1
- [X] T047 [US4] Connect Wizard completion to API (create task) and refresh Task List

## Phase 7: User Story 5 & 6 - History & Logs (P3)
**Goal**: View execution history and detailed logs.

- [x] T048 [US5] Test & Implement API service for fetching jobs/history in `web/src/api/history.ts`
- [x] T049 [US5] Create "History" tab view in `web/src/modules/connections/views/History.tsx`
- [x] T050 [US5] Test & Implement History Table (Status, Time, Summary) with filtering by Task
- [x] T051 [US5] [Backend] Test & Implement SSE endpoint at GET /api/connections/:name/events
- [x] T052 [US5] [Backend] Test & Implement Job Runner 'job_progress' event emission
- [x] T053 [US5] Integrate SSE subscription in History view to listen for `job_progress` events (status changes, progress)
- [x] T054 [US6] Create "Log" tab view in `web/src/modules/connections/views/Log.tsx`
- [x] T055 [US6] Test & Implement Log Viewer component with filtering controls for Task, Job ID, and Log Level
- [x] T056 [US6] Link "View Logs" action from History table to Log tab

## Phase 8: User Story 7 - Connection Settings (P3)
**Goal**: Edit connection configuration.

- [x] T057 [US7] Create "Settings" tab view in `web/src/modules/connections/views/Settings.tsx`
- [x] T058 [US7] Reuse/Adapt Connection Form for editing existing connection config
- [x] T059 [US7] Implement Connection Deletion with confirmation dialog (FR-018)

## Phase 9: Polish & Cross-Cutting
**Goal**: Final cleanup and refinements.

- [X] T060 Implement "Realtime Sync" toggle in Task settings
- [X] T061 Implement Conflict Resolution options in Task settings
- [X] T062 Verify Accessibility (Keyboard nav, ARIA labels, Touch targets)
- [x] T063 Final UI Polish (Spacing, Colors, Icons)
- [x] T064 Enhance WelcomeView Dashboard
    - **Status**: Completed
    - **Goal**: Transform WelcomeView from empty state to feature-rich dashboard with global overview, recent activity, and quick actions
    - **Implementation Details**:
        - [x] **Phase 1: Basic Structure** - Create layout framework, StatCard component, page title
        - [x] **Phase 2: Statistics Cards** - Display connection count, task count, today's sync count, running/failed status
            - Data sources: `getConnections()`, `getTasks()`, `getJobs()` with date filtering
        - [x] **Phase 3: Recent Activity** - List recent 10 jobs with status icons, relative time, transfer statistics
            - Data source: `getJobs({ limit: 10 })`
            - Features: Status badges, `date-fns` for time formatting, "View All" link
        - [x] **Phase 4: Quick Actions** - Initially planned but removed per user request
        - [x] **Phase 5: Empty State Handling** - Guidance for no connections/tasks/history scenarios
        - [x] **Phase 6: Optimization** - Loading states, error handling, responsive layout
    - **Components Created**:
        - [x] `web/src/modules/core/components/StatCard.tsx` - Reusable statistics card with dark mode
        - [x] `web/src/modules/core/components/RecentActivity.tsx` - Recent jobs list with dark mode and date-fns
        - [x] `web/src/lib/utils.ts` - Shared `formatBytes` utility function
    - **Technical Implementation**:
        - [x] Use SolidJS with `@tanstack/solid-query` for data fetching
        - [x] Implement semantic CSS variables for automatic dark mode support (`bg-background`, `text-foreground`, etc.)
        - [x] All UI text translated to English
        - [x] Reuse existing Solid-UI components (`@/components/ui/`)
        - [x] Tailwind CSS for styling
    - **Cleanup**:
        - [x] Remove unused `QuickActions` component and adjust layout
        - [x] Delete `QuickActions.tsx` file
        - [x] Verify build success
    - **MVP Achieved** (~1 hour implementation):
        - ✅ Statistics cards (connections + tasks + recent activity)
        - ✅ Recent activity list (simplified version)
        - ✅ Full dark mode support

## Dependencies
- Phase 1 -> Phase 2 -> Phase 3 -> Phase 4 -> Phase 5 -> Phase 6 -> Phase 7 -> Phase 8 -> Phase 9
- Most User Stories (3-8) depend on Phase 3 (Sidebar/Connection context) being established.

## Implementation Strategy
- **MVP**: Phases 1, 2, 3, 5 (Basic Connection & Task Management)
- **Incremental**: Add History, Logs, and Advanced Settings in subsequent updates.

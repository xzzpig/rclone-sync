# Feature Specification: UI Refactor to Synology Cloud Sync UX

**Feature Branch**: `002-ui-refactor-synology`
**Created**: 2025-12-07
**Status**: Draft
**Input**: User description: "我需要完全重构ui，使得ui整体使用逻辑与 synology cloud sync 类似"

## Clarifications

### Session 2025-12-07
- Q: Should global views (Dashboard/All Tasks) be retained? → A: **No (Option A)**. The application will be strictly connection-centric. The sidebar will only list Connections and Global Settings.
- Q: How should Real-time Sync be configured? → A: **Explicit Toggle (Option A)**. A specific switch in the Task configuration form.
- Q: How to load Connection Status/Quota? → A: **Load on Demand (Option A)**. Async fetch with skeleton loading state when the Overview tab is accessed.
- Q: What are the performance targets for the UI? → A: **Progressive Enhancement (Option D)**. Basic functionality loads quickly, detailed data loads asynchronously.
- Q: How detailed should the entity relationships be defined? → A: **Already Defined**. Entity relationships are already defined in existing schema files (Task, Job, JobLog) and can be modified in the future.
- Q: How should external API failures be handled? → A: **Basic Error Handling (Option B)**. Display generic error messages and rely on rclone's built-in error handling.
- Q: How should file conflicts be resolved? → A: **Defined Resolution Strategies (Option A)**. Provide options for file conflicts (keep local, keep remote, keep both), with configurable default strategy. Note: rclone's bisync provides relevant options for these settings.
- Q: What is the expected level of accessibility support? → A: **WCAG 2.1 AA Compliance (Option A)**. Ensure standard keyboard navigation (Tab, Enter/Space) is functional and there are no major screen reader issues.
- Q: How should the new sync task creation form be presented? → A: **Step-by-step Wizard (Option B)**. A wizard will guide the user through the process, starting with path selection, then direction, then schedule, etc.
- Q: What specific items should be included in "Global Settings"? → A: Placeholder only for now.
- Q: What should be displayed when no connection is selected? → A: **Guided Welcome Screen (Option A)**. A welcome message with a prominent "Create your first connection" button.
- Q: What is the behavior of the Sidebar "+" button? → A: **Split Actions (Option A)**. The Sidebar "+" button exclusively creates a new Cloud Connection. A separate "Create" button exists inside the "Task List" tab for adding new tasks to the active connection.
- Q: How should Connection deletion be handled? → A: **Cascade Delete (Option A)**. Deleting a Connection shows a confirmation dialog and, upon confirmation, deletes the Connection AND all associated Sync Tasks.
- Q: How to distinguish Connection settings from Global settings? → A: **Context + Global (Option A)**. The connection tab remains "Settings", while the sidebar item is explicitly named "App Settings".

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Connection-Centric Sidebar (Priority: P1)

As a user, I want the main navigation to list my configured Cloud Connections (Remotes) rather than generic application pages, so I can manage my cloud storage providers individually.

**Why this priority**: This is the fundamental navigation paradigm of Synology Cloud Sync.

**Independent Test**:
1.  Load the application.
2.  Verify the sidebar lists existing cloud connections.
3.  Verify a "+" button exists in the sidebar to add a new connection.
4.  Verify a "App Settings" gear icon exists for global application settings.

**Acceptance Scenarios**:

1.  **Given** I have configured "OneDrive" and "GoogleDrive", **When** I load the app, **Then** the sidebar displays these two items.
2.  **Given** I click the Sidebar "+" button, **Then** I am guided to create a new Connection (selecting a Provider).
3.  **Given** I select "OneDrive" in the sidebar, **Then** the main content area updates to show the details for "OneDrive".

---

### User Story 2 - Connection Overview (Priority: P2)

As a user, when I select a connection, I want to see an "Overview" tab that shows its status, storage usage, and a summary of activity.

**Why this priority**: Provides immediate system status feedback.

**Independent Test**: Select a connection. Verify the "Overview" tab is active and displays correct info.

**Acceptance Scenarios**:

1.  **Given** I navigate to the "Overview" tab, **Then** I initially see a loading state (e.g. skeleton UI) while data is fetched.
2.  **Given** data loads successfully, **Then** I see the provider type (e.g., "onedrive") and its current status (Online/Offline).
3.  **Given** the system supports it, **Then** I see a storage quota bar (Used / Total).
3.  **Given** sync is active, **Then** I see a "Current Status" summary (e.g., Disconnected, Connected, Syncing).

---

### User Story 3 - Task Management (Priority: P1)

As a user, I want to manage the specific folder sync pairs (Tasks) for the selected connection in a "Task List" tab.

**Why this priority**: This is where the actual sync configuration happens.

**Independent Test**: Select a connection -> Click "Task List". Verify CRUD operations for Tasks.

**Acceptance Scenarios**:

1.  **Given** I am on the "Task List" tab, **Then** I see a table of Sync Tasks filtered by this Connection.
2.  **Given** the table is displayed, **Then** columns include: "Local Path", "Remote Path", "Direction" (Upload/Download/Bidirectional), and "Schedule".
3.  **Given** I click "Create", **Then** a step-by-step wizard appears to create a new Task. The "Remote" field is pre-filled and locked to the current connection.
4.  **Given** I edit a task, **Then** I can change its `schedule`, `direction`, `realtime` mode, or `options`.
5.  **Given** I want to see past runs of a task, **Then** I click a "History" button on the task row, which redirects me to the "History" tab filtered by that Task.

---

### User Story 4 - Create a New Sync Task (Priority: P3)

As a user, I want to be able to click the "Create" button within the "Task List" tab to launch a wizard that guides me through setting up a new sync task for the current connection.

**Why this priority**: This is a primary action for making the application useful.

**Independent Test**: Navigate to a Connection -> "Task List". Click the "Create" button. Verify wizard opens.

**Acceptance Scenarios**:

1.  **Given** I am on the "Task List" tab of a connection, **When** I click the "Create Task" button, **Then** a "Create Sync Task" step-by-step wizard appears.
2.  **Given** the creation wizard is open, **When** I select a local and remote path and confirm, **Then** a new task is added to the "Task List" for the currently selected connection.

### User Story 5 - History & Logs (Priority: P3)

As a user, I want to see the history of sync operations (Jobs) for this connection to verify files are transferring correctly.

**Why this priority**: Essential for troubleshooting.

**Independent Test**: Select a connection -> Click "History". Verify list of Jobs.

**Acceptance Scenarios**:

1.  **Given** I am on the "History" tab, **Then** I see a list of Jobs associated with the tasks of this connection.
2.  **Given** I want to focus on a specific task, **Then** I can filter the history list by selecting a specific Task.
3.  **Given** I click the "Logs" button/icon on a specific Job in the list, **Then** I am redirected to the "Logs" tab with that Job pre-selected as a filter.

---

### User Story 6 - Dedicated Log View (Priority: P3)

As a user, I want a detailed log view where I can see file-level events, filtered by Task or specific Job history.

**Why this priority**: Detailed troubleshooting often requires digging into specific execution events.

**Independent Test**: Open the "Log" tab. Test filtering by Task and Job ID.

**Acceptance Scenarios**:

1.  **Given** I am on the "Log" tab, **Then** I see a list of file events (transfers, deletions, errors).
2.  **Given** I want to narrow down the noise, **Then** I can filter the logs by "Task" and/or specific "Job History" instance.
3.  **Given** I came from the "History" tab, **Then** the filters are automatically populated with the Job I selected.

---

### User Story 7 - Connection Settings (Priority: P3)

As a user, I want to modify the configuration of the connection itself (e.g., re-authenticate, change polling interval).

**Why this priority**: Maintenance of the connection.

**Independent Test**: Select a connection -> Click "Settings". Verify form to update Connection config.

**Acceptance Scenarios**:

1.  **Given** I am on the "Settings" tab, **Then** I see the configuration parameters for this connection (e.g., `client_id`, `client_secret`).
2.  **Given** I make changes and save, **Then** the system updates the connection config.

---

### User Story 8 - Mobile Navigation (Priority: P2)

As a user on a mobile device (narrow screen), I want the sidebar to act as the primary view, and selecting a connection should navigate to a dedicated detail view, so I can use the app effectively on small screens.

**Why this priority**: Essential for mobile usability.

**Independent Test**: Resize browser to mobile width. Verify sidebar is full width. Click a connection. Verify it navigates to a new view with a "Back" button.

**Acceptance Scenarios**:

1.  **Given** I am on a mobile device, **When** I load the app, **Then** I see the list of Connections (Sidebar) taking up the full screen.
2.  **Given** I tap a Connection, **Then** the view transitions to the Connection Details (Overview/Tasks/etc.), and a "Back" button appears in the header.
3.  **Given** I am viewing Connection Details, **When** I tap "Back", **Then** I return to the Connection list.

### Edge Cases

- What happens when a cloud connection is in an error state? The UI should clearly indicate the error on the connection list and in the overview.
- How does the system handle a very long list of connections or tasks? The UI should implement scrolling within the respective panels. (Note: Technical implementation deferred to planning phase).
- What is displayed in the content panel if no connection is selected? A guided welcome screen with a prominent "Create your first connection" button will be shown.
- How are API failures handled? Display generic error messages and rely on rclone's built-in error handling mechanisms.
- How are file conflicts resolved? Provide options for file conflicts (keep local, keep remote, keep both), with configurable default strategy using rclone's bisync options.

## Requirements *(mandatory)*

### Functional Requirements

-   **FR-001**: **Layout**: The application MUST use a persistent sidebar for navigation between Connections.
-   **FR-002**: **Sidebar**: MUST list all available Connections.
-   **FR-003**: **Sidebar**: MUST provide a "Create Connection" entry point.
-   **FR-004**: **Main View**: MUST be context-sensitive to the selected Connection.
-   **FR-005**: **Tabs**: The Main View MUST contain tabs: "Overview", "Task List", "History", "Log", "Settings".
-   **FR-006**: **Task List**: MUST display Sync Tasks filtered by the selected Connection.
-   **FR-007**: **Task History**: The Task List MUST tab provide a "History" action for each task.
-   **FR-008**: **Task Creation**: Creating a task MUST associate it with the currently selected Connection.
-   **FR-009**: **Task Properties**: The UI MUST support configuring `Direction`, `Schedule`, and a `Realtime Sync` toggle for each task.
-   **FR-009-1**: **Conflict Resolution**: The UI MUST provide options for file conflict resolution (keep local, keep remote, keep both) with configurable default strategy using rclone's bisync options.
-   **FR-010**: **History**: MUST display Jobs filtered by the selected Connection.
-   **FR-011**: **History Filtering**: The History tab MUST allow filtering Jobs by specific "Task".
-   **FR-012**: **History to Log**: The History view MUST provide a "View Logs" action.
-   **FR-013**: **Log View**: MUST display detailed file events (Job Logs) for the connection, filterable by `Task` and `Job`.
-   **FR-014**: **App Settings**: A separate "App Settings" area (accessible from sidebar bottom) MUST exist as a placeholder for future global app configuration, distinct from the connection-specific "Settings" tab.
-   **FR-015**: **Mobile Layout**: On narrow screens (< 768px), the layout MUST switch to a stack navigation model.
-   **FR-016**: **Overview Loading**: The Connection Overview tab MUST use an asynchronous loading pattern (skeleton/spinner) while fetching live status and quota from the backend.
-   **FR-017**: **Accessibility**: The UI MUST comply with WCAG 2.1 AA standards, including keyboard navigability (Tab, Enter, Space) and screen-reader compatibility for primary workflows.
-   **FR-018**: **Connection Deletion**: The UI MUST allow deleting a Connection with confirmation dialog, which cascades to delete all associated tasks.
-   **FR-019**: **Task Deletion**: The UI MUST allow deleting a Task with confirmation dialog.
-   **FR-020**: **Welcome Screen**: When no Connection is selected (e.g., on initial load or after deleting the last connection), the Main View MUST display a Welcome Screen with a call-to-action to create a new Connection.
-   **FR-021**: **Task Creation Wizard**: The Task creation process MUST use a multi-step wizard interface (e.g., Path Selection -> Direction/Schedule -> Options).

### Key Entities

-   **Connection (Remote)**: Represents a cloud storage configuration.
-   **Sync Task**: Defines the sync relationship between a local folder and a remote folder.
-   **Job**: A record of a sync execution event.
-   **Job Log**: Detailed file-level events within a Job.

Note: The entity relationships are already defined in the system: Task has many Jobs, Job has many Job Logs.

### UI/UX Constraints

-   **Style**: Clean, "Enterprise" look similar to Synology DSM.
-   **Responsiveness**:
    -   **Desktop**: Persistent Sidebar + Content Area.
    -   **Mobile**: Stack Navigation (List -> Details). Sidebar collapses/transforms into the root view.
-   **Performance Approach**: Progressive Enhancement - basic UI functionality loads immediately, detailed data (quotas, history, logs) loads asynchronously with loading indicators.

## Success Criteria *(mandatory)*

### Measurable Outcomes

-   **SC-001**: **Navigation Efficiency**: Navigation TTI (Time to Interactive) < 100ms (perceived), Content Paint < 1s.
-   **SC-002**: **Task Visibility**: Users can see all sync tasks for a specific cloud provider on a single screen without filtering manually.
-   **SC-003**: **Configuration Speed**: Adding a new Sync Task to an existing connection takes < 30 seconds.
-   **SC-004**: **Error Visibility**: Connection errors (e.g., "Token Expired") are visible on the Sidebar item itself (e.g., red badge).

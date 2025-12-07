# Feature Specification: Rclone Cloud Sync Manager

**Feature Branch**: `001-cloud-sync`
**Created**: 2025-12-04
**Status**: Draft
**Input**: User description: "我想做一个 基于rclone二次开发，功能类似 群晖 cloud sync 的 软件"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Manage Cloud Connections (Priority: P1)

As a user, I want to configure and manage connections to various cloud storage providers (e.g., Google Drive, S3, OneDrive) so that I can use them as destinations for my data.

**Why this priority**: Foundation for any sync activity. Without connections, no sync is possible.

**Independent Test**: Can be tested by adding a valid rclone remote configuration and verifying connectivity.

**Acceptance Scenarios**:

1. **Given** a fresh installation, **When** I select a cloud provider and enter credentials, **Then** the system saves the connection and verifies it is active.
2. **Given** an existing connection, **When** I update the credentials, **Then** the connection uses the new credentials.
3. **Given** a list of connections, **When** I delete one, **Then** it is removed from the available list.

---

### User Story 2 - Create and Manage Sync Tasks (Priority: P1)

As a user, I want to define sync tasks between local folders and cloud connections, specifying the direction and behavior, so that my data is synchronized according to my needs.

**Why this priority**: Core functionality of the application.

**Independent Test**: Create a task and verify that files are copied/synced when triggered.

**Acceptance Scenarios**:

1. **Given** a configured cloud connection, **When** I create a new task selecting a local folder and a remote path, **Then** the task is saved.
2. **Given** a sync task, **When** I choose "Upload only" mode, **Then** changes in local are sent to remote, but remote changes are ignored.
3. **Given** a sync task, **When** I choose "Bidirectional" mode, **Then** changes are synced both ways (subject to limitations).

---

### User Story 3 - Real-time and Scheduled Sync (Priority: P2)

As a user, I want to see the status of my sync tasks, current transfer speeds, and recent logs, so that I know the system is working correctly.

**Why this priority**: Automates the process, making it "Cloud Sync" rather than just a manual copy tool.

**Independent Test**: Modify a file and observe the sync trigger without manual intervention.

**Acceptance Scenarios**:

1. **Given** a task with "Real-time" enabled, **When** I add a file to the source folder, **Then** the sync starts automatically within a short delay.
2. **Given** a task with a schedule (e.g., every hour), **When** the time is reached, **Then** the sync job executes.

---

### User Story 4 - Dashboard and Monitoring (Priority: P2)

As a user, I want to see the status of my sync tasks, current transfer speeds, and recent logs, so that I know the system is working correctly.

**Why this priority**: Provides visibility and confidence in the system.

**Independent Test**: Run a large sync and check the dashboard for progress updates.

**Acceptance Scenarios**:

1. **Given** a running task, **When** I view the dashboard, **Then** I see the current file being transferred and the speed.
2. **Given** a completed task, **When** I view the history, **Then** the system displays the success/failure status and timestamp.

### Edge Cases

- What happens when the network is disconnected during a sync? (Should retry or pause).
- What happens when a file is locked or in use? (Should skip and log error).
- What happens if the local disk is full? (Should stop and alert).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow management of rclone remotes (create, edit, delete, list).
- **FR-002**: System MUST support defining sync tasks with Source, Destination, and Sync Mode.
- **FR-003**: System MUST support at least three sync modes: Upload (Local to Remote), Download (Remote to Local), and Bidirectional (Simple Merge - new files from both sides are copied, no deletions propagated).
- **FR-004**: System MUST provide a mechanism to trigger syncs based on file system events (inotify/fswatch).
- **FR-005**: System MUST provide a mechanism to trigger syncs based on time schedules (cron-like).
- **FR-006**: System MUST provide a Web UI for management and monitoring.
- **FR-007**: System MUST display real-time status of active transfers (speed, progress).
- **FR-008**: System MUST maintain detailed logs of sync activities (level, date/time, file/folder, event description) for audit and troubleshooting.

### Key Entities

- **Remote**: Represents a cloud storage configuration (Type, Name, Config).
- **Task**: Represents a synchronization job (Name, SourcePath, RemoteID, RemotePath, Direction, Schedule, Options).
- **Job**: Represents an execution instance of a Task (Status, StartTime, EndTime, Summary).
- **JobLog**: Represents individual events within a Job (Level, Time, Path, Message).

## Success Criteria *(mandatory)*

- **SC-001**: Users can configure a new sync task in under 3 minutes.
- **SC-002**: Real-time sync triggers within 30 seconds of a file change.
- **SC-003**: System successfully handles file names with special characters and different encodings.
- **SC-004**: System can run continuously for 24 hours without crashing or memory leaks.

## Assumptions

- The underlying engine will be `rclone`.
- The user has basic knowledge of cloud provider credentials (API keys, etc.).
- The system runs on an OS that supports rclone and file watching.


# Data Model: Synology-style UI

## Frontend State Entities

These entities will be managed in the SolidJS frontend, primarily using TanStack Query for server state and Solid Stores for local UI state.

### 1. Connection (Remote)
Represents a configured rclone remote.

```typescript
interface Connection {
  id: string;              // Unique internal ID (e.g., "onedrive-1")
  type: string;            // Provider type (e.g., "onedrive", "drive", "s3")
  name: string;            // User-friendly name
  status: 'online' | 'offline' | 'error';
  quota?: {
    used: number;          // Bytes
    total: number;         // Bytes
    updated_at: string;    // ISO date
  };
  config: Record<string, any>; // Provider-specific config (partially redacted)
}
```

### 2. Sync Task
A specific folder pair sync configuration.

```typescript
interface SyncTask {
  id: string;
  connection_id: string;   // FK to Connection
  local_path: string;
  remote_path: string;
  direction: 'upload' | 'download' | 'bidirectional';
  schedule: {
    enabled: boolean;
    cron_expression?: string; // e.g., "0 * * * *"
  };
  options: {
    realtime: boolean;     // Enable fsnotify watcher
    delete_mode: 'sync' | 'soft' | 'hard';
    conflict_resolution: 'newer' | 'local' | 'remote';
  };
  last_run?: {
    status: 'success' | 'failed';
    finished_at: string;
  };
}
```

### 3. Job (History)
An execution instance of a Sync Task.

```typescript
interface Job {
  id: string;
  task_id: string;
  start_time: string;
  end_time?: string;
  status: 'running' | 'success' | 'failed' | 'cancelled';
  summary: {
    files_transferred: number;
    bytes_transferred: number;
    errors: number;
  };
}
```

### 4. Job Log (Detail)
Granular file events within a job.

```typescript
interface JobLog {
  id: string;
  job_id: string;
  timestamp: string;
  level: 'info' | 'error' | 'warning';
  action: 'transfer' | 'delete' | 'skip' | 'check';
  file_path: string;
  message: string;
}
```

## API Contracts (New & Updated)

To support the Synology-style UI, we need the following endpoints.

### Connection Management
*   `GET /api/connections` - List all connections (Sidebar).
*   `GET /api/connections/:id` - Get details + status.
*   `GET /api/connections/:id/quota` - (Async) Force refresh quota.
*   `POST /api/connections` - Create new.
*   `DELETE /api/connections/:id` - Cascade delete.

### Task Management
*   `GET /api/connections/:id/tasks` - List tasks for specific connection.
*   `POST /api/tasks` - Create task (requires `connection_id`).
*   `PUT /api/tasks/:id` - Update schedule/options.
*   `POST /api/tasks/:id/run` - Manual run.

### History & Logs
*   `GET /api/jobs?connection_id=...&task_id=...` - Filtered history list.
*   `GET /api/jobs/:id/logs` - Log lines for specific job.

### File Browser (Wizard Support)
*   `GET /api/browse/local?path=...` - List local folders.
*   `GET /api/browse/remote/:connection_id?path=...` - List remote folders.

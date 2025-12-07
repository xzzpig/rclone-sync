# Data Model: Rclone Cloud Sync Manager

**Feature**: `001-cloud-sync`
**Date**: 2025-12-04

## Entities

### Remote
*Stored in `rclone.conf` (managed via rclone lib), but metadata cached in DB if needed.*
- **Name**: `string` (Unique ID)
- **Type**: `string` (e.g., "drive", "s3")
- **Config**: `map[string]string` (Sensitive)

### Task
*Definition of a sync relationship.*
- **ID**: `string` (UUID)
- **Name**: `string`
- **SourcePath**: `string` (Local path)
- **RemoteName**: `string` (Ref -> Remote.Name)
- **RemotePath**: `string` (Path on remote)
- **Direction**: `enum` ("upload", "download", "bidirectional")
- **Schedule**: `string` (Cron expression, empty if manual/watch-only)
- **Realtime**: `boolean` (Enable file watching)
- **Options**: `json` (Includes filters, conflict resolution, bandwidth limits)
- **CreatedAt**: `datetime`
- **UpdatedAt**: `datetime`

### Job
*Execution history of a Task.*
- **ID**: `string` (UUID)
- **TaskID**: `string` (Ref -> Task.ID)
- **Status**: `enum` ("pending", "running", "success", "failed", "cancelled")
- **Trigger**: `enum` ("manual", "schedule", "realtime")
- **StartTime**: `datetime`
- **EndTime**: `datetime`
- **FilesTransferred**: `int`
- **BytesTransferred**: `int64`
- **Errors**: `text` (Summary of errors)

### JobLog
*Detailed operation log for a specific job.*
- **ID**: `int` (Auto-increment)
- **JobID**: `string` (Ref -> Job.ID)
- **Level**: `enum` ("info", "warning", "error")
- **Time**: `datetime`
- **Path**: `string` (File or folder path involved)
- **Message**: `string` (Description of the event, e.g., "Uploaded", "Deleted", "Conflict detected")

### AppConfig
*Global application settings. Stored in `config.toml` (via Viper), NOT in database.*
- **Port**: `int` (Web UI port)
- **LogLevel**: `string`
- **DatabasePath**: `string`
- **RcloneConfigPath**: `string`
- **LogRetentionDays**: `int`
- **MigrationMode**: `enum` ("auto", "versioned", "none")

## Storage Schema (Ent Schema)

*Note: The following SQL is illustrative. The actual schema will be defined in Go code using `ent`.*

```sql
-- AppConfig is stored in config.toml, not in DB.

CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    source_path TEXT NOT NULL,
    remote_name TEXT NOT NULL,
    remote_path TEXT NOT NULL,
    direction TEXT NOT NULL,
    schedule TEXT,
    realtime BOOLEAN DEFAULT 0,
    options TEXT, -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    status TEXT NOT NULL,
    trigger TEXT NOT NULL,
    start_time DATETIME,
    end_time DATETIME,
    files_transferred INTEGER DEFAULT 0,
    bytes_transferred INTEGER DEFAULT 0,
    errors TEXT,
    FOREIGN KEY(task_id) REFERENCES tasks(id)
);

CREATE TABLE job_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT NOT NULL,
    level TEXT NOT NULL,
    time DATETIME DEFAULT CURRENT_TIMESTAMP,
    path TEXT,
    message TEXT,
    FOREIGN KEY(job_id) REFERENCES jobs(id)
);

CREATE INDEX idx_jobs_task_id ON jobs(task_id);
CREATE INDEX idx_job_logs_job_id ON job_logs(job_id);
```

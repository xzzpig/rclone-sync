# Quickstart: Versioned Database Migration

**Feature**: 005-versioned-migration  
**Date**: 2025-12-19  
**Status**: Complete

## Overview

This document provides a quick development guide for the versioned migration feature, using golang-migrate as the migration engine.

---

## Development Environment Setup

### 1. Add Dependencies

```bash
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/sqlite3
go get github.com/golang-migrate/migrate/v4/source/iofs
```

### 2. Create Migration Directory Structure

```bash
mkdir -p internal/core/db/migrations
```

### 3. Create embed.go File

```go
// internal/core/db/embed.go
package db

import "embed"

//go:embed migrations/*.sql
var migrations embed.FS
```

Note: `embed.go` is placed in the `internal/core/db/` directory, and the `migrations/` directory only stores pure SQL files.

---

## Common Development Operations

### Development Environment Preparation

Ensure you have entered the nix development environment and the atlas CLI is automatically installed (used for generating migration SQL):

```bash
# Enter development environment
nix develop

# Verify atlas installation
atlas version
```

### Generate Initial Baseline Migration

When setting up for the first time, use Atlas to generate migration files in golang-migrate format from the ent schema:

```bash
# Generate migration files in golang-migrate format
atlas migrate diff initial \
  --dir "file://internal/core/db/migrations" \
  --dir-format golang-migrate \
  --to "ent://internal/core/db/schema" \
  --dev-url "sqlite://file?mode=memory"
```

This will automatically generate:
- `{timestamp}_initial.up.sql`
- `{timestamp}_initial.down.sql`

### Create New Migration

After modifying the ent schema:

```bash
# 1. Modify ent schema
vim internal/core/db/schema/job.go

# 2. Regenerate ent code
go generate ./internal/core/ent

# 3. Use migration generation script (recommended)
./scripts/gen-migration.sh add_sync_options

# Or use Atlas CLI directly
atlas migrate diff add_sync_options \
  --dir "file://internal/core/db/migrations" \
  --dir-format golang-migrate \
  --to "ent://internal/core/db/schema" \
  --dev-url "sqlite://file?mode=memory"
```

Atlas will automatically generate:
- `{timestamp}_add_sync_options.up.sql`
- `{timestamp}_add_sync_options.down.sql`

### Manually Create Migration

You can also manually write migration files directly:

```bash
# Create migration files
touch internal/core/db/migrations/000003_add_feature.up.sql
touch internal/core/db/migrations/000003_add_feature.down.sql

# Write SQL
vim internal/core/db/migrations/000003_add_feature.up.sql
vim internal/core/db/migrations/000003_add_feature.down.sql
```

### Run Migrations

Migrations are automatically executed when the application starts:

```bash
# Start application (automatically executes migrations)
go run ./cmd/cloud-sync serve
```

---

## Migration File Format

### Naming Convention

```
{version}_{name}.up.sql    # Upgrade script
{version}_{name}.down.sql  # Rollback script
```

- version: 6 digits, e.g., `000001`, `000002`
- name: Descriptive name, separated by underscores

### Example

**000001_initial.up.sql**:
```sql
-- Create connections table
CREATE TABLE connections (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    encrypted_config BLOB NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

-- Create jobs table
CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    source_connection_id TEXT NOT NULL,
    target_connection_id TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (source_connection_id) REFERENCES connections(id),
    FOREIGN KEY (target_connection_id) REFERENCES connections(id)
);
```

**000001_initial.down.sql**:
```sql
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS connections;
```

---

## Testing and Validation

### 1. Unit Testing

```bash
# Run migration related tests
go test ./internal/core/db/... -v
```

### 2. Clean Database Test

```bash
# Delete existing database
rm -f app_data/cloud-sync.db

# Start application, verify migrations execute normally
go run ./cmd/cloud-sync serve
```

Logs should show:
```
INFO Migrations completed successfully
```

### 3. No-change Test

Start the application again:
```bash
go run ./cmd/cloud-sync serve
```

Logs should show:
```
INFO No pending migrations
```

### 4. Dirty State Test

```bash
# Simulate dirty state (for testing only)
sqlite3 app_data/cloud-sync.db "INSERT INTO schema_migrations VALUES (999, 1);"

# Start application, verify error
go run ./cmd/cloud-sync serve
```

Logs should show that migration failed.

---

## Data Migration Examples

When data transformation is needed alongside schema changes, you can write data migration SQL in the `.up.sql` file.

### Example 1: Adding a Column with a Default Value

```sql
-- 000002_add_status_column.up.sql

-- Add new column
ALTER TABLE tasks ADD COLUMN status TEXT NOT NULL DEFAULT 'active';

-- Set default values for existing records (if conditional settings are needed)
UPDATE tasks SET status = 'paused' WHERE realtime = 0 AND schedule IS NULL;
```

### Example 2: Renaming a Column (Not directly supported by SQLite)

```sql
-- 000003_rename_source_to_source_path.up.sql

-- SQLite does not support direct column renaming; a new table must be created and data migrated
CREATE TABLE tasks_new (
    id uuid NOT NULL PRIMARY KEY,
    name text NOT NULL,
    source_path text NOT NULL,  -- New column name
    -- ... other columns
);

-- Copy data
INSERT INTO tasks_new SELECT id, name, source AS source_path, ... FROM tasks;

-- Delete old table
DROP TABLE tasks;

-- Rename new table
ALTER TABLE tasks_new RENAME TO tasks;
```

### Example 3: Data Format Conversion

```sql
-- 000004_convert_json_format.up.sql

-- Update JSON data format
UPDATE tasks 
SET options = json_set(options, '$.newFormat', json_extract(options, '$.oldFormat'))
WHERE json_extract(options, '$.oldFormat') IS NOT NULL;

-- Remove old field (if needed)
UPDATE tasks 
SET options = json_remove(options, '$.oldFormat')
WHERE json_extract(options, '$.oldFormat') IS NOT NULL;
```

### Important Notes

1. **Test Migrations**: Thoroughly test data migration logic in a development environment.
2. **Backup Data**: For production environments, back up the database before running migrations.
3. **Transactions**: Each migration file is executed within a transaction; it will roll back if it fails.
4. **Irreversible Operations**: Be cautious with data deletion; ensure there is a down migration or backup.

---

## Troubleshooting

### Problem 1: Migration Execution Failed

**Symptom**: "migration failed" error at startup.

**Solution**:
1. Check specific error messages in the logs.
2. If tables already exist, the database is in an inconsistent state.
3. Back up data, then delete the database and retry.

### Problem 2: Dirty State

**Symptom**: "dirty database" error at startup.

**Solution**:
```bash
# View current status
sqlite3 app_data/cloud-sync.db "SELECT * FROM schema_migrations;"

# If dirty=1, manual repair is required
# Option 1: Delete database and start over
rm -f app_data/cloud-sync.db

# Option 2: Manually fix dirty state
sqlite3 app_data/cloud-sync.db "UPDATE schema_migrations SET dirty=0 WHERE version=<version>;"
```

### Problem 3: embed.FS Cannot Find Files

**Symptom**: "pattern matches no files" error during compilation.

**Solution**:
```bash
# Ensure migration directory has SQL files
ls internal/core/db/migrations/*.sql

# Ensure there are .up.sql files
ls internal/core/db/migrations/*.up.sql
```

### Problem 4: Incompatible Database

**Symptom**: Migration failed, table already exists.

**Solution**:
1. If the application is in the development stage, delete the database and start over.
2. After backing up important data: `rm -f app_data/cloud-sync.db`

---

## Migration Mode Configuration

### Migration Mode Overview

The system supports two migration modes:

| Mode | Value | Description | Use Case |
|------|-------|-------------|----------|
| Versioned Migration | `versioned` | Uses golang-migrate to execute SQL migration scripts | Production environment (Default) |
| Auto Migration | `auto` | Uses ent ORM's `Client.Schema.Create` | Development environment, Unit tests |

### Setting via Configuration File

Set in `config.yaml`:

```yaml
database:
  path: "app_data/cloud-sync.db"
  migration_mode: "auto"  # versioned (default) or auto
```

### Setting via Environment Variables

```bash
# Start using auto migration mode
DATABASE_MIGRATION_MODE=auto go run ./cmd/cloud-sync serve
```

### Using Auto Migration in Unit Tests

In unit tests, specify the migration mode directly via parameters:

```go
package mypackage_test

import (
    "testing"
    
    "github.com/xzzpig/rclone-sync/internal/core/config"
    "github.com/xzzpig/rclone-sync/internal/core/db"
)

func TestSomething(t *testing.T) {
    // Use memory database
    config.Cfg.Database.Path = ":memory:"
    
    // Initialize database using auto migration mode
    db.InitDB(db.MigrationModeAuto)
    defer db.CloseDB()
    
    // Test code...
}
```

### Call at Application Startup

In `cmd/cloud-sync/serve.go`, read the migration mode from config and pass it to InitDB:

```go
// Read migration mode from config and pass to InitDB
db.InitDB(db.MigrationMode(config.Cfg.Database.MigrationMode))
defer db.CloseDB()
```

### Quick Start in Development Environment

```bash
# Option 1: Using environment variables
DATABASE_MIGRATION_MODE=auto air

# Option 2: Set in .envrc (Recommended)
echo 'export DATABASE_MIGRATION_MODE=auto' >> .envrc
direnv allow

# Then run directly
air
```

### Important Notes

1. **Production must use versioned migration**: The default value is `versioned` to ensure data safety.
2. **Development switching**: It is recommended to set `auto` for the development environment in `.envrc`.
3. **Database incompatibility**: The database structure created by auto migration may not be perfectly identical to that created by versioned migration.
4. **Schema synchronization**: Migration scripts should be generated immediately after modifying the ent schema to ensure consistency between both modes.

---

## Development Checklist

Before submitting code, ensure the following checks are completed:

- [ ] ent schema changes have corresponding migration files generated.
- [ ] Migration files are named correctly (`{version}_{name}.up.sql`).
- [ ] Down migration files are created (if rollback support is needed).
- [ ] Migration files passed code review.
- [ ] Unit tests passed.
- [ ] Clean database startup test passed.
- [ ] Upgrade scenario tests passed (if applicable).

---

## References

- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [golang-migrate iofs source](https://pkg.go.dev/github.com/golang-migrate/migrate/v4/source/iofs)
- [Atlas CLI Documentation](https://atlasgo.io/docs) (Used for generating migration SQL)
- [Go embed Documentation](https://pkg.go.dev/embed)

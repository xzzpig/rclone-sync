# Research: Versioned Database Migration

**Feature**: 005-versioned-migration  
**Date**: 2025-12-19  
**Status**: Complete

## Research Topics

### 1. golang-migrate Architecture Analysis

**Decision**: Use golang-migrate instead of Atlas for versioned migration

**Comparative Analysis**:

| Dimension | Atlas | golang-migrate |
|------|-------|----------------|
| **API Complexity** | Complex (Requires Driver + RevisionReadWriter) | Simple (Requires only DB connection) |
| **embed.FS Support** | Requires custom Dir implementation | Native support via `iofs.New()` |
| **Migration Record Table** | Requires RevisionReadWriter interface | Automatically manages `schema_migrations` |
| **Integration with ent** | Officially recommended by ent | Independent tool |
| **Maturity** | Newer | Mature and stable |

**Rationale**:
1. **Simple API**: golang-migrate only needs `*sql.DB` to work, without complex interface implementations.
2. **Native embed.FS**: Supports embedded file systems directly through `iofs.New()`.
3. **Automatic Record Management**: Automatically creates and manages the `schema_migrations` table.

---

### 2. Core Dependency Packages

```go
import (
    "embed"
    "database/sql"
    
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/sqlite3"
    "github.com/golang-migrate/migrate/v4/source/iofs"
)
```

---

### 3. Migration Directory Structure

```
internal/core/db/
├── db.go                             # Database initialization
├── migrate.go                        # Migration execution logic
├── embed.go                          # Go embed declaration
└── migrations/                       # Migration scripts directory (pure SQL files)
    ├── 000001_initial.up.sql         # First migration (upgrade)
    ├── 000001_initial.down.sql       # First migration (rollback)
    ├── 000002_add_feature.up.sql     # Subsequent migration
    └── 000002_add_feature.down.sql
```

**Naming Convention**:
- `{version}_{name}.up.sql` - Upgrade script
- `{version}_{name}.down.sql` - Rollback script
- version uses 6 digits, e.g., `000001`

---

### 4. Migration Record Table (schema_migrations)

golang-migrate automatically creates a migration record table in the database:

```sql
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,  -- Current migration version number
    dirty BOOLEAN                 -- Whether it is in a dirty state (migration failed)
);
```

**Features**:
- Only records the current version, not the history.
- The `dirty` flag is used to detect interrupted migrations.

---

### 5. Migration Execution Implementation

**embed.go** (`internal/core/db/embed.go`):

```go
package db

import "embed"

//go:embed migrations/*.sql
var migrations embed.FS
```

**migrate.go** (`internal/core/db/migrate.go`):

```go
package db

import (
    "database/sql"
    "errors"
    "fmt"

    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/sqlite3"
    "github.com/golang-migrate/migrate/v4/source/iofs"
    "github.com/xzzpig/rclone-sync/internal/core/logger"
)

// Migrate executes database migrations
func Migrate(db *sql.DB) error {
    // 1. Create migration source from embed.FS (specify migrations subdirectory)
    source, err := iofs.New(migrations, "migrations")
    if err != nil {
        return fmt.Errorf("failed to create migration source: %w", err)
    }

    // 2. Create SQLite database driver
    driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
    if err != nil {
        return fmt.Errorf("failed to create database driver: %w", err)
    }

    // 3. Create migrate instance
    m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
    if err != nil {
        return fmt.Errorf("failed to create migrate instance: %w", err)
    }

    // 4. Execute migrations
    if err := m.Up(); err != nil {
        if errors.Is(err, migrate.ErrNoChange) {
            logger.L.Info("No pending migrations")
            return nil
        }
        return fmt.Errorf("migration failed: %w", err)
    }

    logger.L.Info("Migrations completed successfully")
    return nil
}
```

**db.go modifications**:

```go
// InitDB initializes the database connection and runs migrations.
func InitDB() {
    var err error

    // Open database connection
    db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_fk=1", config.Cfg.Database.Path))
    if err != nil {
        log.Fatalf("failed opening connection to sqlite: %v", err)
    }

    // Execute versioned migrations (replacing the original auto-migration)
    if err := Migrate(db); err != nil {
        log.Fatalf("failed to run migrations: %v", err)
    }

    // Create ent driver
    drv := entsql.OpenDB("sqlite3", db)

    // ... rest of the code remains unchanged
}
```

---

### 6. Database State Detection

**golang-migrate automatic handling scenarios**:

| Scenario | schema_migrations | Business Tables | Behavior |
|------|-------------------|-------|------|
| Fresh Installation | Does not exist | Do not exist | ✅ Executes all migrations normally |
| Versioned Migration Used | Exists | Exist | ✅ Executes pending migrations |
| Upgrade from Auto-migration | Does not exist | Exist | ❌ `CREATE TABLE` fails |
| Dirty State | dirty=true | - | ❌ Errors out, requiring manual handling |

**Rationale**: 
1. **Application in Development Phase**: The application has not been officially released; the auto-migration mode only affects developers.
2. **Simplified Implementation**: Developers only need to delete the development database.

---

### 7. Go embed Migration File Embedding

**Decision**: Use `//go:embed` + `iofs.New()` for native support.

**Advantages**:
1. **Single-file Distribution**: The application is compiled into a single executable file.
2. **Version Consistency**: Migration files are bound to the application version.
3. **No External Dependencies**: No need to deploy migration files in the runtime environment.
4. **Native Support**: No custom implementation required; uses golang-migrate's iofs source directly.

---

### 8. Developer Workflow

**Decision**: Use Atlas CLI to generate migration files in golang-migrate format, and golang-migrate to execute migrations.

**Rationale**: 
- Atlas CLI can automatically generate SQL diffs from the ent schema.
- Use `--dir-format golang-migrate` to generate the correct format directly.
- golang-migrate executes migrations at runtime.

**Development Environment Configuration** (`flake.nix`):
```nix
pkgs.atlas  # Atlas CLI for generating migrations from ent schema
```

**Workflow**:

```bash
# Use Atlas to generate golang-migrate format migration files from ent schema
atlas migrate diff add_feature \
  --dir "file://internal/core/db/migrations" \
  --dir-format golang-migrate \
  --to "ent://internal/core/db/schema" \
  --dev-url "sqlite://file?mode=memory"

# Atlas automatically generates:
# - {timestamp}_add_feature.up.sql
# - {timestamp}_add_feature.down.sql

# Migrations are automatically executed upon application startup
```

**Or manual migration writing**:
```bash
# Create migration files directly
touch internal/core/db/migrations/000002_add_feature.up.sql
touch internal/core/db/migrations/000002_add_feature.down.sql
```

---

### 9. Error Handling

**Decision**: Migration errors use English logs directly, without using I18nError.

**Rationale**:
1. **No Language Context During Startup**: Migrations occur when the application starts; at this point, the HTTP server has not started, there is no request context, and the user's language preference cannot be obtained from the `Accept-Language` header.
2. **Operations Standards**: Migration errors are system-level/operations-level errors intended for developers and operations personnel; English logs are the industry standard.
3. **Simplified Implementation**: Avoids introducing additional language detection logic (e.g., reading system environment variables).

**Error Message Examples**:

```go
// Using logger
logger.L.Error("Migration failed", zap.Error(err))
```

**Common Error Scenarios**:
- `failed to create migration source: %v` - Failed to create migration source.
- `failed to create database driver: %v` - Failed to create database driver.
- `migration failed: %v` - Migration execution failed.
- `database is dirty, please fix manually or delete the database file` - Database is in a dirty state.

---

### 10. Migration Log Output

**Decision**: Output migration details via golang-migrate's log callback.

```go
// Create migrate instance with logging
m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
if err != nil {
    return err
}

// Set log
m.Log = &migrateLogger{}

type migrateLogger struct{}

func (l *migrateLogger) Printf(format string, v ...interface{}) {
    logger.L.Info(fmt.Sprintf(format, v...))
}

func (l *migrateLogger) Verbose() bool {
    return config.Cfg.App.Environment == "development"
}
```

---

---

### 11. Migration Mode Switching Implementation

**Decision**: Support two migration modes - versioned migration (versioned) and automatic migration (auto).

**Requirements Source**: 
- FR-014: The system must support two migration modes.
- FR-015: The application selects the migration mode based on configuration at startup, defaulting to versioned migration.
- FR-016: Provide a programming interface to allow explicit specification of the migration mode when initializing the database (for unit testing).

**Implementation Plan**:

1. **Define Migration Mode Types**:

```go
// internal/core/db/migrate.go

// MigrationMode defines the migration mode type
type MigrationMode string

const (
    // MigrationModeVersioned Versioned migration mode (default for production)
    MigrationModeVersioned MigrationMode = "versioned"
    // MigrationModeAuto Automatic migration mode (development/testing environment)
    MigrationModeAuto MigrationMode = "auto"
)
```

2. **Configuration Design**:

```go
// internal/core/config/config.go

type DatabaseConfig struct {
    Path          string `mapstructure:"path" default:"app_data/cloud-sync.db"`
    MigrationMode string `mapstructure:"migration_mode" default:"versioned"` // versioned or auto
}
```

**Environment Variable**: `DATABASE_MIGRATION_MODE=auto`

3. **Programming Interface Design**:

```go
// internal/core/db/db.go

// InitDB initializes the database connection and executes migrations
// mode: Migration mode, versioned (versioned migration) or auto (automatic migration)
func InitDB(mode MigrationMode) {
    var err error

    db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_fk=1", config.Cfg.Database.Path))
    if err != nil {
        log.Fatalf("failed opening connection to sqlite: %v", err)
    }

    switch mode {
    case MigrationModeVersioned:
        // Execute versioned migrations
        if err := Migrate(db); err != nil {
            log.Fatalf("failed to run migrations: %v", err)
        }
        LogMigrationStatus(db)
    case MigrationModeAuto:
        // Use ent automatic migration
        drv := entsql.OpenDB("sqlite3", db)
        client := ent.NewClient(ent.Driver(drv))
        if err := client.Schema.Create(context.Background()); err != nil {
            log.Fatalf("failed to run auto migration: %v", err)
        }
        logger.L.Info("Auto migration completed successfully")
        Client = client
        return
    default:
        log.Fatalf("unknown migration mode: %s", mode)
    }

    // Create ent driver
    drv := entsql.OpenDB("sqlite3", db)
    // ... rest of the initialization code
}
```

4. **Caller Usage Example**:

```go
// cmd/cloud-sync/serve.go

// Read migration mode from config and pass to InitDB
db.InitDB(db.MigrationMode(config.Cfg.Database.MigrationMode))
defer db.CloseDB()
```

5. **Unit Test Usage Example**:

```go
// internal/core/db/db_test.go

func TestWithAutoMigration(t *testing.T) {
    // Set test configuration
    config.Cfg.Database.Path = ":memory:"
    
    // Initialize using automatic migration mode
    db.InitDB(db.MigrationModeAuto)
    defer db.CloseDB()
    
    // Test code...
}
```

**Rationale**:
1. **Configuration Flexibility**: Control the migration mode via configuration files or environment variables.
2. **Programming Interface**: Unit tests can explicitly specify the migration mode without modifying the configuration.
3. **Default Security**: Production environments default to using versioned migrations.
4. **Backward Compatibility**: Does not affect existing code structures.

---

## Summary

| Topic | Decision | Key Points |
|-------|----------|------------|
| Migration Engine | golang-migrate | Simple API, native embed.FS support |
| Migration Source | iofs.New() | Uses embed.FS directly, no custom implementation needed |
| Migration Storage | embed.FS embedded in binary | Single-file distribution, version consistency |
| Migration Record | schema_migrations table | Automatically managed by golang-migrate |
| Migration Generation | Atlas CLI | Use --dir-format golang-migrate to generate correct format directly |
| Database State Detection | golang-migrate built-in | Errors out on table conflicts |
| Error Handling | English logs | No language context during startup, use English logs directly |
| Log Output | migrate.Logger interface | Detailed execution logs to standard log stream |
| Migration Mode | versioned/auto dual-mode | Controlled by configuration, programming interface supports testing |
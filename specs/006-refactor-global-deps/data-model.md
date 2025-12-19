# Data Model: Refactoring Global Variable Dependencies

**Feature**: 006-refactor-global-deps
**Date**: 2025-12-19

## Overview

This document describes the structural design of each module after refactoring. Since this is an architectural refactoring rather than a new feature, the focus is on changes in module interfaces rather than data model changes.

---

## Module Structure Changes

### 1. Logger Module (`internal/core/logger`)

#### Current Structure
```go
// Public global variable
var L *zap.Logger
```

#### Refactored Structure
```go
// Private singleton variables
var l *zap.Logger
var defaultL *zap.Logger  // Default logger (Info level)
var once sync.Once

// Init initializes the logger
func Init(environment Environment, logLevel LogLevel)

// Get returns the logger instance
// Returns default logger (Info level) when uninitialized
func Get() *zap.Logger

// Named returns a logger with a hierarchical name
// name format: "{layer}.{module}", e.g., "core.runner"
func Named(name string) *zap.Logger
```

#### Field Descriptions
| Field/Method | Type | Description |
|-----------|------|------|
| `l` | `*zap.Logger` | Private singleton, stores the initialized logger |
| `defaultL` | `*zap.Logger` | Default logger, used for uninitialized cases |
| `Init()` | function | Initializes the logger, sets level and environment |
| `Get()` | function | Gets the logger instance |
| `Named(name)` | function | Gets a named sub-logger |

---

### 2. Config Module (`internal/core/config`)

#### Current Structure
```go
// Public global variable
var Cfg Config

// Initialization function with no return value
func InitConfig(cfgFile string)
```

#### Refactored Structure
```go
// Loading function that returns a config instance
func Load(cfgFile string) (*Config, error)

// Config struct remains unchanged
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Rclone   RcloneConfig   `mapstructure:"rclone"`
    Log      LogConfig      `mapstructure:"log"`
    App      AppConfig      `mapstructure:"app"`
    Security SecurityConfig `mapstructure:"security"`
}
```

#### Change Description
| Change | Description |
|------|------|
| Remove `Cfg` | Global variable no longer used |
| `InitConfig` → `Load` | Function renamed, returns `(*Config, error)` |
| Error Handling | Changed from `os.Exit(1)` to returning error |

---

### 3. Database Module (`internal/core/db`)

#### Current Structure
```go
// Public global variable
var Client *ent.Client

// Initialization function with no return value
func InitDB(opts InitDBOptions)

// Close function uses global variable
func CloseDB()
```

#### Refactored Structure
```go
// InitDBOptions configuration options
type InitDBOptions struct {
    DSN           string        // SQLite DSN connection string (e.g., "file:data.db?cache=shared&_fk=1")
    MigrationMode MigrationMode // Migration mode (versioned or auto)
    EnableDebug   bool          // Enable SQL debug logging
    Environment   string        // Application environment (used for migration)
}

// Initialization function returning client instance
func InitDB(opts InitDBOptions) (*ent.Client, error)

// Close function accepts client parameter
func CloseDB(client *ent.Client)

// Migration function signature change
type MigrateOptions struct {
    DB          *sql.DB
    Environment string  // Replaces config.Cfg.App.Environment
}

func Migrate(opts MigrateOptions) error
```

**DSN Description**:
- Production: `fmt.Sprintf("file:%s?cache=shared&_fk=1", cfg.Database.Path)`
- Test (In-memory DB): `"file:ent?mode=memory&cache=shared&_fk=1"`

#### Change Description
| Change | Description |
|------|------|
| Remove `Client` | Global variable no longer used |
| `InitDB` Return Value | Returns `(*ent.Client, error)` |
| `CloseDB` Parameters | Accepts `*ent.Client` parameter |
| `Migrate` | Removed dependency on `config.Cfg` |

---

## Dependency Injection Interfaces

### Structs Requiring Dependency Injection

```go
// api/routes.go - SetupRouter function signature change
type RouterDeps struct {
    Client      *ent.Client
    Config      *config.Config
    SyncEngine  *rclone.SyncEngine
    TaskRunner  *runner.Runner
    JobService  ports.JobService
    Watcher     ports.Watcher
    Scheduler   ports.Scheduler
}

func SetupRouter(deps RouterDeps) *gin.Engine

// api/server.go - NewServer function signature change
func NewServer(cfg *config.Config) *gin.Engine
```

---

## Dependency Matrix

| Module | db.Client | config.Cfg | logger.L | Refactor Method |
|------|-----------|------------|----------|----------|
| `api/routes.go` | ✅ Uses | ✅ Uses | ❌ | Constructor injection |
| `api/server.go` | ❌ | ✅ Uses | ✅ Uses | Constructor injection + logger.Get() |
| `api/sse/broadcaster.go` | ❌ | ❌ | ✅ Uses | logger.Named() |
| `core/db/db.go` | Definition | ❌ | ✅ Uses | logger.Get() |
| `core/db/migrate.go` | ❌ | ✅ Uses | ✅ Uses | Parameter injection + logger.Get() |
| `core/services/job_service.go` | ❌ | ❌ | ✅ Uses | logger.Named() |
| `core/scheduler/scheduler.go` | ❌ | ❌ | ✅ Uses | logger.Named() |
| `core/runner/runner.go` | ❌ | ❌ | ✅ Uses | logger.Named() |
| `core/watcher/watcher.go` | ❌ | ❌ | ✅ Uses | logger.Named() |
| `core/watcher/recursive.go` | ❌ | ❌ | ✅ Uses | logger.Named() |
| `rclone/sync.go` | ❌ | ❌ | ✅ Uses | logger.Named() |

---

## Testing Helper Structures

```go
// internal/core/db/testing.go
func NewTestClient(t *testing.T) *ent.Client

// internal/core/config/testing.go
func NewTestConfig() *Config
func NewTestConfigWithOverrides(overrides func(*Config)) *Config

// internal/core/logger/testing.go
func NewTestLogger(t *testing.T) *zap.Logger
```

---

## State Transitions

### Application Startup Flow

```
[Start] 
    │
    ▼
[1. Load Config] ─────────────────────► cfg *Config
    │
    ▼
[2. Init Logger] ─────────────────────► logger singleton initialization
    │
    ▼
[3. Init Database] ───────────────────► client *ent.Client
    │
    ▼
[4. Create Services] ─────────────────► services with injected deps
    │
    ▼
[5. Setup Router] ────────────────────► gin.Engine
    │
    ▼
[6. Start Server]
    │
    ▼
[Running]
    │
    ▼
[Shutdown]
    │
    ▼
[7. Close Database]
    │
    ▼
[End]
```

---

## Validation Rules

| Rule | Description | Verification Method |
|------|------|----------|
| No Global Variable Access | Code should not directly access `db.Client`, `config.Cfg`, `logger.L` | Compile-time + grep check |
| Explicit Dependency Passing | All db/config dependencies passed via parameters | Code review |
| Logger Access via Getter | All logger usage via `logger.Get()` or `logger.Named()` | grep check |
| Tests Run Independently | Tests do not rely on global state | Run tests |

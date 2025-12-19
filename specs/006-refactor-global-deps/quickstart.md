# Quickstart: Refactoring Global Variable Dependencies

**Feature**: 006-refactor-global-deps
**Date**: 2025-12-19

## Overview

This document provides a guide for using the refactored modules to help developers quickly adapt to the new dependency acquisition methods.

---

## Quick Reference

### Before vs. After Refactoring

| Before Refactoring | After Refactoring |
|--------|--------|
| `db.Client` | Pass `*ent.Client` via parameters |
| `config.Cfg` | Pass `*config.Config` via parameters |
| `logger.L.Info(...)` | `logger.Get().Info(...)` |
| `logger.L.Named("xxx")` | `logger.Named("xxx")` |

---

## Usage Examples

### 1. Logger Module

```go
import "github.com/xzzpig/rclone-sync/internal/core/logger"

// ✅ Get logger instance
log := logger.Get()
log.Info("message")

// ✅ Get named logger (Recommended)
log := logger.Named("core.runner")
log.Info("runner started")

// ❌ No longer supported: Direct access to global variables
// logger.L.Info("message")  // Compilation error
```

**Named Logger Naming Convention**:
```go
// Format: {layer}.{module}
logger.Named("core.runner")     // Runner module
logger.Named("core.scheduler")  // Scheduler
logger.Named("core.watcher")    // File watcher
logger.Named("service.job")     // Job service
logger.Named("api.sse")         // SSE broadcast
logger.Named("sync.engine")     // Sync engine
```

---

### 2. Config Module

```go
import "github.com/xzzpig/rclone-sync/internal/core/config"

// ✅ Load config (at application entry point)
cfg, err := config.Load(cfgFile)
if err != nil {
    log.Fatal(err)
}

// ✅ Pass to required modules
server := NewServer(cfg)
router := SetupRouter(cfg, dbClient, ...)

// ❌ No longer supported: Direct access to global variables
// port := config.Cfg.Server.Port  // Compilation error
```

---

### 3. Database Module

```go
import "github.com/xzzpig/rclone-sync/internal/core/db"

// ✅ Initialize database (at application entry point)
// DSN is passed directly to sql.Open(), supporting full SQLite connection strings
client, err := db.InitDB(db.InitDBOptions{
    DSN:           fmt.Sprintf("file:%s?cache=shared&_fk=1", cfg.Database.Path),
    MigrationMode: db.MigrationModeVersioned,
    EnableDebug:   cfg.App.Environment == "development",
    Environment:   cfg.App.Environment,
})
if err != nil {
    log.Fatal(err)
}
defer db.CloseDB(client)

// ✅ Use in-memory database in tests
client, err := db.InitDB(db.InitDBOptions{
    DSN:           "file:ent?mode=memory&cache=shared&_fk=1",
    MigrationMode: db.MigrationModeAuto,
})

// ✅ Pass to required modules
router := SetupRouter(client, cfg, ...)

// ❌ No longer supported: Direct access to global variables
// users := db.Client.User.Query().All(ctx)  // Compilation error
```

---

## Application Entry Point Example

```go
// cmd/cloud-sync/serve.go

func runServe(cmd *cobra.Command, args []string) {
    // 1. Load config
    cfg, err := config.Load(cfgFile)
    if err != nil {
        log.Fatalf("failed to load config: %v", err)
    }

    // 2. Initialize logger
    logger.Init(
        logger.Environment(cfg.App.Environment),
        logger.LogLevel(cfg.Log.Level),
    )

    // 3. Initialize database
    dbClient, err := db.InitDB(db.InitDBOptions{
        DSN:           fmt.Sprintf("file:%s?cache=shared&_fk=1", cfg.Database.Path),
        MigrationMode: db.ParseMigrationMode(cfg.Database.MigrationMode),
        EnableDebug:   cfg.App.Environment == "development",
        Environment:   cfg.App.Environment,
    })
    if err != nil {
        logger.Get().Fatal("failed to init database", zap.Error(err))
    }
    defer db.CloseDB(dbClient)

    // 4. Create service instances
    syncEngine := rclone.NewSyncEngine(cfg.Rclone.ConfigPath, cfg.App.DataDir)
    taskRunner := runner.NewRunner(dbClient)
    jobService := services.NewJobService(dbClient, taskRunner, ...)
    // ...

    // 5. Set up router
    router := api.SetupRouter(api.RouterDeps{
        Client:     dbClient,
        Config:     cfg,
        SyncEngine: syncEngine,
        TaskRunner: taskRunner,
        JobService: jobService,
        // ...
    })

    // 6. Start server
    addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
    router.Run(addr)
}
```

---

## Writing Tests

### Unit Test Example

```go
func TestMyService(t *testing.T) {
    // Use test helper functions to create dependencies
    dbClient := db.NewTestClient(t)
    cfg := config.NewTestConfig()
    log := logger.NewTestLogger(t)

    // Inject dependencies to create service
    service := NewMyService(dbClient, cfg, log)

    // Execute test
    result, err := service.DoSomething()
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Test Using Mocks

```go
func TestMyHandler(t *testing.T) {
    // Create mock dependencies
    mockDB := mocks.NewMockEntClient(t)
    mockDB.On("User").Return(...)

    cfg := config.NewTestConfigWithOverrides(func(c *config.Config) {
        c.App.Environment = "test"
    })

    // Inject mock dependencies
    handler := NewMyHandler(mockDB, cfg)

    // Test handler
    // ...
}
```

---

## Migration Checklist

Developers should check the following items when adapting to the new pattern:

- [ ] Replace `logger.L.xxx()` with `logger.Get().xxx()` or `logger.Named("name").xxx()`
- [ ] Change places directly using `db.Client` to accept parameters instead
- [ ] Change places directly using `config.Cfg` to accept parameters instead
- [ ] Update test code to use test helper functions for creating dependencies
- [ ] Ensure all tests pass

---

## FAQs

### Q: Why does the logger retain the singleton pattern instead of full dependency injection?

A: Logger has the highest usage (19+ places), and full dependency injection would require too many changes. The singleton pattern is suitable for stateless, globally shared infrastructure like Logger. Accessing through Getter methods is easier for future expansion and testing than directly exposing variables.

### Q: How to use different loggers in tests?

A: Use `logger.NewTestLogger(t)` to create a test logger, or call `logger.Init()` before testing to set specific configurations.

### Q: What happens if dependency initialization fails?

A: A fail-fast strategy is adopted; any failure in initializing core dependencies will immediately terminate startup and return a clear error message.

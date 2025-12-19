# Research: Refactoring Global Variable Dependencies

**Feature**: 006-refactor-global-deps
**Date**: 2025-12-19

## Research Tasks

### 1. Go Language Manual Dependency Injection Best Practices

**Decision**: Adopt constructor injection pattern

**Rationale**: 
- Go promotes explicit dependencies, clearly declaring dependency relationships through constructor parameters
- No external framework required, aligning with Go's philosophy of simplicity
- Compile-time type checking to avoid runtime errors
- Easy to understand and debug

**Alternatives Considered**:
| Option | Pros | Cons | Reason for Exclusion |
|------|------|------|----------|
| Wire (Google DI) | Compile-time code generation | Increases build complexity | Not needed for project scale |
| Uber fx | Powerful features | High learning curve, runtime reflection | Over-engineered |
| Interface + Global Registration | High flexibility | Implicit dependencies, hard to track | Violates refactoring goals |

**Pattern to Follow**:
```go
// 1. Define struct containing dependencies
type Service struct {
    db     *ent.Client
    config *Config
    logger *zap.Logger
}

// 2. Constructor accepts dependencies as parameters
func NewService(db *ent.Client, cfg *Config, logger *zap.Logger) *Service {
    return &Service{
        db:     db,
        config: cfg,
        logger: logger,
    }
}

// 3. Assemble dependencies at the entry point
func main() {
    cfg := config.Load()
    db := db.InitDB(cfg.Database)
    logger := logger.Get()
    
    service := NewService(db, cfg, logger)
    // ...
}
```

---

### 2. Zap Logger Singleton Pattern and Named Logger

**Decision**: Retain Logger singleton pattern, provide `Get()` and `Named(name string)` methods

**Rationale**:
- Logger is the most widely used (19+ places); full dependency injection would be too disruptive
- Zap natively supports Named loggers, allowing hierarchical logs via `.Named()`
- Singleton pattern is suitable for stateless, globally shared infrastructure like Loggers
- Providing a Getter method is more extensible for the future than exposing a variable directly

**Alternatives Considered**:
| Option | Pros | Cons | Reason for Exclusion |
|------|------|------|----------|
| Full Dependency Injection | Maximum flexibility | Massive changes required | 19+ modifications, high risk |
| Keep as-is | No changes | Difficult to test | Violates refactoring goals |
| Context Passing | Standard Go pattern | Highly intrusive | Massive changes required |

**Pattern to Follow**:
```go
// logger/logger.go
var l *zap.Logger  // lowercase, private

func Init(env Environment, level LogLevel) {
    // Initialize logger
    l = buildLogger(env, level)
}

// Get returns the logger instance, returns default logger if uninitialized
func Get() *zap.Logger {
    if l == nil {
        return defaultLogger() // Default logger at Info level
    }
    return l
}

// Named returns a logger with a name
func Named(name string) *zap.Logger {
    return Get().Named(name)
}
```

**Named Logger Naming Convention**:
- Use hierarchy naming separated by "."
- Format: `{layer}.{module}`
- Examples:
  - `core.runner` - Runner module
  - `core.scheduler` - Scheduler
  - `core.watcher` - File monitoring
  - `service.job` - Job service
  - `api.sse` - SSE broadcast
  - `sync.engine` - Sync engine

---

### 3. Config Module Refactoring Pattern

**Decision**: `InitConfig` returns `*Config` instead of setting a global variable

**Rationale**:
- Config usage is moderate (5 places), changes are manageable
- Return value pattern aligns better with Go conventions
- Facilitates creating different configurations during testing

**Pattern to Follow**:
```go
// config/config.go
func Load(cfgFile string) (*Config, error) {
    // Load configuration
    var cfg Config
    // ...
    return &cfg, nil
}

// Caller
func main() {
    cfg, err := config.Load(cfgFile)
    if err != nil {
        log.Fatal(err)
    }
    // Pass to modules that need it
}
```

---

### 4. Database Client Refactoring Pattern

**Decision**: `InitDB` returns `*ent.Client` instead of setting a global variable; use DSN instead of Path parameter

**Rationale**:
- Least usage (2 places), most suitable as the first refactoring target
- Return value pattern facilitates using in-memory databases for testing
- DSN parameter passed directly to sql.Open(), supporting full SQLite connection strings
- Tests can easily use in-memory database DSN (e.g., `file:ent?mode=memory&cache=shared&_fk=1`)

**Pattern to Follow**:
```go
// db/db.go
type InitDBOptions struct {
    DSN           string        // SQLite DSN connection string
    MigrationMode MigrationMode
    EnableDebug   bool
    Environment   string
}

func InitDB(opts InitDBOptions) (*ent.Client, error) {
    // Open database directly using DSN
    sqlDB, err := sql.Open("sqlite3", opts.DSN)
    if err != nil {
        return nil, err
    }
    // Create ent client
    client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", sqlDB)))
    // Execute migrations
    return client, nil
}

// Production caller
func main() {
    cfg := config.Load()
    dbClient, err := db.InitDB(db.InitDBOptions{
        DSN: fmt.Sprintf("file:%s?cache=shared&_fk=1", cfg.Database.Path),
        // ...
    })
    if err != nil {
        log.Fatal(err)
    }
    defer db.CloseDB(dbClient)
}

// Test caller
func setupTest(t *testing.T) *ent.Client {
    client, err := db.InitDB(db.InitDBOptions{
        DSN:           "file:ent?mode=memory&cache=shared&_fk=1",
        MigrationMode: db.MigrationModeAuto,
    })
    require.NoError(t, err)
    return client
}
```

---

### 5. Migration Strategy

**Decision**: Migrate in three phases, each independently testable

**Rationale**:
- Reduces risk by controlling the scope of each change
- Each phase can be verified independently upon completion
- Aligns with the progressive migration strategy in the spec

**Migration Order**:
1. **Phase A: db.Client** (2 places) - Lowest risk, serves as a pilot
2. **Phase B: config.Cfg** (5 places) - Moderate amount of changes
3. **Phase C: logger.L** (19+ places) - Most changes, but simplest pattern (only changing to Getter calls)

**Verification Checklist per Phase**:
- [ ] Compilation passes
- [ ] All existing tests pass
- [ ] New tests using mock dependencies can be written

---

### 6. Test Helper Functions

**Decision**: Provide test helper functions to simplify test setup

**Pattern to Follow**:
```go
// db/testing.go (Testing only)
func NewTestClient(t *testing.T) *ent.Client {
    client, err := ent.Open("sqlite3", "file:test?mode=memory&cache=shared&_fk=1")
    require.NoError(t, err)
    t.Cleanup(func() { client.Close() })
    return client
}

// config/testing.go
func NewTestConfig() *Config {
    return &Config{
        // Test default values
    }
}

// logger/testing.go
func NewTestLogger(t *testing.T) *zap.Logger {
    return zap.NewNop() // Or use zaptest.NewLogger(t)
}
```

---

## Dependency Diagram

```
cmd/cloud-sync/serve.go (Entry Point)
    │
    ├── config.Load() → *Config
    │
    ├── logger.Init(cfg) 
    │
    ├── db.InitDB(cfg) → *ent.Client
    │
    └── api.SetupRouter(dbClient, cfg, ...)
            │
            ├── handlers.NewConnectionHandler(dbClient, ...)
            ├── handlers.NewTaskHandler(dbClient, ...)
            └── services.NewJobService(dbClient, logger.Named("service.job"), ...)
```

---

## Conclusion

This refactoring adopts the following strategies:

| Global Variable | Refactoring Method | Reason |
|----------|----------|------|
| `db.Client` | Constructor Injection | Low usage, facilitates testing |
| `config.Cfg` | Constructor Injection | Moderate usage, manageable changes |
| `logger.L` | Getter Method | High usage, singleton pattern suitable for Logger |

All NEEDS CLARIFICATION have been resolved through research.

# rclone-sync Development Guidelines

A cloud sync management tool built on rclone, providing a Synology Cloud Sync-like experience.

## Architecture Overview

**Backend (Go 1.25+)**: Gin web framework + GraphQL API (gqlgen) + SQLite (Ent ORM) + embedded rclone library  
**Frontend (SolidJS + TypeScript)**: Single-page app with urql GraphQL client, Tailwind CSS, Kobalte components

### Key Components Flow
```
cmd/rclone-sync/serve.go → Bootstrap & dependency injection
    ↓
internal/api/ → HTTP routes + GraphQL handlers
internal/core/services/ → Business logic (ConnectionService, TaskService, JobService)
internal/core/ent/ → Database entities (auto-generated from schema)
internal/rclone/ → Sync engine wrapping rclone library
internal/core/scheduler/ & watcher/ → Cron jobs + filesystem monitoring
```

### Data Model
- **Connection**: Cloud storage remote configuration (encrypted in DB)
- **Task**: Sync job definition (source path, remote path, direction, schedule)
- **Job**: Execution record of a task
- **JobLog**: File-level operation logs

## Development Commands

```bash
# Backend development (requires Go 1.25+)
air                          # Hot-reload dev server (uses .air.toml)
go generate ./...            # Regenerate Ent models + GraphQL schema
./scripts/gen-migration.sh <name>  # Generate DB migration (requires Atlas CLI via nix develop)

# Frontend development (in /web directory)
pnpm dev                     # Vite dev server with HMR
pnpm build                   # Build for production (includes paraglide i18n compile)
pnpm paraglide               # Compile i18n messages only

# Testing
go test ./...                # Run all backend tests
```

## Code Generation Patterns

**Ent ORM** (schema → generated code):
- Schema definitions: `internal/core/db/schema/*.go`
- Run: `go generate ./internal/core/db/...` → outputs to `internal/core/ent/`

**GraphQL (gqlgen)**:
- Schema: `internal/api/graphql/schema/*.graphql`
- Run: `go generate ./internal/api/graphql/resolver/...`
- Also runs `scripts/merge-schema.js` to create unified `web/src/api/graphql/schema.graphql`

**Frontend i18n (Paraglide)**:
- Messages: `web/project.inlang/messages/{en,zh-CN}.json`
- Import: `import * as m from '@/paraglide/messages'` → use `m.messageKey()`

## Project-Specific Patterns

### Error Handling
Use domain errors from `internal/core/errs/errors.go`:
```go
return errs.ErrNotFound  // NOT fmt.Errorf("not found")
```

### Logging
Use hierarchical named loggers:
```go
logger.Named("core.db.query").Debug("...")  // Configurable per-module via config.toml
```

### Service Layer
Services in `internal/core/services/` accept `*ent.Client` and implement `ports.Interface`:
```go
func NewTaskService(client *ent.Client) *TaskService { ... }
var _ ports.TaskService = (*TaskService)(nil)  // Compile-time interface check
```

### GraphQL Resolvers
Resolvers receive dependencies via `resolver.Dependencies` struct (DI pattern):
```go
func (r *queryResolver) Task(...) { r.deps.TaskService.GetTask(...) }
```

### Frontend GraphQL
Use gql.tada for type-safe queries:
```typescript
import { graphql } from '@/api/graphql/graphql';
const TaskQuery = graphql(`query Task($id: ID!) { task { get(id: $id) { id name } } }`);
```

## Database Migrations

Two modes controlled by `config.toml`:
- `migration_mode = "auto"`: Ent auto-migration (development)
- `migration_mode = "versioned"`: golang-migrate files in `internal/core/db/migrations/` (production)

Generate new migration: `./scripts/gen-migration.sh <name>` (uses Atlas CLI)

## Configuration

Environment prefix: `RCLONESYNC_` (e.g., `RCLONESYNC_AUTH_PASSWORD`)  
Config file: `config.toml` with sections: `[server]`, `[database]`, `[log]`, `[app]`, `[security]`, `[auth]`

## Testing Patterns

Backend tests use `enttest` for in-memory SQLite:
```go
client := enttest.Open(t, "sqlite3", "file:test?mode=memory&_fk=1")
```

## Key Files Reference

| Purpose | Location |
|---------|----------|
| DB Schema | `internal/core/db/schema/*.go` |
| GraphQL Schema | `internal/api/graphql/schema/*.graphql` |
| Service Interfaces | `internal/core/ports/interfaces.go` |
| Main Entry | `cmd/rclone-sync/serve.go` |
| Frontend Routes | `web/src/App.tsx` |
| i18n Messages | `web/project.inlang/messages/*.json` |

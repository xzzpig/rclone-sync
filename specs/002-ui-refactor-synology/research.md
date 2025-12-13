# Research: UI Refactor with Solid-UI

## Decisions

### 1. Solid-UI Integration
**Decision**: Use `solidui-cli` to scaffold and manage UI components.
**Rationale**: The user explicitly requested Solid-UI. It provides a headless, accessible, and customizable component library similar to shadcn/ui but for SolidJS.
**Dependencies**: `solidui-cli`, `tailwindcss`, `class-variance-authority`, `clsx`, `tailwind-merge`.

**Setup Steps**:
```bash
# In web/ directory
pnpm dlx solidui-cli@latest init
# Installs needed dependencies and creates components.json
# Add key components
pnpm dlx solidui-cli@latest add button card dialog dropdown-menu input label sheet table tabs toast toggle
```

### 2. Architecture & Directory Structure
**Decision**: Feature-sliced architecture.
**Rationale**: Keeps related logic (API calls, stores, components) together, making the "Connection-Centric" refactor easier to manage than a flat component hierarchy.

**Structure**:
```text
web/src/
├── assets/
├── components/
│   ├── ui/               # Solid-UI components (Button, Input)
│   └── common/           # Shared app-specific components (Loader, ErrorBoundary)
├── modules/              # Business Modules
│   ├── connections/      # Connection list, details, setup wizard
│   │   ├── api.ts        # Enpoints
│   │   ├── components/   # Feature-specific UI
│   │   └── model.ts      # Types/Store
│   ├── tasks/            # Task list, task wizard
│   └── logs/             # History and logs
├── layouts/
│   ├── DashboardLayout/  # Sidebar + Header
│   └── AuthLayout/
├── lib/                  # Utils (cn, fetcher)
├── routes/               # Route definitions
└── App.tsx
```

### 3. State Management & Data Fetching
**Decision**: 
- **Server State**: `@tanstack/solid-query` (v5).
- **Client State**: SolidJS `createStore` for global app settings; `createSignal` for local UI state.
**Rationale**: Cloud Sync is inherently "server-state heavy" (async operations, lists of remote files). React Query/Solid Query handles caching, loading states, and refetching (polling) much better than manual stores.

### 4. API Integration
**Decision**: Manual fetch wrapper around standard `fetch` or `ky`, using generated types if available.
**Rationale**: Since backend is Go/Gin, we can export OpenAPI spec and generate TS types using `openapi-typescript` to ensure contract safety.

**Type Generation**:
1. Go backend serves `/swagger/doc.json`.
2. `pnpm dlx openapi-typescript http://localhost:8080/swagger/doc.json -o ./src/lib/api-types.ts`.

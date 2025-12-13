# Quickstart: UI Development

## Prerequisites
*   Node.js 20+
*   Go 1.22+
*   Rclone installed
*   VSCode with SolidJS and Go extensions

## Initial Setup

1.  **Backend Start**:
    ```bash
    go run cmd/cloud-sync/main.go
    # Serves API at http://localhost:8080
    ```

2.  **Frontend Setup**:
    ```bash
    cd web
    pnpm install
    
    # Initialize Solid-UI (Already done in repo, but if regenerating)
    # pnpm dlx solidui-cli@latest init
    ```

3.  **Frontend Start**:
    ```bash
    cd web
    pnpm dev
    # Serves UI at http://localhost:3000
    ```

## Development Workflow

1.  **Adding a Component**:
    Use the CLI to add standard UI components:
    ```bash
    cd web
    pnpm dlx solidui-cli@latest add [component-name]
    # e.g., pnpm dlx solidui-cli@latest add popover
    ```

2.  **Creating a Module**:
    *   Create folder `web/src/modules/[module-name]`
    *   Define `api.ts` for endpoints.
    *   Create `model.ts` for types.
    *   Build components in `components/` subfolder.

3.  **API Type Generation**:
    If backend structs change `internal/api/`, regenerate types:
    ```bash
    # Ensure backend is running
    cd web
    pnpm dlx openapi-typescript http://localhost:8080/swagger/doc.json -o src/lib/api-types.ts
    ```

## Key Commands

| Command | Description |
| :--- | :--- |
| `pnpm dev` | Start dev server |
| `pnpm build` | Build for production |
| `pnpm dlx solidui-cli add` | Add new UI component |
| `go test ./...` | Run backend tests |


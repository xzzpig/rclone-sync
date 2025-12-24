# Rclone Cloud Sync Manager

A "Synology Cloud Sync"-like application built on top of `rclone`. This application allows you to manage rclone remotes, define sync tasks, and monitor their status through a modern web interface.

## Features

- **Web Interface**: Manage remotes and tasks from a browser.
- **Task Management**: Create, edit, and delete sync tasks.
- **Scheduled Sync**: Run tasks automatically on a schedule (Cron syntax).
- **Real-time Sync**: Monitor file system changes and sync automatically (File Watcher).
- **Bidirectional Sync**: Keep two locations in sync using `rclone bisync` logic.
- **Monitoring**: Real-time progress updates and job history logs.
- **Cross-Platform**: Run on Linux, Windows, macOS (Single binary).
- **GraphQL API**: Modern GraphQL API with real-time subscriptions.

## Requirements

- Go 1.21+ (for building)
- Node.js & pnpm (for frontend development)
- `rclone` (library is embedded, but some external config might be read)
- Modern web browser

## Installation

### From Source

1. Clone the repository:
   ```bash
   git clone https://github.com/xzzpig/rclone-sync.git
   cd rclone-sync
   ```

2. Build the frontend:
   ```bash
   cd web
   pnpm install
   pnpm build
   cd ..
   ```

3. Build the backend (with embedded frontend):
   ```bash
   # Make sure web/dist exists from previous step
   go build -o cloud-sync ./cmd/cloud-sync
   ```

## Usage

1. Start the server:
   ```bash
   ./cloud-sync serve
   ```

2. Open your browser and navigate to:
   ```
   http://localhost:8080
   ```

3. (Optional) Provide a custom rclone config path:
   ```bash
   ./cloud-sync serve --config /path/to/rclone.conf
   ```

## API

### GraphQL Endpoint

The application exposes a GraphQL API at `/api/graphql`.

- **HTTP**: `POST /api/graphql` - Send GraphQL queries and mutations
- **WebSocket**: `GET /api/graphql` - Connect for GraphQL subscriptions (real-time updates)
- **Playground**: `GET /api/graphql/playground` - Interactive GraphQL IDE (development only)

#### Example Queries

**List all connections:**
```graphql
query {
  connection {
    list {
      id
      name
      type
      loadStatus
    }
  }
}
```

**List all tasks:**
```graphql
query {
  task {
    list {
      id
      name
      status
      connection {
        name
      }
    }
  }
}
```

**Create a new task:**
```graphql
mutation {
  task {
    create(input: {
      connectionId: "conn-id"
      name: "My Sync Task"
      localPath: "/data/sync"
      remotePath: "/"
      direction: DOWNLOAD
      mode: COPY
    }) {
      ... on Task {
        id
        name
      }
      ... on Error {
        message
        code
      }
    }
  }
}
```

**Subscribe to job progress:**
```graphql
subscription {
  jobProgress {
    jobId
    taskId
    status
    progress
    speed
    eta
    currentFile
    transferredBytes
    totalBytes
    transferredFiles
    totalFiles
    error
  }
}
```

#### Schema Overview

The GraphQL schema provides the following main types:

- **Connection**: Represents a cloud storage connection (e.g., Google Drive, S3)
- **Task**: A sync task definition with source, destination, and sync settings
- **Job**: An execution instance of a task with progress and logs
- **JobLog**: Log entries for a job execution
- **Provider**: Available storage provider types and their configuration options
- **FileEntry**: File and directory entries for browsing local/remote paths

For the complete schema, visit the GraphQL Playground at `/api/graphql/playground` in development mode.

## Development

### Backend

Running in dev mode (without embedded frontend):

```bash
go run ./cmd/cloud-sync serve
```

### Frontend

Running the frontend dev server:

```bash
cd web
pnpm dev
```

### GraphQL Development

Regenerate GraphQL code after schema changes:

```bash
go generate ./internal/api/graphql/...
```

## Configuration

The application uses `config.toml` for configuration.

```toml
[app]
environment = "production" # or "development"
data_dir = "app_data"

[server]
host = "0.0.0.0"
port = 8080

[rclone]
config_path = "rclone.conf"
log_level = "INFO" # DEBUG, INFO, NOTICE, ERROR
```

## License

MIT

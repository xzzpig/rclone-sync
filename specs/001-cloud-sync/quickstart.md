# Quickstart: Rclone Cloud Sync Manager

## Prerequisites

- **Go**: Version 1.21 or later
- **Node.js**: Version 18 or later (for frontend build)
- **Rclone**: Not required as external binary (embedded in application)

## Build Instructions

1. **Clone the repository**:
   ```bash
   git clone <repo-url>
   cd rclone-sync
   ```

2. **Build Frontend**:
   ```bash
   cd web
   npm install
   npm run build
   cd ..
   ```

3. **Build Backend**:
   ```bash
   go build -o cloud-sync cmd/cloud-sync/main.go
   ```

## Running the Application

1. **Start the server**:
   ```bash
   ./cloud-sync
   ```
   By default, it will listen on `http://localhost:8080`.

2. **Access the UI**:
   Open your browser and navigate to `http://localhost:8080`.

## Configuration

- The application stores its database in `data/cloud-sync.db` by default.
- Rclone configuration is read from the standard location (`~/.config/rclone/rclone.conf`) or can be specified via environment variable `RCLONE_CONFIG`.

## Development Mode

To run with hot-reloading for frontend:

1. Start backend: `go run cmd/cloud-sync/main.go`
2. Start frontend: `cd web && npm start`
3. Access via frontend dev server port (usually 3000), ensuring it proxies API requests to 8080.

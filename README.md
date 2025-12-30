<p align="center">
  <img src="web/src/public/icon.svg" width="128" height="128" alt="Rclone Cloud Sync Manager Icon">
</p>

# Rclone Cloud Sync Manager

A cloud sync management tool based on `rclone` development, designed to provide a user experience similar to "Synology Cloud Sync". Through a modern web interface, easily manage cloud storage connections, define sync tasks, and monitor sync status in real-time.

## ✨ Core Features

- **Modern Web Interface**: Clean and intuitive UI to easily manage all cloud connections and sync tasks.
- **Multi-Cloud Storage Support**: Based on powerful `rclone`, supports dozens of cloud storage services such as Google Drive, S3, OneDrive, Dropbox, etc.
- **Flexible Sync Modes**:
  - **One-way Upload**: Local -> Cloud (Suitable for backup)
  - **One-way Download**: Cloud -> Local (Suitable for fetching resources)
  - **Two-way Sync**: Keep data on both ends consistent (Suitable for multi-end collaboration)
- **Advanced Task Options**:
  - **File Filters**: Include/exclude files using powerful rclone filter patterns
  - **Keep Deleted Files**: Prevent deletion of files in destination (one-way sync only)
  - **Parallel Transfers**: Configure concurrent transfer count (1-64) per task
- **Smart Trigger Mechanism**:
  - **Real-time Sync**: Listen for file system changes and trigger sync immediately.
  - **Scheduled Tasks**: Support custom schedules (Cron) for automatic execution.
- **Visual Monitoring**:
  - **Real-time Progress**: View current transfer files, speed, and remaining time.
  - **Task History**: Detailed execution logs and result records for easy review.
- **Secure and Reliable**:
  - **Encrypted Storage**: Sensitive configuration information is encrypted and stored in the local database.
  - **Data Security**: Strict sync logic prevents accidental data loss.
- **Internationalization Support**: Natively supports **Simplified Chinese** and **English** interfaces.
- **Cross-Platform**: Supports Linux, Windows, macOS.

## ☁️ Supported Cloud Storage

Thanks to the powerful ecosystem of Rclone, this tool supports over 40 cloud storage services, including but not limited to:

- **Public Cloud**: Google Drive, OneDrive, Dropbox, Box, pCloud
- **Object Storage**: Amazon S3 (and compatible protocols like MinIO, Aliyun OSS, Tencent Cloud COS), Backblaze B2, Wasabi
- **Standard Protocols**: WebDAV, FTP, SFTP, HTTP
- **Local/Network**: Local Disk, SMB/CIFS (Windows Sharing)

## 🚀 Installation and Running

### Method One: Download and Run Directly (Recommended)

Please go to the [Releases](https://github.com/xzzpig/rclone-sync/releases) page to download the binary file for your system.

1.  Unzip the downloaded file.
2.  Run in terminal or command line:
    ```bash
    # Linux / macOS
    ./cloud-sync serve

    # Windows
    .\cloud-sync.exe serve
    ```
3.  Open your browser and visit `http://localhost:8080` to start using it.

### Method Two: Build from Source

If you are a developer or want to experience the latest features:

1.  **Clone Repository**:
    ```bash
    git clone https://github.com/xzzpig/rclone-sync.git
    cd rclone-sync
    ```
2.  **Build and Run**:
    ```bash
    # Requires Go 1.25+ and Node.js installed
    # Compile frontend
    cd web && pnpm install && pnpm build && cd ..
    # Compile backend
    go build -o cloud-sync ./cmd/cloud-sync
    # Run
    ./cloud-sync serve
    ```

## 📖 User Guide

### 1. Connect Cloud Storage (Connections)
When entering the system for the first time, please click the **"+"** icon in the sidebar to add a connection.
- Select your cloud storage provider (e.g., Google Drive).
- Complete the authorization process according to the wizard prompts.
- After successful authorization, you will see the connection in the sidebar and can browse the files inside.

### 2. Create Sync Task (Tasks)
On the connection details page, click the **"New Task"** button.
- **Local Path**: Select the folder on your computer that needs to be synced.
- **Remote Path**: Select the folder in the cloud.
- **Sync Direction**:
    - **Upload**: Only push local changes to the cloud.
    - **Download**: Only pull cloud changes to the local.
    - **Bidirectional**: Keep both ends consistent; modifications on either end will sync to the other.
- **Trigger Method**:
    - **Manual**: Sync only when you click "Run".
    - **Schedule**: Set scheduled tasks (e.g., "2 AM every day").
    - **Real-time**: When enabled, automatically starts syncing when files change (with a short delay to optimize performance).

### 3. Monitoring and Logs
- **Dashboard**: In the task list, you can intuitively see the current status of each task (Idle, Syncing, Error).
- **Task Details**: Click the task card to view detailed transfer speed, remaining file count, and historical run logs.
- **History**: The system retains recent sync logs for easy troubleshooting of file transfer issues.

## ❓ Frequently Asked Questions (FAQ)

**Q: Is real-time sync immediate?**
A: To avoid frequent triggers, the system has a "debounce" delay of a few seconds. Sync will start a few seconds after you stop modifying files.

**Q: Where is the configuration file stored?**
A: By default, data is stored in the `app_data` folder in the program's running directory. You can modify this using the `--data-dir` parameter.

**Q: How do I reset the administrator password?**
A: The current version defaults to no login authentication (suitable for personal local use). If you need to deploy on the public network, please use Nginx or other reverse proxies for authentication protection.

## ⚙️ Configuration Instructions

The program reads the `config.toml` file in the current directory by default when starting. You can also specify other paths using the command line parameter `--config`.

Here is a complete configuration example and description:

```toml
[app]
# Operating environment: "development" or "production"
# In production mode, some debugging functions are disabled, and logs are more concise
environment = "production"

# Data storage directory
# Used to store database files, log files, etc.
# Default value: "./app_data"
data_dir = "./app_data"

[server]
# Listening address
# 0.0.0.0 allows LAN/Public access
# 127.0.0.1 allows localhost access only
host = "0.0.0.0"

# Listening port
port = 8080

[log]
# Log level: "debug", "info", "warn", "error"
# "info" is recommended for production environments
level = "info"

# Hierarchical log levels by module name
# Names are case-sensitive, separated by "."
# Example: "core.db" matches "core.db", "core.db.query", etc.
[log.levels]
# "core.db" = "debug"        # core.db and sub-modules use debug level
# "core.scheduler" = "warn"  # core.scheduler uses warn level
# "rclone" = "error"         # rclone module uses error level

[app.job]
# Maximum number of logs retained per connection
# 0 = unlimited (no cleanup)
# Default: 1000
max_logs_per_connection = 1000

# Cron expression for log cleanup task
# Format: minute hour day month weekday
# Default: "0 * * * *" (every hour)
cleanup_schedule = "0 * * * *"

[app.sync]
# Global default parallel transfer count
# Range: 1-64
# Default: 4
transfers = 4

[database]
# Database migration mode
# "auto": Automatic migration (Suitable for development or simple upgrades)
# "versioned": Versioned migration (Suitable for production environments, safer)
migration_mode = "versioned"

# Database file path (Relative to data_dir)
# Default value: "cloud-sync.db"
path = "cloud-sync.db"

[security]
# Encryption key for sensitive data in database, such as cloud storage credentials
# Leave empty to disable encryption (not recommended for production)
encryption_key = ""
```

### Environment Variables

Configuration can also be set via environment variables with the prefix `CLOUDSYNC_`. Nested fields use `_` as a separator.

Examples:
- `CLOUDSYNC_SERVER_PORT=9090`
- `CLOUDSYNC_APP_DATA_DIR=/data`
- `CLOUDSYNC_LOG_LEVEL=debug`

### Command Line Parameters

In addition to the configuration file, you can also override some settings via command line parameters:

- `--config`: Specify configuration file path (Default: `config.toml`)
- `--data-dir`: Specify data storage directory (Overrides setting in config file)
- `--port`: Specify listening port (Overrides setting in config file)
- `--help`: View all available parameters

## 📄 License

MIT License

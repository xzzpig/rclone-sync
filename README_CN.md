<p align="center">
  <img src="web/src/public/icon.svg" width="128" height="128" alt="Rclone Cloud Sync Manager Icon">
</p>

# Rclone Cloud Sync Manager

一个基于 `rclone` 二次开发的云同步管理工具，旨在提供类似 "群晖 Cloud Sync" 的用户体验。通过现代化的 Web 界面，轻松管理云存储连接、定义同步任务，并实时监控同步状态。

## ✨ 核心特性

- **现代化 Web 界面**: 简洁直观的 UI，轻松管理所有云连接和同步任务。
- **多云存储支持**: 基于强大的 `rclone`，支持 Google Drive, S3, OneDrive, Dropbox 等数十种云存储服务。
- **灵活的同步模式**:
  - **单向上传**: 本地 -> 云端 (适合备份)
  - **单向下载**: 云端 -> 本地 (适合拉取资源)
  - **双向同步**: 保持两端数据一致 (适合多端协作)
- **智能触发机制**:
  - **实时同步**: 监听文件系统变动，即时触发同步。
  - **计划任务**: 支持自定义时间表 (Cron)，按计划自动执行。
- **可视化监控**:
  - **实时进度**: 查看当前传输文件、速度、剩余时间。
  - **任务历史**: 详细的执行日志和结果记录，随时回溯。
- **安全可靠**:
  - **加密存储**: 敏感配置信息加密存储于本地数据库。
  - **数据安全**: 严格的同步逻辑，防止意外数据丢失。
- **国际化支持**: 原生支持 **简体中文** 和 **English** 界面。
- **跨平台**: 支持 Linux, Windows, macOS。

## ☁️ 支持的云存储

得益于 Rclone 的强大生态，本工具支持超过 40 种云存储服务，包括但不限于：

- **公有云**: Google Drive, OneDrive, Dropbox, Box, pCloud
- **对象存储**: Amazon S3 (及兼容协议如 MinIO, 阿里云 OSS, 腾讯云 COS), Backblaze B2, Wasabi
- **标准协议**: WebDAV, FTP, SFTP, HTTP
- **本地/网络**: 本地磁盘, SMB/CIFS (Windows 共享)

## 🚀 安装与运行

### 方式一：直接下载运行 (推荐)

请前往 [Releases](https://github.com/xzzpig/rclone-sync/releases) 页面下载对应您系统的二进制文件。

1.  解压下载的文件。
2.  在终端或命令行中运行：
    ```bash
    # Linux / macOS
    ./cloud-sync serve

    # Windows
    .\cloud-sync.exe serve
    ```
3.  打开浏览器访问 `http://localhost:8080` 即可开始使用。

### 方式二：从源码构建

如果您是开发者或希望体验最新功能：

1.  **克隆仓库**:
    ```bash
    git clone https://github.com/xzzpig/rclone-sync.git
    cd rclone-sync
    ```
2.  **构建并运行**:
    ```bash
    # 需要安装 Go 1.25+ 和 Node.js
    # 编译前端
    cd web && pnpm install && pnpm build && cd ..
    # 编译后端
    go build -o cloud-sync ./cmd/cloud-sync
    # 运行
    ./cloud-sync serve
    ```

## 📖 使用指南

### 1. 连接云存储 (Connections)
首次进入系统，请点击侧边栏的 **"+"** 号添加连接。
- 选择您的云存储提供商（如 Google Drive）。
- 根据向导提示完成授权流程。
- 授权成功后，您可以在侧边栏看到该连接，并浏览其中的文件。

### 2. 创建同步任务 (Tasks)
在连接详情页，点击 **"新建任务"** 按钮。
- **本地路径**: 选择您电脑上需要同步的文件夹。
- **远程路径**: 选择云端的文件夹。
- **同步方向**:
    - **上传**: 仅将本地修改推送到云端。
    - **下载**: 仅将云端修改拉取到本地。
    - **双向**: 两端保持一致，任何一端的修改都会同步到另一端。
- **触发方式**:
    - **手动**: 仅在您点击"运行"时同步。
    - **计划**: 设置定时任务（如"每天凌晨2点"）。
    - **实时**: 开启后，文件发生变化时自动开始同步（有短暂延迟以优化性能）。

### 3. 监控与日志
- **仪表盘**: 在任务列表中，您可以直观地看到每个任务的当前状态（空闲、同步中、错误）。
- **任务详情**: 点击任务卡片，查看详细的传输速度、剩余文件数以及历史运行日志。
- **历史记录**: 系统会保留最近的同步日志，方便您排查文件传输问题。

## ❓ 常见问题 (FAQ)

**Q: 实时同步是立即发生的吗？**
A: 为了避免频繁触发，系统会有几秒钟的"防抖"延迟。当您停止修改文件几秒后，同步才会开始。

**Q: 配置文件存储在哪里？**
A: 默认情况下，数据存储在程序运行目录下的 `app_data` 文件夹中。您可以通过 `--data-dir` 参数修改。

**Q: 如何重置管理员密码？**
A: 目前版本默认无登录认证（适合个人本地使用）。如需部署在公网，请配合 Nginx 等反向代理进行认证保护。

## ⚙️ 配置说明

程序启动时会默认读取当前目录下的 `config.toml` 文件。您也可以通过命令行参数 `--config` 指定其他路径。

以下是完整的配置示例及说明：

```toml
[app]
# 运行环境: "development" (开发) 或 "production" (生产)
# 生产模式下会禁用部分调试功能，日志更精简
environment = "production"

# 数据存储目录
# 用于存放数据库文件、日志文件等
# 默认值: "./app_data"
data_dir = "./app_data"

[server]
# 监听地址
# 0.0.0.0 表示允许局域网/公网访问
# 127.0.0.1 表示仅允许本机访问
host = "0.0.0.0"

# 监听端口
port = 8080

[log]
# 日志级别: "debug", "info", "warn", "error"
# 生产环境建议使用 "info"
level = "info"

[database]
# 数据库迁移模式
# "auto": 自动迁移 (适合开发或简单升级)
# "versioned": 版本化迁移 (适合生产环境，更安全)
migration_mode = "versioned"

# 数据库文件路径 (相对于 data_dir)
# 默认值: "cloud-sync.db"
path = "cloud-sync.db"

[security]
# 数据库敏感数据加密密钥，如云存储凭据
# 留空则不加密 (不建议在生产环境使用)
encryption_key = ""
```

### 环境变量

配置也可以通过环境变量设置，前缀为 `CLOUDSYNC_`。嵌套字段使用 `_` 分隔。

示例:
- `CLOUDSYNC_SERVER_PORT=9090`
- `CLOUDSYNC_APP_DATA_DIR=/data`
- `CLOUDSYNC_LOG_LEVEL=debug`

### 命令行参数

除了配置文件，您还可以通过命令行参数覆盖部分设置：

- `--config`: 指定配置文件路径 (默认: `config.toml`)
- `--data-dir`: 指定数据存储目录 (覆盖配置文件中的设置)
- `--port`: 指定监听端口 (覆盖配置文件中的设置)
- `--help`: 查看所有可用参数

## 📄 许可证

MIT License

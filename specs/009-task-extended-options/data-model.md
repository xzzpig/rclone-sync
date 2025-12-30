# Data Model: Task 扩展选项配置

**Feature Branch**: `009-task-extended-options`  
**Created**: 2025-12-28

---

## Entity Changes

### 数据库变更

Task 表不需要新增列，扩展选项存储在现有的 `options` JSON 字段中。

---

## TaskSyncOptions 扩展

### 现有字段

| 字段名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `conflictResolution` | `string` | `"newer"` | 冲突解决策略（仅用于双向同步） |

### 新增字段

| 字段名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `filters` | `[]string` | `[]` | 文件过滤规则列表（rclone filter 语法，每个元素一条规则） |
| `noDelete` | `bool` | `false` | 保留删除文件（仅单向同步有效） |
| `transfers` | `*int` | `nil` | 并行传输数量（1-64，nil 表示使用全局默认值） |

### JSON 存储格式

```json
{
  "conflictResolution": "newer",
  "filters": ["- node_modules/**", "- .git/**", "+ **"],
  "noDelete": false,
  "transfers": 8
}
```

---

## 过滤器规则格式

### 存储格式

字符串数组，每个元素为一条 rclone filter 规则：

```json
["- node_modules/**", "- .git/**", "- *.tmp", "+ **"]
```

### UI 规则列表（前端使用）

```typescript
// 直接使用字符串数组，无需额外转换
const filters: string[] = ["- node_modules/**", "- .git/**", "+ **"];
```

### 规则语法

- `- pattern`: 排除匹配的文件/目录
- `+ pattern`: 包含匹配的文件/目录
- 规则按顺序匹配，第一个匹配的规则生效

### 常用规则示例

```
- node_modules/**     # 排除 node_modules 目录
- .git/**             # 排除 .git 目录
- *.tmp               # 排除所有 .tmp 文件
- ~$*                 # 排除 Office 临时文件
- .DS_Store           # 排除 macOS 元数据文件
+ **                  # 包含其他所有文件
```

---

## GraphQL Schema Changes

详细定义请参考 [contracts/schema.graphql](./contracts/schema.graphql)

### 变更概览

| 类型 | 变更类型 | 说明 |
|------|----------|------|
| `TaskSyncOptions` | 扩展 | 新增 `filters`, `noDelete`, `transfers` |
| `TaskSyncOptionsInput` | 扩展 | 新增 `filters`, `noDelete`, `transfers` |
| `Query.file.remote` | 扩展 | 新增 `filters`, `includeFiles` 参数用于过滤器预览 |

---

## Configuration Changes

### config.toml 新增配置项

```toml
[sync]
# 全局默认并行传输数量
# 范围: 1-64，默认: 4
transfers = 4
```

### Config struct 扩展

```go
type Config struct {
  // ... 现有字段
  Sync struct {
    Transfers int `mapstructure:"transfers"` // 默认 4
  } `mapstructure:"sync"`
}
```

### 优先级逻辑

```
任务级 transfers → 配置文件 sync.transfers → rclone 默认值 (4)
```

---

## Validation Rules

| 字段 | 验证规则 |
|------|----------|
| `filters` | 每条规则必须以 `+` 或 `-` 开头，后跟空格和模式；使用 rclone filter.AddRule 验证语法 |
| `noDelete` | 仅在 `direction` 为 `UPLOAD` 或 `DOWNLOAD` 时有效；双向同步模式下后端静默忽略 |
| `transfers` | 必须在 1-64 范围内（如果设置） |

### 验证时机

- **过滤器规则**: 仅在保存任务时验证（不进行实时校验）
- **transfers 范围**: 在保存任务时验证

---

## Dependencies

| 组件 | 依赖 | 说明 |
|------|------|------|
| SyncEngine | `filter.ReplaceConfig` | 注入过滤器到 context |
| SyncEngine | `fs.AddConfig` | 注入并行传输数量到 context |
| SyncEngine | `sync.CopyDir` | 实现保留删除文件功能 |
| TaskService | `rclone/filter` 包 | 验证过滤器规则语法 |
| ListRemoteDir | `filter.ReplaceConfig` | 过滤器预览功能（在 connection.go 中） |

---

## Data Flow

### 保存任务时

```
[用户提交表单]
    ↓
[前端序列化为 TaskSyncOptionsInput]
    ↓
[GraphQL Mutation: task.create/update]
    ↓
[TaskService.validateSyncOptions() 验证]
    ↓ (验证通过)
[序列化到 Task.Options JSON 字段]
    ↓
[保存到数据库]
```

### 执行同步时

```
[读取 Task.Options JSON]
    ↓
[解析 SyncOptions]
    ↓
[应用 Filters: filter.ReplaceConfig]
    ↓
[应用 Transfers: fs.AddConfig]
    ↓
[判断 NoDelete: CopyDir vs Sync]
    ↓
[执行 rclone 同步]
```

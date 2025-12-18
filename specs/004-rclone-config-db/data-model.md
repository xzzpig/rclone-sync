# Data Model: Rclone 连接配置数据库存储

**Feature**: 004-rclone-config-db  
**Date**: 2025-12-15  
**Status**: Complete

## Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        Connection                            │
├─────────────────────────────────────────────────────────────┤
│ id: UUID (PK)                                               │
│ name: String (UNIQUE, NOT NULL)                             │
│ type: String (NOT NULL)  -- e.g., onedrive, s3, local       │
│ encrypted_config: Bytes (NOT NULL)  -- AES-GCM 加密的配置    │
│ created_at: DateTime                                        │
│ updated_at: DateTime                                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ 1:N (CASCADE DELETE)
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                          Task                                │
├─────────────────────────────────────────────────────────────┤
│ id: UUID (PK)                                               │
│ connection_id: UUID (FK → Connection.id)                    │
│ name: String                                                │
│ source_path: String                                         │
│ remote_path: String                                         │
│ direction: Enum                                             │
│ ...                                                         │
└─────────────────────────────────────────────────────────────┘
```

## Connection Entity

### Ent Schema Definition

```go
// internal/core/db/schema/connection.go
package schema

import (
    "time"

    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
    "github.com/google/uuid"
)

// Connection holds the schema definition for the Connection entity.
type Connection struct {
    ent.Schema
}

// Fields of the Connection.
func (Connection) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("id", uuid.UUID{}).
            Default(uuid.New),
        field.String("name").
            NotEmpty().
            Unique().
            Comment("Remote name, must be unique across the system"),
        field.String("type").
            NotEmpty().
            Comment("Provider type, e.g., onedrive, s3, drive, local"),
        field.Bytes("encrypted_config").
            Comment("AES-GCM encrypted configuration JSON"),
        field.Time("created_at").
            Default(time.Now).
            Immutable(),
        field.Time("updated_at").
            Default(time.Now).
            UpdateDefault(time.Now),
    }
}

// Indexes of the Connection.
func (Connection) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("name").Unique(),
        index.Fields("type"),
    }
}

// Edges of the Connection.
func (Connection) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("tasks", Task.Type).
            Annotations(entsql.OnDelete(entsql.Cascade)),
    }
}
```

### Field Descriptions

| Field            | Type     | Constraints        | Description                                   |
| ---------------- | -------- | ------------------ | --------------------------------------------- |
| id               | UUID     | PK, auto-generated | 唯一标识符                                    |
| name             | String   | UNIQUE, NOT EMPTY  | 连接名称，对应 rclone remote 名称             |
| type             | String   | NOT EMPTY          | rclone 提供商类型 (onedrive, s3, drive, etc.) |
| encrypted_config | Bytes    | NOT NULL           | AES-GCM 加密的完整配置 JSON                   |
| created_at       | DateTime | IMMUTABLE          | 创建时间                                      |
| updated_at       | DateTime | AUTO UPDATE        | 更新时间                                      |

### Encrypted Config

配置使用 AES-256-GCM 进行整体加密存储，支持可选加密：

**加密模式**（配置了加密密钥）:

```
config map[string]string → JSON 序列化 → AES-GCM 加密 → encrypted_config bytes
```

**Plaintext 模式**（未配置加密密钥）:

```
config map[string]string → JSON 序列化 → encrypted_config bytes (明文JSON)
```

**解密流程**:

```
encrypted_config bytes → (如需要)AES-GCM 解密 → JSON 反序列化 → config map[string]string
```

**加密密钥**:

- 支持任意长度的密钥（通过 SHA-256 自动转换为 32 字节）
- 密钥为空时启用 plaintext 模式（不加密，适合开发环境）
- 密钥从配置文件或环境变量 `CLOUDSYNC_SECURITY_ENCRYPTION_KEY` 获取

**原始配置结构** (加密前/解密后):

```json
{
  "type": "onedrive",
  "token": "{\"access_token\":\"...\",\"refresh_token\":\"...\"}",
  "drive_id": "abc123",
  "drive_type": "personal"
}
```

**优点**:

- 实现简单，无需识别敏感字段
- 所有配置信息都被保护
- 无需维护敏感字段列表

## Validation Rules

### Connection Name

- 不能为空
- 必须唯一（系统范围内）
- 只能包含字母、数字、下划线和连字符
- 不能以数字开头
- 最大长度 64 字符

```go
// ValidateConnectionName 验证连接名称
func ValidateConnectionName(name string) error {
    if name == "" {
        return errors.New("name cannot be empty")
    }
    if len(name) > 64 {
        return errors.New("name too long (max 64 characters)")
    }
    // 使用与 rclone 相同的命名规则
    matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_-]*$`, name)
    if !matched {
        return errors.New("invalid name format")
    }
    return nil
}
```

### Config Validation

- `type` 键必须存在且为有效的 rclone 提供商
- 必填参数必须提供（根据提供商类型不同）

## Database Migration

### Migration Strategy

1. **创建新表**: 添加 `connections` 表
2. **修改 Task 表**:
   - 添加 `connection_id` 字段 (UUID, 外键到 Connection.id)
   - 移除 `remote_name` 字段
3. **数据迁移**: 无需自动迁移（应用尚未正式发布）

### Ent Migration

```bash
# 生成迁移文件
go generate ./internal/core/ent

# 应用迁移
go run -mod=mod entgo.io/ent/cmd/ent generate ./internal/core/db/schema
```

## Frontend Type Definitions

```typescript
// web/src/lib/types.ts

export type LoadStatus = "loaded" | "loading" | "error";

export interface Connection {
  id: string;
  name: string;
  type: string;
  created_at: string;
  updated_at: string;

  // 加载状态（从内存获取，非数据库字段）
  load_status: LoadStatus;
  load_error?: string;
}

// 创建/更新请求（config 不在响应中返回，只在请求中使用）
export interface ConnectionRequest {
  name: string;
  type: string;
  config: Record<string, string>;
}

// 导入预览项
export interface ImportPreviewItem {
  name: string;
  type: string;
  config: Record<string, string>; // 导入预览时显示，仅在导入流程中使用
  test_status: "pending" | "success" | "failed";
  test_error?: string;
  will_overwrite: boolean;
}

// 导入验证结果
export interface ImportValidation {
  valid: boolean;
  connections: ImportPreviewItem[];
  errors: ImportError[];
}

export interface ImportError {
  name: string;
  error: string;
}
```

## API Response Formats

### List Connections Response

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "my-onedrive",
    "type": "onedrive",
    "created_at": "2024-12-01T08:00:00Z",
    "updated_at": "2024-12-15T10:30:00Z",
    "load_status": "loaded",
    "load_error": null
  }
]
```

### Get Connection Detail Response

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "my-onedrive",
  "type": "onedrive",
  "created_at": "2024-12-01T08:00:00Z",
  "updated_at": "2024-12-15T10:30:00Z",
  "load_status": "loaded",
  "load_error": null
}
```

**说明**：

- `load_status` 从内存获取（检查 rclone cache），不存储在数据库
- 配置使用整体加密存储，API 响应中不返回 config 字段
- 如需编辑配置，使用 `GET /connections/{id}/config` 获取完整配置

## Relationship with Existing Entities

### Connection → Task (1:N, CASCADE DELETE)

Task 实体通过 `connection_id` 外键与 Connection 建立强关联。

#### Task Schema 更新

```go
// internal/core/db/schema/task.go (修改部分)
func (Task) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("id", uuid.UUID{}).Default(uuid.New),
        field.String("name").NotEmpty(),
        field.String("source_path").NotEmpty(),
        field.UUID("connection_id", uuid.UUID{}).Optional(),  // 外键
        field.String("remote_path").NotEmpty(),
        // ... 其他字段
    }
}

func (Task) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("jobs", Job.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
        edge.From("connection", Connection.Type).
            Ref("tasks").
            Unique().
            Field("connection_id"),  // 绑定到外键字段
    }
}
```

#### 级联删除策略

删除 Connection 时会自动级联删除所有关联的 Task 及其 Job、JobLog：

```
Connection (删除)
    └─> Tasks (CASCADE 删除)
            └─> Jobs (CASCADE 删除)
                    └─> JobLogs (CASCADE 删除)
```

**设计理由**:

- 连接是任务的核心依赖，无连接则任务无法执行
- 避免遗留无效的任务记录
- 简化用户操作，无需手动清理关联任务

**UI 提示**: 删除连接前应显示警告，告知用户会同时删除 N 个关联任务。

#### 与 rclone 的兼容性

虽然 Task 通过 ID 关联 Connection，但在执行同步时仍通过 Connection.name 调用 rclone：

```go
// 执行任务时
task := taskService.GetTask(ctx, taskID)
task.Edges.Connection.Name  // 获取关联的 Connection.name

remotePath := fmt.Sprintf("%s:%s", task.Edges.Connection.Name, task.RemotePath)
f, err := fs.NewFs(ctx, remotePath)  // rclone 通过 name 从 DBStorage 读取配置
```

这种设计的好处：

1. Task 与 Connection 强关联，数据完整性更好
2. Connection.name 可以修改（Task 通过 ID 引用，不受影响）
3. 符合 rclone 的工作方式（通过 name 查找配置）

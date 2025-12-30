# Quickstart: Task 扩展选项配置

**Feature Branch**: `009-task-extended-options`  
**Created**: 2025-12-28

---

## Prerequisites

- Go 1.21+
- Node.js 18+ with pnpm
- 已有可用的开发环境

---

## Quick Verification

### 1. 验证过滤器规则校验

保存任务时会自动校验 filter 规则语法，无效的规则会被拒绝。

```graphql
# 测试无效的过滤器规则
mutation {
  task {
    update(id: "YOUR_TASK_ID", input: {
      options: {
        filters: [
          "**.jpg",       # 无效规则：缺少 + 或 - 前缀
          "+ *.png"
        ]
      }
    }) {
      id
      options {
        filters
      }
    }
  }
}
```

**预期结果**:
- 请求返回错误，类似: `规则 #1 "**.jpg" 无效: ...`
- 任务未被保存

```graphql
# 测试有效的过滤器规则
mutation {
  task {
    update(id: "YOUR_TASK_ID", input: {
      options: {
        filters: [
          "+ *.jpg",      # 包含所有 jpg 文件
          "+ *.png",      # 包含所有 png 文件
          "- *"           # 排除其他所有文件
        ]
      }
    }) {
      id
      options {
        filters
      }
    }
  }
}

# 触发同步任务，验证过滤器生效
```

**预期结果**:
- 有效规则保存成功
- 只有 `.jpg` 和 `.png` 文件被同步
- 其他文件被排除

---

### 2. 验证过滤器预览

过滤器预览复用现有的 `file.remote` 接口，通过 `filters` 和 `includeFiles` 参数实现。

```graphql
# 使用 file.remote 接口预览过滤器规则效果
query {
  file {
    remote(
      connectionId: "YOUR_CONNECTION_ID"
      path: "/"
      filters: [
        "+ *.jpg",
        "+ *.png",
        "- *"
      ]
      includeFiles: true
      pagination: { first: 100 }
    ) {
      items {
        name
        isDir
        size
        modTime
      }
      pageInfo {
        hasNextPage
      }
    }
  }
}
```

**预期结果**:
- `items`: 返回符合过滤规则的文件和目录列表
- 只有 `.jpg` 和 `.png` 文件会显示在列表中
- 其他类型的文件被过滤掉不显示
- `pageInfo.hasNextPage`: 指示是否有更多结果

**前端集成**:
- 过滤器预览面板默认折叠，点击展开后才发起查询（懒加载）
- 复用 `FileBrowser.tsx` 组件展示过滤后的文件列表
- 文件项显示对应的文件图标（图片/视频/文档等）

---

### 3. 验证保留删除文件

```graphql
# 设置 noDelete 选项
mutation {
  task {
    update(id: "YOUR_TASK_ID", input: {
      options: {
        noDelete: true
      }
    }) {
      id
      options {
        noDelete
      }
    }
  }
}
```

**测试流程**:
1. 在源目录创建文件 A、B、C
2. 执行同步，目标目录有 A、B、C
3. 在源目录删除文件 C
4. 再次执行同步

```bash
# 验证目标目录
ls -la /path/to/dest/
```

**预期结果**:
- 当 `noDelete: true` 时，目标目录仍保留文件 C
- 当 `noDelete: false`（默认）时，目标目录的文件 C 会被删除

**注意**: `noDelete` 选项仅在单向同步模式（Upload/Download）下有效，双向同步模式下 UI 应隐藏该选项。

---

### 4. 验证并行传输数量

```graphql
# 设置 transfers 选项
mutation {
  task {
    update(id: "YOUR_TASK_ID", input: {
      options: {
        transfers: 8  # 使用 8 个并行传输
      }
    }) {
      id
      options {
        transfers
      }
    }
  }
}

# 触发同步任务（使用多个大文件以便观察效果）
# 查看 rclone 日志或使用 htop 观察并发传输
```

**预期结果**:
- 同步过程中最多有 8 个并行传输
- 默认值为 4，范围 1-64

**配置优先级**:
```
任务级 transfers → 配置文件 sync.transfers → rclone 默认值 (4)
```

---

### 5. 验证配置文件

```toml
# config.toml
[sync]
# 全局默认并行传输数量
# 范围: 1-64，默认: 4
transfers = 4
```

---

## Development Workflow

### Step 1: 修改 GraphQL Schema

```bash
# 编辑 schema 文件
vim internal/api/graphql/schema/task.graphql
vim internal/api/graphql/schema/file.graphql

# 重新生成代码
go generate ./...
```

### Step 2: 实现过滤器验证

```bash
# 创建过滤器验证函数
vim internal/rclone/filter_validator.go
vim internal/rclone/filter_validator_test.go

# 运行测试
go test ./internal/rclone/... -v
```

### Step 3: 修改 Sync Engine

```bash
# 编辑 sync.go 添加过滤器和传输数量支持
vim internal/rclone/sync.go

# 运行测试
go test ./internal/rclone/... -v
```

### Step 4: 修改 Connection 模块（过滤器预览）

```bash
# 编辑 connection.go 添加过滤器预览支持（扩展 ListRemoteDir）
vim internal/rclone/connection.go

# 运行测试
go test ./internal/rclone/... -v
```

### Step 5: 实现 Resolver

```bash
# 编辑 resolver 文件
vim internal/api/graphql/resolver/task.resolvers.go
vim internal/api/graphql/resolver/file.resolvers.go

# 编写测试
vim internal/api/graphql/resolver/task_test.go
vim internal/api/graphql/resolver/file_test.go

# 运行测试
go test ./internal/api/graphql/resolver/... -v
```

### Step 6: 前端开发

```bash
cd web

# 更新 GraphQL 查询
vim src/api/graphql/queries/tasks.ts
vim src/api/graphql/queries/files.ts

# 重新生成类型
pnpm codegen

# 创建过滤器编辑器组件
vim src/modules/connections/components/FilterRulesEditor.tsx
vim src/modules/connections/components/FilterPreviewPanel.tsx

# 更新任务设置页面
vim src/modules/connections/views/Tasks.tsx

# 添加翻译
vim project.inlang/messages/en.json
vim project.inlang/messages/zh-CN.json

# 启动开发服务器
pnpm dev
```

---

## Testing Checklist

### Backend Tests

```bash
# 全量测试
go test ./... -v

# 特定模块
go test ./internal/rclone/... -v
go test ./internal/api/graphql/resolver/... -v
go test ./internal/core/services/... -v
```

### Frontend Manual Tests

1. **过滤器规则配置**
   - [ ] 打开任务设置页面
   - [ ] 点击 "过滤器" Tab 标签
   - [ ] 添加规则（选择 Include/Exclude + 输入模式）
   - [ ] 验证规则可以上移、下移、删除
   - [ ] 保存任务，验证规则保存成功
   - [ ] 输入无效规则（如空模式），验证保存时显示错误

2. **过滤器预览**
   - [ ] 配置过滤器规则后
   - [ ] 点击 "预览过滤后的文件" 折叠面板
   - [ ] 验证显示符合过滤规则的文件列表
   - [ ] 验证文件显示正确的图标

3. **保留删除文件**
   - [ ] 打开单向同步任务设置页面
   - [ ] 验证显示 "保留删除文件" 开关
   - [ ] 打开双向同步任务设置页面
   - [ ] 验证 "保留删除文件" 选项隐藏
   - [ ] 启用选项后执行同步，验证目标端文件不被删除

4. **并行传输数量**
   - [ ] 打开任务设置页面
   - [ ] 验证显示 "并行传输数量" 数字输入框
   - [ ] 输入有效值（1-64），验证保存成功
   - [ ] 输入无效值（0 或 100），验证显示错误

5. **任务详情展示**
   - [ ] 配置扩展选项后
   - [ ] 打开任务详情页面
   - [ ] 验证显示已配置的过滤器规则
   - [ ] 验证显示 "保留删除文件" 状态
   - [ ] 验证显示 "并行传输数量" 值

---

## Configuration Reference

```toml
# config.toml

[sync]
# 全局默认并行传输数量
# 范围: 1-64
# 默认: 4
transfers = 4
```

---

## Common Issues

### Q: 过滤器规则保存失败

**原因**: 规则语法错误  
**解决**: 确保每条规则以 `+` 或 `-` 开头，后跟空格和模式

### Q: 过滤器预览显示空列表

**原因**: 没有文件匹配过滤规则  
**解决**: 检查规则是否正确，确保最后有 `+ **` 或 `- **` 规则

### Q: noDelete 选项不生效

**原因**: 任务是双向同步模式  
**解决**: 该选项仅在单向同步模式下有效

### Q: 并行传输数量设置不生效

**原因**: 可能被全局配置覆盖  
**解决**: 检查 config.toml 中的 `[sync].transfers` 设置

### Q: 过滤器预览不显示文件

**原因**: `includeFiles` 参数未设置为 true  
**解决**: 确保在查询 file.remote 时传递 `includeFiles: true`

---

## Code Implementation References

### filter_validator.go 完整实现

```go
package rclone

import (
    "fmt"
    "github.com/rclone/rclone/fs/filter"
)

// ValidateFilterRules 校验过滤器规则语法
// 返回 nil 表示所有规则有效，否则返回第一个无效规则的错误信息
func ValidateFilterRules(rules []string) error {
    fi, err := filter.NewFilter(nil)
    if err != nil {
        return fmt.Errorf("failed to create filter: %w", err)
    }
    
    for i, rule := range rules {
        // AddRule 会解析规则并返回语法错误
        if err := fi.AddRule(rule); err != nil {
            return fmt.Errorf("规则 #%d %q 无效: %w", i+1, rule, err)
        }
    }
    return nil
}
```

### TaskService.validateSyncOptions 实现

```go
func (s *TaskService) validateSyncOptions(opts *TaskSyncOptionsInput) error {
    // 1. 校验 filters 规则语法
    if len(opts.Filters) > 0 {
        if err := rclone.ValidateFilterRules(opts.Filters); err != nil {
            return errs.NewValidationError("syncOptions.filters", err.Error())
        }
    }
    
    // 2. 校验 transfers 范围
    if opts.Transfers != nil && (*opts.Transfers < 1 || *opts.Transfers > 64) {
        return errs.NewValidationError("syncOptions.transfers", "must be between 1 and 64")
    }
    
    return nil
}
```

### Sync 方法中应用选项

```go
// Filters: 使用 rclone filter 包
if len(opts.Filters) > 0 {
    fi, err := filter.NewFilter(nil)
    if err != nil {
        return err
    }
    for _, rule := range opts.Filters {
        if err := fi.AddRule(rule); err != nil {
            return fmt.Errorf("invalid filter rule '%s': %w", rule, err)
        }
    }
    ctx = filter.ReplaceConfig(ctx, fi)
}

// Transfers: 使用 fs.AddConfig
if opts.Transfers > 0 {
    ci := fs.GetConfig(ctx)
    ci.Transfers = opts.Transfers
    ctx = fs.AddConfig(ctx, ci)
}

// NoDelete: 使用 CopyDir 替代 Sync
if opts.NoDelete {
    return sync.CopyDir(ctx, dstFs, srcFs, false)
}
return sync.Sync(ctx, dstFs, srcFs, false)
```

### FilterPreviewPanel 组件示例

```tsx
import { FileBrowser } from '@/components/common/FileBrowser';

export const FilterPreviewPanel: Component<{
  connectionId: string;
  path: string;
  filters: string[];
}> = (props) => {
  const [isOpen, setIsOpen] = createSignal(false);
  
  const loadDirectory = async (path: string, refresh?: boolean) => {
    // 调用 file.remote 时传入 filters 和 includeFiles: true
    const result = await client.query({
      file: {
        remote: [{
          connectionId: props.connectionId,
          path,
          filters: props.filters,
          includeFiles: true,
        }]
      }
    });
    return result.file.remote;
  };
  
  return (
    <Collapsible open={isOpen()} onOpenChange={setIsOpen}>
      <CollapsibleTrigger class="flex items-center gap-2">
        <span>{isOpen() ? '▼' : '▶'}</span>
        <span>{m.task_previewFilteredFiles()}</span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <Show when={isOpen()}>
          <FileBrowser
            initialPath={props.path}
            loadDirectory={loadDirectory}
            onSelect={() => {}}
          />
        </Show>
      </CollapsibleContent>
    </Collapsible>
  );
};
```

---

## Related Files

| 类型 | 文件 |
|------|------|
| Spec | `specs/009-task-extended-options/spec.md` |
| Plan | `specs/009-task-extended-options/plan.md` |
| Research | `specs/009-task-extended-options/research.md` |
| Data Model | `specs/009-task-extended-options/data-model.md` |
| Contract | `specs/009-task-extended-options/contracts/schema.graphql` |

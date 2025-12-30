# Data Model: Rclone Fs Cache Optimization

**Feature**: 010-rclone-fs-cache  
**Date**: 2025-12-30  
**Status**: N/A - No Data Model Changes

## Overview

本功能为内部性能优化，**不涉及任何数据模型变更**。

## Entities Analysis

### Affected Components

本功能仅涉及 rclone Fs 实例的运行时缓存管理，不涉及持久化数据：

| Component | Type | Change Required |
|-----------|------|-----------------|
| Connection (Ent Entity) | Database | ❌ No change |
| Task (Ent Entity) | Database | ❌ No change |
| Job (Ent Entity) | Database | ❌ No change |
| Fs Cache | Runtime Memory | ✅ Use rclone built-in cache |

### Runtime State (Non-Persistent)

rclone 内置的 Fs 缓存是运行时内存状态，不需要持久化：

```go
// rclone 内部缓存结构（参考，非我们维护）
// 位于 github.com/rclone/rclone/fs/cache/cache.go

var (
    cacheMu sync.Mutex
    cache   = map[string]*cacheEntry{}  // remote path -> cached Fs
)

type cacheEntry struct {
    f     fs.Fs      // 缓存的 Fs 实例
    pinCount int     // 引用计数
    // ...
}
```

## Validation Rules

不适用 - 本功能不引入新的数据验证规则。

## State Transitions

不适用 - 本功能不涉及实体状态变更。

## Relationships

不适用 - 本功能不修改实体关系。

## Migration

不适用 - 无数据库 schema 变更，无需迁移。

## Summary

本功能是一个纯粹的内部重构优化：
- 将 `fs.NewFs` 调用替换为 `cache.Get` 调用
- 配置变更时使用 `cache.ClearConfig` 清除缓存
- 所有变更都在运行时层面，不影响持久化数据结构

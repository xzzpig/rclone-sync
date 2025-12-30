# Feature Specification: HTTP Basic Auth 认证

**Feature Branch**: `011-basic-auth`  
**Created**: 2024-12-30  
**Status**: Draft  
**Input**: 系统没有任何认证，添加 HTTP Basic Auth 基本认证

## 概述

为 rclone-sync 系统添加 HTTP Basic Auth 认证。用户在配置文件或环境变量中设置用户名和密码，浏览器访问时会弹出标准的 HTTP 认证对话框。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 访问受保护资源 (Priority: P1)

用户访问系统时，浏览器弹出 HTTP Basic Auth 对话框要求输入凭据。

**Why this priority**: 这是唯一的核心功能。

**Independent Test**: 访问任意页面，验证浏览器弹出认证对话框，输入正确凭据后可正常访问。

**Acceptance Scenarios**:

1. **Given** 用户访问系统, **When** 未提供凭据, **Then** 浏览器弹出 HTTP Basic Auth 对话框
2. **Given** 用户在认证对话框, **When** 输入正确的用户名和密码, **Then** 正常访问系统
3. **Given** 用户在认证对话框, **When** 输入错误的凭据, **Then** 返回 401 状态码，继续要求认证
4. **Given** 认证成功后, **When** 继续访问其他页面, **Then** 浏览器自动携带凭据，无需重复输入

---

### User Story 2 - 配置凭据 (Priority: P1)

管理员通过配置文件或环境变量设置认证凭据。

**Why this priority**: 凭据配置是认证的前提。

**Acceptance Scenarios**:

1. **Given** 配置文件设置了 `auth.username` 和 `auth.password`, **When** 启动服务, **Then** 使用该凭据进行认证
2. **Given** 设置了环境变量 `CLOUDSYNC_AUTH_USERNAME` 和 `CLOUDSYNC_AUTH_PASSWORD`, **When** 启动服务, **Then** 环境变量优先于配置文件
3. **Given** 未配置任何凭据, **When** 启动服务, **Then** 认证禁用，系统保持开放访问（向后兼容）

---

### Edge Cases

- 用户名或密码为空时，视为未配置认证
- 健康检查端点 `/health` 无需认证
- GraphQL 端点 `/api/graphql` 需要认证（与其他受保护资源一致）
- 认证失败时记录 Warn 级别日志（包含 IP、时间、尝试的用户名），但不做自动限流
- 所有静态资源（HTML/JS/CSS）都需要认证
- 启动时验证凭据配置有效性，配置无效（如仅设置用户名未设置密码）则拒绝启动并报错

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 支持 HTTP Basic Auth 认证，Realm 名称为 "Login Required"
- **FR-002**: 系统 MUST 支持通过配置文件 [auth] 块设置用户名和密码
- **FR-003**: 系统 MUST 支持通过环境变量 (CLOUDSYNC_AUTH_*) 覆盖凭据配置
- **FR-004**: 未配置凭据时，系统 MUST 保持开放访问（向后兼容）
- **FR-005**: `/health` 端点 MUST 无需认证

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 未认证请求返回 401 状态码
- **SC-002**: 认证成功后可正常访问所有功能
- **SC-003**: 未配置凭据时系统行为与当前一致

## Assumptions

- 使用标准 HTTP Basic Auth，由浏览器处理认证对话框
- 无状态验证：后端不存储会话，每次请求均对比当前配置凭据
- 密码在配置文件中明文存储（用户自行保护配置文件安全，详见 quickstart.md 安全建议）
- 单用户模式，只需一组凭据
- 不强制 HTTPS，但文档中建议生产环境使用 HTTPS 或反向代理以保护凭据传输安全

## Clarifications

### Session 2024-12-30

- Q: HTTPS 传输要求 → A: 不强制，但在文档中建议生产环境使用 HTTPS 或反向代理
- Q: GraphQL 端点认证 → A: GraphQL 端点 `/api/graphql` 需要认证（与其他受保护资源一致）
- Q: 暴力破解防护 → A: 仅记录失败尝试日志，不做自动限流
- Q: 静态资源认证 → A: 所有静态资源都需要认证（与 API 一致）
- Q: 凭据验证时机 → A: 启动时验证，配置无效则拒绝启动并报错
- Q: HTTP Basic Auth Realm 名称 → A: 使用 "Login Required" 作为 Realm 名称
- Q: 配置数据结构位置 → A: 在 Config 结构体中新增顶层 Auth 字段 (对应 [auth] 配置块)
- Q: 认证失败日志内容 → A: 记录 IP、时间、以及尝试使用的用户名
- Q: 认证状态持久化 → A: 无状态验证，每次请求直接对比当前配置凭据

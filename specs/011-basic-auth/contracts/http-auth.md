# HTTP API Contracts: Basic Auth

**Feature**: 011-basic-auth  
**Date**: 2024-12-30  
**Type**: HTTP Authentication Layer

## Overview

HTTP Basic Auth 在 HTTP 层实现，不影响 GraphQL schema。本文档定义 HTTP 认证行为契约。

## Authentication Behavior Contract

### Realm Name

```
WWW-Authenticate: Basic realm="Login Required"
```

### Protected Endpoints

| Endpoint Pattern | Auth Required | Notes |
|-----------------|---------------|-------|
| `GET /health` | ❌ No | 健康检查，始终开放 |
| `GET /api/*` | ✅ Yes | 所有 API 端点 |
| `POST /api/*` | ✅ Yes | 所有 API 端点 |
| `GET /api/graphql` | ✅ Yes | GraphQL endpoint |
| `POST /api/graphql` | ✅ Yes | GraphQL mutations/queries |
| `GET /*` (static files) | ✅ Yes | HTML/JS/CSS 静态资源 |

### Response Codes

| Scenario | Status Code | Headers | Body |
|----------|-------------|---------|------|
| No auth header (auth enabled) | 401 | `WWW-Authenticate: Basic realm="Login Required"` | - |
| Invalid credentials | 401 | `WWW-Authenticate: Basic realm="Login Required"` | - |
| Valid credentials | (pass through) | - | - |
| Auth disabled | (pass through) | - | - |

## HTTP Request/Response Examples

### Scenario 1: Unauthenticated Request (Auth Enabled)

**Request:**
```http
GET /api/graphql HTTP/1.1
Host: localhost:8080
```

**Response:**
```http
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Basic realm="Login Required"
Content-Length: 0
```

### Scenario 2: Invalid Credentials

**Request:**
```http
GET /api/graphql HTTP/1.1
Host: localhost:8080
Authorization: Basic aW52YWxpZDppbnZhbGlk
```
*(Base64 of "invalid:invalid")*

**Response:**
```http
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Basic realm="Login Required"
Content-Length: 0
```

### Scenario 3: Valid Credentials

**Request:**
```http
GET /api/graphql HTTP/1.1
Host: localhost:8080
Authorization: Basic YWRtaW46c2VjcmV0MTIz
```
*(Base64 of "admin:secret123")*

**Response:**
```http
HTTP/1.1 200 OK
Content-Type: application/json
...
```

### Scenario 4: Health Check (Always Open)

**Request:**
```http
GET /health HTTP/1.1
Host: localhost:8080
```

**Response:**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok"}
```

## Configuration Contract

### TOML Schema

```toml
[auth]
# Username for HTTP Basic Auth
# Type: string
# Default: "" (empty = auth disabled)
username = ""

# Password for HTTP Basic Auth
# Type: string
# Default: "" (empty = auth disabled)
password = ""
```

### Environment Variables Schema

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `CLOUDSYNC_AUTH_USERNAME` | string | "" | Authentication username |
| `CLOUDSYNC_AUTH_PASSWORD` | string | "" | Authentication password |

### Validation Contract

| Configuration State | Startup Behavior | Runtime Behavior |
|--------------------|------------------|------------------|
| username="" && password="" | ✅ Start normally | Auth disabled, open access |
| username="x" && password="y" | ✅ Start normally | Auth enabled, require credentials |
| username="x" && password="" | ❌ Refuse to start | N/A |
| username="" && password="y" | ❌ Refuse to start | N/A |

### Startup Error Messages

**English:**
```
Invalid auth configuration: username and password must both be set or both be empty
```

**Chinese (zh-CN):**
```
认证配置无效：用户名和密码必须同时设置或同时为空
```

## GraphQL Schema Changes

**None** - HTTP Basic Auth operates at the HTTP transport layer and does not require GraphQL schema modifications.

## OpenAPI Specification (Reference)

```yaml
openapi: 3.0.3
info:
  title: rclone-sync API
  version: 1.0.0
components:
  securitySchemes:
    basicAuth:
      type: http
      scheme: basic
      description: HTTP Basic Authentication (when enabled via configuration)
security:
  - basicAuth: []
paths:
  /health:
    get:
      security: []  # No authentication required
      summary: Health check endpoint
      responses:
        '200':
          description: Service is healthy
  /api/graphql:
    post:
      summary: GraphQL endpoint
      security:
        - basicAuth: []
      responses:
        '200':
          description: GraphQL response
        '401':
          description: Authentication required
```

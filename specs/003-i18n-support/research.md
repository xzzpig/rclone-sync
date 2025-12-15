# Research: Multi-Language Support (i18n)

**Feature**: 003-i18n-support
**Date**: 2025-12-14
**Status**: Complete

## Executive Summary

This document researches i18n implementation solutions for the Cloud Sync application, covering key technical decisions such as internationalization library selection for the SolidJS frontend and Go backend, translation resource organization, and date formatting.

---

## Research Topics

### 1. SolidJS i18n Library Selection

**Background**: Need to select a lightweight, type-safe i18n solution for the SolidJS frontend.

**Candidate Solutions**:

| Library                 | Size  | Type Safety | Compile-time Optimization | Active Maintenance |
| ----------------------- | ----- | ----------- | ------------------------- | ------------------ |
| paraglide-js            | ~0KB  | ✅ Complete | ✅ Yes                    | ✅ Yes             |
| @solid-primitives/i18n  | ~3KB  | ✅ Complete | ❌ No                     | ✅ Yes             |
| i18next + solid-i18next | ~40KB | Partial     | ❌ No                     | ✅ Yes             |
| Custom Implementation   | ~1KB  | ✅ Complete | ❌ No                     | N/A                |

**Decision**: Use paraglide-js

**Reasons**:

1.  **Zero Runtime Overhead**: Translations are converted into JavaScript functions at compile time, adding no bundle size.
2.  **Full Type Safety**: Type definitions are generated at compile time, enabling IDE auto-completion and type checking.
3.  **Tree-shaking Friendly**: Unused translations are automatically removed.
4.  **Framework Agnostic**: Can be used with any framework, including SolidJS.
5.  **Inlang Ecosystem**: Well integrated with VS Code extensions, translation management tools, etc.
6.  **Simple Message Format**: Uses `.json` files to store translations, easy to maintain.

**Reasons for rejecting alternatives**:

- @solid-primitives/i18n: Runtime library, has extra overhead compared to paraglide-js.
- i18next: Overly large bundle (40KB+), over-engineered for simple scenarios.
- Custom implementation: Requires self-maintenance, not as good as using a mature compile-time solution.

---

### 2. Go Backend i18n Solution

**Background**: Need to implement localization for API response messages in the Go backend.

**Candidate Solutions**:

| Solution        | Complexity | Type Safety     | Runtime Overhead | Plural Support |
| --------------- | ---------- | --------------- | ---------------- | -------------- |
| go-i18n library | Medium     | Runtime         | Low              | ✅ Yes         |
| Embedded Go map | Low        | ✅ Compile-time | Very Low         | ❌ No          |
| JSON/YAML Files | Medium     | Runtime         | Medium           | ❌ No          |

**Decision**: Use go-i18n library (github.com/nicksnyder/go-i18n/v2)

**Reasons**:

1.  **Mature and Stable**: Widely used standard Go i18n library in the community, actively maintained.
2.  **Plural Handling**: Built-in CLDR plural rules support (e.g., "1 file" vs "2 files").
3.  **Message Templates**: Supports message templates with variables `{{.Count}} files`.
4.  **Multi-format Support**: Supports TOML, JSON, YAML message files (TOML recommended).
5.  **Embedding Support**: Translation files can be embedded at compile time via `embed.FS`.
6.  **Toolchain**: Provides `goi18n` CLI tool for translation management.
7.  **Hot Reloading**: Supports loading from local files in development mode, enabling hot reloading.

**Implementation Pattern**:

```go
package i18n

import (
    "embed"
    "os"
    "path/filepath"

    "github.com/BurntSushi/toml"
    "github.com/nicksnyder/go-i18n/v2/i18n"
    "golang.org/x/text/language"
)

//go:embed locales/*.toml
var localeFS embed.FS

var bundle *i18n.Bundle

// devMode controls whether to load from local files (supports hot reloading)
var devMode = os.Getenv("GO_ENV") == "development"

func init() {
    bundle = i18n.NewBundle(language.English)
    bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

    if devMode {
        // Development mode: load from local files, supports hot reloading
        bundle.LoadMessageFile("internal/i18n/locales/en.toml")
        bundle.LoadMessageFile("internal/i18n/locales/zh-CN.toml")
    } else {
        // Production mode: load from embedded files
        bundle.LoadMessageFileFS(localeFS, "locales/en.toml")
        bundle.LoadMessageFileFS(localeFS, "locales/zh-CN.toml")
    }
}

func NewLocalizer(lang string) *i18n.Localizer {
    return i18n.NewLocalizer(bundle, lang)
}

func T(localizer *i18n.Localizer, msgID string, data map[string]interface{}) string {
    msg, _ := localizer.Localize(&i18n.LocalizeConfig{
        MessageID:    msgID,
        TemplateData: data,
    })
    return msg
}
```

**Message File Format (TOML)**:

```toml
# internal/i18n/locales/en.toml

[error_task_not_found]
one = "Task not found"
other = "Task not found"

[error_connection_failed]
other = "Connection failed: {{.Reason}}"

[status_syncing_files]
one = "Syncing {{.Count}} file"
other = "Syncing {{.Count}} files"
```

```toml
# internal/i18n/locales/zh-CN.toml

[error_task_not_found]
other = "任务未找到"

[error_connection_failed]
other = "连接失败: {{.Reason}}"

[status_syncing_files]
other = "正在同步 {{.Count}} 个文件"
```

---

### 3. Language Detection and Persistence Strategy

**Background**: Need to automatically detect user language preferences and persist them across sessions.

**Decision**: Layered detection strategy

**Priority**:

1.  User selection saved in localStorage (highest priority)
2.  `navigator.language` browser language setting
3.  Default to English (fallback)

**Implementation**:

```typescript
function detectLanguage(): Locale {
  // 1. Check localStorage
  const saved = localStorage.getItem("locale");
  if (saved === "zh-CN" || saved === "en") {
    return saved;
  }

  // 2. Check browser language
  const browserLang = navigator.language;
  if (browserLang.startsWith("zh")) {
    return "zh-CN";
  }

  // 3. Default to English
  return "en";
}
```

---

### 4. Date/Time Localization

**Background**: Need to format dates and relative times according to language settings.

**Decision**: Use date-fns with locales

**Reasons**:

1.  Project already depends on date-fns.
2.  date-fns provides `zh-CN` and `en-US` locales.
3.  Supports relative time formatting (e.g., "5 minutes ago").

**Implementation**:

```typescript
import { format, formatDistanceToNow } from "date-fns";
import { zhCN, enUS } from "date-fns/locale";

const locales = { "zh-CN": zhCN, en: enUS };

function formatDate(date: Date, locale: Locale): string {
  return format(date, "PPP", { locale: locales[locale] });
}

function formatRelative(date: Date, locale: Locale): string {
  return formatDistanceToNow(date, {
    addSuffix: true,
    locale: locales[locale],
  });
}
```

---

### 5. Translation Resource Organization

**Background**: Need to design an easy-to-maintain translation file structure.

**Decision**: Use paraglide-js's JSON message format + Inlang project configuration

**Frontend Translation Structure (paraglide-js)**:

```
web/
├── project.inlang/           # Inlang project configuration
│   ├── settings.json         # Language settings
│   └── messages/
│       ├── en.json           # English translations
│       └── zh-CN.json        # Chinese (Simplified) translations
├── src/
│   └── paraglide/            # Generated at compile time
│       ├── messages.js       # Translation functions
│       └── runtime.js        # Runtime utilities
```

**Message File Format**:

```json
// web/project.inlang/messages/en.json
{
  "common_save": "Save",
  "common_cancel": "Cancel",
  "common_delete": "Delete",
  "common_loading": "Loading...",
  "nav_overview": "Overview",
  "nav_connections": "Connections",
  "nav_tasks": "Tasks",
  "nav_history": "History",
  "task_status_running": "Running",
  "task_status_completed": "Completed",
  "task_status_failed": "Failed",
  "error_generic": "An error occurred",
  "error_notFound": "Not found"
}
```

```json
// web/project.inlang/messages/zh-CN.json
{
  "common_save": "保存",
  "common_cancel": "取消",
  "common_delete": "删除",
  "common_loading": "加载中...",
  "nav_overview": "概览",
  "nav_connections": "连接",
  "nav_tasks": "任务",
  "nav_history": "历史",
  "task_status_running": "运行中",
  "task_status_completed": "已完成",
  "task_status_failed": "失败",
  "error_generic": "发生错误",
  "error_notFound": "未找到"
}
```

**Usage**:

```typescript
// Used in a component
import * as m from "../paraglide/messages";

function MyComponent() {
  return (
    <div>
      <button>{m.common_save()}</button>
      <span>{m.task_status_running()}</span>
    </div>
  );
}
```

**Backend Translation Structure (go-i18n)**:

```
internal/
└── i18n/
    ├── i18n.go           # Bundle initialization and utility functions
    ├── keys.go           # Message ID constants
    └── locales/
        ├── en.toml       # English translations
        └── zh-CN.toml    # Chinese (Simplified) translations
```

```go
// internal/i18n/keys.go
const (
    // Error messages
    ErrTaskNotFound      = "error.task_not_found"
    ErrConnectionFailed  = "error.connection_failed"
    ErrInvalidInput      = "error.invalid_input"

    // Status messages
    StatusSyncing        = "status.syncing"
    StatusSyncingFiles   = "status.syncing_files"
    StatusCompleted      = "status.completed"
    StatusFailed         = "status.failed"
)
```

---

### 6. API Accept-Language Header Handling

**Background**: Need to parse the language preference from the request in the backend.

**Decision**: Gin middleware to parse Accept-Language

**Implementation**:

```go
func LocaleMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        lang := c.GetHeader("Accept-Language")
        locale := i18n.EN // default

        if strings.HasPrefix(lang, "zh") {
            locale = i18n.ZH
        }

        c.Set("locale", locale)
        c.Next()
    }
}

// Usage in handler
func (h *Handler) GetTask(c *gin.Context) {
    locale := c.GetString("locale")
    // ...
    if err != nil {
        c.JSON(404, gin.H{
            "error": i18n.T(locale, i18n.ErrTaskNotFound),
        })
    }
}
```

---

### 7. Language Switcher UI Design

**Background**: Need to design a user-friendly and WCAG-compliant language switcher.

**Decision**: Dropdown selector, located in the user area at the bottom of the sidebar.

**Design Considerations**:

1.  Consistent style with the theme switcher (ModeToggle).
2.  Displays current language icon and name.
3.  Supports keyboard navigation (Tab, Enter, Arrow keys).
4.  Complete ARIA labels.

**Component Sketch**:

```
┌──────────────────┐
│ 🌐 English    ▼  │
├──────────────────┤
│   English        │
│ ✓ 简体中文       │
└──────────────────┘
```

---

## Summary of Decisions

| Decision Point       | Selection                                    | Reason                                             |
| -------------------- | -------------------------------------------- | -------------------------------------------------- |
| Frontend i18n        | paraglide-js                                 | Zero runtime, compile-time type safety             |
| Backend i18n         | go-i18n                                      | Mature, stable, plural support, template variables |
| Language Detection   | localStorage > navigator > Default           | User priority, automatic detection                 |
| Date Formatting      | date-fns + locale                            | Existing dependency, full functionality            |
| Translation Org.     | JSON files (frontend) / TOML files (backend) | Easy to maintain, go-i18n recommended format       |
| API Language         | Accept-Language middleware                   | HTTP standard, easy to implement                   |
| Language Switcher UI | Dropdown selector                            | Aligns with design system, accessibility           |
| Development Mode     | Local file loading                           | Supports hot reloading                             |

---

## Unresolved Items

No unresolved items. All technical decisions have been made, ready to proceed to Phase 1 design.

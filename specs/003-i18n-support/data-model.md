# Data Model: Multi-Language Support (i18n)

**Feature**: 003-i18n-support
**Date**: 2025-12-14
**Status**: Complete

## Overview

This document defines the data structure for i18n features, including frontend translation resources, backend message mapping, and language configuration.

---

## 1. Frontend Translation Resources (paraglide-js)

### 1.1 Inlang Project Configuration

```json
// web/project.inlang/settings.json
{
  "$schema": "https://inlang.com/schema/project-settings",
  "sourceLanguageTag": "en",
  "languageTags": ["en", "zh-CN"],
  "modules": [
    "https://cdn.jsdelivr.net/npm/@inlang/message-lint-rule-empty-pattern@latest/dist/index.js",
    "https://cdn.jsdelivr.net/npm/@inlang/message-lint-rule-missing-translation@latest/dist/index.js",
    "https://cdn.jsdelivr.net/npm/@inlang/plugin-message-format@latest/dist/index.js",
    "https://cdn.jsdelivr.net/npm/@inlang/plugin-m-function-matcher@latest/dist/index.js"
  ],
  "plugin.inlang.messageFormat": {
    "pathPattern": "./messages/{languageTag}.json"
  }
}
```

### 1.2 English Messages

```json
// web/project.inlang/messages/en.json
{
  "common_save": "Save",
  "common_cancel": "Cancel",
  "common_delete": "Delete",
  "common_confirm": "Confirm",
  "common_loading": "Loading...",
  "common_error": "Error",
  "common_success": "Success",
  "common_retry": "Retry",
  "common_close": "Close",
  "common_search": "Search",
  "common_noData": "No data",
  "common_actions": "Actions",

  "nav_overview": "Overview",
  "nav_connections": "Connections",
  "nav_tasks": "Tasks",
  "nav_history": "History",
  "nav_logs": "Logs",
  "nav_settings": "Settings",

  "connection_title": "Connections",
  "connection_addNew": "Add Connection",
  "connection_name": "Name",
  "connection_type": "Type",
  "connection_status": "Status",
  "connection_edit": "Edit",
  "connection_delete": "Delete",
  "connection_deleteConfirm": "Are you sure you want to delete this connection?",
  "connection_provider": "Provider",
  "connection_configure": "Configure",
  "connection_testConnection": "Test Connection",
  "connection_testSuccess": "Connection successful",
  "connection_testFailed": "Connection failed",

  "task_title": "Tasks",
  "task_create": "Create Task",
  "task_edit": "Edit Task",
  "task_delete": "Delete Task",
  "task_deleteConfirm": "Are you sure you want to delete this task?",
  "task_name": "Task Name",
  "task_source": "Source",
  "task_destination": "Destination",
  "task_schedule": "Schedule",
  "task_syncMode": "Sync Mode",
  "task_status_idle": "Idle",
  "task_status_running": "Running",
  "task_status_completed": "Completed",
  "task_status_failed": "Failed",
  "task_status_cancelled": "Cancelled",
  "task_syncNow": "Sync Now",
  "task_lastSync": "Last Sync",
  "task_nextSync": "Next Sync",
  "task_noSchedule": "No schedule",

  "history_title": "Sync History",
  "history_date": "Date",
  "history_duration": "Duration",
  "history_filesTransferred": "Files Transferred",
  "history_bytesTransferred": "Bytes Transferred",
  "history_errors": "Errors",
  "history_noHistory": "No sync history",

  "error_generic": "An error occurred",
  "error_notFound": "Not found",
  "error_networkError": "Network error",
  "error_unauthorized": "Unauthorized",
  "error_validationFailed": "Validation failed",
  "error_connectionFailed": "Connection failed",
  "error_syncFailed": "Sync failed",
  "error_taskNotFound": "Task not found",
  "error_connectionNotFound": "Connection not found",

  "settings_title": "Settings",
  "settings_language": "Language",
  "settings_theme": "Theme",
  "settings_themeLight": "Light",
  "settings_themeDark": "Dark",
  "settings_themeSystem": "System",

  "time_justNow": "Just now",
  "time_minutesAgo": "{count} minutes ago",
  "time_hoursAgo": "{count} hours ago",
  "time_daysAgo": "{count} days ago",
  "time_never": "Never"
}
```

### 1.3 Chinese Messages

```json
// web/project.inlang/messages/zh-CN.json
{
  "common_save": "保存",
  "common_cancel": "取消",
  "common_delete": "删除",
  "common_confirm": "确认",
  "common_loading": "加载中...",
  "common_error": "错误",
  "common_success": "成功",
  "common_retry": "重试",
  "common_close": "关闭",
  "common_search": "搜索",
  "common_noData": "暂无数据",
  "common_actions": "操作",

  "nav_overview": "概览",
  "nav_connections": "连接",
  "nav_tasks": "任务",
  "nav_history": "历史",
  "nav_logs": "日志",
  "nav_settings": "设置",

  "connection_title": "连接管理",
  "connection_addNew": "添加连接",
  "connection_name": "名称",
  "connection_type": "类型",
  "connection_status": "状态",
  "connection_edit": "编辑",
  "connection_delete": "删除",
  "connection_deleteConfirm": "确定要删除此连接吗？",
  "connection_provider": "服务商",
  "connection_configure": "配置",
  "connection_testConnection": "测试连接",
  "connection_testSuccess": "连接成功",
  "connection_testFailed": "连接失败",

  "task_title": "任务管理",
  "task_create": "创建任务",
  "task_edit": "编辑任务",
  "task_delete": "删除任务",
  "task_deleteConfirm": "确定要删除此任务吗？",
  "task_name": "任务名称",
  "task_source": "源路径",
  "task_destination": "目标路径",
  "task_schedule": "调度",
  "task_syncMode": "同步模式",
  "task_status_idle": "空闲",
  "task_status_running": "运行中",
  "task_status_completed": "已完成",
  "task_status_failed": "失败",
  "task_status_cancelled": "已取消",
  "task_syncNow": "立即同步",
  "task_lastSync": "上次同步",
  "task_nextSync": "下次同步",
  "task_noSchedule": "未设置调度",

  "history_title": "同步历史",
  "history_date": "日期",
  "history_duration": "耗时",
  "history_filesTransferred": "传输文件数",
  "history_bytesTransferred": "传输字节数",
  "history_errors": "错误",
  "history_noHistory": "暂无同步历史",

  "error_generic": "发生错误",
  "error_notFound": "未找到",
  "error_networkError": "网络错误",
  "error_unauthorized": "未授权",
  "error_validationFailed": "验证失败",
  "error_connectionFailed": "连接失败",
  "error_syncFailed": "同步失败",
  "error_taskNotFound": "任务未找到",
  "error_connectionNotFound": "连接未找到",

  "settings_title": "设置",
  "settings_language": "语言",
  "settings_theme": "主题",
  "settings_themeLight": "浅色",
  "settings_themeDark": "深色",
  "settings_themeSystem": "跟随系统",

  "time_justNow": "刚刚",
  "time_minutesAgo": "{count} 分钟前",
  "time_hoursAgo": "{count} 小时前",
  "time_daysAgo": "{count} 天前",
  "time_never": "从未"
}
```

### 1.4 Generated Types (Auto-generated by paraglide-js)

```typescript
// web/src/paraglide/messages.d.ts (AUTO-GENERATED)
// This file is generated by paraglide-js - do not edit manually

export function common_save(): string;
export function common_cancel(): string;
export function common_delete(): string;
export function common_loading(): string;
export function nav_overview(): string;
export function nav_connections(): string;
export function nav_tasks(): string;
export function task_status_running(): string;
export function task_status_completed(): string;
export function time_minutesAgo(params: { count: number }): string;
// ... more generated functions
```

---

## 2. Backend Message Mapping (go-i18n)

### 2.1 Directory Structure

```
internal/
└── i18n/
    ├── i18n.go           # Bundle initialization and utility functions
    ├── keys.go           # Message ID constants
    └── locales/
        ├── en.toml       # English translations (TOML format)
        └── zh-CN.toml    # Chinese translations (TOML format)
```

### 2.2 Bundle Initialization

```go
// internal/i18n/i18n.go

package i18n

import (
    "embed"
    "os"
    "strings"

    "github.com/BurntSushi/toml"
    "github.com/nicksnyder/go-i18n/v2/i18n"
    "golang.org/x/text/language"
)

//go:embed locales/*.toml
var localeFS embed.FS

var bundle *i18n.Bundle

// Init initializes the i18n bundle
// Should be called at application startup, depends on config.Cfg being loaded
func Init() {
    bundle = i18n.NewBundle(language.English)
    bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

    // Determine if in development mode from config
    isDev := config.Cfg.App.Environment == "development"

    if isDev {
        // Development mode: load from local files, supports hot reloading
        bundle.LoadMessageFile("internal/i18n/locales/en.toml")
        bundle.LoadMessageFile("internal/i18n/locales/zh-CN.toml")
    } else {
        // Production mode: load from embedded files
        bundle.LoadMessageFileFS(localeFS, "locales/en.toml")
        bundle.LoadMessageFileFS(localeFS, "locales/zh-CN.toml")
    }
}

// ReloadMessages reloads translation files in development mode (hot reload)
func ReloadMessages() error {
    if config.Cfg.App.Environment != "development" {
        return nil
    }
    bundle = i18n.NewBundle(language.English)
    bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
    if _, err := bundle.LoadMessageFile("internal/i18n/locales/en.toml"); err != nil {
        return err
    }
    if _, err := bundle.LoadMessageFile("internal/i18n/locales/zh-CN.toml"); err != nil {
        return err
    }
    return nil
}

// NewLocalizer creates a new localizer for the given language
func NewLocalizer(lang string) *i18n.Localizer {
    return i18n.NewLocalizer(bundle, lang)
}

// ParseLocale normalizes a language string to a supported locale
func ParseLocale(s string) string {
    if strings.HasPrefix(s, "zh") {
        return "zh-CN"
    }
    return "en"
}

// T translates a message with the given localizer
func T(localizer *i18n.Localizer, msgID string) string {
    msg, err := localizer.Localize(&i18n.LocalizeConfig{
        MessageID: msgID,
    })
    if err != nil {
        return msgID // fallback to key
    }
    return msg
}

// TWithData translates a message with template data
func TWithData(localizer *i18n.Localizer, msgID string, data map[string]interface{}) string {
    msg, err := localizer.Localize(&i18n.LocalizeConfig{
        MessageID:    msgID,
        TemplateData: data,
    })
    if err != nil {
        return msgID
    }
    return msg
}

// TPlural translates a message with plural support
func TPlural(localizer *i18n.Localizer, msgID string, count int, data map[string]interface{}) string {
    if data == nil {
        data = make(map[string]interface{})
    }
    data["Count"] = count
    msg, err := localizer.Localize(&i18n.LocalizeConfig{
        MessageID:    msgID,
        TemplateData: data,
        PluralCount:  count,
    })
    if err != nil {
        return msgID
    }
    return msg
}

// ========== context.Context related functions ==========

// contextKey is the type for keys storing values in context.Context
type contextKey string

const (
    // ContextKeyLocalizer is the key for Localizer in context.Context
    ContextKeyLocalizer contextKey = "i18n.localizer"
    // ContextKeyLocale is the key for the locale string in context.Context
    ContextKeyLocale contextKey = "i18n.locale"
)

// WithLocalizer stores Localizer in context.Context
func WithLocalizer(ctx context.Context, localizer *i18n.Localizer) context.Context {
    return context.WithValue(ctx, ContextKeyLocalizer, localizer)
}

// WithLocale stores the locale string in context.Context
func WithLocale(ctx context.Context, locale string) context.Context {
    return context.WithValue(ctx, ContextKeyLocale, locale)
}

// LocalizerFromContext retrieves Localizer from context.Context
// Returns an English Localizer if not found
func LocalizerFromContext(ctx context.Context) *i18n.Localizer {
    if localizer, ok := ctx.Value(ContextKeyLocalizer).(*i18n.Localizer); ok {
        return localizer
    }
    return NewLocalizer("en")
}

// LocaleFromContext retrieves the locale string from context.Context
// Returns "en" if not found
func LocaleFromContext(ctx context.Context) string {
    if locale, ok := ctx.Value(ContextKeyLocale).(string); ok {
        return locale
    }
    return "en"
}

// Ctx is a convenient translation function for business logic
// Directly retrieves Localizer from context.Context and translates messages
func Ctx(ctx context.Context, msgID string) string {
    return T(LocalizerFromContext(ctx), msgID)
}

// CtxWithData is a convenient translation function with data for business logic
func CtxWithData(ctx context.Context, msgID string, data map[string]interface{}) string {
    return TWithData(LocalizerFromContext(ctx), msgID, data)
}

// CtxPlural is a convenient plural translation function for business logic
func CtxPlural(ctx context.Context, msgID string, count int, data map[string]interface{}) string {
    return TPlural(LocalizerFromContext(ctx), msgID, count, data)
}

// ========== I18nError error type ==========

// I18nError is a translatable error type
// Used to return translatable errors at the business logic layer
type I18nError struct {
    // MsgID is the key for the translation message
    MsgID string
    // Data is the data for the translation template (optional)
    Data map[string]interface{}
    // StatusCode is the HTTP status code (default 400)
    StatusCode int
    // Cause is the original error (optional)
    Cause error
}

// Error implements the error interface
// Returns the message ID (for logging and other scenarios)
func (e *I18nError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %v", e.MsgID, e.Cause)
    }
    return e.MsgID
}

// Unwrap returns the original error
func (e *I18nError) Unwrap() error {
    return e.Cause
}

// Translate translates the error message using the given localizer
func (e *I18nError) Translate(localizer *i18n.Localizer) string {
    if e.Data != nil {
        return TWithData(localizer, e.MsgID, e.Data)
    }
    return T(localizer, e.MsgID)
}

// TranslateCtx translates the error message using the localizer from context
func (e *I18nError) TranslateCtx(ctx context.Context) string {
    return e.Translate(LocalizerFromContext(ctx))
}

// NewI18nError creates a new I18nError
func NewI18nError(msgID string) *I18nError {
    return &I18nError{
        MsgID:      msgID,
        StatusCode: 400,
    }
}

// NewI18nErrorWithData creates an I18nError with data
func NewI18nErrorWithData(msgID string, data map[string]interface{}) *I18nError {
    return &I18nError{
        MsgID:      msgID,
        Data:       data,
        StatusCode: 400,
    }
}

// WithStatus sets the HTTP status code
func (e *I18nError) WithStatus(code int) *I18nError {
    e.StatusCode = code
    return e
}

// WithCause sets the original error
func (e *I18nError) WithCause(err error) *I18nError {
    e.Cause = err
    return e
}

// WithData sets translation data
func (e *I18nError) WithData(data map[string]interface{}) *I18nError {
    e.Data = data
    return e
}

// Common error constructors

// ErrNotFoundI18n returns a 404 error
func ErrNotFoundI18n(msgID string) *I18nError {
    return NewI18nError(msgID).WithStatus(404)
}

// ErrBadRequestI18n returns a 400 error
func ErrBadRequestI18n(msgID string) *I18nError {
    return NewI18nError(msgID).WithStatus(400)
}

// ErrInternalI18n returns a 500 error
func ErrInternalI18n(msgID string) *I18nError {
    return NewI18nError(msgID).WithStatus(500)
}

// ErrUnauthorizedI18n returns a 401 error
func ErrUnauthorizedI18n(msgID string) *I18nError {
    return NewI18nError(msgID).WithStatus(401)
}

// IsI18nError checks if an error is an I18nError
func IsI18nError(err error) (*I18nError, bool) {
    var i18nErr *I18nError
    if errors.As(err, &i18nErr) {
        return i18nErr, true
    }
    return nil, false
}
```

### 2.3 Message Keys

```go
// internal/i18n/keys.go

package i18n

// Error message keys
const (
    ErrGeneric            = "error.generic"
    ErrNotFound           = "error.not_found"
    ErrUnauthorized       = "error.unauthorized"
    ErrValidationFailed   = "error.validation_failed"
    ErrConnectionFailed   = "error.connection_failed"
    ErrSyncFailed         = "error.sync_failed"
    ErrTaskNotFound       = "error.task_not_found"
    ErrConnectionNotFound = "error.connection_not_found"
    ErrInvalidInput       = "error.invalid_input"
    ErrDatabaseError      = "error.database_error"
)

// Status message keys
const (
    StatusSyncing       = "status.syncing"
    StatusSyncingFiles  = "status.syncing_files"
    StatusCompleted     = "status.completed"
    StatusFailed        = "status.failed"
    StatusIdle          = "status.idle"
    StatusCancelled     = "status.cancelled"
)

// Success message keys
const (
    SuccessCreated   = "success.created"
    SuccessUpdated   = "success.updated"
    SuccessDeleted   = "success.deleted"
    SuccessSyncStart = "success.sync_started"
)
```

### 2.4 English Messages (TOML format)

```toml
# internal/i18n/locales/en.toml

# Error messages
[error_generic]
other = "An error occurred"

[error_not_found]
other = "Resource not found"

[error_unauthorized]
other = "Unauthorized access"

[error_validation_failed]
other = "Validation failed"

[error_connection_failed]
other = "Connection failed: {{.Reason}}"

[error_sync_failed]
other = "Sync operation failed"

[error_task_not_found]
other = "Task not found"

[error_connection_not_found]
other = "Connection not found"

[error_invalid_input]
other = "Invalid input provided"

[error_database_error]
other = "Database error"

# Status messages
[status_syncing]
other = "Syncing"

[status_syncing_files]
one = "Syncing {{.Count}} file"
other = "Syncing {{.Count}} files"

[status_completed]
other = "Completed"

[status_failed]
other = "Failed"

[status_idle]
other = "Idle"

[status_cancelled]
other = "Cancelled"

# Success messages
[success_created]
other = "Created successfully"

[success_updated]
other = "Updated successfully"

[success_deleted]
other = "Deleted successfully"

[success_sync_started]
other = "Sync started"
```

### 2.5 Chinese Messages (TOML format)

```toml
# internal/i18n/locales/zh-CN.toml

# Error messages
[error_generic]
other = "发生错误"

[error_not_found]
other = "资源未找到"

[error_unauthorized]
other = "未授权访问"

[error_validation_failed]
other = "验证失败"

[error_connection_failed]
other = "连接失败: {{.Reason}}"

[error_sync_failed]
other = "同步操作失败"

[error_task_not_found]
other = "任务未找到"

[error_connection_not_found]
other = "连接未找到"

[error_invalid_input]
other = "输入无效"

[error_database_error]
other = "数据库错误"

# Status messages
[status_syncing]
other = "同步中"

[status_syncing_files]
other = "正在同步 {{.Count}} 个文件"

[status_completed]
other = "已完成"

[status_failed]
other = "失败"

[status_idle]
other = "空闲"

[status_cancelled]
other = "已取消"

# Success messages
[success_created]
other = "创建成功"

[success_updated]
other = "更新成功"

[success_deleted]
other = "删除成功"

[success_sync_started]
other = "同步已开始"
```

---

## 3. i18n Runtime & Hooks

### 3.1 Locale Store (SolidJS Context/Provider)

```typescript
// web/src/store/locale.tsx

import { ParentComponent, createContext, useContext } from "solid-js";
import { createStore } from "solid-js/store";
import {
  setLanguageTag,
  availableLanguageTags,
  sourceLanguageTag,
} from "../paraglide/runtime";

// Types
export type Locale = (typeof availableLanguageTags)[number];

export const SUPPORTED_LOCALES = availableLanguageTags;
export const DEFAULT_LOCALE = sourceLanguageTag;

export const LOCALE_NAMES: Record<Locale, string> = {
  en: "English",
  "zh-CN": "简体中文",
};

// State interface
interface LocaleState {
  locale: Locale;
  isLoading: boolean;
}

// Actions interface
interface LocaleActions {
  setLocale: (locale: Locale) => void;
  getLocale: () => Locale;
  getLocaleName: () => string;
  detectAndSetLocale: () => void;
}

// Initial state
const initialState: LocaleState = {
  locale: detectLanguage(),
  isLoading: false,
};

// Language detection function
function detectLanguage(): Locale {
  // 1. Check localStorage
  const saved = localStorage.getItem("locale") as Locale | null;
  if (saved && SUPPORTED_LOCALES.includes(saved)) {
    return saved;
  }

  // 2. Check browser language
  const browserLang = navigator.language;
  if (browserLang.startsWith("zh")) {
    return "zh-CN";
  }

  // 3. Default to source language
  return DEFAULT_LOCALE;
}

// Context
const LocaleContext = createContext<[LocaleState, LocaleActions]>();

// Provider component
export const LocaleProvider: ParentComponent = (props) => {
  const [state, setState] = createStore<LocaleState>(initialState);

  // Initialize paraglide with detected language
  setLanguageTag(state.locale);

  const actions: LocaleActions = {
    setLocale: (newLocale: Locale) => {
      if (SUPPORTED_LOCALES.includes(newLocale)) {
        setState("locale", newLocale);
        setLanguageTag(newLocale);
        localStorage.setItem("locale", newLocale);
      }
    },

    getLocale: () => state.locale,

    getLocaleName: () => LOCALE_NAMES[state.locale] || state.locale,

    detectAndSetLocale: () => {
      const detected = detectLanguage();
      actions.setLocale(detected);
    },
  };

  return (
    <LocaleContext.Provider value={[state, actions]}>
      {props.children}
    </LocaleContext.Provider>
  );
};

// Hook to use locale store
export const useLocale = () => {
  const context = useContext(LocaleContext);
  if (!context) {
    throw new Error("useLocale must be used within a LocaleProvider");
  }
  return context;
};

// Re-export for convenience
export { availableLanguageTags, sourceLanguageTag };
```

### 3.2 Provider Setup in App

```typescript
// web/src/App.tsx (excerpt)

import { LocaleProvider } from "./store/locale";
import { TaskProvider } from "./store/tasks";

function App() {
  return (
    <LocaleProvider>
      <TaskProvider>{/* ... rest of app */}</TaskProvider>
    </LocaleProvider>
  );
}
```

### 3.3 Using Translations in Components

```typescript
// Example component usage
import * as m from "../paraglide/messages";
import { getLocale, setLocale, LOCALE_NAMES, SUPPORTED_LOCALES } from "../i18n";

function TaskCard() {
  return (
    <div>
      <h2>{m.task_title()}</h2>
      <span class="status">{m.task_status_running()}</span>
      <button>{m.common_save()}</button>
    </div>
  );
}

function LanguageSwitcher() {
  return (
    <select
      value={getLocale()}
      onChange={(e) => setLocale(e.target.value as Locale)}
    >
      {SUPPORTED_LOCALES.map((loc) => (
        <option value={loc}>{LOCALE_NAMES[loc]}</option>
      ))}
    </select>
  );
}
```

### 3.2 Backend Translation Function Usage

```go
// Example usage in handlers

package handlers

import (
    "github.com/gin-gonic/gin"
    "your-project/internal/i18n"
)

func (h *Handler) GetTask(c *gin.Context) {
    // Get localizer from context (set by middleware)
    localizer := c.MustGet("localizer").(*i18n.Localizer)

    task, err := h.taskService.GetTask(id)
    if err != nil {
        c.JSON(404, gin.H{
            "error": i18n.T(localizer, i18n.ErrTaskNotFound),
        })
        return
    }

    // With template data
    c.JSON(200, gin.H{
        "message": i18n.TWithData(localizer, i18n.SuccessCreated, map[string]interface{}{
            "Name": task.Name,
        }),
    })
}

// With plural support
func (h *Handler) GetSyncStatus(c *gin.Context) {
    localizer := c.MustGet("localizer").(*i18n.Localizer)
    fileCount := 5

    c.JSON(200, gin.H{
        "status": i18n.TPlural(localizer, i18n.StatusSyncingFiles, fileCount, nil),
        // English: "Syncing 5 files"
        // Chinese: "正在同步 5 个文件"
    })
}
```

### 3.4 Locale Middleware (Gin Context + context.Context)

The middleware needs to store the localizer in both `gin.Context` and `context.Context` to allow:

- **Handler layer**: direct access via `gin.Context`
- **Business logic layer**: access via `context.Context` (business services do not have `gin.Context`)

```go
// internal/api/middleware/locale.go

package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/nicksnyder/go-i18n/v2/i18n"
    i18npkg "github.com/xzzpig/rclone-sync/internal/i18n"
)

// GinContextKeyLocalizer is the key for storing Localizer in Gin context
const GinContextKeyLocalizer = "localizer"

// LocaleMiddleware parses Accept-Language header and stores Localizer in both contexts
func LocaleMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        lang := c.GetHeader("Accept-Language")
        locale := i18npkg.ParseLocale(lang)
        localizer := i18npkg.NewLocalizer(locale)

        // 1. Store in Gin context (for handlers)
        c.Set("locale", locale)
        c.Set(GinContextKeyLocalizer, localizer)

        // 2. Store in context.Context (for business logic / services)
        // Pass to the business layer by modifying c.Request's Context
        ctx := c.Request.Context()
        ctx = i18npkg.WithLocalizer(ctx, localizer)
        ctx = i18npkg.WithLocale(ctx, locale)
        c.Request = c.Request.WithContext(ctx)

        c.Next()
    }
}

// GetLocalizer retrieves the Localizer from Gin context
func GetLocalizer(c *gin.Context) *i18n.Localizer {
    if localizer, exists := c.Get(GinContextKeyLocalizer); exists {
        return localizer.(*i18n.Localizer)
    }
    // Fallback: try to get from request context
    return i18npkg.LocalizerFromContext(c.Request.Context())
}
```

### 3.5 I18nError Handling Middleware

Automatically handles errors of type `I18nError`, translates error messages, and returns a JSON response:

```go
// internal/api/middleware/i18n_error.go

package middleware

import (
    "github.com/gin-gonic/gin"
    i18npkg "github.com/xzzpig/rclone-sync/internal/i18n"
)

// I18nErrorMiddleware automatically handles errors of type I18nError
// Uses Gin's error handling mechanism: c.Error(err)
func I18nErrorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        // Check for errors
        if len(c.Errors) == 0 {
            return
        }

        // Get localizer
        localizer := GetLocalizer(c)

        // Handle the last error
        lastErr := c.Errors.Last()
        if lastErr == nil {
            return
        }

        // Check if it's an I18nError
        if i18nErr, ok := i18npkg.IsI18nError(lastErr.Err); ok {
            // Translate error message
            translatedMsg := i18nErr.Translate(localizer)

            // Return JSON response
            c.JSON(i18nErr.StatusCode, gin.H{
                "error":   translatedMsg,
                "code":    i18nErr.MsgID,
                "success": false,
            })
            return
        }

        // Not an I18nError, return a generic error
        c.JSON(500, gin.H{
            "error":   i18npkg.T(localizer, i18npkg.ErrGeneric),
            "code":    "internal_error",
            "success": false,
        })
    }
}
```

**Route Registration**:

```go
// internal/api/routes.go

func SetupRoutes(r *gin.Engine) {
    // Register LocaleMiddleware first, then I18nErrorMiddleware
    r.Use(middleware.LocaleMiddleware())
    r.Use(middleware.I18nErrorMiddleware())

    // ... other routes
}
```

**Usage Example - Business Layer returns I18nError**:

```go
// internal/core/services/task_service.go

func (s *TaskService) GetTask(ctx context.Context, id string) (*Task, error) {
    task, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            // Return I18nError, will be automatically handled by middleware
            return nil, i18npkg.ErrNotFoundI18n(i18npkg.ErrTaskNotFound)
        }
        return nil, i18npkg.ErrInternalI18n(i18npkg.ErrDatabaseError).WithCause(err)
    }
    return task, nil
}

func (s *TaskService) CreateTask(ctx context.Context, task *Task) error {
    if task.Name == "" {
        // Error with parameters
        return i18npkg.NewI18nErrorWithData(i18npkg.ErrValidationFailed, map[string]interface{}{
            "Field": "name",
        }).WithStatus(400)
    }
    // ...
}
```

**Usage Example - Handler uses c.Error()**:

```go
// internal/api/handlers/task.go

func (h *TaskHandler) GetTask(c *gin.Context) {
    ctx := c.Request.Context()
    id := c.Param("id")

    task, err := h.service.GetTask(ctx, id)
    if err != nil {
        // Add the error to Gin's error chain
        // I18nErrorMiddleware will handle it automatically
        c.Error(err)
        return
    }

    c.JSON(200, task)
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
    ctx := c.Request.Context()
    var req CreateTaskRequest

    if err := c.ShouldBindJSON(&req); err != nil {
        c.Error(i18npkg.ErrBadRequestI18n(i18npkg.ErrValidationFailed).WithCause(err))
        return
    }

    task, err := h.service.CreateTask(ctx, &req)
    if err != nil {
        c.Error(err) // Automatically handles I18nError or regular errors
        return
    }

    c.JSON(201, task)
}
```

**Usage Example - Handler Layer**:

```go
// Get Localizer in handler
func (h *Handler) GetTask(c *gin.Context) {
    // Method 1: Use helper function
    localizer := middleware.GetLocalizer(c)

    // Method 2: Get directly from Gin context
    // localizer := c.MustGet(middleware.GinContextKeyLocalizer).(*i18n.Localizer)

    // Pass context when calling business service
    ctx := c.Request.Context()
    task, err := h.service.GetTask(ctx, id)
    if err != nil {
        c.JSON(404, gin.H{
            "error": i18npkg.T(localizer, i18npkg.ErrTaskNotFound),
        })
        return
    }
    // ...
}
```

**Usage Example - Business Logic Layer**:

```go
// internal/core/services/task_service.go

package services

import (
    "context"
    i18npkg "github.com/xzzpig/rclone-sync/internal/i18n"
)

func (s *TaskService) CreateTask(ctx context.Context, task *Task) error {
    // Method 1: Use convenient function Ctx
    if task.Name == "" {
        return fmt.Errorf(i18npkg.Ctx(ctx, i18npkg.ErrInvalidInput))
    }

    // Method 2: Get localizer and then use it
    localizer := i18npkg.LocalizerFromContext(ctx)
    msg := i18npkg.T(localizer, i18npkg.SuccessCreated)

    // Method 3: Translation with data
    msg := i18npkg.CtxWithData(ctx, i18npkg.StatusSyncingFiles, map[string]interface{}{
        "Count": 5,
    })

    // ...
}
```

---

## 4. Entity Relationships

```
┌─────────────────────────────────────────────────────────────┐
│                   Frontend (paraglide-js)                    │
├─────────────────────────────────────────────────────────────┤
│  paraglide/runtime.js                                        │
│  ├── languageTag: () => Locale                              │
│  ├── setLanguageTag: (locale: Locale) => void               │
│  ├── availableLanguageTags: Locale[]                        │
│  └── sourceLanguageTag: Locale                              │
│                                                              │
│  paraglide/messages.js (COMPILED)                           │
│  ├── common_save(): string                                  │
│  ├── nav_overview(): string                                 │
│  ├── task_status_running(): string                          │
│  ├── time_minutesAgo({count}): string                       │
│  └── ... (all message functions)                            │
│                                                              │
│  i18n/index.ts (helpers)                                    │
│  ├── getLocale(): Locale                                    │
│  ├── setLocale(locale): void                                │
│  ├── LOCALE_NAMES: Record<Locale, string>                   │
│  └── SUPPORTED_LOCALES: Locale[]                            │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ Accept-Language header
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Backend (go-i18n)                         │
├─────────────────────────────────────────────────────────────┤
│  LocaleMiddleware                                            │
│  └── parses Accept-Language → creates Localizer             │
│                                                              │
│  i18n.Bundle (go-i18n)                                      │
│  ├── LoadMessageFileFS() - loads embedded JSON              │
│  └── NewLocalizer(lang) → *i18n.Localizer                   │
│                                                              │
│  i18n.T(localizer, key)                                     │
│  i18n.TWithData(localizer, key, data)                       │
│  i18n.TPlural(localizer, key, count, data)                  │
│                                                              │
│  locales/*.toml (embedded via go:embed)                     │
│  ├── en.toml                                                │
│  └── zh-CN.toml                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. Storage

| Data Type                | Storage Location                                   | Description                            |
| ------------------------ | -------------------------------------------------- | -------------------------------------- |
| User language preference | localStorage ("locale")                            | Frontend persistence                   |
| Frontend translations    | JSON → JS functions (generated at compile time)    | Zero runtime overhead                  |
| Backend translations     | TOML files (embedded via go:embed at compile time) | go-i18n Bundle loading                 |
| Gin Context              | Localizer object                                   | Created per request, stored in context |

**Note**: No database schema changes are required. Language preferences are stored only on the client side.

**go-i18n Features**:

- Uses `go:embed` to embed TOML translation files at compile time
- Supports CLDR plural rules (one/other)
- Supports Go text/template syntax for message templates
- Bundle loaded once at startup, zero I/O at runtime
- **Development mode**: Set `GO_ENV=development` to load from local files, supporting hot reloading
- **Gin Context**: Localizer stored in `c.Get("localizer")`, convenient for Handler usage

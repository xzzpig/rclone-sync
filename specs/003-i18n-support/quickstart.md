# Quickstart Guide: Multi-Language Support (i18n)

**Feature**: 003-i18n-support
**Date**: 2025-12-14
**Audience**: Developers implementing i18n support

## Overview

This guide helps developers quickly get started with implementing internationalization (i18n) features for Cloud Sync applications, using paraglide-js as a frontend compile-time i18n library.

---

## Prerequisites

- Node.js 18+ (Frontend development)
- Go 1.25+ (Backend development)
- pnpm (Frontend package manager)
- Basic understanding of SolidJS and Gin frameworks

---

## Quick Start

### 1. Frontend i18n Setup - paraglide-js (5 minutes)

#### 1.1 Initialize paraglide-js

Use the official CLI to initialize the project (it will automatically install dependencies, create configuration files and directory structure):

```bash
cd web
pnpx @inlang/paraglide-js@latest init
```

The initialization wizard will ask:

- Select languages (Choose English as the source language, add Chinese (Simplified) as the target language)
- Select output directory (default `./src/paraglide`)
- Whether to configure Vite plugin (Choose yes)

**Or manual setup** (if custom configuration is needed):

```bash
pnpm add @inlang/paraglide-js
pnpm add -D @inlang/paraglide-vite
mkdir -p project.inlang/messages
```

Create configuration file:

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

#### 1.3 Create translation files

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
  "nav_history": "History"
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
  "nav_history": "历史"
}
```

#### 1.4 Configure Vite Plugin

```typescript
// web/vite.config.ts
import { defineConfig } from "vite";
import solid from "vite-plugin-solid";
import paraglide from "@inlang/paraglide-vite";

export default defineConfig({
  plugins: [
    solid(),
    paraglide({
      project: "./project.inlang",
      outdir: "./src/paraglide",
    }),
  ],
});
```

#### 1.5 Create Locale Store

Use SolidJS Context/Provider pattern to manage language state (consistent with TaskProvider, etc.):

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
  const saved = localStorage.getItem("locale") as Locale | null;
  if (saved && SUPPORTED_LOCALES.includes(saved)) {
    return saved;
  }
  const browserLang = navigator.language;
  if (browserLang.startsWith("zh")) {
    return "zh-CN";
  }
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

#### 1.6 Configure Provider in App

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

#### 1.7 Use in Components

```typescript
// Any component
import * as m from "../paraglide/messages";

function MyComponent() {
  return (
    <div>
      <button>{m.common_save()}</button>
      <button>{m.common_cancel()}</button>
    </div>
  );
}
```

---

### 2. Backend i18n Setup - go-i18n (5 minutes)

#### 2.1 Install Dependencies

```bash
go get github.com/nicksnyder/go-i18n/v2/i18n
go get github.com/BurntSushi/toml
go get golang.org/x/text/language
```

#### 2.2 Create i18n Directory Structure

```bash
mkdir -p internal/i18n/locales
```

#### 2.3 Create Translation Files (TOML format)

```toml
# internal/i18n/locales/en.toml

[error_generic]
other = "An error occurred"

[error_task_not_found]
other = "Task not found"

[error_connection_failed]
other = "Connection failed: {{.Reason}}"

[success_created]
other = "Created successfully"

[success_deleted]
other = "Deleted successfully"

[status_syncing_files]
one = "Syncing {{.Count}} file"
other = "Syncing {{.Count}} files"
```

```toml
# internal/i18n/locales/zh-CN.toml

[error_generic]
other = "发生错误"

[error_task_not_found]
other = "任务未找到"

[error_connection_failed]
other = "连接失败: {{.Reason}}"

[success_created]
other = "创建成功"

[success_deleted]
other = "删除成功"

[status_syncing_files]
other = "正在同步 {{.Count}} 个文件"
```

#### 2.4 Create i18n Package

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
        // Development mode: load from local files, supports hot reload
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
        return msgID
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
```

#### 2.5 Create Message Key Constants

```go
// internal/i18n/keys.go

package i18n

// Message key constants (note: TOML uses underscores, not dots)
const (
    ErrGeneric          = "error_generic"
    ErrTaskNotFound     = "error_task_not_found"
    ErrConnectionFailed = "error_connection_failed"
    SuccessCreated      = "success_created"
    SuccessDeleted      = "success_deleted"
    StatusSyncingFiles  = "status_syncing_files"
)
```

#### 2.6 Create Middleware (store in both Gin Context and context.Context)

The middleware needs to store the localizer in two contexts:

- `gin.Context`: Directly used by the Handler layer
- `context.Context`: Used by the business logic layer (passed via `c.Request.Context()`)

```go
// internal/api/middleware/locale.go

package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/nicksnyder/go-i18n/v2/i18n"
    i18npkg "github.com/xzzpig/rclone-sync/internal/i18n"
)

// GinContextKeyLocalizer is the key for Localizer in Gin context
const GinContextKeyLocalizer = "localizer"

// LocaleMiddleware parses the Accept-Language header and stores the Localizer in two contexts
func LocaleMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        lang := c.GetHeader("Accept-Language")
        locale := i18npkg.ParseLocale(lang)
        localizer := i18npkg.NewLocalizer(locale)

        // 1. Store in Gin context (for Handler layer)
        c.Set("locale", locale)
        c.Set(GinContextKeyLocalizer, localizer)

        // 2. Store in context.Context (for business logic layer)
        ctx := c.Request.Context()
        ctx = i18npkg.WithLocalizer(ctx, localizer)
        ctx = i18npkg.WithLocale(ctx, locale)
        c.Request = c.Request.WithContext(ctx)

        c.Next()
    }
}

// GetLocalizer retrieves Localizer from Gin context
func GetLocalizer(c *gin.Context) *i18n.Localizer {
    if localizer, exists := c.Get(GinContextKeyLocalizer); exists {
        return localizer.(*i18n.Localizer)
    }
    // Fallback: try to get from request context
    return i18npkg.LocalizerFromContext(c.Request.Context())
}
```

#### 2.7 Use in Handler (pass context to business layer)

```go
// internal/api/handlers/task.go

import (
    "github.com/gin-gonic/gin"
    "github.com/xzzpig/rclone-sync/internal/api/middleware"
    i18npkg "github.com/xzzpig/rclone-sync/internal/i18n"
)

func (h *TaskHandler) GetTask(c *gin.Context) {
    // Method 1: Use GetLocalizer helper function
    localizer := middleware.GetLocalizer(c)

    // Method 2: Get directly from context
    // localizer := c.MustGet(middleware.ContextKeyLocalizer).(*i18n.Localizer)

    task, err := h.service.GetTask(id)
    if err != nil {
        c.JSON(404, gin.H{
            "error": i18npkg.T(localizer, i18npkg.ErrTaskNotFound),
            "code":  "TASK_NOT_FOUND",
        })
        return
    }
    // ...
}

// Use message with parameters
func (h *TaskHandler) SyncStatus(c *gin.Context) {
    localizer := middleware.GetLocalizer(c)
    fileCount := 5

    c.JSON(200, gin.H{
        "status": i18npkg.TPlural(localizer, i18npkg.StatusSyncingFiles, fileCount, nil),
        // English: "Syncing 5 files"
        // Chinese: "正在同步 5 个文件"
    })
}
```

#### 2.8 Use in Business Logic Layer (via context.Context)

Business services obtain the localizer via `context.Context`, without relying on gin.Context:

```go
// internal/core/services/task_service.go

package services

import (
    "context"
    "fmt"

    i18npkg "github.com/xzzpig/rclone-sync/internal/i18n"
)

func (s *TaskService) CreateTask(ctx context.Context, task *Task) error {
    // Method 1: Use convenience function Ctx (recommended)
    if task.Name == "" {
        return fmt.Errorf(i18npkg.Ctx(ctx, i18npkg.ErrInvalidInput))
    }

    // Method 2: Get localizer then use
    localizer := i18npkg.LocalizerFromContext(ctx)
    msg := i18npkg.T(localizer, i18npkg.SuccessCreated)
    _ = msg // use msg

    // Method 3: Translation with data
    statusMsg := i18npkg.CtxWithData(ctx, i18npkg.StatusSyncingFiles, map[string]interface{}{
        "Count": 5,
    })
    _ = statusMsg // use statusMsg

    // Method 4: Plural translation
    fileCount := 5
    pluralMsg := i18npkg.CtxPlural(ctx, i18npkg.StatusSyncingFiles, fileCount, nil)
    _ = pluralMsg // use pluralMsg

    // Continue business logic...
    return nil
}
```

**Note**: When a Handler calls a business service, it needs to pass `c.Request.Context()`:

```go
// Handler call example
func (h *TaskHandler) CreateTask(c *gin.Context) {
    // Get context from request (already contains localizer)
    ctx := c.Request.Context()

    // Call business service
    err := h.service.CreateTask(ctx, task)
    // ...
}
```

#### 2.9 Use I18nError (recommended approach)

`I18nError` is a specially designed translatable error type, which, when combined with `I18nErrorMiddleware`, can automatically handle error translation:

```go
// Return I18nError in the business logic layer
func (s *TaskService) GetTask(ctx context.Context, id string) (*Task, error) {
    task, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            // Return 404 error, will be automatically translated by middleware
            return nil, i18npkg.ErrNotFoundI18n(i18npkg.ErrTaskNotFound)
        }
        // Return 500 error, with original error
        return nil, i18npkg.ErrInternalI18n(i18npkg.ErrDatabaseError).WithCause(err)
    }
    return task, nil
}

// Error with parameters
func (s *TaskService) CreateTask(ctx context.Context, task *Task) error {
    if task.Name == "" {
        return i18npkg.NewI18nErrorWithData(i18npkg.ErrValidationFailed, map[string]interface{}{
            "Field": "name",
        }).WithStatus(400)
    }
    // ...
}
```

**Use `c.Error()` in Handler**:

```go
func (h *TaskHandler) GetTask(c *gin.Context) {
    ctx := c.Request.Context()
    id := c.Param("id")

    task, err := h.service.GetTask(ctx, id)
    if err != nil {
        // Add error to Gin error chain, I18nErrorMiddleware will handle it automatically
        c.Error(err)
        return
    }

    c.JSON(200, task)
}
```

**Register middleware** (routes.go):

```go
func SetupRoutes(r *gin.Engine) {
    // Order matters: LocaleMiddleware first, then I18nErrorMiddleware
    r.Use(middleware.LocaleMiddleware())
    r.Use(middleware.I18nErrorMiddleware())
    // ... Other routes
}
```

**Common `I18nError` constructors**:

| Function                               | HTTP Status Code | Purpose                  |
| -------------------------------------- | ---------------- | ------------------------ |
| `ErrNotFoundI18n(msgID)`               | 404              | Resource not found       |
| `ErrBadRequestI18n(msgID)`             | 400              | Request parameter error  |
| `ErrUnauthorizedI18n(msgID)`           | 401              | Unauthorized             |
| `ErrInternalI18n(msgID)`               | 500              | Internal error           |
| `NewI18nError(msgID).WithStatus(code)` | Custom           | Custom status code       |
| `NewI18nErrorWithData(msgID, data)`    | 400              | Error with template data |

#### 2.10 Hot Reload in Development Mode

Start the service in development mode:

```bash
GO_ENV=development go run ./cmd/cloud-sync serve
```

After modifying TOML translation files, call `i18n.ReloadMessages()` to reload:

```go
// Can be called at a debug endpoint
func (h *Handler) ReloadI18n(c *gin.Context) {
    if err := i18npkg.ReloadMessages(); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"message": "i18n messages reloaded"})
}
```

---

### 3. API Request Language Header (2 minutes)

Update the frontend API client to send the Accept-Language header. Since the API module is outside the store, a callback function is needed to get the current language:

```typescript
// web/src/lib/api.ts

import axios from "axios";

// Language retrieval callback (set by LocaleProvider)
let getLocaleCallback: () => string = () => "en";

export function setLocaleCallback(callback: () => string) {
  getLocaleCallback = callback;
}

const api = axios.create({
  baseURL: "/api",
});

api.interceptors.request.use((config) => {
  config.headers["Accept-Language"] = getLocaleCallback();
  return config;
});

export default api;
```

Set the callback in LocaleProvider:

```typescript
// web/src/store/locale.tsx (add to LocaleProvider)

import { setLocaleCallback } from "../lib/api";
import { onMount } from "solid-js";

export const LocaleProvider: ParentComponent = (props) => {
  // ... existing code ...

  onMount(() => {
    // Set callback for API request language header
    setLocaleCallback(() => state.locale);
  });

  // ... rest of provider ...
};
```

---

### 4. Language Switcher Component (3 minutes)

Use the `useLocale` hook to get and set the language from the store:

```typescript
// web/src/components/common/LanguageSwitcher.tsx

import { For } from "solid-js";
import {
  useLocale,
  SUPPORTED_LOCALES,
  LOCALE_NAMES,
  type Locale,
} from "../../store/locale";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";
import { Button } from "../ui/button";

export function LanguageSwitcher() {
  const [state, actions] = useLocale();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" aria-label="Select language">
          <span class="text-lg">🌐</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <For each={[...SUPPORTED_LOCALES]}>
          {(loc) => (
            <DropdownMenuItem
              onClick={() => actions.setLocale(loc)}
              class={state.locale === loc ? "font-bold" : ""}
            >
              {state.locale === loc && "✓ "}
              {LOCALE_NAMES[loc]}
            </DropdownMenuItem>
          )}
        </For>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
```

---

## Build and Development

### Development Mode

paraglide-js automatically compiles translations when the Vite development server starts:

```bash
cd web
pnpm dev
```

### Production Build

Translations are compiled into optimized JavaScript during the build:

```bash
cd web
pnpm build
```

---

## Adding New Translations

1. Add new messages to `web/project.inlang/messages/en.json`
2. Add corresponding Chinese translations to `web/project.inlang/messages/zh-CN.json`
3. Vite will automatically recompile after saving the files

```json
// en.json
{
  "new_message": "New message text"
}

// zh-CN.json
{
  "new_message": "新消息文本"
}
```

Then use in components:

```typescript
import * as m from "../paraglide/messages";

function Component() {
  return <span>{m.new_message()}</span>;
}
```

---

## Messages with Parameters

paraglide-js supports ICU message format:

```json
// en.json
{
  "items_count": "You have {count} items"
}

// zh-CN.json
{
  "items_count": "您有 {count} 个项目"
}
```

Usage:

```typescript
import * as m from "../paraglide/messages";

function Component() {
  return <span>{m.items_count({ count: 5 })}</span>;
}
```

---

## VS Code Extension (Recommended)

Install the [Inlang VS Code extension](https://marketplace.visualstudio.com/items?itemName=inlang.vs-code-extension) to get:

- Inline translation preview
- Message key auto-completion
- Missing translation warnings
- Quick jump to translation files

---

## Testing

### Backend Testing

```go
// internal/i18n/i18n_test.go

package i18n

import "testing"

func TestT(t *testing.T) {
    tests := []struct {
        lang string
        key  string
        want string
    }{
        {"en", ErrTaskNotFound, "Task not found"},
        {"zh-CN", ErrTaskNotFound, "任务未找到"},
        {"en", "unknown.key", "unknown.key"},
    }

    for _, tt := range tests {
        localizer := NewLocalizer(tt.lang)
        got := T(localizer, tt.key)
        if got != tt.want {
            t.Errorf("T(%q, %q) = %q, want %q", tt.lang, tt.key, got, tt.want)
        }
    }
}

func TestTPlural(t *testing.T) {
    tests := []struct {
        lang  string
        key   string
        count int
        want  string
    }{
        {"en", StatusSyncingFiles, 1, "Syncing 1 file"},
        {"en", StatusSyncingFiles, 5, "Syncing 5 files"},
        {"zh-CN", StatusSyncingFiles, 5, "正在同步 5 个文件"},
    }

    for _, tt := range tests {
        localizer := NewLocalizer(tt.lang)
        got := TPlural(localizer, tt.key, tt.count, nil)
        if got != tt.want {
            t.Errorf("TPlural(%q, %q, %d) = %q, want %q", tt.lang, tt.key, tt.count, got, tt.want)
        }
    }
}

func TestParseLocale(t *testing.T) {
    tests := []struct {
        input string
        want  string
    }{
        {"zh-CN", "zh-CN"},
        {"zh-TW", "zh-CN"},
        {"zh", "zh-CN"},
        {"en-US", "en"},
        {"en", "en"},
        {"fr", "en"}, // Unknown language falls back to English
    }

    for _, tt := range tests {
        got := ParseLocale(tt.input)
        if got != tt.want {
            t.Errorf("ParseLocale(%q) = %q, want %q", tt.input, got, tt.want)
        }
    }
}
```

---

## Frequently Asked Questions

### Q: How to add new translation keys?

1. Add the key in `web/project.inlang/messages/en.json`
2. Add the corresponding translation in `web/project.inlang/messages/zh-CN.json`
3. Vite will automatically recompile, and TypeScript types will update automatically

### Q: What happens if a translation is missing?

- Compile time: paraglide-js will issue a warning if a translation is missing
- Run time: If a translation is missing, the source language (English) text is displayed

### Q: How to debug translation issues?

Install the Inlang VS Code extension to see translation previews and missing warnings directly in the editor.

---

## Next Steps

1. Read the full [data-model.md](./data-model.md) for detailed data structure
2. View [contracts/openapi.yaml](./contracts/openapi.yaml) for API contracts
3. Run `/speckit.tasks` to get a detailed task breakdown

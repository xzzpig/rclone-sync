---
name: check-i18n
description: This skill validates i18n (internationalization) key consistency across Go source and TOML locale files. Use when adding, modifying, or reviewing i18n keys in this project to ensure `internal/i18n/keys.go` stays in sync with `internal/i18n/locales/*.toml` files.
---

# Check I18n

## Overview

Validate that i18n keys defined in Go source code match the keys in TOML translation files. Ensures all keys in `internal/i18n/keys.go` have corresponding entries in `en.toml` and `zh-CN.toml`, and vice versa.

## When to Use

- After adding new i18n keys to `internal/i18n/keys.go`
- After adding translations to `internal/i18n/locales/en.toml` or `zh-CN.toml`
- Before committing i18n-related changes
- To diagnose missing translation issues

## Workflow

### Step 1: Run the Check Script

Execute the validation script from the project root:

```bash
go run scripts/check_i18n.go
```

Or use the bundled script in this skill:

```bash
go run .claude/skills/check-i18n/scripts/check_i18n.go
```

### Step 2: Interpret Results

The script compares keys.go with both locale files and reports:

**OK Output** (all keys match):
```
Comparing keys.go with en.toml:
  OK

Comparing keys.go with zh-CN.toml:
  OK
```

**Error Output** (mismatches found):
```
Comparing keys.go with en.toml:
  Missing in TOML (defined in keys.go):
    - error_new_key
  Extra in TOML (not in keys.go):
    - error_deprecated_key
```

### Step 3: Fix Mismatches

| Issue Type | Resolution |
|------------|------------|
| **Missing in TOML** | Add the missing key to the TOML file(s) with appropriate translation |
| **Extra in TOML** | Either remove from TOML or add the key to `keys.go` if it should exist |

## File Locations

| File | Purpose |
|------|---------|
| `internal/i18n/keys.go` | Go constants defining all i18n keys |
| `internal/i18n/locales/en.toml` | English translations |
| `internal/i18n/locales/zh-CN.toml` | Chinese (Simplified) translations |

## Key Format

In `keys.go`, keys are defined as:
```go
const (
    ErrGeneric = "error_generic"
    ErrNotFound = "error_not_found"
)
```

In TOML files, keys are defined as sections:
```toml
[error_generic]
other = "An error occurred"

[error_not_found]
other = "Resource not found"
```

## Resources

### scripts/

- `check_i18n.go` - Validation script that extracts keys from Go source and TOML files, then compares them

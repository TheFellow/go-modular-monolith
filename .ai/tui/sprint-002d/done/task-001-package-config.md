# Task 001: Package-Level Config

## Goal

Convert styles and keys from per-instance construction to package-level variables computed once at init.

## Files to Modify

```
main/tui/
├── config.go       # NEW: Package-level config exports
├── styles.go       # Rename NewStyles → newStyles (unexported)
├── keys.go         # Rename NewKeyMap → newKeyMap (unexported)
└── viewmodel_types.go  # Rename XxxFrom → xxxFrom (unexported)
```

## Implementation

### 1. Create `main/tui/config.go`

```go
package tui

import (
    "github.com/TheFellow/go-modular-monolith/pkg/tui"
    "github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
    "github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
)

// Package-level styles and keys, computed once at init.
// ViewModels import these directly instead of receiving them as parameters.
var (
    appStyles = newStyles()
    appKeys   = newKeyMap()
)

// Pre-computed style/key subsets for ViewModels to import.
var (
    ListViewStyles = listViewStylesFrom(appStyles)
    ListViewKeys   = listViewKeysFrom(appKeys)
    FormStyles     = formStylesFrom(appStyles)
    FormKeys       = formKeysFrom(appKeys)
    DialogStyles   = dialogStylesFrom(appStyles)
    DialogKeys     = dialogKeysFrom(appKeys)
)

// AppStyles returns the full application styles (used by App for status bar, etc.)
func AppStyles() Styles { return appStyles }

// AppKeys returns the full application key map (used by App for global bindings).
func AppKeys() KeyMap { return appKeys }
```

### 2. Update `main/tui/styles.go`

```go
// NewStyles → newStyles (unexported, called once by config.go)
func newStyles() Styles {
    // ... existing implementation unchanged
}
```

### 3. Update `main/tui/keys.go`

```go
// NewKeyMap → newKeyMap (unexported, called once by config.go)
func newKeyMap() KeyMap {
    // ... existing implementation unchanged
}
```

### 4. Update `main/tui/viewmodel_types.go`

Rename exports to unexported (only used by config.go now):

```go
// ListViewStylesFrom → listViewStylesFrom
func listViewStylesFrom(s Styles) tui.ListViewStyles { ... }

// ListViewKeysFrom → listViewKeysFrom
func listViewKeysFrom(k KeyMap) tui.ListViewKeys { ... }

// FormStylesFrom → formStylesFrom
func formStylesFrom(s Styles) forms.FormStyles { ... }

// FormKeysFrom → formKeysFrom
func formKeysFrom(k KeyMap) forms.FormKeys { ... }

// DialogStylesFrom → dialogStylesFrom
func dialogStylesFrom(s Styles) dialog.DialogStyles { ... }

// DialogKeysFrom → dialogKeysFrom
func dialogKeysFrom(k KeyMap) dialog.DialogKeys { ... }
```

### 5. Update `main/tui/app.go`

Change NewApp to use package-level config:

```go
func NewApp(ctx *middleware.Context, application *app.App, initialView View) *App {
    // ...
    return &App{
        // ...
        styles: appStyles,  // Was: NewStyles()
        keys:   appKeys,    // Was: NewKeyMap()
        // ...
    }
}
```

## Notes

- This is an additive change - existing code continues to work
- The `XxxFrom` functions become unexported since ViewModels will import the pre-computed values
- App still stores styles/keys for its own use (status bar, global key handling)
- ViewModels will import the exported vars directly in Task 002

## Checklist

- [x] Create `main/tui/config.go` with package-level vars
- [x] Rename `NewStyles()` → `newStyles()` in styles.go
- [x] Rename `NewKeyMap()` → `newKeyMap()` in keys.go
- [x] Rename all `XxxFrom()` → `xxxFrom()` in viewmodel_types.go
- [x] Update `NewApp()` to use `appStyles` and `appKeys`
- [x] Add `AppStyles()` and `AppKeys()` accessor functions
- [x] `go build ./...` passes
- [x] `go test ./...` passes

# Task 001: TUI Error Surface Infrastructure

## Goal

Add TUI surface support to the error generation system, following the existing pattern for HTTP, gRPC, and CLI surfaces.

## Files to Modify/Create

- `pkg/errors/errors.go` - Add TUIStyle type and field to ErrorKind
- `pkg/errors/gen/errors.go.tpl` - Add TUIStyle() method generation
- `pkg/errors/tui.go` - Create TUI surface helpers (new file)
- `main/tui/styles.go` - Add WarningText and InfoText styles

## Pattern Reference

Follow `pkg/errors/cli.go` and the existing `ErrorKind` struct pattern.

## Implementation

### 1. Add TUIStyle type to `pkg/errors/errors.go`

```go
type TUIStyle int

const (
    TUIStyleError   TUIStyle = iota // Red - user errors, permission denied, internal
    TUIStyleWarning                  // Amber - not found, conflicts (recoverable)
    TUIStyleInfo                     // Muted - informational messages
)
```

### 2. Add TUIStyle field to ErrorKind struct

```go
type ErrorKind struct {
    Name     string
    Message  string
    HTTPCode httpCode
    GRPCCode codes.Code
    CLICode  int
    TUIStyle TUIStyle  // NEW
}
```

### 3. Assign TUIStyle to each error kind

- `ErrInvalid` → `TUIStyleError`
- `ErrNotFound` → `TUIStyleWarning`
- `ErrPermission` → `TUIStyleError`
- `ErrConflict` → `TUIStyleWarning`
- `ErrInternal` → `TUIStyleError`

### 4. Update template `pkg/errors/gen/errors.go.tpl`

Add method:
```go
func (e *{{ .Name }}Error) TUIStyle() TUIStyle { return {{ printf "Err%s" .Name }}.TUIStyle }
```

### 5. Create `pkg/errors/tui.go`

```go
package errors

type TUIError struct {
    Style   TUIStyle
    Message string
    Err     error
}

func ToTUIError(err error) TUIError {
    // Check for tuiStyler interface, default to TUIStyleError
}
```

### 6. Add styles to `main/tui/styles.go`

Add to Styles struct:
```go
WarningText lipgloss.Style
InfoText    lipgloss.Style
```

Initialize in NewStyles():
```go
styles.WarningText = lipgloss.NewStyle().Bold(true).Foreground(styles.Warning)
styles.InfoText = lipgloss.NewStyle().Foreground(styles.Muted)
```

### 7. Regenerate error types

```bash
go generate ./pkg/errors/...
```

## Notes

- The TUIStyle determines how errors are displayed in the status bar
- WarningText uses amber/yellow for recoverable errors
- InfoText uses muted color for informational messages
- Run `go generate` after template changes

## Checklist

- [x] Add TUIStyle type and constants to errors.go
- [x] Add TUIStyle field to ErrorKind struct
- [x] Assign TUIStyle to all error kinds
- [x] Update errors.go.tpl to generate TUIStyle() method
- [x] Run `go generate ./pkg/errors/...`
- [x] Create pkg/errors/tui.go with ToTUIError()
- [x] Add WarningText and InfoText to main/tui/styles.go
- [x] `go build ./...` passes
- [x] `go test ./...` passes

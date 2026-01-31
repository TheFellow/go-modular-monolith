# Task 001: Add Bubble Tea Dependencies

## Goal

Add Bubble Tea ecosystem dependencies to go.mod so subsequent tasks can import them.

## Files to Modify

- `go.mod`
- `go.sum` (auto-generated)

## Implementation

Run `go get` to add the three core Charm libraries:

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
```

These packages provide:
- `bubbletea` - The Elm Architecture framework for terminal apps
- `bubbles` - Pre-built components (help, spinner, textinput, list, etc.)
- `lipgloss` - Styling/layout library for terminal UIs

## Notes

- This is a standalone task since it only modifies dependency files
- No Go code changes required
- Verify imports work by checking `go mod tidy` succeeds

## Checklist

- [x] Run `go get` for all three packages
- [x] Run `go mod tidy`
- [x] `go build ./...` passes
- [x] `go test ./...` passes

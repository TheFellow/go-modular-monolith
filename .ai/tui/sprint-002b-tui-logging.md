# Sprint 002b: TUI File-Based Logging

## Overview

Add opt-in file-based logging for TUI mode. When the TUI is active, logs written to stderr corrupt the display or are hidden entirely. This sprint adds a `--log-file` flag to redirect logs to a file when running in TUI mode.

## Problem

1. CLI sets up logging to `os.Stderr`
2. TUI uses `tea.WithAltScreen()` which takes over the terminal
3. Logs to stderr either corrupt the display or are hidden
4. No way to debug TUI issues without log output

## Solution

Add a `--log-file` CLI flag that:
- Redirects application logs to a file when specified
- Works for both CLI and TUI modes
- Uses Bubble Tea's `tea.LogToFile()` pattern for compatibility

## Tasks

### Task 1: Add --log-file flag

**Files to modify:**
- `main/cli/cli.go`

**Implementation:**
```go
&cli.StringFlag{
    Name:        "log-file",
    Usage:       "Write logs to file instead of stderr",
    Destination: &c.logFile,
    Sources:     cli.EnvVars("MIXOLOGY_LOG_FILE"),
},
```

In `Before`:
```go
var logOutput io.Writer = os.Stderr
if c.logFile != "" {
    f, err := os.OpenFile(c.logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
    if err != nil {
        return ctx, fmt.Errorf("open log file: %w", err)
    }
    logOutput = f
    // Store f for cleanup in After
}
logger := pkglog.Setup(c.logLevel, c.logFormat, logOutput)
```

### Task 2: Auto-enable for TUI mode (optional)

Consider auto-creating a log file when `--tui` is specified without `--log-file`:
- Default to `mixology-tui.log` in current directory or temp
- Or just document that `--log-file` should be used with `--tui`

### Task 3: Documentation

Update help text and any README to document:
- `--log-file` flag usage
- Recommended usage with TUI mode
- Log file location and format

## Usage Examples

```bash
# TUI with logging to file
mixology --tui --log-file debug.log

# TUI with logging via env var
MIXOLOGY_LOG_FILE=debug.log mixology --tui

# CLI commands also support it
mixology --log-file debug.log drinks list
```

## Success Criteria

- [ ] `--log-file` flag added and functional
- [ ] Logs redirected to file when flag is set
- [ ] File is properly closed on exit
- [ ] TUI runs cleanly without log corruption
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

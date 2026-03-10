# Sprint 002b Summary: TUI File-Based Logging

**Status:** Complete
**Duration:** Feb 2, 2026

## What Was Accomplished

Added file-based logging support for the TUI to enable debugging without corrupting the display. The TUI now automatically logs to a file when `--log-file` is not explicitly set.

### Features Implemented

1. **`--log-file` CLI flag**: Redirects application logs to a specified file
2. **Environment variable support**: `MIXOLOGY_LOG_FILE` provides an alternative to the flag
3. **TUI default logging**: When `--tui` is active and no log file is specified, logs go to `data/mixology-tui.log` (or `mixology-tui.log` if data/ doesn't exist)
4. **Fresh logger per middleware call**: Each command/query starts with a clean logger to prevent attribute accumulation in long-running TUI sessions
5. **Proper file cleanup**: Log file handle is closed in the `After` hook

## Files Changed

### main/cli/cli.go

Added `--log-file` flag and TUI default logging:

```go
type CLI struct {
    // ...
    logFile       string    // Path from --log-file flag
    logFileHandle *os.File  // Handle for cleanup
}
```

- Added `--log-file` flag with `MIXOLOGY_LOG_FILE` env var support
- TUI mode defaults to `data/mixology-tui.log` when no log file specified
- Creates `data/` directory if it doesn't exist
- File cleanup in `After` hook

### pkg/middleware/logging.go

Added fresh logger reset at start of each middleware invocation to prevent cross-command attribute accumulation.

### pkg/log/context.go

Added `ResetLogger()` function to restore the root logger in context.

## Usage Examples

```bash
# TUI with explicit log file
mixology --tui --log-file debug.log

# TUI with default log file (data/mixology-tui.log)
mixology --tui

# Via environment variable
MIXOLOGY_LOG_FILE=debug.log mixology --tui

# CLI commands also support it
mixology --log-file debug.log drinks list

# Combined with other logging options
mixology --tui --log-file debug.log --log-level debug --log-format json
```

## Deviations from Plan

1. **TUI default logging added**: Originally planned as optional, but implemented automatic logging to `data/mixology-tui.log` for TUI mode to ensure logs are always available for debugging.

2. **Fresh logger per middleware call**: Added after discovering that log attributes accumulated across commands in long-running TUI sessions, causing confusing log output.

## Verification

```bash
go build ./...  # Passes
go test ./...   # Passes
```

## Next Steps

- Sprint 003b (Saga Infrastructure) or Sprint 004 (Workflows) as planned in the TUI roadmap

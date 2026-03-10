# Task 002: Add File Cleanup and TUI Default Log File

## Goal

Ensure the log file handle is properly closed when the application exits, and default TUI mode to log to a file when
`--log-file` is not provided.

## File to Modify

`main/cli/cli.go`

## Pattern Reference

Follow the existing cleanup pattern for `metricsServer` and `metricsShutdown` in the `After` hook (lines 151-161).

## Current State

After Task 001, the CLI struct has a `logFileHandle` field that stores the opened log file.

The current `After` hook:

```go
After: func(ctx context.Context, _ *cli.Command) error {
    if c.app != nil {
        _ = c.app.Close()
    }
    if c.metricsServer != nil {
        _ = c.metricsServer.Shutdown(ctx)
    }
    if c.metricsShutdown != nil {
        _ = c.metricsShutdown(ctx)
    }
    return nil
},
```

## Implementation

### Default log file for TUI

When `--tui` is present and `--log-file` is empty, set a default log file path (e.g. `data/mixology-tui.log`) and create
the log directory if needed before opening the file.

### Update After hook

Add log file cleanup before the return statement:

```go
After: func(ctx context.Context, _ *cli.Command) error {
    if c.app != nil {
        _ = c.app.Close()
    }
    if c.metricsServer != nil {
        _ = c.metricsServer.Shutdown(ctx)
    }
    if c.metricsShutdown != nil {
        _ = c.metricsShutdown(ctx)
    }
    if c.logFileHandle != nil {
        _ = c.logFileHandle.Close()
    }
    return nil
},
```

## Notes

- Cleanup order: app first, then metrics, then log file
- Log file should be closed last so any final logs from other cleanup can be captured
- Errors from Close() are ignored (matching existing pattern)
- The nil check ensures no panic if the file was never opened

## Checklist

- [x] Default `--tui` to a log file when `--log-file` is empty
- [x] Log directory is created for the default path
- [x] Add log file cleanup to After hook
- [x] Cleanup happens after app.Close() and metrics shutdown
- [x] `go build ./...` passes
- [x] `go test ./...` passes

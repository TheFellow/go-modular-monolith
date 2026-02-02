# Sprint 002b: TUI File-Based Logging

**Status:** Complete

## Goal

Add file-based logging for TUI mode. When the TUI is active, logs written to stderr corrupt the display or are hidden
entirely. This sprint adds a `--log-file` flag to redirect logs to a file and defaults TUI logging to a file when the
flag is not provided.

## Scope

**In Scope:**

- Add `--log-file` CLI flag to redirect application logs to a file
- Support both CLI and TUI modes with the same flag
- Default TUI logging to a file when `--log-file` is not set
- Use Bubble Tea's `tea.LogToFile()` pattern for compatibility
- Proper file handle cleanup on application exit

**Out of Scope:**

- Log rotation (users can use external tools like logrotate)
- Log aggregation or remote logging

## Reference

**Pattern to follow:** `vendor/github.com/charmbracelet/bubbletea/logging.go`

Bubble Tea provides `tea.LogToFile()` which demonstrates the pattern of:
1. Opening a file for append
2. Redirecting the standard library logger output
3. Returning the file handle for cleanup

Our implementation integrates with `pkg/log` which uses `log/slog`, so we'll pass the file writer to `pkglog.Setup()`.

## Current State

The CLI currently sets up logging in `main/cli/cli.go`:

```go
func (c *CLI) Command() *cli.Command {
    // ...
    Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
        logger := pkglog.Setup(c.logLevel, c.logFormat, os.Stderr)
        // ...
    },
}
```

Logging always goes to `os.Stderr`. When `--tui` is active, `tea.WithAltScreen()` takes over the terminal and stderr
output either corrupts the display or is hidden.

## Key Implementation Notes

1. **CLI struct needs a new field** for `logFile` path
2. **CLI struct needs a field** for the opened `*os.File` handle (for cleanup)
3. **Before hook** opens the file and passes it to `pkglog.Setup()`
4. **After hook** closes the file if it was opened
5. **Environment variable** `MIXOLOGY_LOG_FILE` provides alternative to flag

---

## Tasks

| Task | Description                                                 | Status  |
|------|-------------------------------------------------------------|---------|
| 001  | [Add --log-file flag](done/task-001-log-file-flag.md)       | Done    |
| 002  | [Add file cleanup + TUI default log file](done/task-002-file-cleanup.md) | Done    |
| 003  | [Fresh logger + docs/testing](done/task-003-documentation.md) | Done |

### Task Dependencies

```
001 (flag) ── 002 (cleanup) ── 003 (docs/testing)
```

Tasks are sequential - each depends on the previous.

---

## Success Criteria

- [ ] `--log-file` flag added to CLI
- [ ] Environment variable `MIXOLOGY_LOG_FILE` works
- [ ] Logs redirected to file when flag is set
- [ ] TUI defaults to a log file when `--log-file` is not set
- [ ] File is properly closed on exit (both normal and error)
- [ ] TUI runs cleanly without log corruption when using `--log-file`
- [ ] CLI commands also work with `--log-file`
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

## Usage Examples

```bash
# TUI with logging to file
mixology --tui --log-file debug.log

# TUI defaults to data/mixology-tui.log when no log file is specified
mixology --tui

# TUI with logging via env var
MIXOLOGY_LOG_FILE=debug.log mixology --tui

# CLI commands also support it (useful for debugging)
mixology --log-file debug.log drinks list

# Combine with other logging options
mixology --tui --log-file debug.log --log-level debug --log-format json
```

# Task 003: Documentation and Testing

## Goal

Verify the logging functionality works correctly and document its usage.

## Files to Modify/Create

- None (manual testing and verification)

## Implementation

### 1. Manual Testing

Test the following scenarios:

**TUI with log file:**
```bash
# Start TUI with logging
mixology --tui --log-file /tmp/tui-debug.log

# In another terminal, tail the log
tail -f /tmp/tui-debug.log

# Exercise TUI functionality, verify logs appear
# Exit TUI, verify file is closed properly
```

**CLI with log file:**
```bash
# CLI command with logging
mixology --log-file /tmp/cli-debug.log drinks list

# Verify logs were written
cat /tmp/cli-debug.log
```

**Environment variable:**
```bash
# Via environment variable
MIXOLOGY_LOG_FILE=/tmp/env-debug.log mixology --tui
```

**Combined with other log options:**
```bash
# Debug level, JSON format, to file
mixology --tui --log-file /tmp/debug.log --log-level debug --log-format json
```

**Error cases:**
```bash
# Invalid path (should fail gracefully)
mixology --tui --log-file /nonexistent/path/debug.log

# No write permission (should fail gracefully)
mixology --tui --log-file /etc/debug.log
```

### 2. Verify Existing Tests Pass

```bash
go test ./main/cli/...
go test ./...
```

### 3. Verify Help Text

```bash
mixology --help

# Should show:
#   --log-file value   Write logs to file instead of stderr [$MIXOLOGY_LOG_FILE]
```

## Notes

- No new test files needed - this is infrastructure that's hard to unit test
- Manual verification ensures the feature works end-to-end
- The sprint-002b-tui-logging.md file serves as documentation

## Checklist

- [ ] TUI with `--log-file` works, logs appear in file
- [ ] CLI with `--log-file` works
- [ ] Environment variable `MIXOLOGY_LOG_FILE` works
- [ ] Combined options (`--log-level`, `--log-format`) work with `--log-file`
- [ ] Invalid file paths produce clear error messages
- [ ] Help text shows the new flag
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

# Task 003: Cleanup

## Goal

Remove the "fresh logger" hack and clean up unused code after the architecture update.

## Files to Modify

```
pkg/middleware/logging.go    # Remove ResetLogger call
pkg/log/context.go           # Remove ResetLogger function (if unused)
main/tui/viewmodel_types.go  # Remove if empty after unexport
```

## Implementation

### 1. Remove fresh logger hack from middleware

The "fresh logger" hack was added in sprint-002b to work around log attribute accumulation. With fresh context per command/query, this is no longer needed.

**In `pkg/middleware/logging.go`:**

```go
// Remove this line that resets the logger at the start of each middleware call
log.ResetLogger(ctx)
```

### 2. Remove ResetLogger if unused

Check if `pkg/log/context.go` `ResetLogger()` is used elsewhere. If not, remove it:

```go
// Remove if no longer needed
func ResetLogger(ctx context.Context) {
    // ...
}
```

### 3. Verify logging behavior

Test that logging works correctly without the hack:

```bash
# Start TUI with logging
mixology --tui --log-file /tmp/test.log

# Perform multiple operations (create drink, edit, delete, etc.)
# Verify in log file that each operation has clean attributes
# No accumulation of action/resource from previous operations
```

### 4. Clean up viewmodel_types.go

After Task 001 and 002, `main/tui/viewmodel_types.go` may only contain unexported helper functions. If these are only called from `config.go`, consider:

- Moving them inline into `config.go`, or
- Keeping them in `viewmodel_types.go` for organization

Either approach is fine - choose based on readability.

## Notes

- The fresh logger hack was a symptom of the long-lived context problem
- With proper fresh contexts, logging attributes are naturally isolated per operation
- This cleanup validates that the architecture fix is complete

## Checklist

- [ ] Remove `log.ResetLogger(ctx)` call from `pkg/middleware/logging.go`
- [ ] Remove `ResetLogger()` from `pkg/log/context.go` if unused
- [ ] Verify logging works correctly (no attribute accumulation)
- [ ] Clean up or consolidate `viewmodel_types.go` if desired
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `go tool go-check-sumtype ./...` passes
- [ ] `go tool exhaustive ./...` passes

# Task 001: Add --log-file Flag

## Goal

Add a `--log-file` CLI flag that redirects application logs to a file instead of stderr.

## File to Modify

`main/cli/cli.go`

## Pattern Reference

Follow the existing flag pattern in `main/cli/cli.go` (lines 59-85) for `--log-level` and `--log-format`.

## Current State

```go
type CLI struct {
    app             *app.App
    dbPath          string
    actor           string
    logLevel        string
    logFormat       string
    enableMetrics   bool
    metricsServer   *http.Server
    metricsShutdown func(context.Context) error
}
```

The `Before` hook sets up logging:

```go
Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
    logger := pkglog.Setup(c.logLevel, c.logFormat, os.Stderr)
    // ...
}
```

## Implementation

### 1. Add fields to CLI struct

Add two new fields:

```go
type CLI struct {
    // ... existing fields ...
    logFile     string    // Path from --log-file flag
    logFileHandle *os.File // Handle for cleanup (Task 002)
}
```

### 2. Add the flag

Add after the existing `--log-format` flag:

```go
&cli.StringFlag{
    Name:        "log-file",
    Usage:       "Write logs to file instead of stderr",
    Destination: &c.logFile,
    Sources:     cli.EnvVars("MIXOLOGY_LOG_FILE"),
},
```

### 3. Update Before hook

Modify the logging setup in `Before`:

```go
Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
    var logOutput io.Writer = os.Stderr
    if c.logFile != "" {
        f, err := os.OpenFile(c.logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
        if err != nil {
            return ctx, fmt.Errorf("open log file: %w", err)
        }
        logOutput = f
        c.logFileHandle = f  // Store for cleanup in After
    }
    logger := pkglog.Setup(c.logLevel, c.logFormat, logOutput)
    // ... rest unchanged ...
}
```

### 4. Add import if needed

Ensure `io` is imported (check if already present).

## Notes

- File is opened with append mode (`os.O_APPEND`) so logs accumulate across runs
- Permissions `0o600` ensure only the owner can read/write the log file
- The `logFileHandle` field stores the handle for cleanup (implemented in Task 002)
- Error opening the file is fatal - application won't start if it can't write logs

## Checklist

- [ ] Add `logFile` field to CLI struct
- [ ] Add `logFileHandle` field to CLI struct
- [ ] Add `--log-file` flag with `MIXOLOGY_LOG_FILE` env var support
- [ ] Update Before hook to open file and pass to pkglog.Setup
- [ ] Store file handle in `c.logFileHandle`
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

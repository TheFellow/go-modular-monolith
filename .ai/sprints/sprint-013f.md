# Sprint 013f: CLI Exit Codes (Intermezzo)

## Goal

Add CLI exit codes to the error translation layer so the CLI can return meaningful exit codes to the shell.

## Background

urfave/cli v3 supports custom exit codes via `cli.Exit(message, code)` or the `cli.ExitCoder` interface. Currently our `ErrorKind` has HTTP and gRPC codes but no CLI exit code.

## Tasks

- [x] Add `CLICode int` to `ErrorKind` struct
- [x] Define exit code constants with semantic meaning
- [x] Add `ToCLIExit(error) error` helper that returns `cli.Exit()`
- [x] Update CLI error handling to use exit codes

## Exit Code Conventions

Standard Unix conventions:
- `0` - Success
- `1` - General error
- `2` - Misuse of command (invalid args)

Application-specific (we define):
- `10` - Validation/invalid input error
- `20` - Not found error
- `30` - Authorization error
- `50` - Internal error

## Implementation

### Update ErrorKind

```go
// pkg/errors/errors.go

type ErrorKind struct {
    Name     string
    Message  string
    HTTPCode httpCode
    GRPCCode codes.Code
    CLICode  int
}

// Exit codes
const (
    ExitSuccess     = 0
    ExitGeneral     = 1
    ExitUsage       = 2
    ExitInvalid     = 10
    ExitNotFound    = 20
    ExitUnauthorized = 30
    ExitInternal    = 50
)

var (
    ErrInvalid = ErrorKind{
        Name:     "Invalid",
        Message:  "invalid",
        HTTPCode: http.StatusBadRequest,
        GRPCCode: codes.InvalidArgument,
        CLICode:  ExitInvalid,
    }
    ErrNotFound = ErrorKind{
        Name:     "NotFound",
        Message:  "not found",
        HTTPCode: http.StatusNotFound,
        GRPCCode: codes.NotFound,
        CLICode:  ExitNotFound,
    }
    ErrInternal = ErrorKind{
        Name:     "Internal",
        Message:  "internal error",
        HTTPCode: http.StatusInternalServerError,
        GRPCCode: codes.Internal,
        CLICode:  ExitInternal,
    }
)
```

### CLI Exit Helper

```go
// pkg/errors/cli.go

import "github.com/urfave/cli/v3"

// ToCLIExit converts an error to a cli.ExitCoder with appropriate exit code.
func ToCLIExit(err error) error {
    if err == nil {
        return nil
    }

    var appErr *Error
    if As(err, &appErr) {
        return cli.Exit(appErr.Error(), appErr.Kind.CLICode)
    }

    // Unknown error defaults to general error
    return cli.Exit(err.Error(), ExitGeneral)
}
```

### CLI Usage

```go
// main/cli/drinks.go

func createDrinkAction(ctx context.Context, cmd *cli.Command) error {
    drink, err := drinksModule.Create(mwCtx, drink)
    if err != nil {
        return errors.ToCLIExit(err)
    }
    fmt.Printf("Created drink: %s\n", drink.ID)
    return nil
}
```

### Shell Usage

```bash
$ mixology drinks create "Margarita"
Created drink: abc-123
$ echo $?
0

$ mixology drinks get "nonexistent"
Error: drink not found
$ echo $?
20

$ mixology drinks create ""
Error: name is required
$ echo $?
10
```

## Success Criteria

- `ErrorKind` includes `CLICode`
- All error kinds have appropriate exit codes
- `ToCLIExit()` helper converts errors to `cli.Exit()`
- CLI commands return correct exit codes
- `go test ./...` passes

## Dependencies

- Sprint 013e (error handling patterns established)

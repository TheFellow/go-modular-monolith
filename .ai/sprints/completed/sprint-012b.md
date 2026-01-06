# Sprint 012b: CLI Idioms Cleanup (Intermezzo)

## Goal

Align CLI code with urfave/cli v3 idiomatic patterns, reducing boilerplate and improving user experience.

## Problems Identified

### Problem 1: Raw Argument Access

Currently using `cmd.Args().Get(0)` with manual validation:

```go
// Current - manual, no help text, no type conversion
id := cmd.Args().First()
if id == "" {
    return fmt.Errorf("missing id")
}
```

v3 provides typed `Arguments` with validation:

```go
// Idiomatic - typed, validated, documented
Arguments: []cli.Argument{
    &cli.StringArg{
        Name:     "id",
        Usage:    "Drink ID",
        Required: true,
    },
},
Action: func(ctx context.Context, cmd *cli.Command) error {
    id := cmd.StringArg("id")  // Already validated
    // ...
}
```

### Problem 2: Double-Pointer App Pattern

Current pattern to handle `Before` initialization:

```go
func drinksCommands(a **app.App) *cli.Command {
    // ...
    if a == nil || *a == nil {
        return fmt.Errorf("app not initialized")
    }
    (*a).Drinks().List(...)
}
```

This is awkward. Better to use a struct with app stored after init:

```go
type CLI struct {
    app *app.App
}

func (c *CLI) drinksCommands() *cli.Command {
    // ...
    c.app.Drinks().List(...)  // Cleaner access
}
```

### Problem 3: Repeated Context Boilerplate

Every action has:
```go
mctx, err := requireMiddlewareContext(ctx)
if err != nil {
    return err
}
if a == nil || *a == nil {
    return fmt.Errorf("app not initialized")
}
```

Could wrap actions:
```go
func (c *CLI) action(fn func(*middleware.Context, *cli.Command) error) cli.ActionFunc {
    return func(ctx context.Context, cmd *cli.Command) error {
        mctx, ok := ctx.(*middleware.Context)
        if !ok {
            return fmt.Errorf("expected middleware context")
        }
        return fn(mctx, cmd)
    }
}
```

### Problem 4: No Flag Validators

Enum-like flags accept any value:
```go
&cli.StringFlag{
    Name:     "reason",
    Required: true,
    // No validation - accepts "garbage"
}
```

v3 supports validators:
```go
&cli.StringFlag{
    Name:     "reason",
    Required: true,
    Validator: func(s string) error {
        switch models.AdjustmentReason(s) {
        case models.ReasonReceived, models.ReasonUsed,
             models.ReasonSpilled, models.ReasonExpired, models.ReasonCorrected:
            return nil
        }
        return fmt.Errorf("invalid reason: %s", s)
    },
}
```

## Tasks

- [x] Create `CLI` struct to hold app reference, eliminating double-pointer
- [x] Add `action()` wrapper method to reduce context boilerplate
- [x] Convert positional args to typed `Arguments` with validation
- [x] Add `Validator` functions to enum-like flags (reason, category, unit)
- [x] Remove manual `if id == \"\"` checks (use typed args)
- [x] Update all command files (drinks.go, ingredients.go, inventory.go)

## Revised Structure

```go
// main/cli/cli.go
type CLI struct {
    app *app.App
}

func New() (*CLI, error) {
    a, err := app.New()
    if err != nil {
        return nil, err
    }
    return &CLI{app: a}, nil
}

func (c *CLI) action(fn func(*middleware.Context, *cli.Command) error) cli.ActionFunc {
    return func(ctx context.Context, cmd *cli.Command) error {
        mctx, ok := ctx.(*middleware.Context)
        if !ok {
            return fmt.Errorf("expected middleware context")
        }
        return fn(mctx, cmd)
    }
}

func (c *CLI) Command() *cli.Command {
    return &cli.Command{
        Name:  "mixology",
        Usage: "Mixology as a Service",
        Flags: []cli.Flag{...},
        Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
            // Setup principal from --as flag
            return middleware.NewContext(ctx, middleware.WithPrincipal(p)), nil
        },
        Commands: []*cli.Command{
            c.drinksCommands(),
            c.ingredientsCommands(),
            c.inventoryCommands(),
        },
    }
}
```

## Example: Refactored Command

Before:
```go
{
    Name:      "get",
    Usage:     "Get a drink by ID",
    ArgsUsage: "<id>",
    Flags:     []cli.Flag{JSONFlag},
    Action: func(ctx context.Context, cmd *cli.Command) error {
        id := cmd.Args().First()
        if id == "" {
            return fmt.Errorf("missing id")
        }
        mctx, err := requireMiddlewareContext(ctx)
        if err != nil {
            return err
        }
        if a == nil || *a == nil {
            return fmt.Errorf("app not initialized")
        }
        // ...
    },
}
```

After:
```go
{
    Name:  "get",
    Usage: "Get a drink by ID",
    Flags: []cli.Flag{JSONFlag},
    Arguments: []cli.Argument{
        &cli.StringArg{
            Name:     "id",
            Usage:    "Drink ID",
            Required: true,
        },
    },
    Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
        id := cmd.StringArg("id")
        // ... no validation needed, no context assertion
    }),
}
```

## Success Criteria

- No `cmd.Args().Get()` or `cmd.Args().First()` calls
- No manual `if arg == ""` validation for required args
- No double-pointer `**app.App` pattern
- Enum flags reject invalid values at parse time
- `go build ./...` passes
- `go test ./...` passes
- CLI help text shows argument names

## Dependencies

- None (CLI-only changes)

# Task 003: CLI - Add menu draft Subcommand

## Goal

Add a CLI subcommand to draft (unpublish) a menu.

## File to Modify

```
main/cli/menu.go
```

## Pattern Reference

Follow the `publish` subcommand pattern at `main/cli/menu.go:289-313`.

## Implementation

Add a new subcommand after the `publish` command:

```go
{
    Name:  "draft",
    Usage: "Return a published menu to draft status",
    Flags: []cli.Flag{
        JSONFlag,
        &cli.StringFlag{Name: "id", Usage: "Menu ID", Required: true},
    },
    Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
        menuID, err := entity.ParseMenuID(cmd.String("id"))
        if err != nil {
            return err
        }
        drafted, err := c.app.Menu.Draft(ctx, &menumodels.Menu{ID: menuID})
        if err != nil {
            return err
        }

        if cmd.Bool("json") {
            return writeJSON(cmd.Writer, menucli.FromDomainMenu(*drafted))
        }

        fmt.Println(drafted.ID.String())
        return nil
    }),
},
```

## Notes

- Place after the `publish` command for logical grouping
- Same flag pattern as publish: `--id` required, `--json` optional
- Returns the menu ID on success (or full JSON with `--json`)

## Checklist

- [x] Add `draft` subcommand to `menu.go`
- [x] `go build ./main/cli/...` passes
- [x] `go test ./main/cli/...` passes
- [x] Manual test: `mixology menu draft --id <published-menu-id>` works

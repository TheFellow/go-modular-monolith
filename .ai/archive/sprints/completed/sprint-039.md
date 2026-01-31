# Sprint 039: CLI Output Consistency

## Goal

1. Simplify create command output to just the ID (script-friendly)
2. Add JSONFlag to all list/get commands for consistency

## Problem

### Create Commands Output Too Much

```bash
# Current - hard to parse in scripts
$ mixology ingredients create "Vodka" --category spirit --unit oz
ing-abc123	Vodka	spirit	oz

# Desired - just the ID
$ mixology ingredients create "Vodka" --category spirit --unit oz
ing-abc123
```

Scripts need to extract the ID:
```bash
# Current - awkward
ID=$(mixology ingredients create "Vodka" -c spirit -u oz | cut -f1)

# Desired - simple
ID=$(mixology ingredients create "Vodka" -c spirit -u oz)
```

### Inconsistent JSONFlag Support

| Command | Has JSONFlag? |
|---------|---------------|
| `drinks list` | ✓ |
| `drinks get` | ✓ |
| `ingredients list` | ✗ |
| `ingredients get` | ✗ |
| `inventory list` | ✗ |
| `inventory get` | ✗ |
| `orders list` | ✗ |
| `orders get` | ✗ |
| `audit list` | ✗ |
| `menu list` | ✓ |
| `menu show` | ✓ |

## Changes

### Part 1: Simplify Create Output

All create commands should output only the new entity's ID.

#### ingredients.go

```go
// Before
fmt.Printf("%s\t%s\t%s\t%s\n", res.ID.String(), res.Name, res.Category, res.Unit)

// After
fmt.Println(res.ID.String())
```

#### drinks.go

```go
// Before
fmt.Printf("%s\t%s\n", res.ID.String(), res.Name)

// After
fmt.Println(res.ID.String())
```

#### menu.go

```go
// Before (create)
fmt.Printf("%s\t%s\n", created.ID.String(), created.Name)

// After
fmt.Println(created.ID.String())
```

#### orders.go

```go
// Before (place)
fmt.Printf("%s\t%s\t%s\t%d\n", created.ID.String(), created.MenuID.String(), created.Status, len(created.Items))

// After
fmt.Println(created.ID.String())
```

### Part 2: Add JSONFlag to All List/Get Commands

#### ingredients.go

```go
// list command
{
    Name:  "list",
    Usage: "List ingredients",
    Flags: []cli.Flag{
        JSONFlag,  // ADD
        &cli.StringFlag{...},
    },
    Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
        res, err := c.app.Ingredients.List(ctx, ...)
        if err != nil {
            return err
        }

        if cmd.Bool("json") {  // ADD
            return writeJSON(cmd.Writer, res)
        }

        w := newTabWriter()
        // ... existing table output
    }),
}

// get command
{
    Name:  "get",
    Usage: "Get an ingredient by ID",
    Flags: []cli.Flag{
        JSONFlag,  // ADD
        &cli.StringFlag{Name: "id", ...},
    },
    Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
        // ...
        if cmd.Bool("json") {  // ADD
            return writeJSON(cmd.Writer, res)
        }
        // ... existing detail output
    }),
}
```

#### inventory.go

```go
// list command - add JSONFlag
// get command - add JSONFlag
```

#### orders.go

```go
// list command - add JSONFlag
// get command - add JSONFlag
```

#### audit.go

```go
// list command - add JSONFlag
```

## Audit: Current State

### Create Commands

| Command | Current Output | Change To |
|---------|----------------|-----------|
| `ingredients create` | `ID\tName\tCategory\tUnit` | `ID` |
| `drinks create` | `ID\tName` | `ID` |
| `drinks update` | `ID\tName` | `ID` |
| `menu create` | `ID\tName` | `ID` |
| `menu add-drink` | `ID\tName\tCount` | `ID` |
| `menu remove-drink` | `ID\tName\tCount` | `ID` |
| `menu publish` | `ID\tName\tStatus` | `ID` |
| `orders place` | `ID\tMenuID\tStatus\tCount` | `ID` |
| `orders complete` | `ID\tStatus` | `ID` |
| `orders cancel` | `ID\tStatus` | `ID` |

### List/Get Commands Missing JSONFlag

| File | Command | Needs JSONFlag |
|------|---------|----------------|
| `ingredients.go` | list | ✓ |
| `ingredients.go` | get | ✓ |
| `ingredients.go` | create | ✓ |
| `ingredients.go` | update | ✓ |
| `ingredients.go` | delete | ✓ |
| `inventory.go` | list | ✓ |
| `inventory.go` | get | ✓ |
| `orders.go` | list | ✓ |
| `orders.go` | get | ✓ |
| `audit.go` | list | ✓ |

## JSON Output Structures

For commands that need JSON output, use existing CLI surface DTOs where available, or domain models directly:

```go
// ingredients - use domain model directly (simple structure)
if cmd.Bool("json") {
    return writeJSON(cmd.Writer, res)
}

// drinks - use CLI DTO (has recipe transformation)
if cmd.Bool("json") {
    return writeJSON(cmd.Writer, drinkscli.FromDomainDrink(*res))
}
```

## Tasks

### Phase 1: Simplify Create Output

- [x] `ingredients.go` create: output only ID
- [x] `ingredients.go` update: output only ID
- [x] `drinks.go` create: output only ID (keep JSON option)
- [x] `drinks.go` update: output only ID (keep JSON option)
- [x] `menu.go` create: output only ID
- [x] `menu.go` add-drink: output only ID
- [x] `menu.go` remove-drink: output only ID
- [x] `menu.go` publish: output only ID
- [x] `orders.go` place: output only ID
- [x] `orders.go` complete: output only ID
- [x] `orders.go` cancel: output only ID

### Phase 2: Add JSONFlag to List/Get

- [x] `ingredients.go` list: add JSONFlag
- [x] `ingredients.go` get: add JSONFlag
- [x] `inventory.go` list: add JSONFlag
- [x] `inventory.go` get: add JSONFlag
- [x] `orders.go` list: add JSONFlag
- [x] `orders.go` get: add JSONFlag
- [x] `audit.go` list: add JSONFlag

### Phase 3: Add JSONFlag to Mutations (optional but consistent)

- [x] `ingredients.go` create/update/delete: add JSONFlag
- [x] `inventory.go` adjust/set: add JSONFlag
- [x] `orders.go` place/complete/cancel: add JSONFlag

### Phase 4: Verify

- [x] Test create commands output only ID
- [x] Test `--json` flag on all list/get commands
- [x] Test scripting workflow: `ID=$(mixology ingredients create ...)`

## Acceptance Criteria

- [x] All create/mutation commands output only the entity ID (plain text)
- [x] All list commands support `--json` flag
- [x] All get commands support `--json` flag
- [x] JSON output uses appropriate DTO or domain model
- [x] Scripts can capture IDs directly: `ID=$(mixology ... create ...)`

## Result

```bash
# Create - just the ID
$ mixology ingredients create "Vodka" -c spirit -u oz
ing-2xK7mJ9...

# Easy scripting
$ ID=$(mixology ingredients create "Vodka" -c spirit -u oz)
$ echo "Created: $ID"
Created: ing-2xK7mJ9...

# Chain commands
$ mixology inventory set \
    --ingredient-id $(mixology ingredients create "Lime" -c citrus -u piece) \
    --quantity 50 \
    --cost-per-unit '$0.25'

# JSON for programmatic access
$ mixology ingredients list --json
[
  {"id": "ing-abc", "name": "Vodka", "category": "spirit", "unit": "oz"},
  {"id": "ing-def", "name": "Lime", "category": "citrus", "unit": "piece"}
]

$ mixology ingredients get --id ing-abc --json
{"id": "ing-abc", "name": "Vodka", "category": "spirit", "unit": "oz"}
```

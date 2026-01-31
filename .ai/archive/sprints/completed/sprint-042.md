# Sprint 042: Standardize JSON Input for CLI Create/Update Commands

## Goal

Add `--stdin`, `--file`, and `--template` flag support to all CLI create/update commands, using the view models from Sprint 041 as the JSON contract.

## Problem

Currently only `drinks create` and `drinks update` support JSON input:

```bash
# Drinks - has JSON support
mixology drinks create --stdin < drink.json
mixology drinks create --file drink.json
mixology drinks create --template  # prints template

# Ingredients - flags only
mixology ingredients create "Vodka" --category spirit --unit oz

# Menu - positional only
mixology menu create "Summer Menu"

# Orders - positional only
mixology order place --menu-id mnu-xxx drk-xxx:2 drk-yyy:1

# Inventory - flags only
mixology inventory set --ingredient-id ing-xxx --quantity 10 --cost-per-unit '$5.00'
```

This inconsistency makes scripting harder and prevents bulk operations.

## Solution

Add JSON input support to all create/update commands using the same pattern as drinks:

```bash
# All commands support the same flags
mixology ingredients create --stdin < ingredient.json
mixology ingredients create --file ingredient.json
mixology ingredients create --template

mixology menu create --stdin < menu.json
mixology order place --stdin < order.json
mixology inventory set --stdin < stock.json
```

### Reuse View Models

The view models from Sprint 041 serve double duty:
1. **Output**: Table/detail display and `--json` flag output
2. **Input**: `--template` output and `--stdin`/`--file` input contract

```go
// app/domains/ingredients/surfaces/cli/views.go

type IngredientRow struct {
    ID       string `table:"ID"       json:"id"`
    Name     string `table:"NAME"     json:"name"`
    Category string `table:"CATEGORY" json:"category"`
    Unit     string `table:"UNIT"     json:"unit"`
    Desc     string `table:"-"        json:"description,omitempty"`
}

// TemplateCreate returns a template for JSON input (ID omitted)
func TemplateCreate() IngredientRow {
    return IngredientRow{
        Name:     "Vodka",
        Category: "spirit",
        Unit:     "oz",
        Desc:     "Neutral grain spirit",
    }
}

// DecodeCreate parses JSON input into domain model
func DecodeCreate(r io.Reader) (*models.Ingredient, error) {
    var row IngredientRow
    if err := json.NewDecoder(r).Decode(&row); err != nil {
        return nil, err
    }
    return &models.Ingredient{
        Name:        row.Name,
        Category:    models.Category(row.Category),
        Unit:        measurement.Unit(row.Unit),
        Description: row.Desc,
    }, nil
}
```

## Commands to Update

| Command | Current Input | JSON Input |
|---------|--------------|------------|
| `ingredients create` | `name` + flags | `IngredientRow` |
| `ingredients update` | `--id` + flags | `IngredientRow` |
| `menu create` | `name` positional | `MenuRow` (name only) |
| `order place` | `--menu-id` + `id:qty` args | `OrderInput` |
| `inventory set` | flags | `InventoryRow` |
| `inventory adjust` | flags | `InventoryPatch` |

Note: `drinks create/update` already has this support.

## JSON Templates

### Ingredient

```json
{
  "name": "Vodka",
  "category": "spirit",
  "unit": "oz",
  "description": "Neutral grain spirit"
}
```

### Menu

```json
{
  "name": "Summer Cocktails",
  "description": "Refreshing drinks for warm weather"
}
```

### Order

```json
{
  "menu_id": "mnu-xxx",
  "items": [
    { "drink_id": "drk-xxx", "quantity": 2 },
    { "drink_id": "drk-yyy", "quantity": 1 }
  ]
}
```

### Inventory Set

```json
{
  "ingredient_id": "ing-xxx",
  "quantity": 25.0,
  "unit": "oz",
  "cost_per_unit": "$28.00"
}
```

### Inventory Adjust

```json
{
  "ingredient_id": "ing-xxx",
  "delta": 5.0,
  "reason": "received",
  "cost_per_unit": "$28.00"
}
```

## Implementation

### Shared Input Reader

Extract the input reading logic from `drinks.go` into a helper:

```go
// main/cli/helpers.go

func readJSONInput[T any](cmd *cli.Command) (T, error) {
    var zero T
    fromStdin := cmd.Bool("stdin")
    fromFile := strings.TrimSpace(cmd.String("file"))

    if fromStdin && fromFile != "" {
        return zero, fmt.Errorf("set only one of --stdin or --file")
    }
    if !fromStdin && fromFile == "" {
        return zero, fmt.Errorf("missing input: set --stdin or --file (or use --template)")
    }

    var r io.Reader
    if fromStdin {
        b, err := io.ReadAll(os.Stdin)
        if err != nil {
            return zero, err
        }
        if len(bytes.TrimSpace(b)) == 0 {
            return zero, fmt.Errorf("stdin is empty")
        }
        r = bytes.NewReader(b)
    } else {
        f, err := os.Open(fromFile)
        if err != nil {
            return zero, err
        }
        defer f.Close()
        r = f
    }

    var result T
    if err := json.NewDecoder(r).Decode(&result); err != nil {
        return zero, err
    }
    return result, nil
}
```

### Updated Command Pattern

```go
// Before (ingredients create)
{
    Name:  "create",
    Arguments: []cli.Argument{
        &cli.StringArgs{Name: "name", Min: 1, Max: 1},
    },
    Flags: []cli.Flag{
        JSONFlag,
        &cli.StringFlag{Name: "category", Required: true},
        &cli.StringFlag{Name: "unit", Required: true},
        &cli.StringFlag{Name: "description"},
    },
    Action: func(...) {
        name := cmd.StringArgs("name")[0]
        // ... build from flags
    },
}

// After
{
    Name:  "create",
    Arguments: []cli.Argument{
        &cli.StringArgs{Name: "name", Max: 1},  // Optional now
    },
    Flags: []cli.Flag{
        JSONFlag,
        TemplateFlag,
        StdinFlag,
        FileFlag,
        &cli.StringFlag{Name: "category"},  // Not required (JSON may provide)
        &cli.StringFlag{Name: "unit"},
        &cli.StringFlag{Name: "description"},
    },
    Action: func(...) {
        if cmd.Bool("template") {
            return writeJSON(cmd.Writer, ingredientscli.TemplateCreate())
        }

        var input *models.Ingredient
        if cmd.Bool("stdin") || cmd.String("file") != "" {
            input, err = ingredientscli.DecodeCreate(...)
        } else {
            // Fall back to flags/args
            name := cmd.StringArgs("name")
            if len(name) == 0 {
                return fmt.Errorf("name required (or use --stdin/--file)")
            }
            input = &models.Ingredient{
                Name:     name[0],
                Category: models.Category(cmd.String("category")),
                // ...
            }
        }
        // ... create
    },
}
```

## Tasks

### Phase 1: Shared Helpers

- [x] Extract `readJSONInput[T]` helper to `main/cli/helpers.go`
- [x] Refactor `drinks create/update` to use shared helper

### Phase 2: View Model Input Support

- [x] Add `TemplateCreate()` and `DecodeCreate()` to `ingredientscli`
- [x] Add `TemplateCreate()` and `DecodeCreate()` to `menucli`
- [x] Add `OrderInput` type with `TemplatePlace()` and `DecodePlace()` to `orderscli`
- [x] Add `InventoryInput` type with `TemplateSet()` and `DecodeSet()` to `inventorycli`
- [x] Add `InventoryPatch` type with `TemplateAdjust()` and `DecodeAdjust()` to `inventorycli`

### Phase 3: Update CLI Commands

- [x] Update `ingredients create` to support `--stdin`/`--file`/`--template`
- [x] Update `ingredients update` to support `--stdin`/`--file`/`--template`
- [x] Update `menu create` to support `--stdin`/`--file`/`--template`
- [x] Update `order place` to support `--stdin`/`--file`/`--template`
- [x] Update `inventory set` to support `--stdin`/`--file`/`--template`
- [x] Update `inventory adjust` to support `--stdin`/`--file`/`--template`

### Phase 4: Update Seed Script

- [x] Update `scripts/seed.sh` to use JSON input where beneficial
- [x] Verify seed script still works

### Phase 5: Verify

- [x] All commands accept `--template` and output valid JSON
- [x] All commands accept `--stdin` with template output
- [x] All commands accept `--file` with saved template
- [x] Flag-based input still works (backwards compatible)
- [x] Run `go test ./...`

## Acceptance Criteria

- [x] All create/update commands support `--stdin`, `--file`, `--template`
- [x] `--template` output can be piped to `--stdin` (round-trip works)
- [x] View models from Sprint 041 serve as JSON contract
- [x] Flag-based input remains supported (not breaking change)
- [x] Seed script updated to demonstrate JSON input
- [x] All tests pass

## Notes

### Backwards Compatibility

Flag-based input continues to work. JSON input is additive:

```bash
# Both still work
mixology ingredients create "Vodka" --category spirit --unit oz
mixology ingredients create --stdin < vodka.json
```

### Validation

JSON input goes through the same validation as flag input - the `DecodeCreate()` functions parse into domain models which have their own validation.

### Dependency on Sprint 041

This sprint depends on Sprint 041 for the view model types. If implementing before 041, create minimal input types without `table` tags.

# Sprint 034: Code Generation for CLI Commands

## Goal

Generate consistent CRUD CLI commands from configuration, eliminating boilerplate and ensuring uniformity across all domain modules.

## Problem

Each domain has similar CLI patterns with subtle inconsistencies:

```go
// ingredients.go - list with tabwriter
w := newTabWriter()
fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tUNIT")
for _, i := range res {
    fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", i.ID.String(), i.Name, i.Category, i.Unit)
}

// drinks.go - list with tabwriter (different columns)
w := newTabWriter()
fmt.Fprintln(w, "ID\tNAME")
for _, d := range res {
    fmt.Fprintf(w, "%s\t%s\n", d.ID.String(), d.Name)
}

// menu.go - list without tabwriter header (inconsistent!)
for _, m := range res {
    fmt.Printf("%s\t%s\t%s\t%d\n", m.ID.String(), m.Name, m.Status, len(m.Items))
}
```

Inconsistencies include:
- Some use tabwriter, some don't
- Some have headers, some don't
- JSON flag support varies
- Detail output formatting varies
- Error handling patterns vary

## Analysis: What Can Be Generated?

### Fully Generatable (strict pattern)

| Command | Pattern |
|---------|---------|
| `list` | Fetch all → tabwriter table with header |
| `get` | Parse ID → fetch → tabwriter key-value detail |
| `delete` | Parse ID → delete → confirmation message |

### Semi-Generatable (with hooks)

| Command | Variation |
|---------|-----------|
| `create` (simple) | Flag-based input (ingredients) |
| `create` (complex) | JSON input with template (drinks) |
| `update` (simple) | Flag-based input (ingredients) |
| `update` (complex) | JSON input (drinks) |

### Not Generatable (domain-specific)

| Domain | Custom Commands |
|--------|-----------------|
| Menu | `add-drink`, `remove-drink`, `publish` |
| Orders | `place`, `complete`, `cancel` |
| Inventory | `adjust`, `set` (keyed by ingredient, not PK) |
| Audit | `history`, `actor` (query variations) |

## Solution

### Configuration Structure

```go
// main/cli/gen/config.go

package gen

type DomainDef struct {
    // Identity
    Name       string // "Drink", "Ingredient"
    Plural     string // "Drinks", "Ingredients"
    Command    string // "drink", "ingredients"

    // Entity
    EntityType string // "DrinkID", "IngredientID"

    // Module access
    ModuleField string // "Drinks", "Ingredients"

    // List command
    List *ListDef

    // Get command
    Get *GetDef

    // Delete command
    Delete *DeleteDef

    // Create command (nil = not generated)
    Create *CreateDef

    // Update command (nil = not generated)
    Update *UpdateDef
}

type ListDef struct {
    Columns []ColumnDef
    Filters []FilterDef
    HasJSON bool
}

type ColumnDef struct {
    Header string // "ID", "NAME"
    Field  string // "ID.String()", "Name"
    Format string // "%s", "%.2f"
}

type FilterDef struct {
    Name      string // "category"
    Aliases   []string
    Usage     string
    Type      string // "string", "int", "bool"
    Validator string // "ingredientscli.ValidateCategory" or ""
}

type GetDef struct {
    Fields  []FieldDef
    HasJSON bool
}

type FieldDef struct {
    Label     string // "ID:"
    Field     string // "ID.String()"
    Condition string // "" or "Description != \"\""
}

type DeleteDef struct {
    HasJSON       bool
    Confirmation  string // "deleted %s\t%s" with ID.String(), Name
}

type CreateDef struct {
    Mode string // "flags" or "json"
    // For "flags" mode:
    Arguments []ArgumentDef
    Flags     []FlagDef
    // For "json" mode:
    TemplateFn string // "drinkscli.TemplateCreateDrink"
    DecodeFn   string // "drinkscli.DecodeCreateDrinkJSON"
}

type UpdateDef struct {
    Mode string // "flags" or "json"
    // Similar to CreateDef
}
```

### Domain Definitions

```go
// main/cli/gen/domains.go

var Domains = []DomainDef{
    {
        Name:        "Ingredient",
        Plural:      "Ingredients",
        Command:     "ingredients",
        EntityType:  "IngredientID",
        ModuleField: "Ingredients",
        List: &ListDef{
            HasJSON: false,
            Columns: []ColumnDef{
                {Header: "ID", Field: "ID.String()", Format: "%s"},
                {Header: "NAME", Field: "Name", Format: "%s"},
                {Header: "CATEGORY", Field: "Category", Format: "%s"},
                {Header: "UNIT", Field: "Unit", Format: "%s"},
            },
            Filters: []FilterDef{
                {Name: "category", Aliases: []string{"c"}, Usage: "ingredientscli.CategoryUsage()", Validator: "ingredientscli.ValidateCategory"},
            },
        },
        Get: &GetDef{
            HasJSON: false,
            Fields: []FieldDef{
                {Label: "ID:", Field: "ID.String()"},
                {Label: "Name:", Field: "Name"},
                {Label: "Category:", Field: "Category"},
                {Label: "Unit:", Field: "Unit"},
                {Label: "Description:", Field: "Description", Condition: "Description != \"\""},
            },
        },
        Delete: &DeleteDef{
            HasJSON:      false,
            Confirmation: "deleted %s\t%s",
        },
        Create: &CreateDef{Mode: "flags"}, // Hand-written due to flag complexity
        Update: &UpdateDef{Mode: "flags"}, // Hand-written due to flag complexity
    },
    {
        Name:        "Drink",
        Plural:      "Drinks",
        Command:     "drinks",
        EntityType:  "DrinkID",
        ModuleField: "Drinks",
        List: &ListDef{
            HasJSON: true,
            Columns: []ColumnDef{
                {Header: "ID", Field: "ID.String()", Format: "%s"},
                {Header: "NAME", Field: "Name", Format: "%s"},
            },
            Filters: []FilterDef{
                {Name: "name", Usage: "Filter by exact name match"},
                {Name: "category", Aliases: []string{"c"}, Usage: "Filter by category", Validator: "drinksmodels.DrinkCategory(s).Validate"},
                {Name: "glass", Aliases: []string{"g"}, Usage: "Filter by glass", Validator: "drinksmodels.GlassType(s).Validate"},
            },
        },
        Get: &GetDef{
            HasJSON: true,
            Fields: []FieldDef{
                {Label: "ID:", Field: "ID.String()"},
                {Label: "Name:", Field: "Name"},
            },
        },
        Delete: &DeleteDef{
            HasJSON:      true,
            Confirmation: "deleted %s\t%s",
        },
        Create: &CreateDef{
            Mode:       "json",
            TemplateFn: "drinkscli.TemplateCreateDrink",
            DecodeFn:   "drinkscli.DecodeCreateDrinkJSON",
        },
        Update: &UpdateDef{
            Mode:       "json",
            TemplateFn: "drinkscli.TemplateUpdateDrink",
            DecodeFn:   "drinkscli.DecodeUpdateDrinkJSON",
        },
    },
    {
        Name:        "Menu",
        Plural:      "Menus",
        Command:     "menu",
        EntityType:  "MenuID",
        ModuleField: "Menu",
        List: &ListDef{
            HasJSON: true,
            Columns: []ColumnDef{
                {Header: "ID", Field: "ID.String()", Format: "%s"},
                {Header: "NAME", Field: "Name", Format: "%s"},
                {Header: "STATUS", Field: "Status", Format: "%s"},
                {Header: "ITEMS", Field: "len(Items)", Format: "%d"},
            },
        },
        Get: &GetDef{
            HasJSON: true,
            Fields: []FieldDef{
                {Label: "ID:", Field: "ID.String()"},
                {Label: "Name:", Field: "Name"},
                {Label: "Description:", Field: "Description", Condition: "Description != \"\""},
                {Label: "Status:", Field: "Status"},
            },
        },
        Delete: nil, // Menu uses archive, not delete
        Create: nil, // Hand-written (simple but uses StringArgs)
        Update: nil, // No update command for menu
    },
    {
        Name:        "Order",
        Plural:      "Orders",
        Command:     "order",
        EntityType:  "OrderID",
        ModuleField: "Orders",
        List: &ListDef{
            HasJSON: false,
            Columns: []ColumnDef{
                {Header: "ID", Field: "ID.String()", Format: "%s"},
                {Header: "MENU", Field: "MenuID.String()", Format: "%s"},
                {Header: "STATUS", Field: "Status", Format: "%s"},
                {Header: "CREATED", Field: "CreatedAt.Format(time.RFC3339)", Format: "%s"},
            },
            Filters: []FilterDef{
                {Name: "status", Usage: "Filter by status", Validator: "ordersmodels.OrderStatus(s).Validate"},
            },
        },
        Get: &GetDef{
            HasJSON: false,
            Fields: []FieldDef{
                {Label: "ID:", Field: "ID.String()"},
                {Label: "MenuID:", Field: "MenuID.String()"},
                {Label: "Status:", Field: "Status"},
                {Label: "CreatedAt:", Field: "CreatedAt.Format(time.RFC3339)"},
            },
        },
        Delete: nil, // Orders use cancel, not delete
        Create: nil, // Order uses "place" command
        Update: nil, // Orders use complete/cancel
    },
}
```

### Generated Output Structure

```
main/cli/
├── gen/
│   ├── config.go      # DomainDef types
│   ├── domains.go     # Domains slice (source of truth)
│   └── main.go        # Generator
├── commands_gen.go    # Generated CRUD commands
├── drinks.go          # Custom commands only (create, update with JSON)
├── menu.go            # Custom commands only (add-drink, remove-drink, publish)
├── orders.go          # Custom commands only (place, complete, cancel)
├── inventory.go       # All hand-written (not CRUD)
├── audit.go           # All hand-written (not CRUD)
└── helpers.go         # Shared helpers
```

### Generated Code Example

```go
// main/cli/commands_gen.go
// Code generated by go generate; DO NOT EDIT.

package main

import (
    "fmt"
    "time"

    "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
    drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
    drinkscli "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/cli"
    "github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
    ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
    ingredientscli "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/cli"
    "github.com/TheFellow/go-modular-monolith/app/kernel/entity"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
    "github.com/urfave/cli/v3"
)

// =============================================================================
// Ingredients Commands
// =============================================================================

func (c *CLI) ingredientsListCmd() *cli.Command {
    return &cli.Command{
        Name:  "list",
        Usage: "List ingredients",
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:    "category",
                Aliases: []string{"c"},
                Usage:   ingredientscli.CategoryUsage(),
                Validator: func(s string) error {
                    return ingredientscli.ValidateCategory(s)
                },
            },
        },
        Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
            res, err := c.app.Ingredients.List(ctx, ingredients.ListRequest{
                Category: ingredientsmodels.Category(cmd.String("category")),
            })
            if err != nil {
                return err
            }

            w := newTabWriter()
            fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tUNIT")
            for _, item := range res {
                fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
                    item.ID.String(),
                    item.Name,
                    item.Category,
                    item.Unit,
                )
            }
            return w.Flush()
        }),
    }
}

func (c *CLI) ingredientsGetCmd() *cli.Command {
    return &cli.Command{
        Name:  "get",
        Usage: "Get an ingredient by ID",
        Flags: []cli.Flag{
            &cli.StringFlag{Name: "id", Usage: "Ingredient ID", Required: true},
        },
        Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
            id, err := entity.ParseIngredientID(cmd.String("id"))
            if err != nil {
                return err
            }
            res, err := c.app.Ingredients.Get(ctx, id)
            if err != nil {
                return err
            }

            w := newTabWriter()
            fmt.Fprintf(w, "ID:\t%s\n", res.ID.String())
            fmt.Fprintf(w, "Name:\t%s\n", res.Name)
            fmt.Fprintf(w, "Category:\t%s\n", res.Category)
            fmt.Fprintf(w, "Unit:\t%s\n", res.Unit)
            if res.Description != "" {
                fmt.Fprintf(w, "Description:\t%s\n", res.Description)
            }
            return w.Flush()
        }),
    }
}

func (c *CLI) ingredientsDeleteCmd() *cli.Command {
    return &cli.Command{
        Name:  "delete",
        Usage: "Delete an ingredient by ID",
        Flags: []cli.Flag{
            &cli.StringFlag{Name: "id", Usage: "Ingredient ID", Required: true},
        },
        Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
            id, err := entity.ParseIngredientID(cmd.String("id"))
            if err != nil {
                return err
            }
            res, err := c.app.Ingredients.Delete(ctx, id)
            if err != nil {
                return err
            }

            fmt.Printf("deleted %s\t%s\n", res.ID.String(), res.Name)
            return nil
        }),
    }
}

// =============================================================================
// Drinks Commands
// =============================================================================

func (c *CLI) drinksListCmd() *cli.Command {
    return &cli.Command{
        Name:  "list",
        Usage: "List drinks",
        Flags: []cli.Flag{
            JSONFlag,
            &cli.StringFlag{Name: "name", Usage: "Filter by exact name match"},
            &cli.StringFlag{
                Name:    "category",
                Aliases: []string{"c"},
                Usage:   "Filter by category",
                Validator: func(s string) error {
                    return drinksmodels.DrinkCategory(s).Validate()
                },
            },
            &cli.StringFlag{
                Name:    "glass",
                Aliases: []string{"g"},
                Usage:   "Filter by glass",
                Validator: func(s string) error {
                    return drinksmodels.GlassType(s).Validate()
                },
            },
        },
        Action: c.action(func(ctx *middleware.Context, cmd *cli.Command) error {
            res, err := c.app.Drinks.List(ctx, drinks.ListRequest{
                Name:     cmd.String("name"),
                Category: drinksmodels.DrinkCategory(cmd.String("category")),
                Glass:    drinksmodels.GlassType(cmd.String("glass")),
            })
            if err != nil {
                return err
            }

            if cmd.Bool("json") {
                out := make([]drinkscli.Drink, 0, len(res))
                for _, d := range res {
                    out = append(out, drinkscli.FromDomainDrink(*d))
                }
                return writeJSON(cmd.Writer, out)
            }

            w := newTabWriter()
            fmt.Fprintln(w, "ID\tNAME")
            for _, item := range res {
                fmt.Fprintf(w, "%s\t%s\n",
                    item.ID.String(),
                    item.Name,
                )
            }
            return w.Flush()
        }),
    }
}

// ... similar for drinksGetCmd, drinksDeleteCmd ...

// =============================================================================
// Command Registration
// =============================================================================

func (c *CLI) ingredientsCommands() *cli.Command {
    return &cli.Command{
        Name:  "ingredients",
        Usage: "Manage ingredients",
        Commands: []*cli.Command{
            c.ingredientsListCmd(),
            c.ingredientsGetCmd(),
            c.ingredientsCreateCmd(),  // Hand-written
            c.ingredientsUpdateCmd(),  // Hand-written
            c.ingredientsDeleteCmd(),
        },
    }
}
```

### Hand-Written Commands

Commands that don't fit the CRUD pattern remain hand-written but follow conventions:

```go
// main/cli/menu_custom.go

package main

// Hand-written commands for Menu domain
// Generated commands are in commands_gen.go

func (c *CLI) menuAddDrinkCmd() *cli.Command {
    // Custom implementation
}

func (c *CLI) menuRemoveDrinkCmd() *cli.Command {
    // Custom implementation
}

func (c *CLI) menuPublishCmd() *cli.Command {
    // Custom implementation
}
```

## Alternative: Simpler Approach

If full generation is too complex, a simpler approach uses helper functions:

```go
// main/cli/crud.go

// ListCmd creates a standard list command
func ListCmd[T any](
    name string,
    usage string,
    flags []cli.Flag,
    fetch func(*middleware.Context, *cli.Command) ([]T, error),
    header string,
    format func(T) []any,
) *cli.Command {
    return &cli.Command{
        Name:  name,
        Usage: usage,
        Flags: flags,
        Action: func(ctx context.Context, cmd *cli.Command) error {
            items, err := fetch(appCtx(ctx), cmd)
            if err != nil {
                return err
            }
            w := newTabWriter()
            fmt.Fprintln(w, header)
            for _, item := range items {
                fmt.Fprintf(w, format(item)...)
            }
            return w.Flush()
        },
    }
}

// Usage:
c.ingredientsListCmd = ListCmd(
    "list", "List ingredients",
    []cli.Flag{categoryFlag},
    func(ctx *middleware.Context, cmd *cli.Command) ([]*models.Ingredient, error) {
        return c.app.Ingredients.List(ctx, ingredients.ListRequest{...})
    },
    "ID\tNAME\tCATEGORY\tUNIT",
    func(i *models.Ingredient) []any {
        return []any{i.ID.String(), i.Name, i.Category, i.Unit}
    },
)
```

This approach is less magical but still reduces boilerplate.

## Recommendation

Start with the **simpler helper function approach** first, then evaluate if full code generation is worth the complexity. The helpers provide:
- Consistent patterns
- Reduced boilerplate
- Type safety
- Easy to understand and modify

Full code generation adds:
- Single source of truth
- Zero chance of pattern drift
- But: more complex to modify, harder to debug

## Tasks

### Phase 1: Define Patterns

- [ ] Document the canonical CRUD patterns
- [ ] Identify all variations (JSON support, filters, etc.)
- [ ] Decide: full generation vs. helper functions

### Phase 2: Implement Helpers/Generator

**If helper functions:**
- [ ] Create `ListCmd`, `GetCmd`, `DeleteCmd` helpers
- [ ] Refactor existing commands to use helpers
- [ ] Verify consistency

**If code generation:**
- [ ] Create `gen/config.go` with type definitions
- [ ] Create `gen/domains.go` with domain definitions
- [ ] Create `gen/main.go` with generator
- [ ] Add `//go:generate` directive
- [ ] Generate `commands_gen.go`

### Phase 3: Migrate Commands

- [ ] Migrate ingredients list/get/delete
- [ ] Migrate drinks list/get/delete
- [ ] Migrate menu list/get
- [ ] Migrate orders list/get
- [ ] Keep custom commands hand-written

### Phase 4: Verify

- [ ] Run all CLI commands
- [ ] Verify consistent output formatting
- [ ] Verify JSON output where supported
- [ ] Verify `go generate` is idempotent (if using generation)

## Acceptance Criteria

- [ ] All list commands use tabwriter with header row
- [ ] All get commands use tabwriter with key-value format
- [ ] All delete commands print consistent confirmation
- [ ] JSON flag support is consistent where applicable
- [ ] ID parsing uses `entity.Parse*` consistently
- [ ] Custom commands follow documented patterns
- [ ] Adding new CRUD commands is trivial

## Result

Adding a new domain's CRUD commands:

**With generation:**
```go
// Add to gen/domains.go
{
    Name:        "Widget",
    Plural:      "Widgets",
    Command:     "widgets",
    EntityType:  "WidgetID",
    ModuleField: "Widgets",
    List: &ListDef{...},
    Get: &GetDef{...},
    Delete: &DeleteDef{...},
}

// Run: go generate ./main/cli
// Done - list, get, delete commands generated
```

**With helpers:**
```go
// main/cli/widgets.go
func (c *CLI) widgetsCommands() *cli.Command {
    return &cli.Command{
        Name:  "widgets",
        Usage: "Manage widgets",
        Commands: []*cli.Command{
            ListCmd("list", "List widgets", nil,
                func(ctx *middleware.Context, cmd *cli.Command) ([]*models.Widget, error) {
                    return c.app.Widgets.List(ctx, widgets.ListRequest{})
                },
                "ID\tNAME",
                func(w *models.Widget) string { return fmt.Sprintf("%s\t%s\n", w.ID, w.Name) },
            ),
            GetCmd("get", "Get a widget", entity.ParseWidgetID,
                c.app.Widgets.Get,
                func(w *models.Widget) []FieldDef {
                    return []FieldDef{
                        {Label: "ID:", Value: w.ID.String()},
                        {Label: "Name:", Value: w.Name},
                    }
                },
            ),
            DeleteCmd("delete", "Delete a widget", entity.ParseWidgetID,
                c.app.Widgets.Delete,
                func(w *models.Widget) string { return fmt.Sprintf("deleted %s\t%s", w.ID, w.Name) },
            ),
        },
    }
}
```

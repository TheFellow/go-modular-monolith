# Sprint 033h: Tabular CLI Output with tabwriter

## Goal

Use Go's `text/tabwriter` package to produce well-aligned tabular output in the CLI.

## Problem

Current CLI output uses manual tab characters for column alignment:

```go
fmt.Printf("%s\t%s\t%s\t%s\n", id, name, category, unit)
```

This produces misaligned output when column values vary in length:

```
ing-abc123	Vodka	Spirit	ml
ing-def456789	Simple Syrup	Sweetener	ml
ing-gh	Lime Juice	Citrus	ml
```

## Solution

Use `text/tabwriter` to automatically align columns:

```go
w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "ID", "NAME", "CATEGORY", "UNIT")
for _, ing := range ingredients {
    fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ing.ID, ing.Name, ing.Category, ing.Unit)
}
w.Flush()
```

Produces:

```
ID               NAME          CATEGORY   UNIT
ing-abc123       Vodka         Spirit     ml
ing-def456789    Simple Syrup  Sweetener  ml
ing-gh           Lime Juice    Citrus     ml
```

## Implementation

### Helper Function

```go
// main/cli/helpers.go

import "text/tabwriter"

// newTabWriter creates a tabwriter for aligned columnar output.
// Uses 2-space minimum padding between columns.
func newTabWriter() *tabwriter.Writer {
    return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}
```

### Pattern: List Commands

**Before:**
```go
func ingredientList(ctx context.Context, cmd *cli.Command) error {
    // ...
    for _, ing := range ingredients {
        fmt.Printf("%s\t%s\t%s\t%s\n",
            ing.ID.String(), ing.Name, ing.Category, ing.Unit)
    }
    return nil
}
```

**After:**
```go
func ingredientList(ctx context.Context, cmd *cli.Command) error {
    // ...
    w := newTabWriter()
    fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tUNIT")
    for _, ing := range ingredients {
        fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
            ing.ID.String(), ing.Name, ing.Category, ing.Unit)
    }
    return w.Flush()
}
```

### Pattern: Detail Commands

Key-value detail output also benefits from alignment:

**Before:**
```go
fmt.Printf("ID:       %s\n", ing.ID.String())
fmt.Printf("Name:     %s\n", ing.Name)
fmt.Printf("Category: %s\n", ing.Category)
```

**After:**
```go
w := newTabWriter()
fmt.Fprintf(w, "ID:\t%s\n", ing.ID.String())
fmt.Fprintf(w, "Name:\t%s\n", ing.Name)
fmt.Fprintf(w, "Category:\t%s\n", ing.Category)
w.Flush()
```

### Files to Update

| File | Commands | Output Type |
|------|----------|-------------|
| `ingredients.go` | list, get | List (4 cols), Detail |
| `drinks.go` | list, get | List (2 cols), Detail |
| `menu.go` | list, show | List (4 cols), Detail with analytics |
| `orders.go` | list, get | List (4 cols), Detail |
| `inventory.go` | list | List (3 cols) |
| `audit.go` | list, get | List (7+ cols), Detail |

### Example: ingredients.go

```go
func ingredientList(ctx context.Context, cmd *cli.Command) error {
    ingredients, err := app.Ingredients.List(appCtx(ctx))
    if err != nil {
        return err
    }

    if cmd.Bool("json") {
        return writeJSON(os.Stdout, ingredients)
    }

    w := newTabWriter()
    fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tUNIT")
    for _, ing := range ingredients {
        fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
            ing.ID.String(), ing.Name, ing.Category, ing.Unit)
    }
    return w.Flush()
}

func ingredientGet(ctx context.Context, cmd *cli.Command) error {
    id, err := entity.ParseIngredientID(cmd.String("id"))
    if err != nil {
        return err
    }

    ing, err := app.Ingredients.Get(appCtx(ctx), id)
    if err != nil {
        return err
    }

    if cmd.Bool("json") {
        return writeJSON(os.Stdout, ing)
    }

    w := newTabWriter()
    fmt.Fprintf(w, "ID:\t%s\n", ing.ID.String())
    fmt.Fprintf(w, "Name:\t%s\n", ing.Name)
    fmt.Fprintf(w, "Category:\t%s\n", ing.Category)
    fmt.Fprintf(w, "Unit:\t%s\n", ing.Unit)
    fmt.Fprintf(w, "Cost:\t%s\n", ing.Cost.String())
    return w.Flush()
}
```

### Example: menu.go (with analytics)

```go
func menuShow(ctx context.Context, cmd *cli.Command) error {
    // ... fetch menu with analytics ...

    if cmd.Bool("json") {
        return writeJSON(os.Stdout, menu)
    }

    // Header
    w := newTabWriter()
    fmt.Fprintf(w, "ID:\t%s\n", menu.ID.String())
    fmt.Fprintf(w, "Name:\t%s\n", menu.Name)
    fmt.Fprintf(w, "Status:\t%s\n", menu.Status)
    fmt.Fprintf(w, "Items:\t%d\n", len(menu.Items))
    w.Flush()

    // Items table
    if len(menu.Items) > 0 {
        fmt.Println()
        w = newTabWriter()
        if cmd.Bool("costs") {
            fmt.Fprintln(w, "DRINK\tNAME\tCOST\tPRICE\tMARGIN")
            for _, item := range menu.Items {
                cost := "n/a"
                margin := "n/a"
                if item.Cost != nil && !item.CostUnknown {
                    cost = item.Cost.String()
                    margin = fmt.Sprintf("%.1f%%", item.Margin*100)
                }
                fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
                    item.DrinkID.String(), item.Name, cost, item.Price.String(), margin)
            }
        } else {
            fmt.Fprintln(w, "DRINK\tNAME\tPRICE")
            for _, item := range menu.Items {
                fmt.Fprintf(w, "%s\t%s\t%s\n",
                    item.DrinkID.String(), item.Name, item.Price.String())
            }
        }
        w.Flush()
    }
    return nil
}
```

### Example: audit.go

```go
func auditList(ctx context.Context, cmd *cli.Command) error {
    entries, err := app.Audit.Query(appCtx(ctx), query)
    if err != nil {
        return err
    }

    if cmd.Bool("json") {
        return writeJSON(os.Stdout, entries)
    }

    w := newTabWriter()
    fmt.Fprintln(w, "ID\tTIME\tACTION\tRESOURCE\tOK\tDURATION\tERROR")
    for _, e := range entries {
        errStr := ""
        if e.Error != "" {
            errStr = e.Error
        }
        fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%s\t%s\n",
            e.ID.String(),
            e.StartedAt.Format(time.RFC3339),
            e.Action,
            e.Resource.String(),
            e.Authorized,
            e.Duration.String(),
            errStr,
        )
    }
    return w.Flush()
}
```

## Tasks

### Phase 1: Add Helper

- [x] Add `newTabWriter()` helper to `main/cli/helpers.go`
- [x] Import `text/tabwriter` and `os`

### Phase 2: Update List Commands

- [x] Update `ingredients.go` list command
- [x] Update `drinks.go` list command
- [x] Update `menu.go` list command
- [x] Update `orders.go` list command
- [x] Update `inventory.go` list command
- [x] Update `audit.go` list command

### Phase 3: Update Detail Commands

- [x] Update `ingredients.go` get command
- [x] Update `drinks.go` get command
- [x] Update `menu.go` show command (with analytics)
- [x] Update `orders.go` get command
- [x] Update `audit.go` get command

### Phase 4: Add Column Headers

- [x] Add header row to all list outputs
- [x] Use UPPERCASE for header labels (convention)

### Phase 5: Verify

- [x] Run CLI commands and verify alignment
- [x] Test with varying data lengths
- [x] Ensure JSON output still works

## Acceptance Criteria

- [x] All list commands use tabwriter for aligned output
- [x] All detail commands use tabwriter for key-value alignment
- [x] List outputs include header row
- [x] JSON output unchanged (--json flag)
- [x] Columns properly aligned regardless of data length
- [x] `newTabWriter()` helper used consistently

## Result

**Before:**
```
ing-abc123	Vodka	Spirit	ml
ing-def456789	Simple Syrup	Sweetener	ml
ing-gh	Lime Juice	Citrus	ml
```

**After:**
```
ID               NAME          CATEGORY   UNIT
ing-abc123       Vodka         Spirit     ml
ing-def456789    Simple Syrup  Sweetener  ml
ing-gh           Lime Juice    Citrus     ml
```

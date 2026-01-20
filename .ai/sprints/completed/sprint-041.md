# Sprint 041: Reflection-Based TabWriter Helpers

## Goal

Replace repetitive TabWriter boilerplate with struct-tag-based helpers that use reflection to produce consistent tabular and key-value output.

## Problem

Every CLI list/get command follows the same pattern with minor variations:

### List Commands (Table Format)

```go
// ingredients list
w := newTabWriter()
fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tUNIT")
for _, i := range res {
    fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", i.ID.String(), i.Name, i.Category, i.Unit)
}
return w.Flush()

// orders list
w := newTabWriter()
fmt.Fprintln(w, "ID\tMENU_ID\tSTATUS\tCREATED_AT")
for _, o := range res {
    fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", o.ID.String(), o.MenuID.String(), o.Status, o.CreatedAt.Format(time.RFC3339))
}
return w.Flush()

// ... same pattern in drinks, menu, inventory, audit
```

### Get Commands (Key-Value Format)

```go
// ingredients get
w := newTabWriter()
fmt.Fprintf(w, "ID:\t%s\n", i.ID.String())
fmt.Fprintf(w, "Name:\t%s\n", i.Name)
fmt.Fprintf(w, "Category:\t%s\n", i.Category)
fmt.Fprintf(w, "Unit:\t%s\n", i.Unit)
if i.Description != "" {
    fmt.Fprintf(w, "Description:\t%s\n", i.Description)
}
return w.Flush()

// orders get
w := newTabWriter()
fmt.Fprintf(w, "ID:\t%s\n", o.ID.String())
fmt.Fprintf(w, "Menu ID:\t%s\n", o.MenuID.String())
fmt.Fprintf(w, "Status:\t%s\n", o.Status)
fmt.Fprintf(w, "Created At:\t%s\n", o.CreatedAt.Format(time.RFC3339))
if t, ok := o.CompletedAt.Unwrap(); ok {
    fmt.Fprintf(w, "Completed At:\t%s\n", t.Format(time.RFC3339))
}
// ...
```

**Issues:**

1. 6+ nearly identical code blocks for list commands
2. 5+ nearly identical code blocks for get commands
3. Manual header/field ordering must match
4. Inconsistent formatting (some use `.String()`, some cast, some format times differently)
5. Conditional field display requires manual if-statements

## Solution

### Struct Tags + Reflection

Define view models with struct tags that specify:
- Column/label names
- Display order (field order in struct)
- Conditional display (`omitempty`)

```go
// table:"HEADER" for list output
// json:"label" for get output (reuses standard tag)
// omitempty to skip zero values

type IngredientRow struct {
    ID       string `table:"ID"       json:"id"`
    Name     string `table:"NAME"     json:"name"`
    Category string `table:"CATEGORY" json:"category"`
    Unit     string `table:"UNIT"     json:"unit"`
    Desc     string `table:"-"        json:"description,omitempty"`
}
```

### Helper Functions

```go
// PrintTable prints a slice as a tabular list with headers
func PrintTable[T any](items []T) error

// PrintDetail prints a single item as key-value pairs
func PrintDetail[T any](item T) error
```

### Usage

```go
// Before (ingredients list)
w := newTabWriter()
fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tUNIT")
for _, i := range res {
    fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", i.ID.String(), i.Name, i.Category, i.Unit)
}
return w.Flush()

// After
rows := toIngredientRows(res)
return PrintTable(rows)
```

```go
// Before (ingredients get)
w := newTabWriter()
fmt.Fprintf(w, "ID:\t%s\n", i.ID.String())
fmt.Fprintf(w, "Name:\t%s\n", i.Name)
// ...
return w.Flush()

// After
return PrintDetail(toIngredientDetail(res))
```

## Design

### Package Location

- **Table helpers**: `main/cli/table/` - internal to CLI entry point
- **View models**: `app/domains/<domain>/surfaces/cli/` - each domain owns its CLI view models

This follows the existing pattern where each domain has a `surfaces/cli` package for CLI-specific types (e.g., `drinkscli.FromDomainDrink`).

### Tag Syntax

```
table:"HEADER"          // Include in table with this header
table:"-"               // Exclude from table output
json:"label"            // Used for detail output label (reuses standard json tag)
json:"label,omitempty"  // Exclude if zero value in detail output
json:"-"                // Exclude from detail output
```

This allows the same view model to serve both CLI table/detail output and JSON serialization.

### Stringer Interface

Types implementing `fmt.Stringer` are automatically converted via `.String()`.

This handles:
- Entity IDs (`IngredientID`, `DrinkID`, etc.)
- Cedar `EntityUID`
- Custom string types

### Time Formatting

`time.Time` fields are formatted as RFC3339 by default.

Optional: support `format:"2006-01-02"` tag for custom time formats.

### View Models

Each domain defines view models in its `surfaces/cli` package:

```go
// app/domains/ingredients/surfaces/cli/views.go

type IngredientRow struct {
    ID       string `table:"ID"       json:"id"`
    Name     string `table:"NAME"     json:"name"`
    Category string `table:"CATEGORY" json:"category"`
    Unit     string `table:"UNIT"     json:"unit"`
    Desc     string `table:"-"        json:"description,omitempty"`
}

func ToIngredientRow(i *models.Ingredient) IngredientRow {
    return IngredientRow{
        ID:       i.ID.String(),
        Name:     i.Name,
        Category: string(i.Category),
        Unit:     string(i.Unit),
        Desc:     i.Description,
    }
}

func ToIngredientRows(items []*models.Ingredient) []IngredientRow {
    rows := make([]IngredientRow, len(items))
    for i, item := range items {
        rows[i] = ToIngredientRow(item)
    }
    return rows
}
```

### Implementation

```go
// main/cli/table/table.go

package table

import (
    "fmt"
    "os"
    "reflect"
    "strings"
    "text/tabwriter"
    "time"
)

// PrintTable prints items as a table with headers from struct tags.
func PrintTable[T any](items []T) error {
    if len(items) == 0 {
        return nil
    }

    w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

    // Get field metadata from first item's type
    typ := reflect.TypeOf(items[0])
    fields := getTableFields(typ)

    // Print header
    headers := make([]string, len(fields))
    for i, f := range fields {
        headers[i] = f.header
    }
    fmt.Fprintln(w, strings.Join(headers, "\t"))

    // Print rows
    for _, item := range items {
        val := reflect.ValueOf(item)
        values := make([]string, len(fields))
        for i, f := range fields {
            values[i] = formatValue(val.Field(f.index))
        }
        fmt.Fprintln(w, strings.Join(values, "\t"))
    }

    return w.Flush()
}

// PrintDetail prints a single item as key-value pairs.
// Labels are derived from json tags, converting snake_case to Title Case.
func PrintDetail[T any](item T) error {
    w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

    typ := reflect.TypeOf(item)
    val := reflect.ValueOf(item)
    fields := getJSONFields(typ)

    for _, f := range fields {
        fieldVal := val.Field(f.index)
        if f.omitempty && isZero(fieldVal) {
            continue
        }
        fmt.Fprintf(w, "%s:\t%s\n", f.label, formatValue(fieldVal))
    }

    return w.Flush()
}

type tableField struct {
    index  int
    header string
}

type jsonField struct {
    index     int
    label     string
    omitempty bool
}

func getTableFields(typ reflect.Type) []tableField {
    var fields []tableField
    for i := 0; i < typ.NumField(); i++ {
        tag := typ.Field(i).Tag.Get("table")
        if tag == "" || tag == "-" {
            continue
        }
        fields = append(fields, tableField{index: i, header: tag})
    }
    return fields
}

func getJSONFields(typ reflect.Type) []jsonField {
    var fields []jsonField
    for i := 0; i < typ.NumField(); i++ {
        tag := typ.Field(i).Tag.Get("json")
        if tag == "" || tag == "-" {
            continue
        }
        parts := strings.Split(tag, ",")
        f := jsonField{index: i, label: toLabel(parts[0])}
        for _, opt := range parts[1:] {
            if opt == "omitempty" {
                f.omitempty = true
            }
        }
        fields = append(fields, f)
    }
    return fields
}

// toLabel converts snake_case to Title Case for display
func toLabel(s string) string {
    parts := strings.Split(s, "_")
    for i, p := range parts {
        if len(p) > 0 {
            parts[i] = strings.ToUpper(p[:1]) + p[1:]
        }
    }
    return strings.Join(parts, " ")
}

func formatValue(v reflect.Value) string {
    if !v.IsValid() {
        return ""
    }

    // Handle Stringer interface
    if stringer, ok := v.Interface().(fmt.Stringer); ok {
        return stringer.String()
    }

    // Handle time.Time
    if t, ok := v.Interface().(time.Time); ok {
        if t.IsZero() {
            return ""
        }
        return t.Format(time.RFC3339)
    }

    // Handle pointers
    if v.Kind() == reflect.Ptr {
        if v.IsNil() {
            return ""
        }
        return formatValue(v.Elem())
    }

    return fmt.Sprintf("%v", v.Interface())
}

func isZero(v reflect.Value) bool {
    return v.IsZero()
}
```

## View Models by Domain

### Ingredients (`app/domains/ingredients/surfaces/cli/`)

```go
type IngredientRow struct {
    ID       string `table:"ID"       json:"id"`
    Name     string `table:"NAME"     json:"name"`
    Category string `table:"CATEGORY" json:"category"`
    Unit     string `table:"UNIT"     json:"unit"`
    Desc     string `table:"-"        json:"description,omitempty"`
}
```

### Drinks (`app/domains/drinks/surfaces/cli/`)

```go
type DrinkRow struct {
    ID   string `table:"ID"   json:"id"`
    Name string `table:"NAME" json:"name"`
}
```

### Inventory (`app/domains/inventory/surfaces/cli/`)

```go
type InventoryRow struct {
    IngredientID string  `table:"INGREDIENT_ID" json:"ingredient_id"`
    Quantity     float64 `table:"QUANTITY"      json:"quantity"`
    Unit         string  `table:"UNIT"          json:"unit"`
}
```

### Menu (`app/domains/menu/surfaces/cli/`)

```go
type MenuRow struct {
    ID     string `table:"ID"     json:"id"`
    Name   string `table:"NAME"   json:"name"`
    Status string `table:"STATUS" json:"status"`
    Items  int    `table:"ITEMS"  json:"items"`
    Desc   string `table:"-"      json:"description,omitempty"`
}

type MenuItemRow struct {
    DrinkID      string `table:"DRINK_ID"     json:"drink_id"`
    Availability string `table:"AVAILABILITY" json:"availability"`
}
```

### Orders (`app/domains/orders/surfaces/cli/`)

```go
type OrderRow struct {
    ID        string `table:"ID"         json:"id"`
    MenuID    string `table:"MENU_ID"    json:"menu_id"`
    Status    string `table:"STATUS"     json:"status"`
    CreatedAt string `table:"CREATED_AT" json:"created_at"`
}

type OrderDetail struct {
    ID          string `table:"-" json:"id"`
    MenuID      string `table:"-" json:"menu_id"`
    Status      string `table:"-" json:"status"`
    CreatedAt   string `table:"-" json:"created_at"`
    CompletedAt string `table:"-" json:"completed_at,omitempty"`
    Notes       string `table:"-" json:"notes,omitempty"`
}

type OrderItemRow struct {
    DrinkID  string `table:"DRINK_ID" json:"drink_id"`
    Quantity int    `table:"QUANTITY" json:"quantity"`
}
```

### Audit (`app/domains/audit/surfaces/cli/`)

```go
type AuditRow struct {
    ID        string `table:"ID"         json:"id"`
    StartedAt string `table:"STARTED_AT" json:"started_at"`
    Action    string `table:"ACTION"     json:"action"`
    Resource  string `table:"RESOURCE"   json:"resource"`
    Principal string `table:"PRINCIPAL"  json:"principal"`
    Success   bool   `table:"SUCCESS"    json:"success"`
    Touches   int    `table:"TOUCHES"    json:"touches"`
    Error     string `table:"ERROR"      json:"error,omitempty"`
}
```

## Tasks

### Phase 1: Create Table Package

- [x] Create `main/cli/table/table.go`
- [x] Implement `PrintTable[T any](items []T) error`
- [x] Implement `PrintDetail[T any](item T) error`
- [x] Handle `fmt.Stringer` interface
- [x] Handle `time.Time` formatting
- [x] Handle `omitempty` for detail output
- [x] Add unit tests

### Phase 2: Create View Models

- [x] Add `IngredientRow` to `app/domains/ingredients/surfaces/cli/`
- [x] Add `DrinkRow` to `app/domains/drinks/surfaces/cli/`
- [x] Add `InventoryRow` to `app/domains/inventory/surfaces/cli/`
- [x] Add `MenuRow`, `MenuItemRow` to `app/domains/menu/surfaces/cli/`
- [x] Add `OrderRow`, `OrderDetail`, `OrderItemRow` to `app/domains/orders/surfaces/cli/`
- [x] Add `AuditRow` to `app/domains/audit/surfaces/cli/`

### Phase 3: Migrate CLI Commands

- [x] Migrate `ingredients list` and `ingredients get`
- [x] Migrate `drinks list` and `drinks get`
- [x] Migrate `inventory list` and `inventory get`
- [x] Migrate `menu list` and `menu show` (basic output)
- [x] Migrate `orders list` and `orders get`
- [x] Migrate `audit list`

### Phase 4: Cleanup

- [x] Remove `newTabWriter()` from helpers.go (if unused)
- [x] Run `go test ./...`
- [x] Verify CLI output matches previous format

## Acceptance Criteria

- [x] Single `PrintTable` call replaces all list command formatting
- [x] Single `PrintDetail` call replaces all get command formatting
- [x] `table` tag controls column headers
- [x] `json` tag controls detail labels (snake_case → Title Case)
- [x] `omitempty` skips zero-value fields in detail output
- [x] View models work for both CLI output and JSON serialization
- [x] Output format unchanged (headers, spacing, field order)
- [x] All tests pass

## Notes

### Out of Scope

The `menu show --costs` command has complex conditional formatting (margins, substitutions, analytics summary). This remains manual for now - the helper handles the simple cases.

### Why View Models?

Domain models have:
- Typed IDs (need `.String()`)
- Enums (need string cast)
- Optional fields (need unwrapping)
- Fields not shown in CLI

View models are pre-formatted strings, making the table package simpler and keeping formatting logic in one place.

Placing them in `surfaces/cli/` keeps them with other CLI-specific code for each domain, following the existing pattern (e.g., `drinkscli.FromDomainDrink`).

### Why Reuse `json` Tags?

Using the standard `json` tag for detail labels:
- Eliminates redundant tag definitions
- Same view model works for `--json` flag output
- snake_case is auto-converted to Title Case for display
- Idiomatic Go - follows existing conventions

### Alternative: Format Tags

Could add format specifiers to tags:

```go
type OrderRow struct {
    CreatedAt time.Time `table:"CREATED_AT,format:2006-01-02"`
    Quantity  float64   `table:"QTY,format:%.2f"`
}
```

Decided against this - view models with pre-formatted strings are simpler and more explicit. The conversion functions handle formatting.

# Sprint 045: Improve Dispatcher Generator Variable Naming

## Goal

Update the dispatcher code generator to produce more readable variable names by deriving them from the handler's domain package rather than the handler struct name.

## Problem

When multiple handlers from different domains handle the same event, the generated variable names are confusing:

```go
// Current output (confusing)
case ingredients_events.IngredientDeleted:
    ingredientDeleted := drinks_handlers.NewIngredientDeleted()
    ingredientDeleted2 := inventory_handlers.NewIngredientDeleted()
    ingredientDeleted3 := menu_handlers.NewIngredientDeleted()

    if err := ingredientDeleted.Handle(ctx, e); err != nil { ... }
    if err := ingredientDeleted2.Handle(ctx, e); err != nil { ... }
    if err := ingredientDeleted3.Handle(ctx, e); err != nil { ... }
```

The variable names `ingredientDeleted`, `ingredientDeleted2`, `ingredientDeleted3` don't convey which domain they belong to. You have to trace back to the constructor call to understand what each variable represents.

## Solution

Derive variable names from the domain package path, producing clearer output:

```go
// Desired output (clear)
case ingredients_events.IngredientDeleted:
    drinksHandler := drinks_handlers.NewIngredientDeleted()
    inventoryHandler := inventory_handlers.NewIngredientDeleted()
    menuHandler := menu_handlers.NewIngredientDeleted()

    if err := drinksHandler.Handle(ctx, e); err != nil { ... }
    if err := inventoryHandler.Handle(ctx, e); err != nil { ... }
    if err := menuHandler.Handle(ctx, e); err != nil { ... }
```

### Implementation

Modify `assignHandlerVarNames` in `pkg/dispatcher/gen/main.go` to extract the domain name from the handler's package path:

```go
func assignHandlerVarNames(groups []*eventGroup) {
    for _, g := range groups {
        used := map[string]int{}
        for i := range g.Handlers {
            // Extract domain from package path: .../app/domains/{domain}/handlers
            base := domainFromPkgPath(g.Handlers[i].PkgPath)
            if base == "" {
                base = "handler"
            }
            base = base + "Handler"

            if token.Lookup(base).IsKeyword() {
                base = base + "H"
            }

            used[base]++
            name := base
            if used[base] > 1 {
                name = fmt.Sprintf("%s%d", base, used[base])
            }
            g.Handlers[i].VarName = name
        }
    }
}

func domainFromPkgPath(pkgPath string) string {
    // pkgPath looks like: github.com/.../app/domains/drinks/handlers
    // We want to extract "drinks"
    parts := strings.Split(pkgPath, "/")
    for i, p := range parts {
        if p == "domains" && i+1 < len(parts) {
            return parts[i+1]
        }
    }
    // Fallback: use the parent of "handlers"
    for i, p := range parts {
        if p == "handlers" && i > 0 {
            return parts[i-1]
        }
    }
    return ""
}
```

### Edge Cases

| Package Path | Extracted Domain | Variable Name |
|--------------|------------------|---------------|
| `.../app/domains/drinks/handlers` | `drinks` | `drinksHandler` |
| `.../app/domains/inventory/handlers` | `inventory` | `inventoryHandler` |
| `.../app/domains/menu/handlers` | `menu` | `menuHandler` |
| `.../app/domains/audit/handlers` | `audit` | `auditHandler` |
| `.../pkg/middleware/events` (non-domain) | `middleware` | `middlewareHandler` |

If two handlers from the same domain handle the same event (unlikely but possible), they would get `drinksHandler`, `drinksHandler2`.

## Tasks

- [ ] Add `domainFromPkgPath` function to extract domain name from package path
- [ ] Update `assignHandlerVarNames` to use domain-based naming
- [ ] Run `go generate ./...` to regenerate dispatcher
- [ ] Verify generated code is more readable
- [ ] Run `go test ./...` to ensure no regressions

## Acceptance Criteria

- [ ] Generated variable names reflect the domain (e.g., `drinksHandler`, `menuHandler`)
- [ ] No numeric suffixes unless truly needed (same domain, same event)
- [ ] All tests pass
- [ ] `go generate ./...` succeeds

## Notes

### Why "Handler" Suffix?

Adding `Handler` to the variable name (`drinksHandler` vs `drinks`) provides clarity and avoids potential conflicts with other variables or package aliases.

### Non-Domain Handlers

For handlers not in the `app/domains/` path (like the audit handler responding to `ActivityCompleted`), fall back to extracting the directory name before `handlers`.

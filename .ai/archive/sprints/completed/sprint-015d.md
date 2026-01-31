# Sprint 015d: Exhaustive Enum Switch Linter (Intermezzo)

## Goal

Add the [exhaustive](https://github.com/nishanths/exhaustive) linter to verify switch statements over enum types cover all cases.

## Background

We have two types of sum types in the codebase:

1. **Interface sum types** (e.g., `optional.Value[T]`) - checked by `go-check-sumtype`
2. **Const-based enums** (e.g., `OrderStatus`, `MenuStatus`) - checked by `exhaustive`

The `exhaustive` tool catches missing cases in switch statements over const enums:

```go
type OrderStatus string
const (
    OrderStatusPending   OrderStatus = "pending"
    OrderStatusPreparing OrderStatus = "preparing"
    OrderStatusCompleted OrderStatus = "completed"
    OrderStatusCancelled OrderStatus = "cancelled"
)

// ERROR: missing cases OrderStatusPreparing, OrderStatusCancelled
switch status {
case OrderStatusPending:
    // ...
case OrderStatusCompleted:
    // ...
}
```

## Tasks

- [x] Add `exhaustive` to tool dependencies in `go.mod`
- [x] Add `exhaustive` to CI lint pipeline
- [x] Fix any existing missing enum cases
- [x] Verify `go tool exhaustive ./...` passes

## Tool Setup

Add to `go.mod`:
```go
tool github.com/nishanths/exhaustive/cmd/exhaustive
```

## CI Integration

Update build/lint order:
```bash
go generate ./...
go build ./...
go tool arch-lint
go tool go-check-sumtype ./...    # Interface sum types
go tool exhaustive ./...          # Const-based enums
go test ./...
```

## What It Catches

### Switch Statements

```go
// ERROR: missing cases in switch of type OrderStatus: OrderStatusPreparing, OrderStatusCancelled
func handleOrder(status OrderStatus) {
    switch status {
    case OrderStatusPending:
        // ...
    case OrderStatusCompleted:
        // ...
    }
}
```

### Map Keys (with -check=switch,map)

```go
// ERROR: missing keys in map of key type OrderStatus: OrderStatusPreparing
var statusLabels = map[OrderStatus]string{
    OrderStatusPending:   "Pending",
    OrderStatusCompleted: "Completed",
    OrderStatusCancelled: "Cancelled",
}
```

## Enum Types in Codebase

Types that will be checked:

| Type | Package | Values |
|------|---------|--------|
| `OrderStatus` | `app/domains/orders/models` | Pending, Preparing, Completed, Cancelled |
| `MenuStatus` | `app/domains/menu/models` | Draft, Published, Archived |
| `Availability` | `app/domains/menu/models` | Available, Limited, Unavailable |
| `DrinkCategory` | `app/domains/drinks/models` | Various categories |
| `GlassType` | `app/domains/drinks/models` | Various glass types |
| `AdjustmentReason` | `app/domains/inventory/models` | Order, Restock, Waste, etc. |

## Escape Hatch

For intentionally non-exhaustive switches, use a comment directive:

```go
//exhaustive:ignore
switch status {
case OrderStatusPending:
    // Only handling pending case intentionally
}
```

## Comparison: exhaustive vs go-check-sumtype

| Tool | Checks | Example |
|------|--------|---------|
| `exhaustive` | Const-based enums | `type Status string` with `const` |
| `go-check-sumtype` | Interface sum types | `type Value[T] interface` with `sealed()` |

Both are needed for complete coverage.

## Success Criteria

- `exhaustive` added to tools in `go.mod`
- `go tool exhaustive ./...` passes
- CI runs exhaustive on every PR
- All enum switches are exhaustive (or explicitly ignored)

## Dependencies

- Sprint 015b (go-check-sumtype already added)
- Sprint 015c (domain structure - enum types in place)

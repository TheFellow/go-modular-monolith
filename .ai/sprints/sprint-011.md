# Sprint 011: Inventory Management Context

## Goal

Create the Inventory bounded context to track stock levels for ingredients. This provides meaningful events (depleted, restocked) to exercise the dispatcher.

## Tasks

- [x] Create `app/inventory/models/stock.go` with Stock model
- [x] Create `app/inventory/internal/dao/` with file-based DAO
- [x] Create `app/inventory/authz/` with actions and policies
- [x] Create `app/inventory/queries/` with GetStock, ListStock queries
- [x] Create `app/inventory/internal/commands/` with AdjustStock, SetStock commands
- [x] Create `app/inventory/events/` with stock-related events
- [x] Create `app/inventory/module.go` exposing public API
- [x] Add inventory subcommands to CLI
- [x] Update app.go to include inventory module

## Domain Model

```go
type Stock struct {
    IngredientID   string
    Quantity       float64
    Unit           string
    LastUpdated    time.Time
}

type AdjustmentReason string
const (
    ReasonReceived   AdjustmentReason = "received"    // New shipment
    ReasonUsed       AdjustmentReason = "used"        // Used in drink
    ReasonSpilled    AdjustmentReason = "spilled"     // Waste
    ReasonExpired    AdjustmentReason = "expired"     // Discarded
    ReasonCorrected  AdjustmentReason = "corrected"   // Manual correction
)
```

## Events

One event drives cross-context behavior:

- `StockAdjusted{IngredientID, PreviousQty, NewQty, Delta, Reason}` - any stock change

```go
// app/inventory/events/stock_adjusted.go
type StockAdjusted struct {
    IngredientID cedar.EntityUID
    PreviousQty  float64
    NewQty       float64
    Delta        float64
    Reason       string
}
```

> **Note**: Earlier designs included separate `IngredientDepleted` and `IngredientRestocked` events. These were removed as redundant - handlers can derive threshold states from `StockAdjusted`:
> - Depleted: `NewQty == 0`
> - Restocked: `PreviousQty == 0 && NewQty > 0`

## Command Logic

```go
// app/inventory/internal/commands/adjust.go
func (c *AdjustStock) Execute(ctx *middleware.Context, req AdjustRequest) (*models.Stock, error) {
    stock, err := c.dao.Get(ctx, req.IngredientID)
    // ... handle not found, create if needed

    previousQty := stock.Quantity
    stock.Quantity += req.Delta
    if stock.Quantity < 0 {
        stock.Quantity = 0
    }
    stock.LastUpdated = time.Now()

    if err := c.dao.Save(ctx, stock); err != nil {
        return nil, err
    }

    ctx.AddEvent(events.StockAdjusted{
        IngredientID: req.IngredientID,
        PreviousQty:  previousQty,
        NewQty:       stock.Quantity,
        Delta:        req.Delta,
        Reason:       string(req.Reason),
    })

    return stock, nil
}
```

## CLI Commands

```
mixology inventory list
mixology inventory get <ingredient-id>
mixology inventory adjust <ingredient-id> <delta> --reason=<reason>
mixology inventory set <ingredient-id> <quantity>
```

## Success Criteria

- `go run ./main/cli inventory list` shows stock levels
- `go run ./main/cli inventory adjust <id> -2.0 --reason=used` adjusts stock
- `StockAdjusted` event appears in context after command execution
- `go test ./...` passes

## Dependencies

- Sprint 009 (Ingredients context for validation)
- Sprint 010 (Dispatcher infrastructure)

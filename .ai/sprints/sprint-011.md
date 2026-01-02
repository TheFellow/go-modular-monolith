# Sprint 011: Inventory Management Context

## Goal

Create the Inventory bounded context to track stock levels for ingredients. This provides meaningful events (depleted, restocked) to exercise the dispatcher.

## Tasks

- [ ] Create `app/inventory/models/stock.go` with Stock model
- [ ] Create `app/inventory/internal/dao/` with file-based DAO
- [ ] Create `app/inventory/authz/` with actions and policies
- [ ] Create `app/inventory/queries/` with GetStock, ListStock queries
- [ ] Create `app/inventory/internal/commands/` with AdjustStock, SetStock commands
- [ ] Create `app/inventory/events/` with stock-related events
- [ ] Create `app/inventory/module.go` exposing public API
- [ ] Add inventory subcommands to CLI
- [ ] Update app.go to include inventory module

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

These events drive cross-context behavior:

- `StockAdjusted{IngredientID, PreviousQty, NewQty, Delta, Reason}` - any stock change
- `IngredientDepleted{IngredientID}` - quantity reached zero
- `IngredientRestocked{IngredientID, NewQty}` - quantity went from zero to positive

```go
// app/inventory/events/events.go
type StockAdjusted struct {
    IngredientID string
    PreviousQty  float64
    NewQty       float64
    Delta        float64
    Reason       string
}

type IngredientDepleted struct {
    IngredientID string
}

type IngredientRestocked struct {
    IngredientID string
    NewQty       float64
}
```

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

    // Always emit StockAdjusted
    ctx.AddEvent(events.StockAdjusted{
        IngredientID: req.IngredientID,
        PreviousQty:  previousQty,
        NewQty:       stock.Quantity,
        Delta:        req.Delta,
        Reason:       string(req.Reason),
    })

    // Emit threshold events
    if previousQty > 0 && stock.Quantity == 0 {
        ctx.AddEvent(events.IngredientDepleted{IngredientID: req.IngredientID})
    }
    if previousQty == 0 && stock.Quantity > 0 {
        ctx.AddEvent(events.IngredientRestocked{
            IngredientID: req.IngredientID,
            NewQty:       stock.Quantity,
        })
    }

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
- Depleted/Restocked events fire at correct thresholds
- Events appear in context after command execution
- `go test ./...` passes

## Dependencies

- Sprint 009 (Ingredients context for validation)
- Sprint 010 (Dispatcher infrastructure)

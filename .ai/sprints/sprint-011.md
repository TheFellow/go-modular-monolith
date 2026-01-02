# Sprint 011: Inventory Management Context

## Goal

Create the Inventory bounded context to track stock levels for ingredients.

## Tasks

- [ ] Create `app/inventory/models/stock.go` with Stock model
- [ ] Create `app/inventory/internal/dao/` with file-based DAO
- [ ] Create `app/inventory/authz/` with actions and policies
- [ ] Create `app/inventory/queries/` with GetStock, ListStock, CheckAvailability queries
- [ ] Create `app/inventory/internal/commands/` with AdjustStock, SetStock commands
- [ ] Create `app/inventory/events/` with stock-related events
- [ ] Create `app/inventory/module.go` exposing public API
- [ ] Add inventory subcommands to CLI
- [ ] Seed initial stock data

## Domain Model

```go
type Stock struct {
    IngredientID   string
    Quantity       float64
    Unit           string
    LowThreshold   float64  // Warn when below this
    ReorderPoint   float64  // Suggest reorder when below this
    LastUpdated    time.Time
}

type StockAdjustment struct {
    IngredientID string
    Delta        float64  // Positive = add, negative = remove
    Reason       AdjustmentReason
    Note         string
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
- `IngredientDepleted{IngredientID}` - quantity reached zero (CRITICAL for menus)
- `IngredientRestocked{IngredientID, NewQty}` - quantity went from zero to positive
- `LowStockWarning{IngredientID, CurrentQty, Threshold}` - below warning threshold

## Key Query: CheckAvailability

Used by Menu context to determine drink availability:

```go
// app/inventory/queries/availability.go
type AvailabilityRequest struct {
    IngredientIDs []string
    Amounts       map[string]float64  // Required amounts per ingredient
}

type AvailabilityResponse struct {
    Available    bool
    Missing      []string            // Ingredients with insufficient stock
    LowStock     []string            // Ingredients that would go below threshold
}

func (q *Queries) CheckAvailability(ctx context.Context, req AvailabilityRequest) (AvailabilityResponse, error)
```

## Notes

Inventory is the "truth" for what we physically have. It's queried by Menu to calculate drink availability and emits events that Menu handles to update availability status.

The `IngredientDepleted` event is the key driver: when fired, Menu context must mark affected drinks as unavailable.

## Success Criteria

- `go run ./main/cli inventory list` shows stock levels
- `go run ./main/cli inventory adjust vodka -2.0 --reason=used`
- Depleted/Restocked events fire at correct thresholds
- `go test ./...` passes

## Dependencies

- Sprint 009 (Ingredients context for validation)

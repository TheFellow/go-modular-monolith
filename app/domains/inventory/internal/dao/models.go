package dao

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
)

type StockRow struct {
	IngredientID string
	InventoryID  string `bstore:"index"`
	Quantity     float64
	Unit         string
	CostPerUnit  *money.Price
	LastUpdated  time.Time `bstore:"index"`
}

// ReservationRow is owned by Inventory. OrderID is an external correlation
// identity; Inventory remains authoritative for how much stock is committed.
type ReservationRow struct {
	ID           string
	OrderID      string `bstore:"index"`
	IngredientID string `bstore:"index"`
	Quantity     float64
	Unit         string
}

package dao

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
)

type StockRow struct {
	IngredientID string
	Revision     uint64 `json:"-" store:"revision"`
	InventoryID  string `store:"index"`
	Quantity     float64
	Unit         string
	CostPerUnit  *money.Price
	LastUpdated  time.Time `store:"index"`
}

// ReservationRow is owned by Inventory. OrderID is an external correlation
// identity; Inventory remains authoritative for how much stock is committed.
type ReservationRow struct {
	ID           string
	Revision     uint64 `json:"-" store:"revision"`
	OrderID      string `store:"index"`
	IngredientID string `store:"index"`
	Quantity     float64
	Unit         string
}

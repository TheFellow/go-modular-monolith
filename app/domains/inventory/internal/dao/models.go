package dao

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
)

type StockRow struct {
	IngredientID string
	InventoryID  string
	Quantity     float64
	Unit         string
	CostPerUnit  *money.Price
	LastUpdated  time.Time `bstore:"index"`
}

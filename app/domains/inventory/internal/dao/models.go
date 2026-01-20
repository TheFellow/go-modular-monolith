package dao

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

type StockRow struct {
	IngredientID string
	InventoryID  string
	Quantity     float64
	Unit         string
	CostPerUnit  optional.Value[money.Price]
	LastUpdated  time.Time `bstore:"index"`
}

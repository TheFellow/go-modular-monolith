package events

import "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"

type StockAdjusted struct {
	Previous models.Inventory
	Current  models.Inventory
	Reason   string
}

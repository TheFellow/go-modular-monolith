package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"

// OrdersLoadedMsg is sent when orders have been loaded.
type OrdersLoadedMsg struct {
	Orders []models.Order
	Err    error
}

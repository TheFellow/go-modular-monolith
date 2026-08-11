package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
import "github.com/TheFellow/go-modular-monolith/pkg/paging"

// OrdersLoadedMsg is sent when orders have been loaded.
type OrdersLoadedMsg struct {
	Orders []models.Order
	Next   paging.Cursor
	Err    error
	Token  uint64
}

// OrderCompletedMsg is sent when an order is completed.
type OrderCompletedMsg struct {
	Order *models.Order
}

// OrderCancelledMsg is sent when an order is cancelled.
type OrderCancelledMsg struct {
	Order *models.Order
}

// CompleteErrorMsg is sent when completing fails.
type CompleteErrorMsg struct {
	Err error
}

// CancelErrorMsg is sent when cancellation fails.
type CancelErrorMsg struct {
	Err error
}

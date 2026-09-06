package events

import "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"

type OrderAmended struct {
	Before models.Order
	Order  models.Order
	Reason string
}

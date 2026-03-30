package events

import "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"

type OrderCancelled struct {
	Order models.Order
}

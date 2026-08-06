package events

import "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"

type OrderCompleted struct {
	Order models.Order
}

package events

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
)

type OrderPlaced struct {
	Order models.Order
}

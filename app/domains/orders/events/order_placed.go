package events

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	cedar "github.com/cedar-policy/cedar-go"
)

type OrderPlaced struct {
	Order models.Order
}

type OrderCancelled struct {
	Order models.Order
}

type OrderCompleted struct {
	Order               models.Order
	IngredientUsage     []IngredientUsage
	DepletedIngredients []cedar.EntityUID
}

type IngredientUsage struct {
	IngredientID cedar.EntityUID
	Name         string
	Amount       float64
	Unit         string
}

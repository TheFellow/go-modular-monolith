package events

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
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
	DepletedIngredients []entity.IngredientID
}

type IngredientUsage struct {
	IngredientID entity.IngredientID
	Name         string
	Amount       measurement.Amount
}

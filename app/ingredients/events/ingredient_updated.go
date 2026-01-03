package events

import (
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	cedar "github.com/cedar-policy/cedar-go"
)

type IngredientUpdated struct {
	IngredientID cedar.EntityUID
	Name         string
	Category     models.Category
}

package events

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
)

type IngredientUpdated struct {
	Ingredient models.Ingredient
}

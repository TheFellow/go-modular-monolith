package events

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
)

type IngredientCreated struct {
	Ingredient models.Ingredient
}

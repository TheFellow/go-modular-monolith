package events

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	cedar "github.com/cedar-policy/cedar-go"
)

type DrinkRecipeUpdated struct {
	DrinkID cedar.EntityUID
	Name    string

	PreviousRecipe models.Recipe
	NewRecipe      models.Recipe

	AddedIngredients   []cedar.EntityUID
	RemovedIngredients []cedar.EntityUID
}

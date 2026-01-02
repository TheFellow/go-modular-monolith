package events

import "github.com/TheFellow/go-modular-monolith/app/ingredients/models"

type IngredientCreated struct {
	IngredientID string
	Name         string
	Category     models.Category
}

package events

import "github.com/TheFellow/go-modular-monolith/app/ingredients/models"

type IngredientUpdated struct {
	IngredientID string
	Name         string
	Category     models.Category
}

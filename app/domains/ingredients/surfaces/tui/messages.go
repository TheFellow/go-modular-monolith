package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"

// IngredientsLoadedMsg is sent when ingredients have been loaded.
type IngredientsLoadedMsg struct {
	Ingredients []models.Ingredient
	Err         error
}

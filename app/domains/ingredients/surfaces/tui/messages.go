package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
import "github.com/TheFellow/go-modular-monolith/pkg/paging"

// IngredientsLoadedMsg is sent when ingredients have been loaded.
type IngredientsLoadedMsg struct {
	Ingredients []models.Ingredient
	Err         error
	Next        paging.Cursor
	Token       uint64
}

// IngredientDeletedMsg is sent when an ingredient has been deleted.
type IngredientDeletedMsg struct {
	Ingredient *models.Ingredient
}

// DeleteErrorMsg is sent when a delete operation fails.
type DeleteErrorMsg struct {
	Err error
}

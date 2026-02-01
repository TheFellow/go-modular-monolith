package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
)

type ingredientItem struct {
	ingredient models.Ingredient
}

func (i ingredientItem) Title() string { return i.ingredient.Name }
func (i ingredientItem) Description() string {
	return fmt.Sprintf("%s • %s", i.ingredient.Category, i.ingredient.Unit)
}
func (i ingredientItem) FilterValue() string { return i.ingredient.Name }

package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/mvvm"
)

type ingredientItem = mvvm.ListItem[models.Ingredient]

func newIngredientItem(ingredient models.Ingredient) ingredientItem {
	return mvvm.NewListItem(ingredient, ingredient.Name, fmt.Sprintf("%s • %s", ingredient.Category, ingredient.Unit), ingredient.Name)
}

package tui

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
)

type ingredientItem = tui.ListItem[models.Ingredient]

func newIngredientItem(ingredient models.Ingredient) ingredientItem {
	description := tui.ListSummary(string(ingredient.Category), string(ingredient.Unit), tui.TagSummary(ingredient.Tags.Canonical().String()))
	return tui.NewListItem(ingredient, ingredient.Name, description, ingredient.Name)
}

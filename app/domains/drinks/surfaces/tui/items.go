package tui

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
)

type drinkItem = tui.ListItem[models.Drink]

func newDrinkItem(drink models.Drink) drinkItem {
	description := tui.ListSummary(string(drink.Category), string(drink.Glass), string(drink.Status), tui.TagSummary(drink.Tags.Canonical().String()))
	return tui.NewListItem(drink, drink.Name, description, drink.Name)
}

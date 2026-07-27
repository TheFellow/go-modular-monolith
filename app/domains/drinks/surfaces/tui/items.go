package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
)

type drinkItem = tui.ListItem[models.Drink]

func newDrinkItem(drink models.Drink) drinkItem {
	return tui.NewListItem(drink, drink.Name, fmt.Sprintf("%s • %s", drink.Category, drink.Glass), drink.Name)
}

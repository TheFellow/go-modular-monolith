package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/mvvm"
)

type drinkItem = mvvm.ListItem[models.Drink]

func newDrinkItem(drink models.Drink) drinkItem {
	return mvvm.NewListItem(drink, drink.Name, fmt.Sprintf("%s • %s", drink.Category, drink.Glass), drink.Name)
}

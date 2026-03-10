package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
)

type drinkItem struct {
	drink models.Drink
}

func (i drinkItem) Title() string { return i.drink.Name }

func (i drinkItem) Description() string {
	return fmt.Sprintf("%s • %s", i.drink.Category, i.drink.Glass)
}

func (i drinkItem) FilterValue() string { return i.drink.Name }

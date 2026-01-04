package commands

import (
	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsq "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	inventoryq "github.com/TheFellow/go-modular-monolith/app/domains/inventory/queries"
	menuq "github.com/TheFellow/go-modular-monolith/app/domains/menu/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
)

type Commands struct {
	dao *dao.FileOrderDAO

	menus       *menuq.Queries
	drinks      *drinksq.Queries
	ingredients *ingredientsq.Queries
	inventory   *inventoryq.Queries
}

func New() *Commands {
	return &Commands{
		dao:         dao.New(),
		menus:       menuq.New(),
		drinks:      drinksq.New(),
		ingredients: ingredientsq.New(),
		inventory:   inventoryq.New(),
	}
}

func NewWithDependencies(d *dao.FileOrderDAO, menus *menuq.Queries, drinks *drinksq.Queries, ingredients *ingredientsq.Queries, inventory *inventoryq.Queries) *Commands {
	return &Commands{
		dao:         d,
		menus:       menus,
		drinks:      drinks,
		ingredients: ingredients,
		inventory:   inventory,
	}
}

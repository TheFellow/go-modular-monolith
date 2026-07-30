package commands

import (
	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsq "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	inventoryq "github.com/TheFellow/go-modular-monolith/app/domains/inventory/queries"
	menuq "github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Commands struct {
	dao *dao.DAO

	menus       *menuq.Queries
	drinks      *drinksq.Queries
	ingredients *ingredientsq.Queries
	inventory   *inventoryq.Queries
}

func New(s *store.Store, tags tag.Repository) *Commands {
	return &Commands{
		dao:         dao.New(s, tags),
		menus:       menuq.New(s, tags),
		drinks:      drinksq.New(s, tags),
		ingredients: ingredientsq.New(s, tags),
		inventory:   inventoryq.New(s, tags),
	}
}

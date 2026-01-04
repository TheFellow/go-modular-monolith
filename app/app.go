package app

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type App struct {
	drinks      *drinks.Module
	ingredients *ingredients.Module
	inventory   *inventory.Module
	menu        *menu.Module
	orders      *orders.Module
}

func New() *App {
	middleware.Command = middleware.NewCommandChain(
		middleware.CommandAuthZ(),
		middleware.UnitOfWork(),
		middleware.Dispatcher(dispatcher.New()),
	)

	im := ingredients.NewModule()
	dm := drinks.NewModule()
	invm := inventory.NewModule()
	mm := menu.NewModule()
	om := orders.NewModule()

	return &App{drinks: dm, ingredients: im, inventory: invm, menu: mm, orders: om}
}

func (a *App) Drinks() *drinks.Module {
	return a.drinks
}

func (a *App) Ingredients() *ingredients.Module {
	return a.ingredients
}

func (a *App) Inventory() *inventory.Module {
	return a.inventory
}

func (a *App) Menu() *menu.Module {
	return a.menu
}

func (a *App) Orders() *orders.Module {
	return a.orders
}

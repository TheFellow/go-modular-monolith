package app

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks"
	"github.com/TheFellow/go-modular-monolith/app/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/inventory"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/uow"
)

type App struct {
	drinks      *drinks.Module
	ingredients *ingredients.Module
	inventory   *inventory.Module
}

func New() *App {
	middleware.Command = middleware.NewCommandChain(
		middleware.CommandAuthZ(),
		middleware.UnitOfWork(uow.NewManager()),
		middleware.Dispatcher(dispatcher.New()),
	)

	im := ingredients.NewModule()
	dm := drinks.NewModule()
	invm := inventory.NewModule()

	return &App{drinks: dm, ingredients: im, inventory: invm}
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

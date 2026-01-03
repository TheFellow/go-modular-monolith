package app

import (
	"path/filepath"

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

type options struct {
	drinksDataPath      string
	ingredientsDataPath string
	inventoryDataPath   string
}

type Option func(*options)

func WithDrinksDataPath(path string) Option {
	return func(o *options) {
		o.drinksDataPath = path
	}
}

func WithIngredientsDataPath(path string) Option {
	return func(o *options) {
		o.ingredientsDataPath = path
	}
}

func WithInventoryDataPath(path string) Option {
	return func(o *options) {
		o.inventoryDataPath = path
	}
}

func New(opts ...Option) (*App, error) {
	middleware.Command = middleware.NewCommandChain(
		middleware.CommandAuthZ(),
		middleware.UnitOfWork(uow.NewManager()),
		middleware.Dispatcher(dispatcher.New()),
	)

	o := options{
		drinksDataPath:      filepath.Join("pkg", "data", "drinks.json"),
		ingredientsDataPath: filepath.Join("pkg", "data", "ingredients.json"),
		inventoryDataPath:   filepath.Join("pkg", "data", "inventory.json"),
	}
	for _, opt := range opts {
		opt(&o)
	}

	dm, err := drinks.NewModule(o.drinksDataPath)
	if err != nil {
		return nil, err
	}

	im, err := ingredients.NewModule(o.ingredientsDataPath)
	if err != nil {
		return nil, err
	}

	invm, err := inventory.NewModule(o.inventoryDataPath, im)
	if err != nil {
		return nil, err
	}

	return &App{
		drinks:      dm,
		ingredients: im,
		inventory:   invm,
	}, nil
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

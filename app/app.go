package app

import (
	"path/filepath"

	"github.com/TheFellow/go-modular-monolith/app/drinks"
	"github.com/TheFellow/go-modular-monolith/app/ingredients"
)

type App struct {
	drinks      *drinks.Module
	ingredients *ingredients.Module
}

type options struct {
	drinksDataPath      string
	ingredientsDataPath string
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

func New(opts ...Option) (*App, error) {
	o := options{
		drinksDataPath:      filepath.Join("pkg", "data", "drinks.json"),
		ingredientsDataPath: filepath.Join("pkg", "data", "ingredients.json"),
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

	return &App{
		drinks:      dm,
		ingredients: im,
	}, nil
}

func (a *App) Drinks() *drinks.Module {
	return a.drinks
}

func (a *App) Ingredients() *ingredients.Module {
	return a.ingredients
}

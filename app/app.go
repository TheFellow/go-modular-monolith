package app

import (
	"path/filepath"

	"github.com/TheFellow/go-modular-monolith/app/drinks"
)

type App struct {
	drinks *drinks.Module
}

type options struct {
	drinksDataPath string
}

type Option func(*options)

func WithDrinksDataPath(path string) Option {
	return func(o *options) {
		o.drinksDataPath = path
	}
}

func New(opts ...Option) (*App, error) {
	o := options{
		drinksDataPath: filepath.Join("pkg", "data", "drinks.json"),
	}
	for _, opt := range opts {
		opt(&o)
	}

	m, err := drinks.NewModule(o.drinksDataPath)
	if err != nil {
		return nil, err
	}

	return &App{
		drinks: m,
	}, nil
}

func (a *App) Drinks() *drinks.Module {
	return a.drinks
}

package app

import (
	"path/filepath"
)

type App struct {
	drinks *Drinks
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

	drinks, err := NewDrinks(o.drinksDataPath)
	if err != nil {
		return nil, err
	}

	return &App{
		drinks: drinks,
	}, nil
}

func (a *App) Drinks() *Drinks {
	return a.drinks
}

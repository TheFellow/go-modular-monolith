package app

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

type App struct {
	Store       optional.Value[*store.Store]
	Dispatcher  middleware.EventDispatcher
	Drinks      *drinks.Module
	Ingredients *ingredients.Module
	Inventory   *inventory.Module
	Menu        *menu.Module
	Orders      *orders.Module
}

func New(opts ...Option) *App {
	a := &App{
		Store:       optional.None[*store.Store](),
		Dispatcher:  dispatcher.New(),
		Drinks:      drinks.NewModule(),
		Ingredients: ingredients.NewModule(),
		Inventory:   inventory.NewModule(),
		Menu:        menu.NewModule(),
		Orders:      orders.NewModule(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(a)
		}
	}
	return a
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}
	s, ok := a.Store.Unwrap()
	if !ok || s == nil {
		return nil
	}
	return s.Close()
}

func (a *App) Context(parent context.Context, principal cedar.EntityUID) *middleware.Context {
	opts := []middleware.ContextOpt{
		middleware.WithPrincipal(principal),
	}
	if a != nil {
		if s, ok := a.Store.Unwrap(); ok && s != nil {
			opts = append(opts, middleware.WithStore(s))
		}
		if a.Dispatcher != nil {
			opts = append(opts, middleware.WithEventDispatcher(a.Dispatcher))
		}
	}
	return middleware.NewContext(parent, opts...)
}

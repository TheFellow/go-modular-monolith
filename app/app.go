package app

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type App struct {
	Store       *store.Store
	Drinks      *drinks.Module
	Ingredients *ingredients.Module
	Inventory   *inventory.Module
	Menu        *menu.Module
	Orders      *orders.Module
}

func New() *App {
	return &App{
		Drinks:      drinks.NewModule(),
		Ingredients: ingredients.NewModule(),
		Inventory:   inventory.NewModule(),
		Menu:        menu.NewModule(),
		Orders:      orders.NewModule(),
	}
}

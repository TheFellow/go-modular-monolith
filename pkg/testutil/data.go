package testutil

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
)

func CreateIngredient(t testing.TB, f *Fixture, ingredient ingredientsmodels.Ingredient) *ingredientsmodels.Ingredient {
	t.Helper()
	created, err := f.Ingredients.Create(f.OwnerContext(), &ingredient)
	Ok(t, err)
	return created
}

func CreateDrink(t testing.TB, f *Fixture, drink drinksmodels.Drink) *drinksmodels.Drink {
	t.Helper()
	created, err := f.Drinks.Create(f.OwnerContext(), &drink)
	Ok(t, err)
	return created
}

func SetInventory(t testing.TB, f *Fixture, update inventorymodels.Update) *inventorymodels.Inventory {
	t.Helper()
	stock, err := f.Inventory.Set(f.OwnerContext(), &update)
	Ok(t, err)
	return stock
}

type MenuOption func(*menuOptions)

type menuOptions struct {
	description string
	drinks      []*drinksmodels.Drink
	published   bool
}

func WithDescription(description string) MenuOption {
	return func(options *menuOptions) { options.description = description }
}

func WithDrink(drink *drinksmodels.Drink) MenuOption {
	return func(options *menuOptions) { options.drinks = append(options.drinks, drink) }
}

func Published() MenuOption {
	return func(options *menuOptions) { options.published = true }
}

func CreateMenu(t testing.TB, f *Fixture, name string, opts ...MenuOption) *menumodels.Menu {
	t.Helper()

	options := menuOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	menu, err := f.Menus.Create(f.OwnerContext(), &menumodels.Menu{Name: name, Description: options.description})
	Ok(t, err)
	for _, drink := range options.drinks {
		NotNil(t, drink)
		menu, err = f.Menus.AddDrink(f.OwnerContext(), &menumodels.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
		Ok(t, err)
	}
	if options.published {
		menu, err = f.Menus.Publish(f.OwnerContext(), &menumodels.Menu{ID: menu.ID})
		Ok(t, err)
	}
	return menu
}

func PlaceOrder(t testing.TB, f *Fixture, order ordersmodels.Order) *ordersmodels.Order {
	t.Helper()
	created, err := f.Orders.Place(f.OwnerContext(), &order)
	Ok(t, err)
	return created
}

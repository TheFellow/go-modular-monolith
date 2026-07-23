package testutil

import (
	"strings"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
)

type Bootstrap struct {
	fix *Fixture
}

func (b *Bootstrap) WithBasicIngredients() *Bootstrap {
	b.fix.T.Helper()
	ctx := b.fix.OwnerContext()

	basics := []ingredientsmodels.Ingredient{
		{Name: "Tequila", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz},
		{Name: "Lime Juice", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz},
		{Name: "Triple Sec", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz},
		{Name: "Simple Syrup", Category: ingredientsmodels.CategorySyrup, Unit: measurement.UnitOz},
		{Name: "Vodka", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz},
		{Name: "Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz},
	}
	for _, ing := range basics {
		ingredient := ing
		_, err := b.fix.Ingredients.Create(ctx, &ingredient)
		Ok(b.fix.T, err)
	}
	return b
}

func (b *Bootstrap) WithStock(quantity float64) *Bootstrap {
	b.fix.T.Helper()
	ctx := b.fix.OwnerContext()

	ings, err := b.fix.Ingredients.List(ctx, ingredients.ListRequest{})
	Ok(b.fix.T, err)

	for _, ing := range ings.Items {
		_, err := b.fix.Inventory.Set(ctx, &inventorymodels.Update{
			IngredientID: ing.ID,
			Amount:       measurement.MustAmount(quantity, ing.Unit),
			CostPerUnit:  money.NewPriceFromCents(100, currency.USD),
		})
		Ok(b.fix.T, err)
	}
	return b
}

func (b *Bootstrap) WithNoStock() *Bootstrap { return b.WithStock(0) }

func (b *Bootstrap) WithIngredient(name string, unit measurement.Unit) *ingredientsmodels.Ingredient {
	b.fix.T.Helper()

	ings, err := b.fix.Ingredients.List(b.fix.OwnerContext(), ingredients.ListRequest{})
	Ok(b.fix.T, err)

	want := normalizeName(name)
	for _, ing := range ings.Items {
		if normalizeName(ing.Name) == want {
			return ing
		}
	}

	return b.WithIngredientModel(ingredientsmodels.Ingredient{
		Name:     name,
		Category: ingredientsmodels.CategoryOther,
		Unit:     unit,
	})
}

func (b *Bootstrap) WithIngredientModel(ingredient ingredientsmodels.Ingredient) *ingredientsmodels.Ingredient {
	b.fix.T.Helper()

	created, err := b.fix.Ingredients.Create(b.fix.OwnerContext(), &ingredient)
	Ok(b.fix.T, err)
	return created
}

func (b *Bootstrap) WithDrink(drink drinksmodels.Drink) *drinksmodels.Drink {
	b.fix.T.Helper()

	created, err := b.fix.Drinks.Create(b.fix.OwnerContext(), &drink)
	Ok(b.fix.T, err)
	return created
}

func (b *Bootstrap) WithMenu(name string) *menumodels.Menu {
	b.fix.T.Helper()
	return b.WithMenuModel(menumodels.Menu{Name: name})
}

func (b *Bootstrap) WithMenuModel(menu menumodels.Menu) *menumodels.Menu {
	b.fix.T.Helper()

	created, err := b.fix.Menus.Create(b.fix.OwnerContext(), &menu)
	Ok(b.fix.T, err)
	return created
}

func (b *Bootstrap) WithInventory(ingredient *ingredientsmodels.Ingredient, quantity float64) *inventorymodels.Inventory {
	b.fix.T.Helper()

	stock, err := b.fix.Inventory.Set(b.fix.OwnerContext(), &inventorymodels.Update{
		IngredientID: ingredient.ID,
		Amount:       measurement.MustAmount(quantity, ingredient.Unit),
		CostPerUnit:  money.NewPriceFromCents(100, currency.USD),
	})
	Ok(b.fix.T, err)
	return stock
}

func (b *Bootstrap) WithPublishedMenu(menu menumodels.Menu, drinks ...*drinksmodels.Drink) *menumodels.Menu {
	b.fix.T.Helper()

	created := b.AddDrinks(b.WithMenuModel(menu), drinks...)
	published, err := b.fix.Menus.Publish(b.fix.OwnerContext(), &menumodels.Menu{ID: created.ID})
	Ok(b.fix.T, err)
	return published
}

func (b *Bootstrap) AddDrinks(menu *menumodels.Menu, drinks ...*drinksmodels.Drink) *menumodels.Menu {
	b.fix.T.Helper()
	NotNil(b.fix.T, menu)

	updated := menu
	for _, drink := range drinks {
		NotNil(b.fix.T, drink)
		var err error
		updated, err = b.fix.Menus.AddDrink(b.fix.OwnerContext(), &menumodels.MenuPatch{
			MenuID: updated.ID, DrinkID: drink.ID,
		})
		Ok(b.fix.T, err)
	}
	return updated
}

func (b *Bootstrap) WithOrder(order ordersmodels.Order) *ordersmodels.Order {
	b.fix.T.Helper()

	created, err := b.fix.Orders.Place(b.fix.OwnerContext(), &order)
	Ok(b.fix.T, err)
	return created
}

func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

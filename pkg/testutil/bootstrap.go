package testutil

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
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

	for _, ing := range ings {
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
	ctx := b.fix.OwnerContext()

	ings, err := b.fix.Ingredients.List(ctx, ingredients.ListRequest{})
	Ok(b.fix.T, err)

	want := normalizeName(name)
	for _, ing := range ings {
		if normalizeName(ing.Name) == want {
			return ing
		}
	}

	created, err := b.fix.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{
		Name:     name,
		Category: ingredientsmodels.CategoryOther,
		Unit:     unit,
	})
	Ok(b.fix.T, err)
	return created
}

func (b *Bootstrap) WithDrink(drink models.Drink) *models.Drink {
	b.fix.T.Helper()

	created, err := b.fix.Drinks.Create(b.fix.OwnerContext(), &drink)
	Ok(b.fix.T, err)
	return created
}

func (b *Bootstrap) WithMenu(name string) *menumodels.Menu {
	b.fix.T.Helper()

	created, err := b.fix.Menu.Create(b.fix.OwnerContext(), &menumodels.Menu{Name: name})
	Ok(b.fix.T, err)
	return created
}

func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

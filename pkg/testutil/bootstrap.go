package testutil

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
)

type Bootstrap struct {
	fix *Fixture
}

func (b *Bootstrap) WithBasicIngredients() *Bootstrap {
	b.fix.T.Helper()

	basics := []ingredientsmodels.Ingredient{
		{Name: "Tequila", Category: ingredientsmodels.CategorySpirit, Unit: ingredientsmodels.UnitOz},
		{Name: "Lime Juice", Category: ingredientsmodels.CategoryJuice, Unit: ingredientsmodels.UnitOz},
		{Name: "Triple Sec", Category: ingredientsmodels.CategoryOther, Unit: ingredientsmodels.UnitOz},
		{Name: "Simple Syrup", Category: ingredientsmodels.CategorySyrup, Unit: ingredientsmodels.UnitOz},
		{Name: "Vodka", Category: ingredientsmodels.CategorySpirit, Unit: ingredientsmodels.UnitOz},
		{Name: "Gin", Category: ingredientsmodels.CategorySpirit, Unit: ingredientsmodels.UnitOz},
	}
	for _, ing := range basics {
		_, err := b.fix.Ingredients.Create(b.fix.Ctx, ing)
		Ok(b.fix.T, err)
	}
	return b
}

func (b *Bootstrap) WithStock(quantity float64) *Bootstrap {
	b.fix.T.Helper()

	res, err := b.fix.Ingredients.List(b.fix.Ctx, ingredients.ListRequest{})
	Ok(b.fix.T, err)

	for _, ing := range res.Ingredients {
		_, err := b.fix.Inventory.Set(b.fix.Ctx, inventorymodels.StockUpdate{
			IngredientID: ing.ID,
			Quantity:     quantity,
			CostPerUnit:  money.NewPriceFromCents(100, "USD"),
		})
		Ok(b.fix.T, err)
	}
	return b
}

func (b *Bootstrap) WithNoStock() *Bootstrap { return b.WithStock(0) }

func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

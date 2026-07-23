package app_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDomainListsUseCursorPages(t *testing.T) {
	t.Parallel()

	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	ing1, err := f.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{Name: "Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	testutil.Ok(t, err)
	ing2, err := f.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{Name: "Tonic", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.Ok(t, err)

	drink1 := f.CreateDrink("Gin One").With("Gin", 1).Build()
	drink2 := f.CreateDrink("Gin Two").With("Gin", 2).Build()

	price := money.NewPriceFromCents(100, currency.USD)
	_, err = f.Inventory.Set(ctx, &inventorymodels.Update{IngredientID: ing1.ID, Amount: measurement.MustAmount(10, measurement.UnitOz), CostPerUnit: price})
	testutil.Ok(t, err)
	_, err = f.Inventory.Set(ctx, &inventorymodels.Update{IngredientID: ing2.ID, Amount: measurement.MustAmount(20, measurement.UnitOz), CostPerUnit: price})
	testutil.Ok(t, err)

	menu1, err := f.Menus.Create(ctx, &menusmodels.Menu{Name: "First"})
	testutil.Ok(t, err)
	_, err = f.Menus.Create(ctx, &menusmodels.Menu{Name: "Second"})
	testutil.Ok(t, err)
	menu1, err = f.Menus.AddDrink(ctx, &menusmodels.MenuPatch{MenuID: menu1.ID, DrinkID: drink1.ID})
	testutil.Ok(t, err)
	menu1, err = f.Menus.AddDrink(ctx, &menusmodels.MenuPatch{MenuID: menu1.ID, DrinkID: drink2.ID})
	testutil.Ok(t, err)
	menu1, err = f.Menus.Publish(ctx, &menusmodels.Menu{ID: menu1.ID})
	testutil.Ok(t, err)

	_, err = f.Orders.Place(ctx, &ordersmodels.Order{MenuID: menu1.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink1.ID, Quantity: 1}}})
	testutil.Ok(t, err)
	_, err = f.Orders.Place(ctx, &ordersmodels.Order{MenuID: menu1.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink2.ID, Quantity: 1}}})
	testutil.Ok(t, err)

	assertCursorPages(t, func(cursor paging.Cursor) (paging.Page[*ingredientsmodels.Ingredient], error) {
		return f.Ingredients.List(ctx, ingredients.ListRequest{Cursor: cursor, Limit: 1})
	}, "ing-not-a-ksuid", func(v *ingredientsmodels.Ingredient) string { return v.ID.String() })
	assertCursorPages(t, func(cursor paging.Cursor) (paging.Page[*drinksmodels.Drink], error) {
		return f.Drinks.List(ctx, drinks.ListRequest{Cursor: cursor, Limit: 1})
	}, "drk-not-a-ksuid", func(v *drinksmodels.Drink) string { return v.ID.String() })
	assertCursorPages(t, func(cursor paging.Cursor) (paging.Page[*inventorymodels.Inventory], error) {
		return f.Inventory.List(ctx, inventory.ListRequest{Cursor: cursor, Limit: 1})
	}, "inv-not-a-ksuid", func(v *inventorymodels.Inventory) string { return v.ID.String() })
	assertCursorPages(t, func(cursor paging.Cursor) (paging.Page[*menusmodels.Menu], error) {
		return f.Menus.List(ctx, menus.ListRequest{Cursor: cursor, Limit: 1})
	}, "mnu-not-a-ksuid", func(v *menusmodels.Menu) string { return v.ID.String() })
	assertCursorPages(t, func(cursor paging.Cursor) (paging.Page[*ordersmodels.Order], error) {
		return f.Orders.List(ctx, orders.ListRequest{Cursor: cursor, Limit: 1})
	}, "ord-not-a-ksuid", func(v *ordersmodels.Order) string { return v.ID.String() })
}

func TestResidualOrFilterFillsCursorPages(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	_, err := f.Ingredients.List(ctx, ingredients.ListRequest{Filter: `unknown == "value"`})
	testutil.ErrorIsInvalid(t, err)
	for _, ingredient := range []ingredientsmodels.Ingredient{
		{Name: "Needle", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz},
		{Name: "Hay One", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz},
		{Name: "Hay Two", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz},
		{Name: "Other Match", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz},
		{Name: "Hay Three", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz},
	} {
		_, err := f.Ingredients.Create(ctx, &ingredient)
		testutil.Ok(t, err)
	}

	const expression = `name == "Needle" || category == "other"`
	first, err := f.Ingredients.List(ctx, ingredients.ListRequest{Filter: expression, Limit: 1})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(first.Items) != 1, "first filtered page has %d items", len(first.Items))
	testutil.StringNonEmpty(t, string(first.Next), "first filtered page missing cursor")

	second, err := f.Ingredients.List(ctx, ingredients.ListRequest{Filter: expression, Cursor: first.Next, Limit: 1})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(second.Items) != 1, "second filtered page has %d items", len(second.Items))
	testutil.ErrorIf(t, first.Items[0].ID == second.Items[0].ID, "filtered cursor repeated an item")
	testutil.ErrorIf(t, second.Next != "", "second filtered page has cursor %q", second.Next)
}

func assertCursorPages[T any](t *testing.T, list func(paging.Cursor) (paging.Page[T], error), malformed paging.Cursor, id func(T) string) {
	t.Helper()
	_, err := list(malformed)
	testutil.ErrorIsInvalid(t, err)

	first, err := list("")
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(first.Items) != 1, "first page has %d items, want 1", len(first.Items))
	testutil.StringNonEmpty(t, string(first.Next), "first page missing next cursor")

	second, err := list(first.Next)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(second.Items) != 1, "second page has %d items, want 1", len(second.Items))
	testutil.ErrorIf(t, id(first.Items[0]) == id(second.Items[0]), "cursor page repeated item %q", id(first.Items[0]))
	testutil.ErrorIf(t, second.Next != "", "final page has unexpected next cursor %q", second.Next)
}

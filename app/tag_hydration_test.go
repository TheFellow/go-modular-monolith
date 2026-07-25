package app_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	appLog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestOperationalEntitiesHydrateTagsThroughGetListAndCedar(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Hydration Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Hydration Highball", Category: drinksmodels.DrinkCategoryHighball, Glass: drinksmodels.GlassTypeHighball,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"build"},
		},
	})
	stock := testutil.SetInventory(t, f, inventorymodels.Update{
		IngredientID: ingredient.ID, Amount: measurement.MustAmount(12, measurement.UnitOz),
		CostPerUnit: money.NewPriceFromCents(200, currency.USD),
	})
	menu := testutil.CreateMenu(t, f, "Hydration Menu", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{
		MenuID: menu.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
	})

	want := tag.Tags{{Key: "audience", Value: "members"}, {Key: "featured"}}
	targets := []cedar.EntityUID{ingredient.EntityUID(), drink.EntityUID(), stock.EntityUID(), menu.EntityUID(), order.EntityUID()}
	for _, target := range targets {
		for _, value := range want {
			_, err := f.App.Tags.Upsert(ctx, target, value)
			testutil.Ok(t, err)
		}
	}

	gotIngredient, err := f.Ingredients.Get(ctx, ingredient.ID)
	testutil.Ok(t, err)
	gotDrink, err := f.Drinks.Get(ctx, drink.ID)
	testutil.Ok(t, err)
	gotStock, err := f.Inventory.Get(ctx, ingredient.ID)
	testutil.Ok(t, err)
	gotMenu, err := f.Menus.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	gotOrder, err := f.Orders.Get(ctx, order.ID)
	testutil.Ok(t, err)

	assertHydrated(t, gotIngredient.Tags, gotIngredient.CedarEntity(), want)
	assertHydrated(t, gotDrink.Tags, gotDrink.CedarEntity(), want)
	assertHydrated(t, gotStock.Tags, gotStock.CedarEntity(), want)
	assertHydrated(t, gotMenu.Tags, gotMenu.CedarEntity(), want)
	assertHydrated(t, gotOrder.Tags, gotOrder.CedarEntity(), want)
	updatedIngredient, err := f.Ingredients.Update(ctx, &ingredientsmodels.Ingredient{ID: ingredient.ID, Description: "updated"})
	testutil.Ok(t, err)
	assertHydrated(t, updatedIngredient.Tags, updatedIngredient.CedarEntity(), want)

	ingredientPage, err := f.Ingredients.List(ctx, ingredients.ListRequest{})
	testutil.Ok(t, err)
	drinkPage, err := f.Drinks.List(ctx, drinks.ListRequest{})
	testutil.Ok(t, err)
	stockPage, err := f.Inventory.List(ctx, inventory.ListRequest{})
	testutil.Ok(t, err)
	menuPage, err := f.Menus.List(ctx, menus.ListRequest{})
	testutil.Ok(t, err)
	orderPage, err := f.Orders.List(ctx, orders.ListRequest{})
	testutil.Ok(t, err)

	testutil.Equals(t, ingredientPage.Items[0].Tags, want)
	testutil.Equals(t, drinkPage.Items[0].Tags, want)
	testutil.Equals(t, stockPage.Items[0].Tags, want)
	testutil.Equals(t, menuPage.Items[0].Tags, want)
	testutil.Equals(t, orderPage.Items[0].Tags, want)
}

func TestTagAssociationsRemainIsolatedAndFollowDeletionSemantics(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Deletion Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	stock := testutil.SetInventory(t, f, inventorymodels.Update{
		IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz),
		CostPerUnit: money.NewPriceFromCents(100, currency.USD),
	})
	value := tag.Tag{Key: "lifecycle", Value: "test"}
	_, err := f.App.Tags.Upsert(ctx, ingredient.EntityUID(), value)
	testutil.Ok(t, err)
	_, err = f.App.Tags.Upsert(ctx, stock.EntityUID(), value)
	testutil.Ok(t, err)

	_, err = f.Ingredients.Delete(ctx, ingredient.ID)
	testutil.Ok(t, err)
	repository := tagging.NewRepository(f.Store)
	ingredientTags, err := repository.List(ctx, ingredient.EntityUID())
	testutil.Ok(t, err)
	stockTags, err := repository.List(ctx, stock.EntityUID())
	testutil.Ok(t, err)
	testutil.Equals(t, ingredientTags, tag.Tags{value})
	testutil.Equals(t, stockTags, tag.Tags(nil))
}

func TestHydratedTagsPersistAcrossApplicationRestart(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "restart.db")
	principal, err := authn.ParseActor("owner")
	testutil.Ok(t, err)
	baseCtx := appLog.ToContext(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	baseCtx = authn.ToContext(baseCtx, principal)
	requestCtx := middleware.NewContext(baseCtx)

	s, err := store.Open(baseCtx, path)
	testutil.Ok(t, err)
	first := app.New(baseCtx, app.Config{Store: s})
	ingredient, err := first.Ingredients.Create(requestCtx, &ingredientsmodels.Ingredient{
		Name: "Restart Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	testutil.Ok(t, err)
	want := tag.Tags{{Key: "region", Value: "west"}}
	_, err = first.Tags.Upsert(requestCtx, ingredient.EntityUID(), want[0])
	testutil.Ok(t, err)
	testutil.Ok(t, first.Close())

	s, err = store.Open(baseCtx, path)
	testutil.Ok(t, err)
	second := app.New(baseCtx, app.Config{Store: s})
	t.Cleanup(func() { testutil.Ok(t, second.Close()) })
	got, err := second.Ingredients.Get(requestCtx, ingredient.ID)
	testutil.Ok(t, err)
	assertHydrated(t, got.Tags, got.CedarEntity(), want)
}

func assertHydrated(t *testing.T, got tag.Tags, entity cedar.Entity, want tag.Tags) {
	t.Helper()
	testutil.Equals(t, got, want)
	testutil.Equals(t, entity.Tags, cedar.NewRecord(tagRecord(want)))
}

func tagRecord(values tag.Tags) cedar.RecordMap {
	out := make(cedar.RecordMap, len(values))
	for _, value := range values {
		out[cedar.String(value.Key)] = cedar.String(value.Value)
	}
	return out
}

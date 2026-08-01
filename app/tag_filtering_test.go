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
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestOperationalListsFilterCanonicalTags(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	for _, target := range operationalTagTargets(t, f) {
		_, err := f.App.Tags.Upsert(ctx, target.uid, tag.Tag{Key: "featured"})
		testutil.Ok(t, err)
		_, err = f.App.Tags.Upsert(ctx, target.uid, tag.Tag{Key: "region", Value: "west"})
		testutil.Ok(t, err)

		for _, expression := range []string{
			`tags contains "featured"`,
			`tags contains "region=west"`,
			`!(tags contains "missing") && (tags contains "featured" || id == "missing")`,
		} {
			got := listUIDs(t, f, target.uid.Type, expression, "")
			testutil.Equals(t, got, []cedar.EntityUID{target.uid})
		}
		got := listUIDs(t, f, target.uid.Type, `tags contains "region=east"`, "")
		testutil.Equals(t, len(got), 0)
	}
}

func TestTagFilteredPagingFillsPagesWithoutDuplicates(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	for i := range 5 {
		ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
			Name: "Paged Ingredient " + string(rune('A'+i)), Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitMl,
		})
		if i%2 == 0 {
			_, err := f.App.Tags.Upsert(ctx, ingredient.EntityUID(), tag.Tag{Key: "paged"})
			testutil.Ok(t, err)
		}
	}

	first, err := f.Ingredients.List(ctx, ingredients.ListRequest{Filter: `tags contains "paged"`, Limit: 2})
	testutil.Ok(t, err)
	testutil.Equals(t, len(first.Items), 2)
	testutil.ErrorIf(t, first.Next == "", "expected a second filtered page")
	second, err := f.Ingredients.List(ctx, ingredients.ListRequest{Filter: `tags contains "paged"`, Cursor: first.Next, Limit: 2})
	testutil.Ok(t, err)
	testutil.Equals(t, len(second.Items), 1)
	testutil.Equals(t, second.Next, paging.Cursor(""))
	seen := map[entity.IngredientID]bool{}
	for _, item := range append(first.Items, second.Items...) {
		testutil.ErrorIf(t, seen[item.ID], "duplicate ingredient %s", item.ID)
		seen[item.ID] = true
	}
	testutil.Equals(t, len(seen), 3)
}

func TestTagFilteringPrecedesAuthorizationElision(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	base := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Authorization Base", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})

	for i, category := range []drinksmodels.DrinkCategory{
		drinksmodels.DrinkCategoryCocktail,
		drinksmodels.DrinkCategoryWine,
		drinksmodels.DrinkCategoryHighball,
		drinksmodels.DrinkCategoryWine,
	} {
		drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
			Name: "Visible Tagged " + string(rune('A'+i)), Category: category, Glass: drinksmodels.GlassTypeCoupe,
			Recipe: drinksmodels.Recipe{
				Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
				Steps:       []string{"serve"},
			},
		})
		_, err := f.App.Tags.Upsert(ctx, drink.EntityUID(), tag.Tag{Key: "visible"})
		testutil.Ok(t, err)
	}

	page, err := f.Drinks.List(f.ActorContext("sommelier"), drinks.ListRequest{
		Filter: `name.startsWith("Visible Tagged") && tags contains "visible"`, Limit: 2,
	})
	testutil.Ok(t, err)
	testutil.Equals(t, len(page.Items), 2)
	for _, item := range page.Items {
		testutil.Equals(t, item.Category, drinksmodels.DrinkCategoryWine)
	}
}

func listUIDs(t *testing.T, f *testutil.Fixture, typ cedar.EntityType, expression string, cursor paging.Cursor) []cedar.EntityUID {
	t.Helper()
	ctx := f.OwnerContext()
	switch typ {
	case entity.TypeDrink:
		page, err := f.Drinks.List(ctx, drinks.ListRequest{Filter: expression, Cursor: cursor})
		testutil.Ok(t, err)
		return mapUIDs(page.Items, func(v *drinksmodels.Drink) cedar.EntityUID { return v.EntityUID() })
	case entity.TypeIngredient:
		page, err := f.Ingredients.List(ctx, ingredients.ListRequest{Filter: expression, Cursor: cursor})
		testutil.Ok(t, err)
		return mapUIDs(page.Items, func(v *ingredientsmodels.Ingredient) cedar.EntityUID { return v.EntityUID() })
	case entity.TypeInventory:
		page, err := f.Inventory.List(ctx, inventory.ListRequest{Filter: expression, Cursor: cursor})
		testutil.Ok(t, err)
		return mapUIDs(page.Items, func(v *inventorymodels.Inventory) cedar.EntityUID { return v.EntityUID() })
	case entity.TypeMenu:
		page, err := f.Menus.List(ctx, menus.ListRequest{Filter: expression, Cursor: cursor})
		testutil.Ok(t, err)
		return mapUIDs(page.Items, func(v *menusmodels.Menu) cedar.EntityUID { return v.EntityUID() })
	case entity.TypeOrder:
		page, err := f.Orders.List(ctx, orders.ListRequest{Filter: expression, Cursor: cursor})
		testutil.Ok(t, err)
		return mapUIDs(page.Items, func(v *ordersmodels.Order) cedar.EntityUID { return v.EntityUID() })
	default:
		testutil.ErrorIf(t, true, "unexpected entity type %s", typ)
		return nil
	}
}

func mapUIDs[T any](values []T, uid func(T) cedar.EntityUID) []cedar.EntityUID {
	out := make([]cedar.EntityUID, len(values))
	for i, value := range values {
		out[i] = uid(value)
	}
	return out
}

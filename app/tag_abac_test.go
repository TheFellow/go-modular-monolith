package app_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDrinkAudienceTagExtendsSommelierReadPolicy(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	owner := f.OwnerContext()
	manager := f.ActorContext("manager")
	sommelier := f.ActorContext("sommelier")
	base := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "ABAC Cellar Base", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})
	recipe := drinksmodels.Recipe{
		Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
		Steps:       []string{"serve"},
	}

	wine := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "ABAC Cellar Wine", Category: drinksmodels.DrinkCategoryWine, Glass: drinksmodels.GlassTypeCoupe, Recipe: recipe,
	})
	cocktail := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "ABAC Cellar Cocktail", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe, Recipe: recipe,
	})
	for _, target := range []*drinksmodels.Drink{wine, cocktail} {
		_, err := f.App.Tags.Upsert(owner, target.EntityUID(), tag.Tag{Key: "collection", Value: "cellar"})
		testutil.Ok(t, err)
	}
	_, err := f.App.Tags.Upsert(owner, cocktail.EntityUID(), tag.Tag{Key: "featured"})
	testutil.Ok(t, err)

	// Existing category policy still admits wine and rejects a non-wine drink.
	_, err = f.Drinks.Get(sommelier, wine.ID)
	testutil.Ok(t, err)
	_, err = f.Drinks.Get(sommelier, cocktail.ID)
	testutil.ErrorIsPermission(t, err)
	assertDrinkListIDs(t, f, sommelier, `tags contains "collection=cellar"`, wine.EntityUID())

	// Neither an unrelated key/value tag nor a label has implicit policy meaning,
	// and the denied principal cannot tag the drink to escalate its own access.
	_, err = f.App.Tags.Upsert(sommelier, cocktail.EntityUID(), tag.Tag{Key: "audience", Value: "sommelier"})
	testutil.ErrorIsPermission(t, err)
	_, err = f.Drinks.Get(sommelier, cocktail.ID)
	testutil.ErrorIsPermission(t, err)

	// A manager already authorized to mutate this drink can opt it into the
	// Cedar-authored audience policy. The same tag composes with list filtering.
	_, err = f.App.Tags.Upsert(manager, cocktail.EntityUID(), tag.Tag{Key: "audience", Value: "sommelier"})
	testutil.Ok(t, err)
	got, err := f.Drinks.Get(sommelier, cocktail.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Tags, tag.Tags{
		{Key: "audience", Value: "sommelier"},
		{Key: "collection", Value: "cellar"},
		{Key: "featured"},
	})
	assertDrinkListIDs(t, f, sommelier, `tags contains "collection=cellar"`, wine.EntityUID(), cocktail.EntityUID())
	assertDrinkListIDs(t, f, sommelier, `tags contains "audience=sommelier"`, cocktail.EntityUID())

	// Replacing the value and removing the key both revoke only the tag-derived
	// grant. Existing category rules continue to authorize the wine.
	_, err = f.App.Tags.Upsert(manager, cocktail.EntityUID(), tag.Tag{Key: "audience", Value: "bartender"})
	testutil.Ok(t, err)
	_, err = f.Drinks.Get(sommelier, cocktail.ID)
	testutil.ErrorIsPermission(t, err)
	assertDrinkListIDs(t, f, sommelier, `tags contains "collection=cellar"`, wine.EntityUID())

	_, err = f.App.Tags.Upsert(manager, cocktail.EntityUID(), tag.Tag{Key: "audience", Value: "sommelier"})
	testutil.Ok(t, err)
	_, err = f.App.Tags.Remove(manager, cocktail.EntityUID(), "audience")
	testutil.Ok(t, err)
	_, err = f.Drinks.Get(sommelier, cocktail.ID)
	testutil.ErrorIsPermission(t, err)
	assertDrinkListIDs(t, f, sommelier, `tags contains "collection=cellar"`, wine.EntityUID())

	_, err = f.Drinks.Get(f.ActorContext("bartender"), cocktail.ID)
	testutil.Ok(t, err)
}

func assertDrinkListIDs(
	t *testing.T,
	f *testutil.Fixture,
	ctx *middleware.Context,
	filter string,
	want ...cedar.EntityUID,
) {
	t.Helper()
	page, err := f.Drinks.List(ctx, drinks.ListRequest{Filter: filter})
	testutil.Ok(t, err)
	got := make([]cedar.EntityUID, len(page.Items))
	for i, item := range page.Items {
		got[i] = item.EntityUID()
	}
	testutil.Equals(t, got, want, cmpopts.SortSlices(func(a, b cedar.EntityUID) bool {
		return a.String() < b.String()
	}))
}

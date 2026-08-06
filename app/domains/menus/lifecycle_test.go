package menus_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMenuLifecycleCommandsRejectInvalidStateWithoutMutation(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	ctx := fix.OwnerContext()

	empty := testutil.CreateMenu(t, fix, "Empty menu")
	_, err := fix.Menus.Publish(ctx, &models.Menu{ID: empty.ID})
	testutil.ErrorIsFailedPrecondition(t, err)
	got, err := fix.Menus.Get(ctx, empty.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, empty, cmpopts.EquateEmpty())

	drink := createMenuTestDrink(t, fix, "Lifecycle drink")
	published := testutil.CreateMenu(t, fix, "Published menu", testutil.WithDrink(drink), testutil.Published())
	publishedBefore := *published

	_, err = fix.Menus.Publish(ctx, &models.Menu{ID: published.ID})
	testutil.ErrorIsFailedPrecondition(t, err)
	got, err = fix.Menus.Get(ctx, published.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, &publishedBefore, cmpopts.EquateEmpty())

	drafted, err := fix.Menus.Draft(ctx, &models.Menu{ID: published.ID})
	testutil.Ok(t, err)
	draftBefore := *drafted
	_, err = fix.Menus.Draft(ctx, &models.Menu{ID: drafted.ID})
	testutil.ErrorIsFailedPrecondition(t, err)
	got, err = fix.Menus.Get(ctx, drafted.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, &draftBefore, cmpopts.EquateEmpty())
}

func TestMenuReadinessBlocksKnownUnavailableDraft(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Unstocked", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Unavailable", Category: drinksmodels.DrinkCategoryCocktail,
		Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, ingredient.Unit)}}, Steps: []string{"Mix"}},
	})
	menu := testutil.CreateMenu(t, f, "Not ready", testutil.WithDrink(drink))

	report, err := f.Menus.Readiness(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, !report.HasBlockers(), "expected readiness blocker")
	_, err = f.Menus.Publish(ctx, &models.Menu{ID: menu.ID})
	testutil.ErrorIsFailedPrecondition(t, err)
	got, err := f.Menus.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Status, models.MenuStatusDraft)
}

func TestPublishedMenuDegradesWithTemporarySubstitutionAndRemainsPublished(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	primary := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Primary spirit", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	substitute := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Temporary spirit", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: primary.ID, Amount: measurement.MustAmount(10, primary.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: substitute.ID, Amount: measurement.MustAmount(10, substitute.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Substitutable", Category: drinksmodels.DrinkCategoryCocktail,
		Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: primary.ID, Amount: measurement.MustAmount(1, primary.Unit), Substitutes: []entity.IngredientID{substitute.ID}}}, Steps: []string{"Mix"}},
	})
	menu := testutil.CreateMenu(t, f, "Published degradation", testutil.WithDrink(drink), testutil.Published())

	_, err := f.Ingredients.Retire(ctx, primary.ID, ingredientsmodels.Retirement{})
	testutil.Ok(t, err)
	got, err := f.Menus.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Status, models.MenuStatusPublished)
	testutil.Equals(t, got.Items[0].Availability, models.AvailabilityLimited)
	report, err := f.Menus.Readiness(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, !report.HasBlockers(), "expected degraded published menu blockers")
	var temporary bool
	for _, finding := range report.Findings {
		temporary = temporary || finding.Code == models.ReadinessTemporarySubstitution
	}
	testutil.ErrorIf(t, !temporary, "expected temporary substitution finding")
}

func createMenuTestDrink(t testing.TB, fix *testutil.Fixture, name string) *drinksmodels.Drink {
	t.Helper()
	ingredient := testutil.CreateIngredient(t, fix, ingredientsmodels.Ingredient{
		Name: name + " ingredient", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})
	testutil.SetInventory(t, fix, inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	return testutil.CreateDrink(t, fix, drinksmodels.Drink{
		Name: name, Category: drinksmodels.DrinkCategoryCocktail,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"Mix"},
		},
	})
}

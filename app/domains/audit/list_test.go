package audit_test

import (
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/cedar-policy/cedar-go"
)

func TestAudit_RecordsActivityForCommand(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	created, err := f.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{
		Name:     "Vodka",
		Category: ingredientsmodels.CategorySpirit,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	entries, err := f.App.Audit.List(ctx, audit.ListRequest{})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(entries) != 1, "expected 1 audit entry, got %d", len(entries))
	entry := entries[0]
	testutil.ErrorIf(t, entry.Action != ingredientsauthz.ActionCreate.String(), "expected action %q, got %q", ingredientsauthz.ActionCreate.String(), entry.Action)
	testutil.ErrorIf(t, entry.Principal != ctx.Principal(), "expected principal %s, got %s", ctx.Principal(), entry.Principal)
	testutil.ErrorIf(t, !touchesContain(entry.Touches, created.ID.EntityUID()), "expected touches to include %s", created.ID.String())
}

func TestAudit_TouchesIncludeHandlerUpdates(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	ingredient, err := f.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{
		Name:     "Gin",
		Category: ingredientsmodels.CategorySpirit,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	drink, err := f.Drinks.Create(ctx, &drinksmodels.Drink{
		Name:     "Gin and Tonic",
		Category: drinksmodels.DrinkCategoryHighball,
		Glass:    drinksmodels.GlassTypeHighball,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{
					IngredientID: ingredient.ID,
					Amount:       measurement.MustAmount(2, measurement.UnitOz),
				},
			},
			Steps: []string{"Build in glass"},
		},
	})
	testutil.Ok(t, err)

	menu, err := f.Menu.Create(ctx, &menumodels.Menu{Name: "Happy Hour"})
	testutil.Ok(t, err)

	_, err = f.Menu.AddDrink(ctx, &menumodels.MenuPatch{
		MenuID:  menu.ID,
		DrinkID: drink.ID,
	})
	testutil.Ok(t, err)

	_, err = f.Drinks.Delete(ctx, drink.ID)
	testutil.Ok(t, err)

	entries, err := f.App.Audit.List(ctx, audit.ListRequest{Action: drinksauthz.ActionDelete})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(entries) != 1, "expected 1 audit entry, got %d", len(entries))
	entry := entries[0]
	testutil.ErrorIf(t, !touchesContain(entry.Touches, drink.ID.EntityUID()), "expected touches to include drink %s", drink.ID.String())
	testutil.ErrorIf(t, !touchesContain(entry.Touches, menu.ID.EntityUID()), "expected touches to include menu %s", menu.ID.String())
}

func TestAudit_TouchesIncludeIngredientUpdateDrinks(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	ingredient, err := f.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{
		Name:     "Gin",
		Category: ingredientsmodels.CategorySpirit,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	drink, err := f.Drinks.Create(ctx, &drinksmodels.Drink{
		Name:     "Gin and Tonic",
		Category: drinksmodels.DrinkCategoryHighball,
		Glass:    drinksmodels.GlassTypeHighball,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{
					IngredientID: ingredient.ID,
					Amount:       measurement.MustAmount(2, measurement.UnitOz),
				},
			},
			Steps: []string{"Build in glass"},
		},
	})
	testutil.Ok(t, err)

	_, err = f.Ingredients.Update(ctx, &ingredientsmodels.Ingredient{
		ID:   ingredient.ID,
		Name: "Gin (Updated)",
	})
	testutil.Ok(t, err)

	entries, err := f.App.Audit.List(ctx, audit.ListRequest{Action: ingredientsauthz.ActionUpdate})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(entries) != 1, "expected 1 audit entry, got %d", len(entries))
	entry := entries[0]
	testutil.ErrorIf(t, !touchesContain(entry.Touches, ingredient.ID.EntityUID()), "expected touches to include ingredient %s", ingredient.ID.String())
	testutil.ErrorIf(t, !touchesContain(entry.Touches, drink.ID.EntityUID()), "expected touches to include drink %s", drink.ID.String())
}

func TestAudit_TouchesIncludeIngredientUpdateMenus(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	ingredient, err := f.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{
		Name:     "Lime Juice",
		Category: ingredientsmodels.CategoryJuice,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	drink, err := f.Drinks.Create(ctx, &drinksmodels.Drink{
		Name:     "Gimlet",
		Category: drinksmodels.DrinkCategoryCocktail,
		Glass:    drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{
					IngredientID: ingredient.ID,
					Amount:       measurement.MustAmount(1, measurement.UnitOz),
				},
			},
			Steps: []string{"Shake"},
		},
	})
	testutil.Ok(t, err)

	menu, err := f.Menu.Create(ctx, &menumodels.Menu{Name: "Citrus Menu"})
	testutil.Ok(t, err)
	_, err = f.Menu.AddDrink(ctx, &menumodels.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)

	_, err = f.Ingredients.Update(ctx, &ingredientsmodels.Ingredient{
		ID:   ingredient.ID,
		Name: "Fresh Lime Juice",
	})
	testutil.Ok(t, err)

	entries, err := f.App.Audit.List(ctx, audit.ListRequest{Action: ingredientsauthz.ActionUpdate})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(entries) != 1, "expected 1 audit entry, got %d", len(entries))
	entry := entries[0]
	testutil.ErrorIf(t, !touchesContain(entry.Touches, menu.ID.EntityUID()), "expected touches to include menu %s", menu.ID.String())
}

func TestAudit_ListFilters(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	ing1, err := f.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{
		Name:     "Bourbon",
		Category: ingredientsmodels.CategorySpirit,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	_, err = f.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{
		Name:     "Vermouth",
		Category: ingredientsmodels.CategoryOther,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	_, err = f.Ingredients.Update(ctx, &ingredientsmodels.Ingredient{
		ID:   ing1.ID,
		Name: "Bourbon (Updated)",
	})
	testutil.Ok(t, err)

	ownerEntries, err := f.App.Audit.List(ctx, audit.ListRequest{Principal: ctx.Principal()})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(ownerEntries) != 3, "expected 3 owner entries, got %d", len(ownerEntries))

	anonymousEntries, err := f.App.Audit.List(ctx, audit.ListRequest{Principal: authn.Anonymous()})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(anonymousEntries) != 0, "expected 0 anonymous entries, got %d", len(anonymousEntries))

	updateEntries, err := f.App.Audit.List(ctx, audit.ListRequest{Action: ingredientsauthz.ActionUpdate})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(updateEntries) != 1, "expected 1 update entry, got %d", len(updateEntries))

	entityEntries, err := f.App.Audit.List(ctx, audit.ListRequest{Entity: ing1.ID.EntityUID()})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(entityEntries) != 2, "expected 2 entries for entity, got %d", len(entityEntries))
}

func TestAudit_ListFiltersByTime(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	_, err := f.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{
		Name:     "Lime Juice",
		Category: ingredientsmodels.CategoryJuice,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	cutoff := time.Now().UTC()
	time.Sleep(10 * time.Millisecond)

	_, err = f.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{
		Name:     "Simple Syrup",
		Category: ingredientsmodels.CategorySyrup,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	afterEntries, err := f.App.Audit.List(ctx, audit.ListRequest{From: cutoff})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(afterEntries) != 1, "expected 1 entry after cutoff, got %d", len(afterEntries))

	beforeEntries, err := f.App.Audit.List(ctx, audit.ListRequest{To: cutoff})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(beforeEntries) != 1, "expected 1 entry before cutoff, got %d", len(beforeEntries))
}

func touchesContain(touches []cedar.EntityUID, uid cedar.EntityUID) bool {
	for _, touch := range touches {
		if touch == uid {
			return true
		}
	}
	return false
}

package audit_test

import (
	"slices"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/cedar-policy/cedar-go"
)

func TestAudit_ListPageUsesCursorWithoutDuplicates(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	for _, name := range []string{"A", "B", "C", "D", "E"} {
		_, err := f.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{
			Name: name, Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
		})
		testutil.Ok(t, err)
	}

	var got []string
	var cursor paging.Cursor
	for {
		page, err := f.App.Audit.List(ctx, audit.ListRequest{Cursor: cursor, Limit: 2})
		testutil.Ok(t, err)
		for _, entry := range page.Items {
			got = append(got, entry.ID.String())
		}
		if page.Next == "" {
			break
		}
		cursor = page.Next
	}

	if len(got) != 5 {
		t.Fatalf("expected 5 entries across pages, got %d", len(got))
	}
	seen := map[string]bool{}
	for _, id := range got {
		if seen[id] {
			t.Fatalf("duplicate entry %q across cursor pages", id)
		}
		seen[id] = true
	}
}

func TestAudit_ListPageRejectsInvalidCursor(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	_, err := f.App.Audit.List(f.OwnerContext(), audit.ListRequest{
		Cursor: "not-an-audit-entry", Limit: 10,
	})
	testutil.ErrorIsInvalid(t, err)
}

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

	page, err := f.App.Audit.List(ctx, audit.ListRequest{})
	testutil.Ok(t, err)
	entries := page.Items
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

	menu, err := f.Menus.Create(ctx, &menumodels.Menu{Name: "Happy Hour"})
	testutil.Ok(t, err)

	_, err = f.Menus.AddDrink(ctx, &menumodels.MenuPatch{
		MenuID:  menu.ID,
		DrinkID: drink.ID,
	})
	testutil.Ok(t, err)

	_, err = f.Drinks.Delete(ctx, drink.ID)
	testutil.Ok(t, err)

	page, err := f.App.Audit.List(ctx, audit.ListRequest{Action: drinksauthz.ActionDelete})
	testutil.Ok(t, err)
	entries := page.Items
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

	page, err := f.App.Audit.List(ctx, audit.ListRequest{Action: ingredientsauthz.ActionUpdate})
	testutil.Ok(t, err)
	entries := page.Items
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

	menu, err := f.Menus.Create(ctx, &menumodels.Menu{Name: "Citrus Menu"})
	testutil.Ok(t, err)
	_, err = f.Menus.AddDrink(ctx, &menumodels.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)

	_, err = f.Ingredients.Update(ctx, &ingredientsmodels.Ingredient{
		ID:   ingredient.ID,
		Name: "Fresh Lime Juice",
	})
	testutil.Ok(t, err)

	page, err := f.App.Audit.List(ctx, audit.ListRequest{Action: ingredientsauthz.ActionUpdate})
	testutil.Ok(t, err)
	entries := page.Items
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
	testutil.ErrorIf(t, len(ownerEntries.Items) != 3, "expected 3 owner entries, got %d", len(ownerEntries.Items))

	anonymousEntries, err := f.App.Audit.List(ctx, audit.ListRequest{Principal: authn.Anonymous()})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(anonymousEntries.Items) != 0, "expected 0 anonymous entries, got %d", len(anonymousEntries.Items))

	updateEntries, err := f.App.Audit.List(ctx, audit.ListRequest{Action: ingredientsauthz.ActionUpdate})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(updateEntries.Items) != 1, "expected 1 update entry, got %d", len(updateEntries.Items))

	entityEntries, err := f.App.Audit.List(ctx, audit.ListRequest{Entity: ing1.ID.EntityUID()})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(entityEntries.Items) != 2, "expected 2 entries for entity, got %d", len(entityEntries.Items))
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
	testutil.ErrorIf(t, len(afterEntries.Items) != 1, "expected 1 entry after cutoff, got %d", len(afterEntries.Items))

	beforeEntries, err := f.App.Audit.List(ctx, audit.ListRequest{To: cutoff})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(beforeEntries.Items) != 1, "expected 1 entry before cutoff, got %d", len(beforeEntries.Items))
}

func touchesContain(touches []cedar.EntityUID, uid cedar.EntityUID) bool {
	return slices.Contains(touches, uid)
}

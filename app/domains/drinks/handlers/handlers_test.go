package handlers_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestIngredientDeletedHandlersRemoveDependentsAndPreserveUnrelatedEntities(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	target := b.WithIngredientModel(ingredientsmodels.Ingredient{Name: "Target", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	other := b.WithIngredientModel(ingredientsmodels.Ingredient{Name: "Other", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	targetStock := b.WithInventory(target, 10)
	otherStock := b.WithInventory(other, 10)
	affectedA := handlerDrink(b, "Affected A", target)
	affectedB := handlerDrink(b, "Affected B", target)
	survivor := handlerDrink(b, "Survivor", other)
	affectedMenu := b.WithPublishedMenu(menumodels.Menu{Name: "Affected"}, affectedA, survivor, affectedB)
	unrelatedMenu := b.WithPublishedMenu(menumodels.Menu{Name: "Unrelated"}, survivor)

	_, err := f.Ingredients.Delete(ctx, target.ID)
	testutil.Ok(t, err)
	_, err = f.Drinks.Get(ctx, affectedA.ID)
	testutil.ErrorIsNotFound(t, err)
	_, err = f.Drinks.Get(ctx, affectedB.ID)
	testutil.ErrorIsNotFound(t, err)
	_, err = f.Inventory.Get(ctx, target.ID)
	testutil.ErrorIsNotFound(t, err)
	gotSurvivor, err := f.Drinks.Get(ctx, survivor.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotSurvivor, survivor, testutil.EquateAmounts(0.000001), cmpopts.EquateEmpty())
	gotOtherStock, err := f.Inventory.Get(ctx, other.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotOtherStock, otherStock, testutil.EquateAmounts(0.000001))
	gotMenu, err := f.Menus.Get(ctx, affectedMenu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, menuDrinkIDs(gotMenu), []entity.DrinkID{survivor.ID})
	gotUnrelatedMenu, err := f.Menus.Get(ctx, unrelatedMenu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotUnrelatedMenu, unrelatedMenu, cmpopts.EquateEmpty())

	entry := f.LatestAuditEntry(ingredientsauthz.ActionDelete)
	testutil.AuditTouches(t, entry,
		target.ID.EntityUID(), affectedA.ID.EntityUID(), affectedB.ID.EntityUID(),
		targetStock.EntityUID(), affectedMenu.ID.EntityUID(),
	)
}

func TestIngredientUpdatedHandlersAuditEveryDependentWithoutMutatingThem(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	target := b.WithIngredientModel(ingredientsmodels.Ingredient{Name: "Target", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	other := b.WithIngredientModel(ingredientsmodels.Ingredient{Name: "Other", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	affectedA := handlerDrink(b, "Affected A", target)
	affectedB := handlerDrink(b, "Affected B", target)
	survivor := handlerDrink(b, "Survivor", other)
	menuA := b.WithMenu("Menu A")
	menuA = addDrinks(t, f, menuA, affectedA, affectedB)
	menuB := b.WithMenu("Menu B")
	menuB = addDrinks(t, f, menuB, affectedA)
	unrelatedMenu := b.WithMenu("Unrelated")
	unrelatedMenu = addDrinks(t, f, unrelatedMenu, survivor)

	_, err := f.Ingredients.Update(ctx, &ingredientsmodels.Ingredient{ID: target.ID, Name: "Renamed Target"})
	testutil.Ok(t, err)
	for _, want := range []*drinksmodels.Drink{affectedA, affectedB, survivor} {
		got, err := f.Drinks.Get(ctx, want.ID)
		testutil.Ok(t, err)
		testutil.Equals(t, got, want, testutil.EquateAmounts(0.000001), cmpopts.EquateEmpty())
	}
	for _, want := range []*menumodels.Menu{menuA, menuB, unrelatedMenu} {
		got, err := f.Menus.Get(ctx, want.ID)
		testutil.Ok(t, err)
		testutil.Equals(t, got, want, cmpopts.EquateEmpty())
	}

	entry := f.LatestAuditEntry(ingredientsauthz.ActionUpdate)
	testutil.AuditTouches(t, entry,
		target.ID.EntityUID(), affectedA.ID.EntityUID(), affectedB.ID.EntityUID(),
		menuA.ID.EntityUID(), menuB.ID.EntityUID(),
	)
}

func handlerDrink(b *testutil.Bootstrap, name string, ingredient *ingredientsmodels.Ingredient) *drinksmodels.Drink {
	return b.WithDrink(drinksmodels.Drink{
		Name: name, Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, ingredient.Unit)}},
			Steps:       []string{"Mix"},
		},
	})
}

func addDrinks(t *testing.T, f *testutil.Fixture, menu *menumodels.Menu, drinks ...*drinksmodels.Drink) *menumodels.Menu {
	t.Helper()
	for _, drink := range drinks {
		var err error
		menu, err = f.Menus.AddDrink(f.OwnerContext(), &menumodels.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
		testutil.Ok(t, err)
	}
	return menu
}

func menuDrinkIDs(menu *menumodels.Menu) []entity.DrinkID {
	ids := make([]entity.DrinkID, 0, len(menu.Items))
	for _, item := range menu.Items {
		ids = append(ids, item.DrinkID)
	}
	return ids
}

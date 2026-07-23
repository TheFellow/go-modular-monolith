package handlers_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestIngredientDeletedHandlersRemoveDependentsAndPreserveUnrelatedEntities(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	target := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Target", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	other := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Other", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	targetStock := testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: target.ID, Amount: measurement.MustAmount(10, target.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	otherStock := testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: other.ID, Amount: measurement.MustAmount(10, other.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	affectedA := testutil.CreateDrink(t, f, handlerDrink("Affected A", target.ID))
	affectedB := testutil.CreateDrink(t, f, handlerDrink("Affected B", target.ID))
	survivor := testutil.CreateDrink(t, f, handlerDrink("Survivor", other.ID))
	affectedMenu := testutil.CreateMenu(t, f, "Affected", testutil.WithDrink(affectedA), testutil.WithDrink(survivor), testutil.WithDrink(affectedB), testutil.Published())
	unrelatedMenu := testutil.CreateMenu(t, f, "Unrelated", testutil.WithDrink(survivor), testutil.Published())

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
	testutil.Equals(t, gotSurvivor, survivor, cmpopts.EquateEmpty())
	gotOtherStock, err := f.Inventory.Get(ctx, other.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotOtherStock, otherStock)
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
	ctx := f.OwnerContext()

	target := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Target", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	other := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Other", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	affectedA := testutil.CreateDrink(t, f, handlerDrink("Affected A", target.ID))
	affectedB := testutil.CreateDrink(t, f, handlerDrink("Affected B", target.ID))
	survivor := testutil.CreateDrink(t, f, handlerDrink("Survivor", other.ID))
	menuA := testutil.CreateMenu(t, f, "Menu A", testutil.WithDrink(affectedA), testutil.WithDrink(affectedB))
	menuB := testutil.CreateMenu(t, f, "Menu B", testutil.WithDrink(affectedA))
	unrelatedMenu := testutil.CreateMenu(t, f, "Unrelated", testutil.WithDrink(survivor))

	_, err := f.Ingredients.Update(ctx, &ingredientsmodels.Ingredient{ID: target.ID, Name: "Renamed Target"})
	testutil.Ok(t, err)
	for _, want := range []*drinksmodels.Drink{affectedA, affectedB, survivor} {
		got, err := f.Drinks.Get(ctx, want.ID)
		testutil.Ok(t, err)
		testutil.Equals(t, got, want, cmpopts.EquateEmpty())
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

func handlerDrink(name string, ingredientID entity.IngredientID) drinksmodels.Drink {
	return drinksmodels.Drink{
		Name: name, Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredientID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"Mix"},
		},
	}
}

func menuDrinkIDs(menu *menumodels.Menu) []entity.DrinkID {
	ids := make([]entity.DrinkID, 0, len(menu.Items))
	for _, item := range menu.Items {
		ids = append(ids, item.DrinkID)
	}
	return ids
}

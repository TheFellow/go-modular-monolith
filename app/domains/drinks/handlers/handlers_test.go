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

	target := b.WithIngredient("Target", measurement.UnitOz)
	other := b.WithIngredient("Other", measurement.UnitOz)
	targetStock := b.WithInventory(target, 10)
	otherStock := b.WithInventory(other, 10)
	affectedA := f.CreateDrink("Affected A").WithIngredient(target, 1).Build()
	affectedB := f.CreateDrink("Affected B").WithIngredient(target, 1).Build()
	survivor := f.CreateDrink("Survivor").WithIngredient(other, 1).Build()
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

	target := b.WithIngredient("Target", measurement.UnitOz)
	other := b.WithIngredient("Other", measurement.UnitOz)
	affectedA := f.CreateDrink("Affected A").WithIngredient(target, 1).Build()
	affectedB := f.CreateDrink("Affected B").WithIngredient(target, 1).Build()
	survivor := f.CreateDrink("Survivor").WithIngredient(other, 1).Build()
	menuA := b.AddDrinks(b.WithMenu("Menu A"), affectedA, affectedB)
	menuB := b.AddDrinks(b.WithMenu("Menu B"), affectedA)
	unrelatedMenu := b.AddDrinks(b.WithMenu("Unrelated"), survivor)

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

func menuDrinkIDs(menu *menumodels.Menu) []entity.DrinkID {
	ids := make([]entity.DrinkID, 0, len(menu.Items))
	for _, item := range menu.Items {
		ids = append(ids, item.DrinkID)
	}
	return ids
}

package handlers_test

import (
	"testing"

	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menusaudit "github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDrinkDeletedHandlerRemovesOnlyDeletedDrinkFromMenus(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()
	targetIngredient := b.WithIngredient("Target", measurement.UnitOz)
	otherIngredient := b.WithIngredient("Other", measurement.UnitOz)
	b.WithInventory(targetIngredient, 10)
	b.WithInventory(otherIngredient, 10)
	target := f.CreateDrink("Target").WithIngredient(targetIngredient, 1).Build()
	survivor := f.CreateDrink("Survivor").WithIngredient(otherIngredient, 1).Build()
	affectedMenu := b.WithPublishedMenu(menumodels.Menu{Name: "Affected"}, target, survivor)
	unrelatedMenu := b.WithPublishedMenu(menumodels.Menu{Name: "Unrelated"}, survivor)

	_, err := f.Drinks.Delete(ctx, target.ID)
	testutil.Ok(t, err)
	gotAffected, err := f.Menus.Get(ctx, affectedMenu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, menuHandlerDrinkIDs(gotAffected), []entity.DrinkID{survivor.ID})
	gotUnrelated, err := f.Menus.Get(ctx, unrelatedMenu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotUnrelated, unrelatedMenu, cmpopts.EquateEmpty())
	_, err = f.Drinks.Get(ctx, target.ID)
	testutil.ErrorIsNotFound(t, err)
	gotSurvivor, err := f.Drinks.Get(ctx, survivor.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotSurvivor, survivor, testutil.EquateAmounts(0.000001), cmpopts.EquateEmpty())

	entry := f.LatestAuditEntry(drinksauthz.ActionDelete)
	testutil.AuditTouches(t, entry, target.ID.EntityUID(), affectedMenu.ID.EntityUID())
}

func TestDrinkUpdatedHandlerChangesOnlyAffectedPublishedMenuItems(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()
	base := b.WithIngredient("Base", measurement.UnitOz)
	other := b.WithIngredient("Other", measurement.UnitOz)
	rare := b.WithIngredient("Rare", measurement.UnitOz)
	b.WithInventory(base, 10)
	b.WithInventory(other, 10)
	target := f.CreateDrink("Target").WithIngredient(base, 1).Build()
	survivor := f.CreateDrink("Survivor").WithIngredient(other, 1).Build()
	affectedMenu := b.WithPublishedMenu(menumodels.Menu{Name: "Affected"}, target, survivor)
	draftMenu := b.AddDrinks(b.WithMenu("Draft"), target)
	unrelatedMenu := b.WithPublishedMenu(menumodels.Menu{Name: "Unrelated"}, survivor)

	update := *target
	update.Recipe.Ingredients = append(update.Recipe.Ingredients, drinksmodels.RecipeIngredient{
		IngredientID: rare.ID, Amount: measurement.MustAmount(1, rare.Unit),
	})
	_, err := f.Drinks.Update(ctx, &update)
	testutil.Ok(t, err)
	gotAffected, err := f.Menus.Get(ctx, affectedMenu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, menuHandlerAvailability(gotAffected, target.ID), menumodels.AvailabilityUnavailable)
	testutil.Equals(t, menuHandlerAvailability(gotAffected, survivor.ID), menumodels.AvailabilityAvailable)
	gotDraft, err := f.Menus.Get(ctx, draftMenu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotDraft, draftMenu, cmpopts.EquateEmpty())
	gotUnrelated, err := f.Menus.Get(ctx, unrelatedMenu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotUnrelated, unrelatedMenu, cmpopts.EquateEmpty())

	entry := f.LatestAuditEntry(drinksauthz.ActionUpdate)
	testutil.AuditTouches(t, entry, target.ID.EntityUID(), affectedMenu.ID.EntityUID())
}

func TestStockAdjustedHandlerRecalculatesOnlyPublishedMenusUsingIngredient(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()
	targetIngredient := b.WithIngredient("Target", measurement.UnitOz)
	otherIngredient := b.WithIngredient("Other", measurement.UnitOz)
	targetStock := b.WithInventory(targetIngredient, 10)
	b.WithInventory(otherIngredient, 10)
	target := f.CreateDrink("Target").WithIngredient(targetIngredient, 1).Build()
	survivor := f.CreateDrink("Survivor").WithIngredient(otherIngredient, 1).Build()
	affectedMenu := b.WithPublishedMenu(menumodels.Menu{Name: "Affected"}, target, survivor)
	draftMenu := b.AddDrinks(b.WithMenu("Draft"), target)
	unrelatedMenu := b.WithPublishedMenu(menumodels.Menu{Name: "Unrelated"}, survivor)

	_, err := f.Inventory.Adjust(ctx, &inventorymodels.Patch{
		IngredientID: targetIngredient.ID, Reason: inventorymodels.ReasonUsed,
		Delta: optional.Some(measurement.MustAmount(-10, targetIngredient.Unit)),
	})
	testutil.Ok(t, err)
	gotAffected, err := f.Menus.Get(ctx, affectedMenu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, menuHandlerAvailability(gotAffected, target.ID), menumodels.AvailabilityUnavailable)
	testutil.Equals(t, menuHandlerAvailability(gotAffected, survivor.ID), menumodels.AvailabilityAvailable)
	gotDraft, err := f.Menus.Get(ctx, draftMenu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotDraft, draftMenu, cmpopts.EquateEmpty())
	gotUnrelated, err := f.Menus.Get(ctx, unrelatedMenu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotUnrelated, unrelatedMenu, cmpopts.EquateEmpty())

	entry := f.LatestAuditEntry(inventoryauthz.ActionAdjust)
	testutil.AuditTouches(t, entry, targetStock.EntityUID(), affectedMenu.ID.EntityUID())
}

func TestMenuPublishedHandlerPersistsAvailabilityAndAuditsMenu(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()
	missing := b.WithIngredient("Missing", measurement.UnitOz)
	stocked := b.WithIngredient("Stocked", measurement.UnitOz)
	b.WithInventory(stocked, 10)
	unavailableDrink := f.CreateDrink("Unavailable").WithIngredient(missing, 1).Build()
	availableDrink := f.CreateDrink("Available").WithIngredient(stocked, 1).Build()
	menu := b.AddDrinks(b.WithMenu("Publish"), unavailableDrink, availableDrink)

	published, err := f.Menus.Publish(ctx, &menumodels.Menu{ID: menu.ID})
	testutil.Ok(t, err)
	got, err := f.Menus.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, published, cmpopts.EquateEmpty())
	testutil.Equals(t, menuHandlerAvailability(got, unavailableDrink.ID), menumodels.AvailabilityUnavailable)
	testutil.Equals(t, menuHandlerAvailability(got, availableDrink.ID), menumodels.AvailabilityAvailable)

	entry := f.LatestAuditEntry(menusaudit.ActionPublish)
	testutil.AuditTouches(t, entry, menu.ID.EntityUID())
}

func menuHandlerDrinkIDs(menu *menumodels.Menu) []entity.DrinkID {
	ids := make([]entity.DrinkID, 0, len(menu.Items))
	for _, item := range menu.Items {
		ids = append(ids, item.DrinkID)
	}
	return ids
}

func menuHandlerAvailability(menu *menumodels.Menu, drinkID entity.DrinkID) menumodels.Availability {
	for _, item := range menu.Items {
		if item.DrinkID == drinkID {
			return item.Availability
		}
	}
	return ""
}

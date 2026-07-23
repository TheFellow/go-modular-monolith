package handlers_test

import (
	"testing"

	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menusaudit "github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDrinkDeletedHandlerRemovesOnlyDeletedDrinkFromMenus(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	targetIngredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Target", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	otherIngredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Other", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, menuHandlerStock(targetIngredient.ID, targetIngredient.Unit, 10))
	testutil.SetInventory(t, f, menuHandlerStock(otherIngredient.ID, otherIngredient.Unit, 10))
	target := testutil.CreateDrink(t, f, menuHandlerDrink("Target", targetIngredient.ID))
	survivor := testutil.CreateDrink(t, f, menuHandlerDrink("Survivor", otherIngredient.ID))
	affectedMenu := testutil.CreateMenu(t, f, "Affected", testutil.WithDrink(target), testutil.WithDrink(survivor), testutil.Published())
	unrelatedMenu := testutil.CreateMenu(t, f, "Unrelated", testutil.WithDrink(survivor), testutil.Published())

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
	testutil.Equals(t, gotSurvivor, survivor, cmpopts.EquateEmpty())

	entry := f.LatestAuditEntry(drinksauthz.ActionDelete)
	testutil.AuditTouches(t, entry, target.ID.EntityUID(), affectedMenu.ID.EntityUID())
}

func TestDrinkUpdatedHandlerChangesOnlyAffectedPublishedMenuItems(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	base := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Base", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	other := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Other", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	rare := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Rare", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, menuHandlerStock(base.ID, base.Unit, 10))
	testutil.SetInventory(t, f, menuHandlerStock(other.ID, other.Unit, 10))
	target := testutil.CreateDrink(t, f, menuHandlerDrink("Target", base.ID))
	survivor := testutil.CreateDrink(t, f, menuHandlerDrink("Survivor", other.ID))
	affectedMenu := testutil.CreateMenu(t, f, "Affected", testutil.WithDrink(target), testutil.WithDrink(survivor), testutil.Published())
	draftMenu := testutil.CreateMenu(t, f, "Draft", testutil.WithDrink(target))
	unrelatedMenu := testutil.CreateMenu(t, f, "Unrelated", testutil.WithDrink(survivor), testutil.Published())

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
	ctx := f.OwnerContext()
	targetIngredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Target", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	otherIngredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Other", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	targetStock := testutil.SetInventory(t, f, menuHandlerStock(targetIngredient.ID, targetIngredient.Unit, 10))
	testutil.SetInventory(t, f, menuHandlerStock(otherIngredient.ID, otherIngredient.Unit, 10))
	target := testutil.CreateDrink(t, f, menuHandlerDrink("Target", targetIngredient.ID))
	survivor := testutil.CreateDrink(t, f, menuHandlerDrink("Survivor", otherIngredient.ID))
	affectedMenu := testutil.CreateMenu(t, f, "Affected", testutil.WithDrink(target), testutil.WithDrink(survivor), testutil.Published())
	draftMenu := testutil.CreateMenu(t, f, "Draft", testutil.WithDrink(target))
	unrelatedMenu := testutil.CreateMenu(t, f, "Unrelated", testutil.WithDrink(survivor), testutil.Published())

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
	ctx := f.OwnerContext()
	missing := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Missing", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	stocked := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Stocked", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, menuHandlerStock(stocked.ID, stocked.Unit, 10))
	unavailableDrink := testutil.CreateDrink(t, f, menuHandlerDrink("Unavailable", missing.ID))
	availableDrink := testutil.CreateDrink(t, f, menuHandlerDrink("Available", stocked.ID))
	menu := testutil.CreateMenu(t, f, "Publish", testutil.WithDrink(unavailableDrink), testutil.WithDrink(availableDrink))

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

func menuHandlerDrink(name string, ingredientID entity.IngredientID) drinksmodels.Drink {
	return drinksmodels.Drink{
		Name: name, Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredientID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"Mix"},
		},
	}
}

func menuHandlerStock(ingredientID entity.IngredientID, unit measurement.Unit, quantity float64) inventorymodels.Update {
	return inventorymodels.Update{
		IngredientID: ingredientID,
		Amount:       measurement.MustAmount(quantity, unit),
		CostPerUnit:  money.NewPriceFromCents(100, currency.USD),
	}
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

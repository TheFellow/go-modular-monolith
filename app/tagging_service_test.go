package app_test

import (
	"testing"

	drinkauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menuauthz "github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	orderauthz "github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

type tagTarget struct {
	name        string
	uid         cedar.EntityUID
	tagAction   cedar.EntityUID
	untagAction cedar.EntityUID
}

func operationalTagTargets(t *testing.T, f *testutil.Fixture) []tagTarget {
	t.Helper()
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Tagging Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Tagging Highball", Category: drinksmodels.DrinkCategoryHighball, Glass: drinksmodels.GlassTypeHighball,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"build"},
		},
	})
	stock := testutil.SetInventory(t, f, inventorymodels.Update{
		IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, measurement.UnitOz),
		CostPerUnit: money.NewPriceFromCents(100, currency.USD),
	})
	menu := testutil.CreateMenu(t, f, "Tagging Menu", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{
		MenuID: menu.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
	})
	return []tagTarget{
		{name: "ingredient", uid: ingredient.EntityUID(), tagAction: ingredientauthz.ActionTag, untagAction: ingredientauthz.ActionUntag},
		{name: "drink", uid: drink.EntityUID(), tagAction: drinkauthz.ActionTag, untagAction: drinkauthz.ActionUntag},
		{name: "inventory", uid: stock.EntityUID(), tagAction: inventoryauthz.ActionTag, untagAction: inventoryauthz.ActionUntag},
		{name: "menu", uid: menu.EntityUID(), tagAction: menuauthz.ActionTag, untagAction: menuauthz.ActionUntag},
		{name: "order", uid: order.EntityUID(), tagAction: orderauthz.ActionTag, untagAction: orderauthz.ActionUntag},
	}
}

func TestTagsMutateEveryOperationalTargetIdempotently(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	for _, target := range operationalTagTargets(t, f) { //nolint:paralleltest // Subtests share one application fixture.
		t.Run(target.name, func(t *testing.T) {
			first, err := f.App.Tags.Upsert(ctx, target.uid, tag.Tag{Key: "region", Value: "west"})
			testutil.Ok(t, err)
			testutil.IsTrue(t, first.Changed)
			testutil.Equals(t, first.Target, target.uid)
			testutil.Equals(t, first.Tags, tag.Tags{{Key: "region", Value: "west"}})

			entry := f.LatestAuditEntry(target.tagAction)
			testutil.Equals(t, entry.Action, target.tagAction.String())
			testutil.Equals(t, entry.Resource, target.uid)
			testutil.AuditTouches(t, entry, target.uid)
			second, err := f.App.Tags.Upsert(ctx, target.uid, tag.Tag{Key: "audience", Value: "members"})
			testutil.Ok(t, err)
			testutil.Equals(t, second.Tags, tag.Tags{
				{Key: "audience", Value: "members"},
				{Key: "region", Value: "west"},
			})

			unchanged, err := f.App.Tags.Set(ctx, target.uid, tag.Tag{Key: "region", Value: "west"})
			testutil.Ok(t, err)
			testutil.IsFalse(t, unchanged.Changed)
			testutil.Equals(t, len(f.LatestAuditEntry(target.tagAction).Touches), 0)

			replaced, err := f.App.Tags.Upsert(ctx, target.uid, tag.Tag{Key: "region", Value: "east"})
			testutil.Ok(t, err)
			testutil.IsTrue(t, replaced.Changed)
			testutil.Equals(t, replaced.Tags, tag.Tags{
				{Key: "audience", Value: "members"},
				{Key: "region", Value: "east"},
			})

			missing, err := f.App.Tags.Remove(ctx, target.uid, "missing")
			testutil.Ok(t, err)
			testutil.IsFalse(t, missing.Changed)
			testutil.Equals(t, missing.Tags, replaced.Tags)
			testutil.Equals(t, len(f.LatestAuditEntry(target.untagAction).Touches), 0)

			removed, err := f.App.Tags.Remove(ctx, target.uid, "region")
			testutil.Ok(t, err)
			testutil.IsTrue(t, removed.Changed)
			testutil.Equals(t, removed.Tags, tag.Tags{{Key: "audience", Value: "members"}})
			untagEntry := f.LatestAuditEntry(target.untagAction)
			testutil.Equals(t, untagEntry.Action, target.untagAction.String())
			testutil.Equals(t, untagEntry.Resource, target.uid)
			testutil.AuditTouches(t, untagEntry, target.uid)
		})
	}
}

func TestTagPermissionsMirrorDomainMutations(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	targets := operationalTagTargets(t, f)

	tests := []struct {
		name    string
		target  tagTarget
		actor   string
		allowed bool
	}{
		{name: "manager tags ingredient", target: targets[0], actor: "manager", allowed: true},
		{name: "bartender cannot tag ingredient", target: targets[0], actor: "bartender"},
		{name: "bartender tags non-wine drink", target: targets[1], actor: "bartender", allowed: true},
		{name: "sommelier cannot tag non-wine drink", target: targets[1], actor: "sommelier"},
		{name: "manager tags inventory", target: targets[2], actor: "manager", allowed: true},
		{name: "bartender cannot tag inventory", target: targets[2], actor: "bartender"},
		{name: "manager tags menu", target: targets[3], actor: "manager", allowed: true},
		{name: "bartender cannot tag menu", target: targets[3], actor: "bartender"},
		{name: "bartender tags order", target: targets[4], actor: "bartender", allowed: true},
		{name: "sommelier cannot tag order", target: targets[4], actor: "sommelier"},
	}

	for i, tc := range tests { //nolint:paralleltest // Subtests share target state in one application fixture.
		t.Run(tc.name, func(t *testing.T) {
			key := "permission-" + string(rune('a'+i))
			_, err := f.App.Tags.Upsert(f.ActorContext(tc.actor), tc.target.uid, tag.Tag{Key: key})
			if tc.allowed {
				testutil.Ok(t, err)
				return
			}
			testutil.ErrorIsPermission(t, err)
			persisted, listErr := tagging.NewRepository(f.Store).List(f.OwnerContext(), tc.target.uid)
			testutil.Ok(t, listErr)
			for _, value := range persisted {
				testutil.ErrorIf(t, value.Key == key, "denied tag was persisted")
			}
		})
	}
}

func TestTagPermissionsPreserveDrinkCategoryDistinctions(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Wine Base", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})
	wine := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Tagged Wine", Category: drinksmodels.DrinkCategoryWine, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"pour"},
		},
	})

	_, err := f.App.Tags.Upsert(f.ActorContext("sommelier"), wine.EntityUID(), tag.Tag{Key: "cellar"})
	testutil.Ok(t, err)
	_, err = f.App.Tags.Remove(f.ActorContext("sommelier"), wine.EntityUID(), "cellar")
	testutil.Ok(t, err)
	_, err = f.App.Tags.Upsert(f.ActorContext("bartender"), wine.EntityUID(), tag.Tag{Key: "bar"})
	testutil.ErrorIsPermission(t, err)
}

func TestTagListUsesDomainReadPermission(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	targets := operationalTagTargets(t, f)
	for _, target := range targets {
		_, err := f.App.Tags.Upsert(f.OwnerContext(), target.uid, tag.Tag{Key: "visible", Value: "yes"})
		testutil.Ok(t, err)
	}

	for _, target := range targets[:4] {
		values, err := f.App.Tags.List(f.ActorContext("anonymous"), target.uid)
		testutil.Ok(t, err)
		testutil.Equals(t, values, tag.Tags{{Key: "visible", Value: "yes"}})
	}
	_, err := f.App.Tags.List(f.ActorContext("anonymous"), targets[4].uid)
	testutil.ErrorIsPermission(t, err)
	values, err := f.App.Tags.List(f.ActorContext("bartender"), targets[4].uid)
	testutil.Ok(t, err)
	testutil.Equals(t, values, tag.Tags{{Key: "visible", Value: "yes"}})
}

func TestTagsRejectUnsupportedInvalidAndMissingTargets(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	_, err := f.App.Tags.Upsert(ctx, cedar.NewEntityUID("Mixology::AuditEntry", entity.NewAuditEntryID().EntityUID().ID), tag.Tag{Key: "no"})
	testutil.ErrorIsInvalid(t, err)
	_, err = f.App.Tags.Upsert(ctx, cedar.NewEntityUID(entity.TypeDrink, "not-a-drink-id"), tag.Tag{Key: "no"})
	testutil.ErrorIsInvalid(t, err)
	missing := entity.NewDrinkID().EntityUID()
	_, err = f.App.Tags.Upsert(ctx, missing, tag.Tag{Key: "no"})
	testutil.ErrorIsNotFound(t, err)
	_, err = f.App.Tags.List(ctx, missing)
	testutil.ErrorIsNotFound(t, err)
}

package drinks_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func drinkForPermissions(name string, category models.DrinkCategory, ingredientID entity.IngredientID) models.Drink {
	return models.Drink{
		Name:     name,
		Category: category,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: ingredientID, Amount: measurement.MustAmount(1.0, measurement.UnitOz)},
			},
			Steps: []string{"Shake with ice"},
		},
	}
}

func TestPermissions_Drinks(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		canReadWine      bool
		canReadNonWine   bool
		canManageWine    bool
		canManageNonWine bool
	}{
		{name: "owner", canReadWine: true, canReadNonWine: true, canManageWine: true, canManageNonWine: true},
		{name: "manager", canReadWine: true, canReadNonWine: true, canManageWine: true, canManageNonWine: true},
		{name: "sommelier", canReadWine: true, canManageWine: true},
		{name: "bartender", canReadNonWine: true, canManageNonWine: true},
		{name: "anonymous", canReadWine: true, canReadNonWine: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fix := testutil.NewFixture(t)
			b := fix.Bootstrap()
			a := fix.App
			var ctx *middleware.Context
			if tc.name == "owner" {
				ctx = fix.OwnerContext()
			} else {
				ctx = fix.ActorContext(tc.name)
			}

			base := b.WithIngredient("Lime Juice", measurement.UnitOz)
			wineExisting := b.WithDrink(drinkForPermissions("Permissions Wine", models.DrinkCategoryWine, base.ID))
			nonWineExisting := b.WithDrink(drinkForPermissions("Permissions Cocktail", models.DrinkCategoryCocktail, base.ID))

			listed, err := a.Drinks.List(ctx, drinks.ListRequest{})
			testutil.Ok(t, err)
			visible := make(map[entity.DrinkID]bool, len(listed))
			for _, drink := range listed {
				visible[drink.ID] = true
			}
			testutil.ErrorIf(t, visible[wineExisting.ID] != tc.canReadWine, "unexpected wine visibility")
			testutil.ErrorIf(t, visible[nonWineExisting.ID] != tc.canReadNonWine, "unexpected non-wine visibility")

			count, err := a.Drinks.Count(ctx, drinks.ListRequest{})
			testutil.Ok(t, err)
			wantCount := 0
			if tc.canReadWine {
				wantCount++
			}
			if tc.canReadNonWine {
				wantCount++
			}
			testutil.ErrorIf(t, count != wantCount, "expected visible count %d, got %d", wantCount, count)

			_, err = a.Drinks.Get(ctx, wineExisting.ID)
			if tc.canReadWine {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Drinks.Get(ctx, nonWineExisting.ID)
			if tc.canReadNonWine {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			wineCreate := drinkForPermissions("New Wine", models.DrinkCategoryWine, base.ID)
			_, err = a.Drinks.Create(ctx, &wineCreate)
			if tc.canManageWine {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			nonWineCreate := drinkForPermissions("New Cocktail", models.DrinkCategoryCocktail, base.ID)
			_, err = a.Drinks.Create(ctx, &nonWineCreate)
			if tc.canManageNonWine {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			wineUpdated := *wineExisting
			wineUpdated.Description = "Updated"
			_, err = a.Drinks.Update(ctx, &wineUpdated)
			if tc.canManageWine {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			nonWineUpdated := *nonWineExisting
			nonWineUpdated.Description = "Updated"
			_, err = a.Drinks.Update(ctx, &nonWineUpdated)
			if tc.canManageNonWine {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Drinks.Delete(ctx, wineExisting.ID)
			if tc.canManageWine {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Drinks.Delete(ctx, nonWineExisting.ID)
			if tc.canManageNonWine {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}
		})
	}
}

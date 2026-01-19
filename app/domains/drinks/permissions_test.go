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
		canManageWine    bool
		canManageNonWine bool
	}{
		{name: "owner", canManageWine: true, canManageNonWine: true},
		{name: "manager", canManageWine: true, canManageNonWine: true},
		{name: "sommelier", canManageWine: true, canManageNonWine: false},
		{name: "bartender", canManageWine: false, canManageNonWine: true},
		{name: "anonymous", canManageWine: false, canManageNonWine: false},
	}

	for _, tc := range cases {
		tc := tc
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

			_, err := a.Drinks.List(ctx, drinks.ListRequest{})
			testutil.PermissionTestPass(t, err)

			_, err = a.Drinks.Get(ctx, models.NewDrinkID("does-not-exist"))
			testutil.PermissionTestPass(t, err)

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

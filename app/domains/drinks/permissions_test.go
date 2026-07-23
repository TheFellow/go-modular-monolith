package drinks_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
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
			a := fix.App
			owner := fix.OwnerContext()
			var ctx *middleware.Context
			if tc.name == "owner" {
				ctx = owner
			} else {
				ctx = fix.ActorContext(tc.name)
			}

			base := testutil.CreateIngredient(t, fix, ingredientsmodels.Ingredient{
				Name: "Lime Juice", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz,
			})
			wineExisting := testutil.CreateDrink(t, fix, drinkForPermissions("Permissions Wine", models.DrinkCategoryWine, base.ID))
			nonWineExisting := testutil.CreateDrink(t, fix, drinkForPermissions("Permissions Cocktail", models.DrinkCategoryCocktail, base.ID))

			listed, err := a.Drinks.List(ctx, drinks.ListRequest{})
			testutil.Ok(t, err)
			visible := make(map[entity.DrinkID]bool, len(listed.Items))
			for _, drink := range listed.Items {
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
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}

			_, err = a.Drinks.Get(ctx, nonWineExisting.ID)
			if tc.canReadNonWine {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}

			wineCreate := drinkForPermissions("New Wine", models.DrinkCategoryWine, base.ID)
			_, err = a.Drinks.Create(ctx, &wineCreate)
			if tc.canManageWine {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}

			nonWineCreate := drinkForPermissions("New Cocktail", models.DrinkCategoryCocktail, base.ID)
			_, err = a.Drinks.Create(ctx, &nonWineCreate)
			if tc.canManageNonWine {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}
			persistedCount, err := a.Drinks.Count(owner, drinks.ListRequest{})
			testutil.Ok(t, err)
			wantPersistedCount := 2
			if tc.canManageWine {
				wantPersistedCount++
			}
			if tc.canManageNonWine {
				wantPersistedCount++
			}
			testutil.Equals(t, persistedCount, wantPersistedCount)

			wineUpdated := *wineExisting
			wineUpdated.Description = "Updated"
			_, err = a.Drinks.Update(ctx, &wineUpdated)
			if tc.canManageWine {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}
			gotWine, err := a.Drinks.Get(owner, wineExisting.ID)
			testutil.Ok(t, err)
			wantWineDescription := ""
			if tc.canManageWine {
				wantWineDescription = "Updated"
			}
			testutil.Equals(t, gotWine.Description, wantWineDescription)

			nonWineUpdated := *nonWineExisting
			nonWineUpdated.Description = "Updated"
			_, err = a.Drinks.Update(ctx, &nonWineUpdated)
			if tc.canManageNonWine {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}
			gotNonWine, err := a.Drinks.Get(owner, nonWineExisting.ID)
			testutil.Ok(t, err)
			wantNonWineDescription := ""
			if tc.canManageNonWine {
				wantNonWineDescription = "Updated"
			}
			testutil.Equals(t, gotNonWine.Description, wantNonWineDescription)

			_, err = a.Drinks.Delete(ctx, wineExisting.ID)
			if tc.canManageWine {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}
			_, err = a.Drinks.Get(owner, wineExisting.ID)
			if tc.canManageWine {
				testutil.ErrorIsNotFound(t, err)
			} else {
				testutil.Ok(t, err)
			}

			_, err = a.Drinks.Delete(ctx, nonWineExisting.ID)
			if tc.canManageNonWine {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}
			_, err = a.Drinks.Get(owner, nonWineExisting.ID)
			if tc.canManageNonWine {
				testutil.ErrorIsNotFound(t, err)
			} else {
				testutil.Ok(t, err)
			}
		})
	}
}

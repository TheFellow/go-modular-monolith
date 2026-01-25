package menu_test

import (
	"testing"

	drinksM "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	menuM "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Menu(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		canWrite bool
	}{
		{name: "owner", canWrite: true},
		{name: "manager", canWrite: true},
		{name: "sommelier", canWrite: false},
		{name: "bartender", canWrite: false},
		{name: "anonymous", canWrite: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			b := f.Bootstrap()
			a := f.App
			var ctx *middleware.Context
			if tc.name == "owner" {
				ctx = f.OwnerContext()
			} else {
				ctx = f.ActorContext(tc.name)
			}

			base := b.WithIngredient("Menu Permissions Base", measurement.UnitOz)
			drink := b.WithDrink(drinksM.Drink{
				Name:     "Menu Permissions Drink",
				Category: drinksM.DrinkCategoryCocktail,
				Glass:    drinksM.GlassTypeCoupe,
				Recipe: drinksM.Recipe{
					Ingredients: []drinksM.RecipeIngredient{
						{IngredientID: base.ID, Amount: measurement.MustAmount(1.0, measurement.UnitOz)},
					},
					Steps: []string{"Shake"},
				},
			})
			menuRecord := b.WithMenu("Permissions Menu")

			_, err := a.Menu.List(ctx, menu.ListRequest{})
			testutil.PermissionTestPass(t, err)

			_, err = a.Menu.Get(ctx, menuM.NewMenuID("does-not-exist"))
			testutil.PermissionTestPass(t, err)

			_, err = a.Menu.Create(ctx, &menuM.Menu{})
			if tc.canWrite {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Menu.AddDrink(ctx, &menuM.MenuPatch{
				MenuID:  menuRecord.ID,
				DrinkID: drink.ID,
			})
			if tc.canWrite {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Menu.RemoveDrink(ctx, &menuM.MenuPatch{
				MenuID:  menuRecord.ID,
				DrinkID: drink.ID,
			})
			if tc.canWrite {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Menu.Publish(ctx, &menuM.Menu{ID: menuRecord.ID})
			if tc.canWrite {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}
		})
	}
}

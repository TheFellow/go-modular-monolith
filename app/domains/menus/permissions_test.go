package menus_test

import (
	"testing"

	drinksM "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menuM "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
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
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			b := f.Bootstrap()
			a := f.App
			owner := f.OwnerContext()
			var ctx *middleware.Context
			if tc.name == "owner" {
				ctx = owner
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
			menuForAdd := b.WithMenu("Permissions Add Menu")
			menuForRemove := b.AddDrinks(b.WithMenu("Permissions Remove Menu"), drink)
			menuForPublish := b.AddDrinks(b.WithMenu("Permissions Publish Menu"), drink)
			menuForDraft := b.WithPublishedMenu(menuM.Menu{Name: "Permissions Draft Menu"}, drink)

			_, err := a.Menus.List(ctx, menus.ListRequest{})
			testutil.Ok(t, err)

			_, err = a.Menus.Get(ctx, menuForAdd.ID)
			testutil.Ok(t, err)

			_, err = a.Menus.Create(ctx, &menuM.Menu{Name: "Created Permissions Menu"})
			if tc.canWrite {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}

			_, err = a.Menus.AddDrink(ctx, &menuM.MenuPatch{
				MenuID:  menuForAdd.ID,
				DrinkID: drink.ID,
			})
			if tc.canWrite {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}

			_, err = a.Menus.RemoveDrink(ctx, &menuM.MenuPatch{
				MenuID:  menuForRemove.ID,
				DrinkID: drink.ID,
			})
			if tc.canWrite {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}

			_, err = a.Menus.Publish(ctx, &menuM.Menu{ID: menuForPublish.ID})
			if tc.canWrite {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}

			_, err = a.Menus.Draft(ctx, &menuM.Menu{ID: menuForDraft.ID})
			if tc.canWrite {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}

			count, err := a.Menus.Count(owner, menus.ListRequest{})
			testutil.Ok(t, err)
			wantCount := 4
			if tc.canWrite {
				wantCount++
			}
			testutil.Equals(t, count, wantCount)

			gotAdd, err := a.Menus.Get(owner, menuForAdd.ID)
			testutil.Ok(t, err)
			wantAddItems := 0
			if tc.canWrite {
				wantAddItems = 1
			}
			testutil.Equals(t, len(gotAdd.Items), wantAddItems)
			gotRemove, err := a.Menus.Get(owner, menuForRemove.ID)
			testutil.Ok(t, err)
			wantRemoveItems := 1
			if tc.canWrite {
				wantRemoveItems = 0
			}
			testutil.Equals(t, len(gotRemove.Items), wantRemoveItems)
			gotPublish, err := a.Menus.Get(owner, menuForPublish.ID)
			testutil.Ok(t, err)
			wantPublishStatus := menuM.MenuStatusDraft
			if tc.canWrite {
				wantPublishStatus = menuM.MenuStatusPublished
			}
			testutil.Equals(t, gotPublish.Status, wantPublishStatus)
			gotDraft, err := a.Menus.Get(owner, menuForDraft.ID)
			testutil.Ok(t, err)
			wantDraftStatus := menuM.MenuStatusPublished
			if tc.canWrite {
				wantDraftStatus = menuM.MenuStatusDraft
			}
			testutil.Equals(t, gotDraft.Status, wantDraftStatus)
		})
	}
}

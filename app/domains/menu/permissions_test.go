package menu_test

import (
	"testing"

	drinksM "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	menuM "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Menu(t *testing.T) {
	t.Parallel()

	t.Run("owner", func(t *testing.T) {
		t.Parallel()
		f := testutil.NewFixture(t)
		b := f.Bootstrap()
		a := f.App
		owner := f.OwnerContext()

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

		_, err := a.Menu.List(owner, menu.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Get(owner, menuM.NewMenuID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Create(owner, &menuM.Menu{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.AddDrink(owner, &menuM.MenuDrinkChange{
			MenuID:  menuRecord.ID,
			DrinkID: drink.ID,
		})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.RemoveDrink(owner, &menuM.MenuDrinkChange{
			MenuID:  menuRecord.ID,
			DrinkID: drink.ID,
		})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Publish(owner, &menuM.Menu{ID: menuRecord.ID})
		testutil.RequireNotDenied(t, err)
	})

	t.Run("anonymous", func(t *testing.T) {
		t.Parallel()
		f := testutil.NewFixture(t)
		b := f.Bootstrap()
		a := f.App
		anon := f.ActorContext("anonymous")

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

		_, err := a.Menu.List(anon, menu.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Get(anon, menuM.NewMenuID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Create(anon, &menuM.Menu{})
		testutil.RequireDenied(t, err)

		_, err = a.Menu.AddDrink(anon, &menuM.MenuDrinkChange{
			MenuID:  menuRecord.ID,
			DrinkID: drink.ID,
		})
		testutil.RequireDenied(t, err)

		_, err = a.Menu.RemoveDrink(anon, &menuM.MenuDrinkChange{
			MenuID:  menuRecord.ID,
			DrinkID: drink.ID,
		})
		testutil.RequireDenied(t, err)

		_, err = a.Menu.Publish(anon, &menuM.Menu{ID: menuRecord.ID})
		testutil.RequireDenied(t, err)
	})
}

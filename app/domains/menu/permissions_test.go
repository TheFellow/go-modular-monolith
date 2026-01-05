package menu_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Menu(t *testing.T) {
	fix := testutil.NewFixture(t)
	a := fix.App

	owner := fix.Ctx
	anon := fix.AsActor("anonymous")

	t.Run("owner", func(t *testing.T) {
		_, err := a.Menu.List(owner, menu.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Get(owner, menu.GetRequest{ID: menumodels.NewMenuID("does-not-exist")})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Create(owner, menumodels.Menu{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.AddDrink(owner, menumodels.MenuDrinkChange{
			MenuID:  menumodels.NewMenuID("does-not-exist"),
			DrinkID: drinksmodels.NewDrinkID("does-not-exist"),
		})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.RemoveDrink(owner, menumodels.MenuDrinkChange{
			MenuID:  menumodels.NewMenuID("does-not-exist"),
			DrinkID: drinksmodels.NewDrinkID("does-not-exist"),
		})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Publish(owner, menumodels.Menu{ID: menumodels.NewMenuID("does-not-exist")})
		testutil.RequireNotDenied(t, err)
	})

	t.Run("anonymous", func(t *testing.T) {
		_, err := a.Menu.List(anon, menu.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Get(anon, menu.GetRequest{ID: menumodels.NewMenuID("does-not-exist")})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Create(anon, menumodels.Menu{})
		testutil.RequireDenied(t, err)

		_, err = a.Menu.AddDrink(anon, menumodels.MenuDrinkChange{
			MenuID:  menumodels.NewMenuID("does-not-exist"),
			DrinkID: drinksmodels.NewDrinkID("does-not-exist"),
		})
		testutil.RequireDenied(t, err)

		_, err = a.Menu.RemoveDrink(anon, menumodels.MenuDrinkChange{
			MenuID:  menumodels.NewMenuID("does-not-exist"),
			DrinkID: drinksmodels.NewDrinkID("does-not-exist"),
		})
		testutil.RequireDenied(t, err)

		_, err = a.Menu.Publish(anon, menumodels.Menu{ID: menumodels.NewMenuID("does-not-exist")})
		testutil.RequireDenied(t, err)
	})
}

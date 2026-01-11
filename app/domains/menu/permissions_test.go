package menu_test

import (
	"testing"

	drinksM "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	menuM "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Menu(t *testing.T) {
	f := testutil.NewFixture(t)
	a := f.App

	owner := f.OwnerContext()
	anon := f.ActorContext("anonymous")

	t.Run("owner", func(t *testing.T) {
		_, err := a.Menu.List(owner, menu.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Get(owner, menuM.NewMenuID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Create(owner, menuM.Menu{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.AddDrink(owner, menuM.MenuDrinkChange{
			MenuID:  menuM.NewMenuID("does-not-exist"),
			DrinkID: drinksM.NewDrinkID("does-not-exist"),
		})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.RemoveDrink(owner, menuM.MenuDrinkChange{
			MenuID:  menuM.NewMenuID("does-not-exist"),
			DrinkID: drinksM.NewDrinkID("does-not-exist"),
		})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Publish(owner, menuM.Menu{ID: menuM.NewMenuID("does-not-exist")})
		testutil.RequireNotDenied(t, err)
	})

	t.Run("anonymous", func(t *testing.T) {
		_, err := a.Menu.List(anon, menu.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Get(anon, menuM.NewMenuID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Menu.Create(anon, menuM.Menu{})
		testutil.RequireDenied(t, err)

		_, err = a.Menu.AddDrink(anon, menuM.MenuDrinkChange{
			MenuID:  menuM.NewMenuID("does-not-exist"),
			DrinkID: drinksM.NewDrinkID("does-not-exist"),
		})
		testutil.RequireDenied(t, err)

		_, err = a.Menu.RemoveDrink(anon, menuM.MenuDrinkChange{
			MenuID:  menuM.NewMenuID("does-not-exist"),
			DrinkID: drinksM.NewDrinkID("does-not-exist"),
		})
		testutil.RequireDenied(t, err)

		_, err = a.Menu.Publish(anon, menuM.Menu{ID: menuM.NewMenuID("does-not-exist")})
		testutil.RequireDenied(t, err)
	})
}

package drinks_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Drinks(t *testing.T) {
	testutil.OpenStore(t)
	a := app.New()

	owner := testutil.ActorContext(t, "owner")
	anon := testutil.ActorContext(t, "anonymous")

	t.Run("owner", func(t *testing.T) {
		_, err := a.Drinks.List(owner, drinks.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Drinks.Get(owner, drinks.GetRequest{ID: models.NewDrinkID("does-not-exist")})
		testutil.RequireNotDenied(t, err)

		_, err = a.Drinks.Create(owner, models.Drink{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Drinks.Update(owner, models.Drink{ID: models.NewDrinkID("does-not-exist")})
		testutil.RequireNotDenied(t, err)
	})

	t.Run("anonymous", func(t *testing.T) {
		_, err := a.Drinks.List(anon, drinks.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Drinks.Get(anon, drinks.GetRequest{ID: models.NewDrinkID("does-not-exist")})
		testutil.RequireNotDenied(t, err)

		_, err = a.Drinks.Create(anon, models.Drink{})
		testutil.RequireDenied(t, err)

		_, err = a.Drinks.Update(anon, models.Drink{ID: models.NewDrinkID("does-not-exist")})
		testutil.RequireDenied(t, err)
	})
}

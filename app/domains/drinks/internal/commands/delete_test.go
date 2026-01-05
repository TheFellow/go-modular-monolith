package commands_test

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/mjl-/bstore"
)

func TestDelete_RemovesDrink(t *testing.T) {
	fix := testutil.NewFixture(t)

	d := dao.New()
	cmds := commands.NewWithDependencies(d, fakeIngredients{})

	var created drinksmodels.Drink
	err := fix.Store.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(fix.Ctx, middleware.WithTransaction(tx))

		var err error
		created, err = cmds.Create(ctx, drinksmodels.Drink{
			Name:     "Margarita",
			Category: drinksmodels.DrinkCategoryCocktail,
			Glass:    drinksmodels.GlassTypeCoupe,
			Recipe: drinksmodels.Recipe{
				Ingredients: []drinksmodels.RecipeIngredient{
					{
						IngredientID: ingredientsmodels.NewIngredientID("lime-juice"),
						Amount:       1.0,
						Unit:         ingredientsmodels.UnitOz,
					},
				},
				Steps: []string{"Shake with ice"},
			},
		})
		return err
	})
	testutil.Ok(t, err)

	// Verify drink exists
	drinks, err := d.List(fix.Ctx, dao.ListFilter{})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(drinks) != 1, "expected 1 drink, got %d", len(drinks))

	// Delete the drink
	var deleted drinksmodels.Drink
	err = fix.Store.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(fix.Ctx, middleware.WithTransaction(tx))

		var err error
		deleted, err = cmds.Delete(ctx, created.ID)
		return err
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, deleted.Name != "Margarita", "expected deleted drink name Margarita, got %q", deleted.Name)

	// Verify drink is gone
	drinks, err = d.List(fix.Ctx, dao.ListFilter{})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(drinks) != 0, "expected 0 drinks after delete, got %d", len(drinks))
}

func TestDelete_NotFound(t *testing.T) {
	fix := testutil.NewFixture(t)

	d := dao.New()
	cmds := commands.NewWithDependencies(d, fakeIngredients{})

	err := fix.Store.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(fix.Ctx, middleware.WithTransaction(tx))

		_, err := cmds.Delete(ctx, drinksmodels.NewDrinkID("nonexistent"))
		return err
	})

	if !errors.IsNotFound(err) {
		t.Fatalf("expected NotFound error, got %v", err)
	}
}

func TestDelete_EmitsDrinkDeletedEvent(t *testing.T) {
	fix := testutil.NewFixture(t)

	d := dao.New()
	cmds := commands.NewWithDependencies(d, fakeIngredients{})

	var created drinksmodels.Drink
	err := fix.Store.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(fix.Ctx, middleware.WithTransaction(tx))

		var err error
		created, err = cmds.Create(ctx, drinksmodels.Drink{
			Name:     "Cosmopolitan",
			Category: drinksmodels.DrinkCategoryCocktail,
			Glass:    drinksmodels.GlassTypeMartini,
			Recipe: drinksmodels.Recipe{
				Ingredients: []drinksmodels.RecipeIngredient{
					{
						IngredientID: ingredientsmodels.NewIngredientID("vodka"),
						Amount:       1.5,
						Unit:         ingredientsmodels.UnitOz,
					},
				},
				Steps: []string{"Shake with ice"},
			},
		})
		return err
	})
	testutil.Ok(t, err)

	var events []any
	err = fix.Store.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(fix.Ctx, middleware.WithTransaction(tx))

		_, err := cmds.Delete(ctx, created.ID)
		events = ctx.Events()
		return err
	})
	testutil.Ok(t, err)

	testutil.ErrorIf(t, len(events) != 1, "expected 1 event, got %d", len(events))
}

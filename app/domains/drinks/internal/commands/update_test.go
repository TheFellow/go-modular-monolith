package commands_test

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

type fakeIngredientsOK struct{}

func (f fakeIngredientsOK) Get(_ context.Context, _ cedar.EntityUID) (ingredientsmodels.Ingredient, error) {
	return ingredientsmodels.Ingredient{}, nil
}

func TestUpdate_PersistsAndEmitsEvent(t *testing.T) {
	testutil.OpenStore(t)

	d := dao.New()
	cmds := commands.NewWithDependencies(d, fakeIngredientsOK{})

	err := store.DB.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(context.Background(), middleware.WithTransaction(tx))
		seed := drinksmodels.Drink{
			ID:       drinksmodels.NewDrinkID("margarita"),
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
		}
		return d.Insert(ctx, seed)
	})
	testutil.Ok(t, err)

	var (
		updated drinksmodels.Drink
		evts    []any
	)
	err = store.DB.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(context.Background(), middleware.WithTransaction(tx))

		var err error
		updated, err = cmds.Update(ctx, drinksmodels.Drink{
			ID:       drinksmodels.NewDrinkID("margarita"),
			Name:     "Margarita",
			Category: drinksmodels.DrinkCategoryCocktail,
			Glass:    drinksmodels.GlassTypeCoupe,
			Recipe: drinksmodels.Recipe{
				Ingredients: []drinksmodels.RecipeIngredient{
					{
						IngredientID: ingredientsmodels.NewIngredientID("lemon-juice"),
						Amount:       1.0,
						Unit:         ingredientsmodels.UnitOz,
					},
				},
				Steps: []string{"Shake hard"},
			},
		})
		evts = ctx.Events()
		return err
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, string(updated.ID.ID) != "margarita", "expected id margarita, got %q", string(updated.ID.ID))
	testutil.ErrorIf(t, len(updated.Recipe.Ingredients) != 1, "expected 1 ingredient")
	testutil.ErrorIf(t, string(updated.Recipe.Ingredients[0].IngredientID.ID) != "lemon-juice", "expected lemon-juice")

	var saw bool
	for _, e := range evts {
		if _, ok := e.(events.DrinkRecipeUpdated); ok {
			saw = true
		}
	}
	testutil.ErrorIf(t, !saw, "expected DrinkRecipeUpdated event")

	got, ok, err := d.Get(context.Background(), drinksmodels.NewDrinkID("margarita"))
	testutil.Ok(t, err)
	testutil.ErrorIf(t, !ok, "expected margarita to exist")
	testutil.ErrorIf(t, string(got.Recipe.Ingredients[0].IngredientID.ID) != "lemon-juice", "expected lemon-juice in persisted recipe")
}

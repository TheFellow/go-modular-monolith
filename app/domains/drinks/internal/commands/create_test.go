package commands_test

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

type fakeIngredients struct{}

func (f fakeIngredients) Get(_ context.Context, _ cedar.EntityUID) (ingredientsmodels.Ingredient, error) {
	return ingredientsmodels.Ingredient{}, nil
}

func TestCreate_PersistsOnCommit(t *testing.T) {
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
			Description: "A classic sour",
		})
		return err
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, string(created.ID.ID) == "", "expected id to be set")

	drinks, err := d.List(fix.Ctx)
	testutil.Ok(t, err)

	testutil.ErrorIf(t, len(drinks) != 1, "expected 1 drink, got %d", len(drinks))
	testutil.ErrorIf(t, drinks[0].Name != "Margarita", "expected Margarita, got %q", drinks[0].Name)
	testutil.ErrorIf(t, drinks[0].Recipe.Ingredients == nil || len(drinks[0].Recipe.Ingredients) != 1, "expected 1 recipe ingredient")
	testutil.ErrorIf(t, string(drinks[0].Recipe.Ingredients[0].IngredientID.ID) != "lime-juice", "expected lime-juice, got %q", string(drinks[0].Recipe.Ingredients[0].IngredientID.ID))
}

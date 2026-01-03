package commands_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/uow"
)

type fakeIngredientsOK struct{}

func (f fakeIngredientsOK) Get(_ *middleware.Context, _ ingredients.GetRequest) (ingredients.GetResponse, error) {
	return ingredients.GetResponse{Ingredient: ingredientsmodels.Ingredient{}}, nil
}

func TestUpdateRecipe_PersistsAndEmitsEvent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "drinks.json")

	const seed = `[
  {
    "id": "margarita",
    "name": "Margarita",
    "recipe": {
      "ingredients": [
        { "ingredient_id": "lime-juice", "amount": 1.0, "unit": "oz" }
      ],
      "steps": ["Shake with ice"]
    }
  }
]`

	err := os.WriteFile(path, []byte(seed), 0o644)
	testutil.ErrorIf(t, err != nil, "write seed: %v", err)

	d := dao.NewFileDrinkDAO(path)
	err = d.Load(context.Background())
	testutil.ErrorIf(t, err != nil, "load: %v", err)

	ctx := middleware.NewContext(context.Background())
	tx, err := uow.NewManager().Begin(ctx)
	testutil.ErrorIf(t, err != nil, "begin tx: %v", err)
	ctx = middleware.NewContext(ctx, middleware.WithUnitOfWork(tx))

	uc := commands.NewUpdateRecipe(d, fakeIngredientsOK{})
	updated, err := uc.Execute(ctx, commands.UpdateRecipeRequest{
		DrinkID: drinksmodels.NewDrinkID("margarita"),
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
	testutil.ErrorIf(t, err != nil, "execute: %v", err)
	testutil.ErrorIf(t, string(updated.ID.ID) != "margarita", "expected id margarita, got %q", string(updated.ID.ID))
	testutil.ErrorIf(t, len(updated.Recipe.Ingredients) != 1, "expected 1 ingredient")
	testutil.ErrorIf(t, string(updated.Recipe.Ingredients[0].IngredientID.ID) != "lemon-juice", "expected lemon-juice")

	var saw bool
	for _, e := range ctx.Events() {
		if _, ok := e.(events.DrinkRecipeUpdated); ok {
			saw = true
		}
	}
	testutil.ErrorIf(t, !saw, "expected DrinkRecipeUpdated event")

	err = tx.Commit()
	testutil.ErrorIf(t, err != nil, "commit: %v", err)

	loaded := dao.NewFileDrinkDAO(path)
	err = loaded.Load(context.Background())
	testutil.ErrorIf(t, err != nil, "reload: %v", err)

	got, ok, err := loaded.Get(context.Background(), "margarita")
	testutil.ErrorIf(t, err != nil, "get: %v", err)
	testutil.ErrorIf(t, !ok, "expected margarita to exist")
	testutil.ErrorIf(t, got.Recipe.Ingredients[0].IngredientID != "lemon-juice", "expected lemon-juice in persisted recipe")
}

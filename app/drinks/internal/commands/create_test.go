package commands_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/uow"
)

type fakeIngredients struct{}

func (f fakeIngredients) Get(_ *middleware.Context, _ ingredients.GetRequest) (ingredients.GetResponse, error) {
	return ingredients.GetResponse{Ingredient: ingredientsmodels.Ingredient{}}, nil
}

func TestCreate_PersistsOnCommit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "drinks.json")

	err := os.WriteFile(path, []byte("[]\n"), 0o644)
	testutil.ErrorIf(t, err != nil, "write seed: %v", err)

	d := dao.NewFileDrinkDAO(path)
	err = d.Load(context.Background())
	testutil.ErrorIf(t, err != nil, "load: %v", err)

	ctx := middleware.NewContext(context.Background())
	tx, err := uow.NewManager().Begin(ctx)
	testutil.ErrorIf(t, err != nil, "begin tx: %v", err)
	ctx = middleware.NewContext(ctx, middleware.WithUnitOfWork(tx))

	uc := commands.NewCreate(d, fakeIngredients{})
	created, err := uc.Execute(ctx, commands.CreateRequest{
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
	testutil.ErrorIf(t, err != nil, "execute: %v", err)
	testutil.ErrorIf(t, string(created.ID.ID) == "", "expected id to be set")

	err = tx.Commit()
	testutil.ErrorIf(t, err != nil, "commit: %v", err)

	loaded := dao.NewFileDrinkDAO(path)
	err = loaded.Load(context.Background())
	testutil.ErrorIf(t, err != nil, "reload: %v", err)

	drinks, err := loaded.List(context.Background())
	testutil.ErrorIf(t, err != nil, "list: %v", err)

	testutil.ErrorIf(t, len(drinks) != 1, "expected 1 drink, got %d", len(drinks))
	testutil.ErrorIf(t, drinks[0].Name != "Margarita", "expected Margarita, got %q", drinks[0].Name)
	testutil.ErrorIf(t, drinks[0].Recipe.Ingredients == nil || len(drinks[0].Recipe.Ingredients) != 1, "expected 1 recipe ingredient")
	testutil.ErrorIf(t, drinks[0].Recipe.Ingredients[0].IngredientID != "lime-juice", "expected lime-juice, got %q", drinks[0].Recipe.Ingredients[0].IngredientID)
}

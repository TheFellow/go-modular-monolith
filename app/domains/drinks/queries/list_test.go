package queries_test

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

type fakeIngredients struct{}

func (f fakeIngredients) Get(_ context.Context, _ cedar.EntityUID) (ingredientsmodels.Ingredient, error) {
	return ingredientsmodels.Ingredient{}, nil
}

func TestList_FilterByName(t *testing.T) {
	fix := testutil.NewFixture(t)

	d := dao.New()
	cmds := commands.NewWithDependencies(d, fakeIngredients{})
	q := queries.NewWithDAO(d)

	// Create multiple drinks
	err := fix.Store.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(fix.Ctx, middleware.WithTransaction(tx))

		_, err := cmds.Create(ctx, drinksmodels.Drink{
			Name:     "Margarita",
			Category: drinksmodels.DrinkCategoryCocktail,
			Glass:    drinksmodels.GlassTypeCoupe,
			Recipe: drinksmodels.Recipe{
				Ingredients: []drinksmodels.RecipeIngredient{
					{IngredientID: entity.IngredientID("tequila"), Amount: 2, Unit: ingredientsmodels.UnitOz},
				},
				Steps: []string{"Shake"},
			},
		})
		if err != nil {
			return err
		}

		_, err = cmds.Create(ctx, drinksmodels.Drink{
			Name:     "Cosmopolitan",
			Category: drinksmodels.DrinkCategoryCocktail,
			Glass:    drinksmodels.GlassTypeMartini,
			Recipe: drinksmodels.Recipe{
				Ingredients: []drinksmodels.RecipeIngredient{
					{IngredientID: entity.IngredientID("vodka"), Amount: 1.5, Unit: ingredientsmodels.UnitOz},
				},
				Steps: []string{"Shake"},
			},
		})
		if err != nil {
			return err
		}

		_, err = cmds.Create(ctx, drinksmodels.Drink{
			Name:     "Old Fashioned",
			Category: drinksmodels.DrinkCategoryCocktail,
			Glass:    drinksmodels.GlassTypeRocks,
			Recipe: drinksmodels.Recipe{
				Ingredients: []drinksmodels.RecipeIngredient{
					{IngredientID: entity.IngredientID("bourbon"), Amount: 2, Unit: ingredientsmodels.UnitOz},
				},
				Steps: []string{"Stir"},
			},
		})
		return err
	})
	testutil.Ok(t, err)

	// List all drinks - should get 3
	all, err := q.List(fix.Ctx, dao.ListFilter{})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(all) != 3, "expected 3 drinks, got %d", len(all))

	// Filter by name "Margarita" - should get 1
	filtered, err := q.List(fix.Ctx, dao.ListFilter{Name: "Margarita"})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(filtered) != 1, "expected 1 drink, got %d", len(filtered))
	testutil.ErrorIf(t, filtered[0].Name != "Margarita", "expected Margarita, got %q", filtered[0].Name)

	// Filter by non-existent name - should get 0
	empty, err := q.List(fix.Ctx, dao.ListFilter{Name: "Nonexistent"})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(empty) != 0, "expected 0 drinks, got %d", len(empty))
}

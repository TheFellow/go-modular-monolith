package queries_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestListFilter_IDs(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	ing1, err := f.Ingredients.Create(f.OwnerContext(), &models.Ingredient{
		Name:     "Lime Juice",
		Category: models.CategoryJuice,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)
	ing2, err := f.Ingredients.Create(f.OwnerContext(), &models.Ingredient{
		Name:     "Simple Syrup",
		Category: models.CategorySyrup,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)
	ing3, err := f.Ingredients.Create(f.OwnerContext(), &models.Ingredient{
		Name:     "Orange Liqueur",
		Category: models.CategoryOther,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	list, err := queries.New().List(f.OwnerContext(), queries.ListFilter{
		IDs: []entity.IngredientID{ing1.ID, ing3.ID},
	})
	testutil.Ok(t, err)

	found := make(map[string]bool, len(list))
	for _, ingredient := range list {
		if ingredient == nil {
			continue
		}
		found[ingredient.ID.String()] = true
	}

	testutil.ErrorIf(t, !found[ing1.ID.String()], "expected ingredient %s in list", ing1.ID.String())
	testutil.ErrorIf(t, !found[ing3.ID.String()], "expected ingredient %s in list", ing3.ID.String())
	testutil.ErrorIf(t, found[ing2.ID.String()], "did not expect ingredient %s in list", ing2.ID.String())
}

package ingredients_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestIngredients_CreateRejectsIDProvided(t *testing.T) {
	t.Parallel()
	fix := testutil.NewFixture(t)

	_, err := fix.Ingredients.Create(fix.OwnerContext(), models.Ingredient{
		ID: entity.IngredientID("explicit-id"),
	})
	testutil.ErrorIsInvalid(t, err)
}

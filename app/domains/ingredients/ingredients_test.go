package ingredients_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestIngredients_CreateRejectsIDProvided(t *testing.T) {
	fix := testutil.NewFixture(t)

	_, err := fix.Ingredients.Create(fix.OwnerContext(), models.Ingredient{
		ID: entity.IngredientID("explicit-id"),
	})
	testutil.ErrorIf(t, err == nil || !errors.IsInvalid(err), "expected invalid error, got %v", err)
}

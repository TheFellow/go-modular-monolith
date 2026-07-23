package ingredients_test

import (
	"fmt"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestIngredients_ListExpressionFilters(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	target := testutil.CreateIngredient(t, f, models.Ingredient{
		Name: "Botanical Gin", Category: models.CategorySpirit,
		Unit: measurement.UnitOz, Description: "Juniper-forward spirit",
	})
	testutil.CreateIngredient(t, f, models.Ingredient{
		Name: "Seasonal Tonic", Category: models.CategoryMixer,
		Unit: measurement.UnitMl, Description: "Seasonal mixer",
	})

	tests := map[string]string{
		"id":          fmt.Sprintf("id == %q", target.ID.String()),
		"name":        `name.contains("Botanical")`,
		"category":    `category == "spirit"`,
		"unit":        `unit == "oz"`,
		"description": `description.contains("Juniper")`,
	}
	for name, expression := range tests {
		ctx := f.ActorContext("owner")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			page, err := f.Ingredients.List(ctx, ingredients.ListRequest{Filter: expression})
			testutil.Ok(t, err)
			testutil.Equals(t, len(page.Items), 1)
			testutil.Equals(t, page.Items[0].ID, target.ID)
		})
	}
}

package main

import (
	"testing"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
)

func TestE2E_ExpandedHelpResizesContentBeforeRendering(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Help Tonic", Category: ingredientsmodels.CategoryMixer, Unit: measurement.UnitMl,
	})
	driver := tuitest.NewDriver(t, NewApp(f.App))
	driver.Resize(100, 40)
	driver.Press("2")
	driver.Press("?")
	driver.RequireText("t manage tags")
	driver.RequireViewport(100, 40)
}

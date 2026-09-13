package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
)

// TestRenderCrossDomainReview captures the composed terminal application for
// visual review without imposing platform-specific pixel goldens.
func TestRenderCrossDomainReview(t *testing.T) {
	t.Parallel()
	directory := os.Getenv("MIXOLOGY_RENDER_DIR")
	if directory == "" {
		t.Skip("set MIXOLOGY_RENDER_DIR to capture terminal review renders")
	}
	testutil.Ok(t, os.MkdirAll(directory, 0o755))
	f := testutil.NewFixture(t)
	gin := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "London Dry Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	replacement := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Replacement Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	for _, ingredient := range []*ingredientsmodels.Ingredient{gin, replacement} {
		testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(24, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(325, currency.USD)})
	}
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{Name: "House Martini", Category: drinksmodels.DrinkCategoryMartini, Glass: drinksmodels.GlassTypeMartini, Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: gin.ID, Amount: measurement.MustAmount(2, measurement.UnitOz)}}, Steps: []string{"Stir with ice", "Strain into a chilled glass"}, Garnish: "Lemon twist"}})
	menu := testutil.CreateMenu(t, f, "Evening Classics", testutil.WithDrink(drink), testutil.Published())
	testutil.PlaceOrder(t, f, ordersmodels.Order{MenuID: menu.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}}, Notes: "Bar seat four"})
	_, err := f.Ingredients.SetSubstitution(f.OwnerContext(), &ingredientsmodels.SubstitutionRule{IngredientID: gin.ID, SubstituteID: replacement.ID, Ratio: 0.75, QualityImpact: ingredientsmodels.Quality("similar"), Notes: "Approved temporary option"})
	testutil.Ok(t, err)
	testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Unstocked Vermouth", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	capture := func(name string, driver *tuitest.Driver) {
		t.Helper()
		driver.RequireRunning()
		driver.RequireViewport(120, 40)
		testutil.Ok(t, os.WriteFile(filepath.Join(directory, "tui-"+name+".ansi"), []byte(driver.Screen()), 0o644))
		driver.Resize(100, 30)
		driver.RequireViewport(100, 30)
		testutil.Ok(t, os.WriteFile(filepath.Join(directory, "tui-"+name+"-compact.ansi"), []byte(driver.Screen()), 0o644))
		driver.Resize(120, 40)
	}
	for _, route := range []struct{ key, name string }{{"1", "drinks"}, {"2", "ingredients"}, {"3", "inventory"}, {"4", "menus"}, {"5", "orders"}, {"6", "audit"}} {
		driver := tuitest.NewDriver(t, NewApp(f.App))
		driver.Resize(120, 40)
		driver.Press(route.key)
		capture(route.name, driver)
		switch route.name {
		case "ingredients":
			selectReviewRow(t, driver, gin.Name)
			driver.Press("s")
			driver.RequireText("1 rules", "Replacement Gin")
			capture("substitutions", driver)
			driver.Press("e")
			driver.RequireText("Ratio", "Quality")
			capture("substitution-edit", driver)
			driver.Press("esc")
			driver.Press("esc")
			driver.Press("R")
			driver.RequireText("Existing stock", "Reason")
			capture("retirement", driver)
		case "inventory":
			driver.Press("c")
			capture("receive-picker", driver)
			driver.Press("ctrl+s")
			capture("receive-form", driver)
			driver.Press("esc")
			selectReviewRow(t, driver, gin.Name)
			driver.Press("x")
			driver.RequireText("Quarantine stock")
			capture("quarantine", driver)
			driver.Press("esc")
			driver.Press("s")
			capture("stock-set", driver)
			driver.Press("esc")
			driver.Press("h")
			capture("stock-history", driver)
		case "orders":
			driver.Press("a")
			capture("amendment", driver)
			driver.Press("esc")
			driver.Press("x")
			capture("cancellation", driver)
		}
	}
	_, err = f.Ingredients.Retire(f.OwnerContext(), gin.ID, ingredientsmodels.Retirement{Reason: "Supplier discontinued this gin"})
	testutil.Ok(t, err)
	for _, route := range []struct{ key, name string }{{"1", "retired-recipe"}, {"3", "retained-stock"}, {"4", "degraded-menu"}, {"5", "accepted-order"}, {"6", "retirement-audit"}} {
		driver := tuitest.NewDriver(t, NewApp(f.App))
		driver.Resize(120, 40)
		driver.Press(route.key)
		if route.name == "retained-stock" {
			selectReviewRow(t, driver, gin.Name)
		}
		capture(route.name, driver)
		if route.name == "retained-stock" {
			driver.Press("d")
			driver.RequireText("Dispose stock")
			capture("disposal", driver)
		}
	}
}

func selectReviewRow(t *testing.T, driver *tuitest.Driver, name string) {
	t.Helper()
	for range 4 {
		driver.Press("up")
	}
	for range 4 {
		for line := range strings.SplitSeq(driver.Screen(), "\n") {
			_, detail, ok := strings.Cut(line, "│")
			if ok && strings.TrimSpace(detail) == name {
				return
			}
		}
		driver.Press("down")
	}
	testutil.Fail(t, "review row %q missing", name)
}

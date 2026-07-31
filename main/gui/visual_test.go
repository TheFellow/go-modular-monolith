//nolint:paralleltest // Fyne application and rendering state are process-global.
package main

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	drinksgui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/gui"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientsgui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/gui"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	inventorygui "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/gui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
)

// TestRenderWorkspaceReview captures real composed workspaces when explicitly
// requested. It is a lightweight review harness, not a platform pixel-golden.
func TestRenderWorkspaceReview(t *testing.T) {
	directory := os.Getenv("MIXOLOGY_RENDER_DIR")
	if directory == "" {
		t.Skip("set MIXOLOGY_RENDER_DIR to capture desktop review renders")
	}
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{dataDirectory: t.TempDir(), actor: "owner"}, deterministicDesktopDependencies(nil))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = desktop.Close() })
	ingredient, err := desktop.session.Ingredients.Create(desktop.session.Context(), &ingredientsmodels.Ingredient{Name: "London Dry Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.session.Tags.Replace(desktop.session.Context(), ingredient.EntityUID(), tag.Tags{{Key: "featured"}, {Key: "env", Value: "development"}, {Key: "region", Value: "west"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.session.Inventory.Set(desktop.session.Context(), &inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(14, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(325, currency.USD)}); err != nil {
		t.Fatal(err)
	}
	drink, err := desktop.session.Drinks.Create(desktop.session.Context(), &drinksmodels.Drink{
		Name: "Negroni", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeRocks,
		Description: "A balanced bittersweet classic.", Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"Stir with ice", "Strain over a large cube"}, Garnish: "Orange peel",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.session.Tags.Replace(desktop.session.Context(), drink.EntityUID(), tag.Tags{{Key: "classic"}}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"drinks", "ingredients", "inventory", "menus", "orders", "audit"} {
		if err := desktop.shell.Navigate(route); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(filepath.Join(directory, route+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		if route == "ingredients" {
			presenter := desktop.presenters[route].(*ingredientsgui.Presenter)
			presenter.Select(ingredient.ID)
			file, err = os.Create(filepath.Join(directory, route+"-london-dry-gin.png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			presenter.Back()
			presenter.Filter("", `name == "missing"`)
			file, err = os.Create(filepath.Join(directory, route+"-empty.png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			presenter.ResetList()
			presenter.StartCreate()
			file, err = os.Create(filepath.Join(directory, route+"-create.png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			presenter.Cancel()
		}
		if route == "drinks" {
			presenter := desktop.presenters[route].(*drinksgui.Presenter)
			presenter.Select(0)
			file, err = os.Create(filepath.Join(directory, route+"-negroni.png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			presenter.Back()
		}
		if route == "inventory" {
			presenter := desktop.presenters[route].(*inventorygui.Presenter)
			state := presenter.Snapshot()
			presenter.Select(state.Rows[0].Inventory.ID)
			captureReview(t, desktop, directory, route+"-london-dry-gin.png")
			presenter.Back()
			presenter.Filter(inventorygui.AllStock, `quantity < 0`, inventorygui.LowStockThreshold, 25)
			captureReview(t, desktop, directory, route+"-empty.png")
			presenter.ResetList()
			presenter.Select(presenter.Snapshot().Rows[0].Inventory.ID)
			presenter.StartAdjust()
			captureReview(t, desktop, directory, route+"-adjust.png")
			presenter.Cancel()
		}
		if route == "audit" {
			openFilterDisclosures(desktop.shell.Content())
			file, err = os.Create(filepath.Join(directory, route+"-expanded.png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func captureReview(t *testing.T, desktop *desktop, directory, name string) {
	t.Helper()
	file, err := os.Create(filepath.Join(directory, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func openFilterDisclosures(object framework.CanvasObject) {
	switch typed := object.(type) {
	case *widget.Accordion:
		typed.Open(0)
	case *framework.Container:
		for _, child := range typed.Objects {
			openFilterDisclosures(child)
		}
	case *container.Split:
		openFilterDisclosures(typed.Leading)
		openFilterDisclosures(typed.Trailing)
	case *container.Scroll:
		openFilterDisclosures(typed.Content)
	}
}

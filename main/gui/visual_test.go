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

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientsgui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/gui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
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
			desktop.presenters[route].(*ingredientsgui.Presenter).StartEdit()
			file, err = os.Create(filepath.Join(directory, route+"-edit.png"))
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
			desktop.presenters[route].(*ingredientsgui.Presenter).Cancel()
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

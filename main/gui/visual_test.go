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

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientsgui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/gui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
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
	for _, route := range desktop.shell.RouteIDs() {
		if err := desktop.shell.Navigate(route); err != nil {
			t.Fatal(err)
		}
		if desktop.shell.Current() != route {
			t.Fatalf("navigate owner to %q remained on %q (possible modal overlay)", route, desktop.shell.Current())
		}
		captureReview(t, desktop, directory, "owner-"+route+"-populated")
		if route == "ingredients" {
			desktop.presenters[route].(*ingredientsgui.Presenter).StartEdit()
			captureReview(t, desktop, directory, "owner-ingredients-edit-form")
			desktop.presenters[route].(*ingredientsgui.Presenter).Cancel()
			desktop.presenters[route].(*ingredientsgui.Presenter).StartCreate()
			captureReview(t, desktop, directory, "owner-ingredients-create-form")
			desktop.presenters[route].(*ingredientsgui.Presenter).Cancel()
			desktop.presenters[route].(*ingredientsgui.Presenter).StartCreate()
			desktop.presenters[route].(*ingredientsgui.Presenter).Submit(ingredientsgui.Form{})
			captureReview(t, desktop, directory, "owner-ingredients-validation-error")
			// Validation deliberately leaves the editor active. Reset it so later
			// route captures cannot become discard-confirmation screenshots.
			desktop.presenters[route].(*ingredientsgui.Presenter).Cancel()
		}
		if route == "audit" {
			openFilterDisclosures(desktop.shell.Content())
			captureReview(t, desktop, directory, "owner-audit-filters-expanded")
		}
	}

	// Every authorization level gets an isolated empty database. RouteIDs is
	// the post-authorization navigation contract, so this captures every and
	// only visible workspace for each actor without hard-coded assumptions.
	for _, actor := range []string{"manager", "sommelier", "bartender", "anonymous"} {
		review, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{dataDirectory: t.TempDir(), actor: actor}, deterministicDesktopDependencies(nil))
		if err != nil {
			t.Fatalf("open %s desktop: %v", actor, err)
		}
		for _, route := range review.shell.RouteIDs() {
			if err := review.shell.Navigate(route); err != nil {
				_ = review.Close()
				t.Fatal(err)
			}
			if review.shell.Current() != route {
				_ = review.Close()
				t.Fatalf("navigate %s to %q remained on %q (possible modal overlay)", actor, route, review.shell.Current())
			}
			captureReview(t, review, directory, actor+"-"+route+"-empty")
		}
		if err := review.Close(); err != nil {
			t.Fatalf("close %s desktop: %v", actor, err)
		}
	}
}

func captureReview(t *testing.T, desktop *desktop, directory, name string) {
	t.Helper()
	if overlays := desktop.window.Canvas().Overlays().List(); len(overlays) != 0 {
		t.Fatalf("capture %q has %d unexpected modal/menu overlays", name, len(overlays))
	}
	file, err := os.Create(filepath.Join(directory, name+".png"))
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
	case *toolkit.SemanticButton:
		if typed.SemanticID() == "filters.more" {
			test.Tap(typed)
		}
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

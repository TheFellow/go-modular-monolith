package tui

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func TestStatusBarView_UsesWarningStyleForNotFound(t *testing.T) {
	app := &App{styles: NewStyles()}
	app.lastError = errors.NotFoundf("ingredient missing")

	expected := app.styles.StatusBar.Render(app.styles.WarningText.Render(app.lastError.Error()))
	if got := app.statusBarView(); got != expected {
		t.Fatalf("unexpected status bar output:\n%s", got)
	}
}

func TestStatusBarView_UsesErrorStyleForInvalid(t *testing.T) {
	app := &App{styles: NewStyles()}
	app.lastError = errors.Invalidf("invalid input")

	expected := app.styles.StatusBar.Render(app.styles.ErrorText.Render(app.lastError.Error()))
	if got := app.statusBarView(); got != expected {
		t.Fatalf("unexpected status bar output:\n%s", got)
	}
}

func TestStatusBarView_UsesErrorStyleForPermission(t *testing.T) {
	app := &App{styles: NewStyles()}
	app.lastError = errors.Permissionf("permission denied")

	expected := app.styles.StatusBar.Render(app.styles.ErrorText.Render(app.lastError.Error()))
	if got := app.statusBarView(); got != expected {
		t.Fatalf("unexpected status bar output:\n%s", got)
	}
}

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

	auditgui "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/gui"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	drinksgui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/gui"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientsgui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/gui"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	inventorygui "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/gui"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	menusgui "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/gui"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	ordersgui "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/gui"
	tagginggui "github.com/TheFellow/go-modular-monolith/app/domains/tagging/surfaces/gui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
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
	testutil.ErrorIf(t, err != nil, "%v", err)
	t.Cleanup(func() { _ = desktop.Close() })
	ingredient, err := desktop.session.Ingredients.Create(desktop.session.Context(), &ingredientsmodels.Ingredient{Name: "London Dry Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	testutil.ErrorIf(t, err != nil, "%v", err)
	{
		_, err := desktop.session.Tags.Replace(desktop.session.Context(), ingredient.EntityUID(), tag.Tags{{Key: "featured"}, {Key: "env", Value: "development"}, {Key: "region", Value: "west"}})
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	{
		_, err := desktop.session.Inventory.Set(desktop.session.Context(), &inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(14, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(325, currency.USD)})
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	drink, err := desktop.session.Drinks.Create(desktop.session.Context(), &drinksmodels.Drink{
		Name: "Negroni", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeRocks,
		Description: "A balanced bittersweet classic.", Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"Stir with ice", "Strain over a large cube"}, Garnish: "Orange peel",
		},
	})
	testutil.ErrorIf(t, err != nil, "%v", err)
	{
		_, err := desktop.session.Tags.Replace(desktop.session.Context(), drink.EntityUID(), tag.Tags{{Key: "classic"}})
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	menu, err := desktop.session.Menus.Create(desktop.session.Context(), &menusmodels.Menu{Name: "Summer Classics", Description: "A concise menu of balanced classics."})
	testutil.ErrorIf(t, err != nil, "%v", err)
	menu, err = desktop.session.Menus.AddDrink(desktop.session.Context(), &menusmodels.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.ErrorIf(t, err != nil, "%v", err)
	publishedMenu, err := desktop.session.Menus.Create(desktop.session.Context(), &menusmodels.Menu{Name: "Published Classics", Description: "The currently published menu."})
	testutil.ErrorIf(t, err != nil, "%v", err)
	publishedMenu, err = desktop.session.Menus.AddDrink(desktop.session.Context(), &menusmodels.MenuPatch{MenuID: publishedMenu.ID, DrinkID: drink.ID})
	testutil.ErrorIf(t, err != nil, "%v", err)
	publishedMenu, err = desktop.session.Menus.Publish(desktop.session.Context(), publishedMenu)
	testutil.ErrorIf(t, err != nil, "%v", err)
	pendingOrder, err := desktop.session.Orders.Place(desktop.session.Context(), &ordersmodels.Order{MenuID: publishedMenu.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 2, Notes: "Orange peel on the side"}}, Notes: "Bar seat four"})
	testutil.ErrorIf(t, err != nil, "%v", err)
	completedOrder, err := desktop.session.Orders.Place(desktop.session.Context(), &ordersmodels.Order{MenuID: publishedMenu.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}}, Notes: "Already served"})
	testutil.ErrorIf(t, err != nil, "%v", err)
	completedOrder, err = desktop.session.Orders.Complete(desktop.session.Context(), completedOrder)
	testutil.ErrorIf(t, err != nil, "%v", err)
	{
		err := os.MkdirAll(directory, 0o755)
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	for _, route := range []string{"drinks", "ingredients", "inventory", "menus", "orders", "audit", "tags"} {
		{
			err := desktop.shell.Navigate(route)
			testutil.ErrorIf(t, err != nil, "%v", err)
		}
		file, err := os.Create(filepath.Join(directory, route+".png"))
		testutil.ErrorIf(t, err != nil, "%v", err)
		if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
			_ = file.Close()
			testutil.ErrorIf(t, err != nil, "%v", err)
		}
		{
			err := file.Close()
			testutil.ErrorIf(t, err != nil, "%v", err)
		}
		if route == "ingredients" {
			presenter := desktop.presenters[route].(*ingredientsgui.Presenter)
			presenter.Select(ingredient.ID)
			file, err = os.Create(filepath.Join(directory, route+"-london-dry-gin.png"))
			testutil.ErrorIf(t, err != nil, "%v", err)
			if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
				_ = file.Close()
				testutil.ErrorIf(t, err != nil, "%v", err)
			}
			{
				err := file.Close()
				testutil.ErrorIf(t, err != nil, "%v", err)
			}
			presenter.Back()
			presenter.Filter("", `name == "missing"`)
			file, err = os.Create(filepath.Join(directory, route+"-empty.png"))
			testutil.ErrorIf(t, err != nil, "%v", err)
			if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
				_ = file.Close()
				testutil.ErrorIf(t, err != nil, "%v", err)
			}
			{
				err := file.Close()
				testutil.ErrorIf(t, err != nil, "%v", err)
			}
			presenter.ResetList()
			presenter.StartCreate()
			file, err = os.Create(filepath.Join(directory, route+"-create.png"))
			testutil.ErrorIf(t, err != nil, "%v", err)
			if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
				_ = file.Close()
				testutil.ErrorIf(t, err != nil, "%v", err)
			}
			{
				err := file.Close()
				testutil.ErrorIf(t, err != nil, "%v", err)
			}
			presenter.Cancel()
		}
		if route == "drinks" {
			presenter := desktop.presenters[route].(*drinksgui.Presenter)
			presenter.Select(0)
			file, err = os.Create(filepath.Join(directory, route+"-negroni.png"))
			testutil.ErrorIf(t, err != nil, "%v", err)
			if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
				_ = file.Close()
				testutil.ErrorIf(t, err != nil, "%v", err)
			}
			{
				err := file.Close()
				testutil.ErrorIf(t, err != nil, "%v", err)
			}
			presenter.Back()
			presenter.StartCreate()
			captureReview(t, desktop, directory, route+"-create.png")
			presenter.Cancel()
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
		if route == "menus" {
			presenter := desktop.presenters[route].(*menusgui.Presenter)
			selectMenu := func(id entity.MenuID) {
				for i, item := range presenter.State().Items {
					if item.ID == id {
						presenter.Select(i)
						return
					}
				}
				testutil.Fail(t, "menu %s missing", id)
			}
			selectMenu(menu.ID)
			captureReview(t, desktop, directory, route+"-summer-classics.png")
			form := presenter.State().Form
			form.Description = "A changed description awaiting review."
			presenter.SetForm(form)
			captureReview(t, desktop, directory, route+"-edit.png")
			presenter.Cancel()
			presenter.StartTags()
			presenter.SetForm(menusgui.Form{Tags: "featured"})
			captureReview(t, desktop, directory, route+"-tags.png")
			presenter.Cancel()
			presenter.ResetList()
			selectMenu(publishedMenu.ID)
			captureReview(t, desktop, directory, route+"-published.png")
			presenter.Back()
			presenter.StartAnalysis()
			captureReview(t, desktop, directory, route+"-analysis.png")
			presenter.Cancel()
			presenter.Back()
			presenter.SetFilter(menusgui.Filter{Expression: `name == "missing"`, Limit: 25})
			presenter.Refresh()
			captureReview(t, desktop, directory, route+"-empty.png")
			presenter.ResetList()
			presenter.StartCreate()
			captureReview(t, desktop, directory, route+"-create.png")
			presenter.Cancel()
		}
		if route == "orders" {
			presenter := desktop.presenters[route].(*ordersgui.Presenter)
			selectOrder := func(id entity.OrderID) {
				for i, row := range presenter.State().Rows {
					if row.Order.ID == id {
						presenter.Select(i)
						return
					}
				}
				testutil.Fail(t, "order %s missing", id)
			}
			selectOrder(pendingOrder.ID)
			captureReview(t, desktop, directory, route+"-pending.png")
			presenter.Back()
			selectOrder(completedOrder.ID)
			captureReview(t, desktop, directory, route+"-completed.png")
			presenter.ResetList()
			presenter.ApplyFilter(ordersgui.Filter{Expression: `notes == "missing"`, Limit: 25})
			captureReview(t, desktop, directory, route+"-empty.png")
			presenter.ResetList()
			presenter.StartPlace()
			captureReview(t, desktop, directory, route+"-place.png")
			presenter.CancelForm()
		}
		if route == "audit" {
			presenter := desktop.presenters[route].(*auditgui.Presenter)
			if len(presenter.State().Rows) > 0 {
				presenter.Select(0)
				captureReview(t, desktop, directory, route+"-detail.png")
				presenter.Back()
			}
			presenter.ApplyFilter(auditgui.Filter{Expression: `action == "missing"`, Limit: 25})
			captureReview(t, desktop, directory, route+"-empty.png")
			presenter.ResetList()
			openFilterDisclosures(desktop.shell.Content())
			file, err = os.Create(filepath.Join(directory, route+"-expanded.png"))
			testutil.ErrorIf(t, err != nil, "%v", err)
			if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
				_ = file.Close()
				testutil.ErrorIf(t, err != nil, "%v", err)
			}
			{
				err := file.Close()
				testutil.ErrorIf(t, err != nil, "%v", err)
			}
		}
		if route == "tags" {
			presenter := desktop.presenters[route].(*tagginggui.Presenter)
			if len(presenter.State().VisibleSummaries) > 0 {
				presenter.SelectSummary(0)
				captureReview(t, desktop, directory, route+"-detail.png")
				presenter.Back()
			}
			presenter.Search("missing")
			captureReview(t, desktop, directory, route+"-empty.png")
			presenter.ResetList()
			presenter.Start(tagginggui.Add)
			captureReview(t, desktop, directory, route+"-tag-entity.png")
			presenter.Back()
		}
	}
}

func captureReview(t *testing.T, desktop *desktop, directory, name string) {
	t.Helper()
	file, err := os.Create(filepath.Join(directory, name))
	testutil.ErrorIf(t, err != nil, "%v", err)
	if err := png.Encode(file, desktop.window.Canvas().Capture()); err != nil {
		_ = file.Close()
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	{
		err := file.Close()
		testutil.ErrorIf(t, err != nil, "%v", err)
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

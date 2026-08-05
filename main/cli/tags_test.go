package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	pkglog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/urfave/cli/v3"
)

//nolint:paralleltest // the CLI fixture intentionally models sequential process lifecycles.
func TestCLIE2EFixtureCapturesExitAndSeparateOutputStreams(t *testing.T) {
	fixture := newCLIE2E(filepath.Join(t.TempDir(), "cli-errors.db"))
	result := fixture.As("bartender").Run("tags", "summary")
	testutil.ErrorIf(t, result.Err == nil, "expected permission failure")
	testutil.Equals(t, result.ExitCode, errors.ExitPermission)
	testutil.Equals(t, result.Stdout, "")
	testutil.StringContains(t, result.Stderr, "authz denied")
}

type cliTagTargets struct {
	drink        string
	ingredient   string
	inventory    string
	menu         string
	order        string
	ingredientID string
	auditEntry   string
}

//nolint:paralleltest // urfave CLI flags hold parse state; this test intentionally exercises separate invocations.
func TestTagsCLIWorkflowsPersistAndAuthorize(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tags.db")
	targets := seedCLITagTargets(t, dbPath)

	out, err := runTagsCLI(dbPath, "owner", "tags", "add", targets.drink, "featured")
	testutil.Ok(t, err)
	testutil.StringContains(t, out, targets.drink+": featured (changed)")

	out, err = runTagsCLI(dbPath, "owner", "tags", "add", targets.drink, "region=west")
	testutil.Ok(t, err)
	testutil.StringContains(t, out, "featured,region=west (changed)")

	out, err = runTagsCLI(dbPath, "owner", "tags", "add", targets.drink, "region=east")
	testutil.Ok(t, err)
	testutil.StringContains(t, out, "featured,region=east (changed)")

	out, err = runTagsCLI(dbPath, "owner", "tags", "add", targets.drink, "region=east")
	testutil.Ok(t, err)
	testutil.StringContains(t, out, "featured,region=east (unchanged)")

	out, err = runTagsCLI(dbPath, "owner", "tags", "list", targets.drink)
	testutil.Ok(t, err)
	testutil.Equals(t, out, targets.drink+": featured,region=east\n")

	out, err = runTagsCLI(dbPath, "owner", "tags", "remove", targets.drink, "missing")
	testutil.Ok(t, err)
	testutil.StringContains(t, out, "(unchanged)")
	out, err = runTagsCLI(dbPath, "owner", "tags", "remove", targets.drink, "region")
	testutil.Ok(t, err)
	testutil.Equals(t, out, targets.drink+": featured (changed)\n")

	all := map[string]string{
		"drink": targets.drink, "ingredient": targets.ingredient, "inventory": targets.inventory,
		"menu": targets.menu, "order": targets.order,
	}
	for name, id := range all {
		out, err = runTagsCLI(dbPath, "owner", "tags", "add", id, "target="+name)
		testutil.Ok(t, err)
		testutil.StringContains(t, out, "target="+name)
	}

	for _, id := range all {
		_, err = runTagsCLI(dbPath, "owner", "tags", "add", id, "shared=visible")
		testutil.Ok(t, err)
	}
	out, err = runTagsCLI(dbPath, "owner", "tags", "show", "shared=visible")
	testutil.Ok(t, err)
	testutil.StringContains(t, out, "ENTITY TYPE")
	for _, name := range []string{"CLI Highball", "CLI Gin", "Inventory for CLI Gin", "CLI Menu", "Order for CLI Menu"} {
		testutil.StringContains(t, out, name)
	}
	for _, id := range all {
		testutil.StringContains(t, out, id)
	}

	out, err = runTagsCLI(dbPath, "owner", "tags", "show", "--key", "target")
	testutil.Ok(t, err)
	for name, id := range all {
		testutil.StringContains(t, out, id)
		testutil.StringContains(t, out, "target="+name)
	}

	_, err = runTagsCLI(dbPath, "bartender", "tags", "show", "shared=visible")
	assertCLIExitCode(t, err, errors.ExitPermission)

	_, err = runTagsCLI(dbPath, "bartender", "tags", "add", targets.ingredient, "denied")
	assertCLIExitCode(t, err, errors.ExitPermission)
	out, err = runTagsCLI(dbPath, "owner", "tags", "list", targets.ingredient)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, bytes.Contains([]byte(out), []byte("denied")), "denied tag was persisted: %s", out)

	_, err = runTagsCLI(dbPath, "owner", "tags", "list", "wat-3BxsD9vQRgeYqJ8v4bFVvytN")
	assertCLIExitCode(t, err, errors.ExitInvalid)
	_, err = runTagsCLI(dbPath, "owner", "tags", "list", "drk-not-a-ksuid")
	assertCLIExitCode(t, err, errors.ExitInvalid)
	_, err = runTagsCLI(dbPath, "owner", "tags", "list", targets.auditEntry)
	assertCLIExitCode(t, err, errors.ExitInvalid)
	_, err = runTagsCLI(dbPath, "owner", "tags", "add", targets.drink)
	assertCLIExitCode(t, err, errors.ExitUsage)

	out, err = runTagsCLI(dbPath, "owner", "tags", "add", "--json", targets.drink, "featured")
	testutil.Ok(t, err)
	var doc tagsOutput
	testutil.Ok(t, json.Unmarshal([]byte(out), &doc))
	testutil.Equals(t, doc.EntityID, targets.drink)
	testutil.NotNil(t, doc.Changed)
	testutil.IsFalse(t, *doc.Changed)
	testutil.Equals(t, doc.Tags.String(), "featured,shared=visible,target=drink")

	// These are separate CLI/application lifecycles over the same database, so
	// the final list demonstrates persistence rather than in-memory state.
	out, err = runTagsCLI(dbPath, "owner", "tags", "list", "--json", targets.drink)
	testutil.Ok(t, err)
	doc = tagsOutput{}
	testutil.Ok(t, json.Unmarshal([]byte(out), &doc))
	testutil.Nil(t, doc.Changed)
	testutil.Equals(t, doc.Tags.String(), "featured,shared=visible,target=drink")

	for _, tc := range []struct {
		noun string
		args []string
		tag  string
	}{
		{"drinks", []string{"list"}, "target=drink"},
		{"ingredients", []string{"list"}, "target=ingredient"},
		{"inventory", []string{"list"}, "target=inventory"},
		{"menus", []string{"list"}, "target=menu"},
		{"orders", []string{"list"}, "target=order"},
	} {
		args := append([]string{tc.noun}, tc.args...)
		out, err = runTagsCLI(dbPath, "owner", args...)
		testutil.Ok(t, err)
		testutil.StringContains(t, out, "TAGS")
		testutil.StringContains(t, out, tc.tag)
	}

	out, err = runTagsCLI(dbPath, "owner", "inventory", "get", "--ingredient-id", targets.ingredientID)
	testutil.Ok(t, err)
	testutil.StringContains(t, out, "ID:")
	testutil.StringContains(t, out, targets.inventory)
	testutil.StringContains(t, out, "Tags:")

	softDeleteCLIDrink(t, dbPath, targets.drink)
	out, err = runTagsCLI(dbPath, "owner", "tags", "show", "shared=visible")
	testutil.Ok(t, err)
	testutil.ErrorIf(t, strings.Contains(out, targets.drink), "deleted drink appeared in tag discovery: %s", out)
	for name, id := range all {
		if name != "drink" {
			testutil.StringContains(t, out, id)
		}
	}

	out, err = runTagsCLI(dbPath, "owner", "tags", "summary")
	testutil.Ok(t, err)
	testutil.StringContains(t, out, "TAG")
	testutil.StringContains(t, out, "shared=visible")
	out, err = runTagsCLI(dbPath, "owner", "tags", "summary", "--json")
	testutil.Ok(t, err)
	var summaries []tagging.Summary
	testutil.Ok(t, json.Unmarshal([]byte(out), &summaries))
	testutil.Equals(t, summaries[0].Tag, "shared=visible")
	testutil.Equals(t, summaries[0].Total, 4)
	testutil.Equals(t, summaries[0].Drinks, 0)
	testutil.Equals(t, summaries[0].Ingredients, 1)
	testutil.Equals(t, summaries[0].Inventory, 1)
	testutil.Equals(t, summaries[0].Menus, 1)
	testutil.Equals(t, summaries[0].Orders, 1)
}

func runTagsCLI(dbPath, actor string, args ...string) (string, error) {
	result := newCLIE2E(dbPath).As(actor).Run(args...)
	return result.Stdout + result.Stderr, result.Err
}

func assertCLIExitCode(t *testing.T, err error, want int) {
	t.Helper()
	var exit cli.ExitCoder
	testutil.ErrorAs(t, err, &exit)
	testutil.Equals(t, exit.ExitCode(), want)
}

func seedCLITagTargets(t *testing.T, dbPath string) cliTagTargets {
	t.Helper()
	ctx := authn.ToContext(context.Background(), authn.Owner())
	ctx = pkglog.ToContext(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s, err := store.Open(ctx, dbPath)
	testutil.Ok(t, err)
	a := app.New(ctx, app.Config{Store: s})
	mctx := middleware.NewContext(ctx)

	ingredient, err := a.Ingredients.Create(mctx, &ingredientsmodels.Ingredient{
		Name: "CLI Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	testutil.Ok(t, err)
	drink, err := a.Drinks.Create(mctx, &drinksmodels.Drink{
		Name: "CLI Highball", Category: drinksmodels.DrinkCategoryHighball, Glass: drinksmodels.GlassTypeHighball,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"build"},
		},
	})
	testutil.Ok(t, err)
	stock, err := a.Inventory.Set(mctx, &inventorymodels.Update{
		IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, measurement.UnitOz),
		CostPerUnit: money.NewPriceFromCents(100, currency.USD),
	})
	testutil.Ok(t, err)
	menu, err := a.Menus.Create(mctx, &menusmodels.Menu{Name: "CLI Menu"})
	testutil.Ok(t, err)
	menu, err = a.Menus.AddDrink(mctx, &menusmodels.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	menu, err = a.Menus.Publish(mctx, &menusmodels.Menu{ID: menu.ID})
	testutil.Ok(t, err)
	order, err := a.Orders.Place(mctx, &ordersmodels.Order{
		MenuID: menu.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
	})
	testutil.Ok(t, err)
	testutil.Ok(t, a.Close())

	return cliTagTargets{
		drink: drink.ID.String(), ingredient: ingredient.ID.String(), inventory: stock.ID.String(),
		menu: menu.ID.String(), order: order.ID.String(), ingredientID: ingredient.ID.String(),
		auditEntry: entity.NewAuditEntryID().String(),
	}
}

func softDeleteCLIDrink(t *testing.T, dbPath, rawID string) {
	t.Helper()
	ctx := authn.ToContext(context.Background(), authn.Owner())
	ctx = pkglog.ToContext(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s, err := store.Open(ctx, dbPath)
	testutil.Ok(t, err)
	a := app.New(ctx, app.Config{Store: s})
	id, err := entity.ParseDrinkID(rawID)
	testutil.Ok(t, err)
	_, err = a.Drinks.Delete(middleware.NewContext(ctx), id)
	testutil.Ok(t, err)
	testutil.Ok(t, a.Close())
}

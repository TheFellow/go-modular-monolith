package tui_test

import (
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	inventorytui "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
)

func TestDetailViewModel_ShowsQuantityAndCost(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap().WithBasicIngredients()

	ingredient := b.WithIngredient("Orgeat", measurement.UnitOz)
	price := money.NewPriceFromCents(120, currency.USD)
	inv, err := f.Inventory.Set(f.OwnerContext(), &models.Update{
		IngredientID: ingredient.ID,
		Amount:       measurement.MustAmount(3, measurement.UnitOz),
		CostPerUnit:  price,
	})
	testutil.Ok(t, err)

	row := inventorytui.InventoryRow{
		Inventory:  *inv,
		Ingredient: *ingredient,
		Quantity:   inv.Amount.String(),
		Cost:       price.String(),
		Status:     "LOW",
	}

	detail := inventorytui.NewDetailViewModel(tuitest.DefaultListViewStyles[tui.ListViewStyles]())
	detail.SetSize(80, 40)
	detail.SetRow(optional.Some(row))

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Orgeat"), "expected ingredient name in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, ingredient.ID.String()), "expected ingredient id in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, inv.ID.String()), "expected inventory id in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, row.Quantity), "expected quantity in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, row.Cost), "expected cost in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "LOW"), "expected status in view, got:\n%s", view)
}

func TestDetailViewModel_NilRow(t *testing.T) {
	t.Parallel()
	detail := inventorytui.NewDetailViewModel(tuitest.DefaultListViewStyles[tui.ListViewStyles]())
	detail.SetRow(optional.None[inventorytui.InventoryRow]())

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Select a stock item"), "expected placeholder view, got:\n%s", view)
}

func TestDetailViewModel_SetSize(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap().WithBasicIngredients()
	ingredient := b.WithIngredient("Orgeat", measurement.UnitOz)
	price := money.NewPriceFromCents(120, currency.USD)
	inv, err := f.Inventory.Set(f.OwnerContext(), &models.Update{
		IngredientID: ingredient.ID,
		Amount:       measurement.MustAmount(3, measurement.UnitOz),
		CostPerUnit:  price,
	})
	testutil.Ok(t, err)

	row := inventorytui.InventoryRow{
		Inventory:  *inv,
		Ingredient: *ingredient,
		Quantity:   inv.Amount.String(),
		Cost:       price.String(),
		Status:     "LOW",
	}

	detail := inventorytui.NewDetailViewModel(tuitest.DefaultListViewStyles[tui.ListViewStyles]())
	detail.SetRow(optional.Some(row))
	detail.SetSize(20, 10)

	view := detail.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view after resizing")
}

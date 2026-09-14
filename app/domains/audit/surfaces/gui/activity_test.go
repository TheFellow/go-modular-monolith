//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package gui

import (
	"strings"
	"testing"

	framework "fyne.io/fyne/v2"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	dm "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	im "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	iv "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	om "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestAuditDetailShowsCommittedAndRolledBackBatchEffects(t *testing.T) {
	for _, success := range []bool{false, true} {
		name := "failed"
		if success {
			name = "succeeded"
		}
		t.Run(name, func(t *testing.T) {
			f := testutil.NewFixture(t)
			original := testutil.CreateIngredient(t, f, im.Ingredient{Name: "Original", Category: im.CategorySpirit, Unit: measurement.UnitOz})
			replacement := testutil.CreateIngredient(t, f, im.Ingredient{Name: "Replacement", Category: im.CategorySpirit, Unit: measurement.UnitOz})
			stock := func(ingredient *im.Ingredient, quantity float64) {
				testutil.SetInventory(t, f, iv.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(quantity, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
			}
			stock(original, 10)
			available := 2.0
			if success {
				available = 10
			}
			stock(replacement, available)
			drink := testutil.CreateDrink(t, f, dm.Drink{Name: "Cocktail", Recipe: dm.Recipe{Steps: []string{"Stir"}, Ingredients: []dm.RecipeIngredient{{IngredientID: original.ID, Amount: measurement.MustAmount(2, original.Unit)}}}})
			menu := testutil.CreateMenu(t, f, "Menu", testutil.WithDrink(drink), testutil.Published())
			var amendments []om.Amendment
			for range 2 {
				order := testutil.PlaceOrder(t, f, om.Order{MenuID: menu.ID, Items: []om.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
				amendments = append(amendments, om.Amendment{OrderID: order.ID, Revision: order.Revision, Reason: "approved", Replacements: []om.Replacement{{OriginalID: original.ID, ReplacementID: replacement.ID, Ratio: 1}}})
			}
			_, err := f.Orders.AmendBatch(f.OwnerContext(), amendments)
			if success {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsFailedPrecondition(t, err)
			}
			page, err := f.Audit.List(f.OwnerContext(), audit.ListRequest{})
			testutil.Ok(t, err)
			presenter := auditPresenter(f)
			view := NewView(presenter)
			for _, field := range view.detailFields {
				if field.MultiLine {
					testutil.Equals(t, field.Wrapping, framework.TextWrapWord)
				}
			}
			view.Activate()
			index := -1
			for i, row := range presenter.State().Rows {
				if strings.Contains(row.Entry.Action, "amend") {
					index = i
					break
				}
			}
			testutil.ErrorIf(t, index < 0, "batch audit missing in %d entries", len(page.Items))
			presenter.Select(index)
			state := presenter.State()
			entry := state.Selected.Entry
			testutil.Equals(t, entry.Success, success)
			testutil.ErrorIf(t, len(entry.Participants) == 0, "batch references missing")
			for _, participant := range entry.Participants {
				testutil.StringContains(t, view.detailFields[10].Text, participant.String())
			}
			heading := "Attempted effects (not committed)"
			if success {
				heading = "Committed effects"
			}
			testutil.StringContains(t, view.detailFields[11].Text, heading)
			testutil.StringContains(t, view.detailFields[11].Text, "Before: ")
			testutil.StringContains(t, view.detailFields[11].Text, "After: ")
			testutil.ErrorIf(t, len(entry.Effects) == 0 || len(entry.Participants) == 0 || len(entry.Effects[0].Changes) == 0, "incomplete activity: %+v", entry)
			for _, snapshot := range []*Row{&state.Rows[index], state.Selected} {
				snapshot.Entry.Participants[0] = entity.NewIngredientID().EntityUID()
				snapshot.Entry.Effects[0].Kind = "corrupt"
				snapshot.Entry.Effects[0].Changes[0].Before = "corrupt"
			}
			fresh := presenter.State()
			testutil.Equals(t, fresh.Rows[index].Entry.Effects[0].Kind, fresh.Selected.Entry.Effects[0].Kind)
			testutil.ErrorIf(t, strings.Contains(view.detailFields[11].Text, "corrupt"), "rendered detail aliases snapshot")
			testutil.NotEquals(t, fresh.Selected.Entry.Effects[0].Kind, "corrupt")
			testutil.NotEquals(t, fresh.Selected.Entry.Effects[0].Changes[0].Before, "corrupt")
			testutil.Equals(t, fresh.Selected.Entry.Participants, page.Items[index].Participants)
		})
	}
}

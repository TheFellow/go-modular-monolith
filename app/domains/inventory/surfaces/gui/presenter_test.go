//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package gui

import (
	"strings"
	"testing"
	"time"

	frameworktest "fyne.io/fyne/v2/test"

	application "github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	fyneui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestInventoryDetailLabelsIncludeExactLastUpdated(t *testing.T) {
	fix, _ := inventoryFixture(t)
	p := NewPresenter(fix.App, fyneui.InlineExecutor{}, fyneui.InlineDispatcher{})
	p.Load()
	row := p.Snapshot().Rows[0]
	labels := strings.Join(inventoryDetailLabels(&row, "None"), "\n")
	testutil.ErrorIf(t, !strings.Contains(labels, "Last updated: "+row.Inventory.LastUpdated.Format(time.RFC3339)), "exact last-updated value missing: %s", labels)
}

func TestValidateSetOptionalCostContract(t *testing.T) {
	existing := optional.Some(money.NewPriceFromCents(123, currency.USD))
	for _, tc := range []struct {
		name, raw, want string
		existing        optional.Value[money.Price]
	}{
		{"blank preserves existing", "", "$1.23", existing},
		{"blank new defaults USD zero", "", "$0.00", optional.None[money.Price]()},
		{"explicit USD", "USD 2.50", "$2.50", existing},
		{"explicit EUR changes currency", "EUR 2.50", "2.50 €", existing},
	} {
		t.Run(tc.name, func(t *testing.T) {
			validated, err := validate(Set, Form{Amount: "1", Cost: tc.raw}, measurement.UnitOz, tc.existing)
			testutil.Ok(t, err)
			price, ok := validated.cost.Unwrap()
			if !ok {
				t.Fatal("cost missing")
			}
			testutil.Equals(t, price.String(), tc.want)
		})
	}
}

func inventoryFixture(t *testing.T) (*testutil.Fixture, *models.Ingredient) {
	t.Helper()
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, models.Ingredient{Name: "London Gin", Category: models.CategorySpirit, Unit: measurement.UnitOz})
	_, err := fix.Inventory.Set(fix.OwnerContext(), &inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(12.5, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(325, currency.USD)})
	testutil.Ok(t, err)
	return fix, ingredient
}

func TestPresenterLoadsJoinedDetailsFiltersAndRefreshes(t *testing.T) {
	fix, ingredient := inventoryFixture(t)
	p := NewPresenter(fix.App, fyneui.InlineExecutor{}, fyneui.InlineDispatcher{})
	p.Load()
	state := p.Snapshot()
	if len(state.Rows) != 1 || state.Selected == nil || state.Selected.Ingredient.ID != ingredient.ID || state.Selected.Quantity != "12.50 oz" || state.Selected.Cost != "$3.25" || state.Selected.Status != "OK" {
		t.Fatalf("joined state = %#v", state)
	}
	p.Filter(LowStock, "", 10, 25)
	if got := p.Snapshot(); len(got.Rows) != 0 || got.Stock != LowStock || got.Limit != 25 {
		t.Fatalf("low stock state = %#v", got)
	}
	p.Filter(AllStock, `quantity <= 13 && unit == "oz"`, 10, 25)
	if got := p.Snapshot(); len(got.Rows) != 1 {
		t.Fatalf("expression state = %#v", got)
	}
	p.Filter(AllStock, "(", 10, 25)
	if got := p.Snapshot(); got.Status != fyneui.Failed || got.Err == nil {
		t.Fatalf("invalid filter state = %#v", got)
	}
}

func TestPresenterAdjustSetAndTagClearPersist(t *testing.T) {
	fix, ingredient := inventoryFixture(t)
	p := NewPresenter(fix.App, fyneui.InlineExecutor{}, fyneui.InlineDispatcher{})
	p.Load()
	p.StartAdjust()
	if !p.Submit(Form{Amount: "-2.25", Reason: inventorymodels.ReasonUsed}) {
		t.Fatal("adjust rejected")
	}
	stock, err := fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	if stock.Amount.Value() != 10.25 {
		t.Fatalf("adjusted=%v", stock.Amount.Value())
	}
	testutil.AuditTouches(t, fix.LatestAuditEntry(inventoryauthz.ActionAdjust), stock.EntityUID())
	p.StartSet()
	if !p.Submit(Form{Amount: "4.50", Cost: "1.75"}) {
		t.Fatal("set rejected")
	}
	stock, err = fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	price, ok := stock.CostPerUnit.Unwrap()
	if !ok {
		t.Fatal("cost missing")
	}
	cents, _ := price.Cents()
	if stock.Amount.Value() != 4.5 || cents != 175 {
		t.Fatalf("set stock=%#v", stock)
	}
	testutil.AuditTouches(t, fix.LatestAuditEntry(inventoryauthz.ActionSet), stock.EntityUID())
	p.StartTags()
	if !p.Submit(Form{Tags: "featured, region=west"}) {
		t.Fatal("tags rejected")
	}
	stock, err = fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	if stock.Tags.Canonical().String() != "featured,region=west" {
		t.Fatalf("tags=%q", stock.Tags.Canonical().String())
	}
	testutil.AuditTouches(t, fix.LatestAuditEntry(inventoryauthz.ActionTag), stock.EntityUID())
	p.StartTags()
	p.Submit(Form{})
	stock, err = fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	if stock.Tags.Canonical().String() != "" {
		t.Fatalf("tags not cleared: %q", stock.Tags.Canonical().String())
	}
}

func TestPresenterPermissionFailureRetainsFormWithoutMutation(t *testing.T) {
	fix, ingredient := inventoryFixture(t)
	denied := application.NewSession(fix.ActorContext("bartender"), fix.App.App)
	p := NewPresenter(denied, fyneui.InlineExecutor{}, fyneui.InlineDispatcher{})
	p.Load()
	p.StartAdjust()
	form := Form{Amount: "2.00", Reason: inventorymodels.ReasonReceived}
	if !p.Submit(form) {
		t.Fatal("valid request was not submitted")
	}
	state := p.Snapshot()
	if state.Mode != Adjust || state.Err == nil || state.Form != form {
		t.Fatalf("permission failure state = %#v", state)
	}
	testutil.ErrorIsPermission(t, state.Err)
	stock, err := fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	if stock.Amount.Value() != 12.5 {
		t.Fatalf("denied adjustment mutated stock: %#v", stock)
	}
}

func TestPresenterValidationRetainsFormAndRejectsDuplicate(t *testing.T) {
	fix, _ := inventoryFixture(t)
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(fix.App, executor, fyneui.InlineDispatcher{})
	p.Load()
	executor.RunNext()
	p.StartAdjust()
	if p.Submit(Form{Amount: "1.234", Reason: inventorymodels.ReasonUsed}) || executor.Pending() != 0 {
		t.Fatal("precision validation scheduled mutation")
	}
	if p.Snapshot().Mode != Adjust || p.Snapshot().Err == nil {
		t.Fatalf("form not retained: %#v", p.Snapshot())
	}
	if !p.Submit(Form{Amount: "1.25", Reason: inventorymodels.ReasonReceived}) || p.Submit(Form{Amount: "1.25", Reason: inventorymodels.ReasonReceived}) || executor.Pending() != 1 {
		t.Fatal("duplicate was not rejected")
	}
}

func TestPresenterRejectsStaleOutOfOrderLoads(t *testing.T) {
	fix, _ := inventoryFixture(t)
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(fix.App, executor, fyneui.InlineDispatcher{})
	p.Filter(AllStock, "", 10, 100)
	p.Filter(LowStock, "", 10, 100)
	if !executor.Run(1) || !executor.RunNext() {
		t.Fatal("expected loads")
	}
	if got := p.Snapshot(); got.Stock != LowStock || len(got.Rows) != 0 {
		t.Fatalf("stale load published: %#v", got)
	}
}

func TestViewDrivesRealRetainedWidgets(t *testing.T) {
	fix, _ := inventoryFixture(t)
	p := NewPresenter(fix.App, fyneui.InlineExecutor{}, fyneui.InlineDispatcher{})
	view := NewView(p)
	view.Activate()
	if len(view.rows) != 1 {
		t.Fatalf("rows=%d", len(view.rows))
	}
	frameworktest.Tap(view.rows[p.Snapshot().Rows[0].Inventory.ID.String()])
	p.StartAdjust()
	selected := p.Snapshot().Selected.Inventory.ID
	p.Select(entity.InventoryID{})
	if p.Snapshot().Selected.Inventory.ID != selected {
		t.Fatal("presenter changed selection during adjustment")
	}
	if !view.rows[p.Snapshot().Rows[0].Inventory.ID.String()].Disabled() {
		t.Fatal("row selection remained enabled during adjustment")
	}
	frameworktest.Type(view.amount, "2.00")
	view.reason.SetSelected(string(inventorymodels.ReasonReceived))
	frameworktest.Tap(view.save)
	if got := p.Snapshot(); got.Mode != Browse || got.Selected.Status != "OK" {
		t.Fatalf("widget state=%#v", got)
	}
}

func TestStockStatusThresholds(t *testing.T) {
	for _, tc := range []struct {
		value float64
		want  string
	}{{0, "OUT"}, {10, "LOW"}, {10.01, "OK"}} {
		{
			if got := StockStatus(measurement.MustAmount(tc.value, measurement.UnitOz), 10); got != tc.want {
				t.Fatalf("status(%v)=%s", tc.value, got)
			}
		}
	}
}

func TestPresenterUsesConfigurableLowStockThreshold(t *testing.T) {
	fix, _ := inventoryFixture(t)
	p := NewPresenter(fix.App, fyneui.InlineExecutor{}, fyneui.InlineDispatcher{})
	if !p.Filter(LowStock, "", 13, 25) {
		t.Fatal("filter rejected")
	}
	state := p.Snapshot()
	if len(state.Rows) != 1 || state.Rows[0].Status != "LOW" || state.LowStock != 13 {
		t.Fatalf("custom threshold state = %#v", state)
	}
	if p.Filter(LowStock, "", -1, 25) || p.Snapshot().Err == nil {
		t.Fatal("negative threshold accepted")
	}
}

func TestPresenterSetBlankCostPreservesExistingPrice(t *testing.T) {
	fix, ingredient := inventoryFixture(t)
	p := NewPresenter(fix.App, fyneui.InlineExecutor{}, fyneui.InlineDispatcher{})
	p.Load()
	p.StartSet()
	if !p.Submit(Form{Amount: "9.00"}) {
		t.Fatal("set rejected")
	}
	stock, err := fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	price, ok := stock.CostPerUnit.Unwrap()
	if !ok {
		t.Fatal("cost missing")
	}
	cents, err := price.Cents()
	testutil.Ok(t, err)
	if cents != 325 {
		t.Fatalf("cost=%d cents", cents)
	}
}

func TestPresenterAdjustsCostWithoutQuantityMutation(t *testing.T) {
	fix, ingredient := inventoryFixture(t)
	p := NewPresenter(fix.App, fyneui.InlineExecutor{}, fyneui.InlineDispatcher{})
	p.Load()
	p.StartAdjust()
	if !p.Submit(Form{Cost: "4.10", Reason: inventorymodels.ReasonCorrected}) {
		t.Fatal("cost-only adjustment rejected")
	}
	stock, err := fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	price, ok := stock.CostPerUnit.Unwrap()
	if !ok {
		t.Fatal("cost missing")
	}
	cents, err := price.Cents()
	testutil.Ok(t, err)
	if stock.Amount.Value() != 12.5 || cents != 410 {
		t.Fatalf("stock=%#v", stock)
	}
}

func TestValidateAdjustAcceptsCurrencyBearingPrice(t *testing.T) {
	validated, err := validate(Adjust, Form{Cost: "EUR 4.10", Reason: inventorymodels.ReasonCorrected}, measurement.UnitOz, optional.None[money.Price]())
	testutil.Ok(t, err)
	price, ok := validated.cost.Unwrap()
	if !ok || price.Currency != currency.EUR || price.String() != "4.10 €" {
		t.Fatalf("currency-bearing cost = %#v", validated.cost)
	}
}

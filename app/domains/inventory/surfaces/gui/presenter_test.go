//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package gui

import (
	"context"
	stderrors "errors"
	"slices"
	"strings"
	"testing"
	"time"

	framework "fyne.io/fyne/v2"
	frameworktest "fyne.io/fyne/v2/test"

	application "github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventory "github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestPresenterProjectsRowsAndRechecksStaleCapabilities(t *testing.T) {
	fix, _ := inventoryFixture(t)
	denied := false
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	p.projector = inventory.ActionProjector{Authorize: func(_ context.Context, _ cedar.EntityUID, action cedar.EntityUID, _ cedar.Entity) error {
		if denied && action == inventoryauthz.ActionAdjust {
			return apperrors.Permissionf("adjust revoked")
		}
		return nil
	}}
	p.Load()
	state := p.Snapshot()
	testutil.Equals(t, state.Rows[0].Actions[inventory.ControlAdjust].Visible, true)
	p.Select(state.Rows[0].Inventory.ID)
	denied = true
	p.StartAdjust()
	state = p.Snapshot()
	testutil.Equals(t, state.Mode, Viewing)
	testutil.Equals(t, state.Actions[inventory.ControlAdjust].Visible, false)
}

func TestPresenterSurfacesProjectionEvaluatorFailure(t *testing.T) {
	fix, _ := inventoryFixture(t)
	want := stderrors.New("policy evaluator unavailable")
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	p.projector = inventory.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return want }}
	p.Load()
	state := p.Snapshot()
	testutil.ErrorIf(t, state.Status != toolkit.Failed || !stderrors.Is(state.Err, want), "failed projection state = %#v", state)
}

func TestPresenterProjectionFailureCanRecover(t *testing.T) {
	fix, _ := inventoryFixture(t)
	want := stderrors.New("policy evaluator unavailable")
	failing := true
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	p.projector = inventory.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error {
		if failing {
			return want
		}
		return nil
	}}
	p.mu.Lock()
	p.state.Selected = &Row{Inventory: inventorymodels.Inventory{ID: entity.NewInventoryID(), IngredientID: entity.NewIngredientID(), Amount: measurement.MustAmount(1, measurement.UnitOz)}}
	err := p.permissionsLocked()
	failing = false
	recovered := p.permissionsLocked()
	state := cloneState(p.state)
	p.mu.Unlock()
	testutil.ErrorIf(t, !stderrors.Is(err, want), "projection error = %v", err)
	testutil.Ok(t, recovered)
	testutil.Equals(t, actionEnabled(state.Actions, inventory.ControlList), true)
}

func TestInventoryDetailLabelsIncludeExactLastUpdated(t *testing.T) {
	fix, _ := inventoryFixture(t)
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
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
			testutil.ErrorIf(t, !ok, "%v", "cost missing")
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
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	p.Load()
	state := p.Snapshot()
	testutil.ErrorIf(t, len(state.Rows) != 1 || state.Selected == nil || state.Selected.Ingredient.ID != ingredient.ID || state.Selected.Quantity != "12.50 oz" || state.Selected.Cost != "$3.25" || state.Selected.Status != "OK", "joined state = %#v", state)
	p.Filter(LowStock, "", 10, 25)
	{
		got := p.Snapshot()
		testutil.ErrorIf(t, len(got.Rows) != 0 || got.Stock != LowStock || got.Limit != 25, "low stock state = %#v", got)
	}
	p.Filter(AllStock, `quantity <= 13 && unit == "oz"`, 10, 25)
	{
		got := p.Snapshot()
		testutil.ErrorIf(t, len(got.Rows) != 1, "expression state = %#v", got)
	}
	p.Filter(AllStock, "(", 10, 25)
	{
		got := p.Snapshot()
		testutil.ErrorIf(t, got.Status != toolkit.Failed || got.Err == nil, "invalid filter state = %#v", got)
	}
}

func TestPresenterAdjustSetAndTagClearPersist(t *testing.T) {
	fix, ingredient := inventoryFixture(t)
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	p.Load()
	p.StartAdjust()
	testutil.ErrorIf(t, !p.Submit(Form{Amount: "-2.25", Reason: inventorymodels.ReasonUsed}), "%v", "adjust rejected")
	stock, err := fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, stock.Amount.Value() != 10.25, "adjusted=%v", stock.Amount.Value())
	testutil.AuditTouches(t, fix.LatestAuditEntry(inventoryauthz.ActionAdjust), stock.EntityUID())
	p.StartSet()
	testutil.ErrorIf(t, !p.Submit(Form{Amount: "4.50", Cost: "1.75"}), "%v", "set rejected")
	stock, err = fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	price, ok := stock.CostPerUnit.Unwrap()
	testutil.ErrorIf(t, !ok, "%v", "cost missing")
	cents, _ := price.Cents()
	testutil.ErrorIf(t, stock.Amount.Value() != 4.5 || cents != 175, "set stock=%#v", stock)
	testutil.AuditTouches(t, fix.LatestAuditEntry(inventoryauthz.ActionSet), stock.EntityUID())
	p.StartTags()
	testutil.ErrorIf(t, !p.Submit(Form{Tags: "featured, region=west"}), "%v", "tags rejected")
	stock, err = fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, stock.Tags.Canonical().String() != "featured,region=west", "tags=%q", stock.Tags.Canonical().String())
	testutil.AuditTouches(t, fix.LatestAuditEntry(inventoryauthz.ActionTag), stock.EntityUID())
	p.StartTags()
	p.Submit(Form{})
	stock, err = fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, stock.Tags.Canonical().String() != "", "tags not cleared: %q", stock.Tags.Canonical().String())
}

func TestPresenterPermissionFailureRetainsFormWithoutMutation(t *testing.T) {
	fix, ingredient := inventoryFixture(t)
	denied := application.NewSession(fix.ActorContext("bartender"), fix.App.App)
	p := NewPresenter(denied, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	p.Load()
	p.StartAdjust()
	state := p.Snapshot()
	testutil.ErrorIf(t, state.Mode != Browse || state.CanAdjust || state.CanSet || state.CanTag, "read-only actor exposed mutation state = %#v", state)
	stock, err := fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, stock.Amount.Value() != 12.5, "denied adjustment mutated stock: %#v", stock)
}

func TestPresenterValidationRetainsFormAndRejectsDuplicate(t *testing.T) {
	fix, _ := inventoryFixture(t)
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(fix.App, executor, toolkit.InlineDispatcher{})
	p.Load()
	executor.RunNext()
	p.StartAdjust()
	testutil.ErrorIf(t, p.Submit(Form{Amount: "1.234", Reason: inventorymodels.ReasonUsed}) || executor.Pending() != 0, "%v", "precision validation scheduled mutation")
	testutil.ErrorIf(t, p.Snapshot().Mode != Adjust || p.Snapshot().Err == nil, "form not retained: %#v", p.Snapshot())
	testutil.ErrorIf(t, !p.Submit(Form{Amount: "1.25", Reason: inventorymodels.ReasonReceived}) || p.Submit(Form{Amount: "1.25", Reason: inventorymodels.ReasonReceived}) || executor.Pending() != 1, "%v", "duplicate was not rejected")
}

func TestPresenterRejectsStaleOutOfOrderLoads(t *testing.T) {
	fix, _ := inventoryFixture(t)
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(fix.App, executor, toolkit.InlineDispatcher{})
	p.Filter(AllStock, "", 10, 100)
	p.Filter(LowStock, "", 10, 100)
	testutil.ErrorIf(t, !executor.Run(1) || !executor.RunNext(), "%v", "expected loads")
	{
		got := p.Snapshot()
		testutil.ErrorIf(t, got.Stock != LowStock || len(got.Rows) != 0, "stale load published: %#v", got)
	}
}

func TestViewDrivesRealRetainedWidgets(t *testing.T) {
	fix, _ := inventoryFixture(t)
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	view := NewView(p)
	view.Activate()
	testutil.ErrorIf(t, len(view.rows) != 1, "rows=%d", len(view.rows))
	frameworktest.Tap(view.rows[p.Snapshot().Rows[0].Inventory.ID.String()])
	p.StartAdjust()
	selected := p.Snapshot().Selected.Inventory.ID
	p.Select(entity.InventoryID{})
	testutil.ErrorIf(t, p.Snapshot().Selected.Inventory.ID != selected, "%v", "presenter changed selection during adjustment")
	testutil.ErrorIf(t, !view.rows[p.Snapshot().Rows[0].Inventory.ID.String()].Disabled(), "%v", "row selection remained enabled during adjustment")
	frameworktest.Type(view.amount, "2.00")
	view.reason.SetSelected(string(inventorymodels.ReasonReceived))
	frameworktest.Tap(view.save)
	{
		got := p.Snapshot()
		testutil.ErrorIf(t, got.Mode != Viewing || got.Selected.Status != "OK", "widget state=%#v", got)
	}
}

func TestStockStatusThresholds(t *testing.T) {
	for _, tc := range []struct {
		value float64
		want  string
	}{{0, "OUT"}, {10, "LOW"}, {10.01, "OK"}} {
		{
			{
				got := StockStatus(measurement.MustAmount(tc.value, measurement.UnitOz), 10)
				testutil.ErrorIf(t, got != tc.want, "status(%v)=%s", tc.value, got)
			}
		}
	}
}

func TestPresenterUsesConfigurableLowStockThreshold(t *testing.T) {
	fix, _ := inventoryFixture(t)
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	testutil.ErrorIf(t, !p.Filter(LowStock, "", 13, 25), "%v", "filter rejected")
	state := p.Snapshot()
	testutil.ErrorIf(t, len(state.Rows) != 1 || state.Rows[0].Status != "LOW" || state.LowStock != 13, "custom threshold state = %#v", state)
	testutil.ErrorIf(t, p.Filter(LowStock, "", -1, 25) || p.Snapshot().Err == nil, "%v", "negative threshold accepted")
}

func TestInventoryListDetailBackAndResetSemantics(t *testing.T) {
	fix, _ := inventoryFixture(t)
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	p.Filter(AllStock, `quantity <= 13`, 10, 25)
	row := p.Snapshot().Rows[0]
	p.Select(row.Inventory.ID)
	{
		got := p.Snapshot()
		testutil.ErrorIf(t, got.Mode != Viewing || got.Expression != `quantity <= 13` || got.Limit != 25, "detail state = %#v", got)
	}
	p.Back()
	{
		got := p.Snapshot()
		testutil.ErrorIf(t, got.Mode != Browse || got.Expression != `quantity <= 13` || got.Limit != 25, "back did not preserve list state = %#v", got)
	}
	p.Select(row.Inventory.ID)
	p.ResetList()
	{
		got := p.Snapshot()
		testutil.ErrorIf(t, got.Mode != Browse || got.Expression != "" || got.Limit != toolkit.PageLimit || got.Cursor != "" || len(got.History) != 0, "breadcrumb did not reset list = %#v", got)
	}
}

func TestInventoryMutationDirtyCancelAndSaveStayInDetail(t *testing.T) {
	fix, _ := inventoryFixture(t)
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	p.Load()
	p.Select(p.Snapshot().Rows[0].Inventory.ID)
	p.StartSet()
	baseline := p.Snapshot().Form
	testutil.ErrorIf(t, p.Snapshot().Dirty, "%v", "fresh set form is dirty")
	changed := baseline
	changed.Amount = "8.00"
	p.SetForm(changed)
	testutil.ErrorIf(t, !p.Snapshot().Dirty, "%v", "edited set form is not dirty")
	p.Cancel()
	{
		got := p.Snapshot()
		testutil.ErrorIf(t, got.Mode != Viewing || got.Dirty, "cancel state = %#v", got)
	}
	p.StartSet()
	changed = p.Snapshot().Form
	changed.Amount = "8.00"
	p.SetForm(changed)
	testutil.ErrorIf(t, !p.Submit(changed), "%v", "save rejected")
	{
		got := p.Snapshot()
		testutil.ErrorIf(t, got.Mode != Viewing || got.Dirty || got.Selected.Inventory.Amount.Value() != 8, "saved detail state = %#v", got)
	}
}

func TestDirtyInventoryNavigationConfirmsThenPreservesOrResetsList(t *testing.T) {
	fix, _ := inventoryFixture(t)
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{}, dialogs)
	p.Filter(AllStock, `quantity <= 13`, 10, 25)
	p.Select(p.Snapshot().Rows[0].Inventory.ID)
	p.StartSet()
	f := p.Snapshot().Form
	f.Amount = "9.00"
	p.SetForm(f)
	p.Back()
	testutil.ErrorIf(t, p.Snapshot().Mode != Set || len(dialogs.Confirmations()) != 1, "%v", "dirty Back did not confirm")
	dialogs.Confirmations()[0].Respond(true)
	{
		got := p.Snapshot()
		testutil.ErrorIf(t, got.Mode != Browse || got.Expression != `quantity <= 13` || got.Limit != 25, "confirmed Back state = %#v", got)
	}
	p.Select(p.Snapshot().Rows[0].Inventory.ID)
	p.StartSet()
	f = p.Snapshot().Form
	f.Amount = "8.00"
	p.SetForm(f)
	p.ResetList()
	testutil.ErrorIf(t, len(dialogs.Confirmations()) != 2, "%v", "dirty breadcrumb did not confirm")
	dialogs.Confirmations()[1].Respond(true)
	{
		got := p.Snapshot()
		testutil.ErrorIf(t, got.Mode != Browse || got.Expression != "" || got.Limit != toolkit.PageLimit, "confirmed breadcrumb state = %#v", got)
	}
}

func TestInlineTypedValidationErrorsRemainVisibleForEveryInventoryMutation(t *testing.T) {
	fix, _ := inventoryFixture(t)
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	v := NewView(p)
	p.Load()
	p.Select(p.Snapshot().Rows[0].Inventory.ID)
	cases := []struct {
		start func()
		form  Form
	}{
		{p.StartAdjust, Form{Amount: "1.234", Reason: inventorymodels.ReasonUsed}},
		{p.StartSet, Form{Amount: "-1.00"}},
		{p.StartTags, Form{Tags: "=missing-key"}},
	}
	for i, tc := range cases {
		tc.start()
		testutil.ErrorIf(t, p.Submit(tc.form), "case %d submitted", i)
		{
			got := p.Snapshot()
			testutil.ErrorIf(t, got.Err == nil || !apperrors.IsInvalid(got.Err) || !strings.Contains(v.formStatus.Text, "Error:"), "case %d did not render typed invalid error: %#v status=%q", i, got, v.formStatus.Text)
		}
		p.Cancel()
	}
}

func TestInventoryStandardFormLayoutDoesNotOverlapAtWorkspaceSize(t *testing.T) {
	fix, _ := inventoryFixture(t)
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	v := NewView(p)
	p.Load()
	p.Select(p.Snapshot().Rows[0].Inventory.ID)
	assertStandardPageRegionsDoNotOverlap(t, v.root.Objects[0], framework.NewSize(900, 720)) // 1100px window minus navigation rail.
	p.StartAdjust()
	assertStandardPageRegionsDoNotOverlap(t, v.root.Objects[0], framework.NewSize(900, 720))
}

func assertStandardPageRegionsDoNotOverlap(t *testing.T, object framework.CanvasObject, size framework.Size) {
	t.Helper()
	page, ok := object.(*framework.Container)
	testutil.ErrorIf(t, !ok, "page type %T", object)
	page.Resize(size)
	page.Refresh()
	testutil.ErrorIf(t, len(page.Objects) != 3, "standard regions = %d", len(page.Objects))
	regions := append([]framework.CanvasObject(nil), page.Objects...)
	slices.SortFunc(regions, func(a, b framework.CanvasObject) int {
		if a.Position().Y < b.Position().Y {
			return -1
		}
		if a.Position().Y > b.Position().Y {
			return 1
		}
		return 0
	})
	heading, body, footer := regions[0], regions[1], regions[2]
	testutil.ErrorIf(t, heading.Position().Y+heading.Size().Height > body.Position().Y || body.Position().Y+body.Size().Height > footer.Position().Y, "overlapping standard page regions: heading=%v/%v body=%v/%v footer=%v/%v", heading.Position(), heading.Size(), body.Position(), body.Size(), footer.Position(), footer.Size())
}

func TestPresenterSetBlankCostPreservesExistingPrice(t *testing.T) {
	fix, ingredient := inventoryFixture(t)
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	p.Load()
	p.StartSet()
	testutil.ErrorIf(t, !p.Submit(Form{Amount: "9.00"}), "%v", "set rejected")
	stock, err := fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	price, ok := stock.CostPerUnit.Unwrap()
	testutil.ErrorIf(t, !ok, "%v", "cost missing")
	cents, err := price.Cents()
	testutil.Ok(t, err)
	testutil.ErrorIf(t, cents != 325, "cost=%d cents", cents)
}

func TestPresenterAdjustsCostWithoutQuantityMutation(t *testing.T) {
	fix, ingredient := inventoryFixture(t)
	p := NewPresenter(fix.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	p.Load()
	p.StartAdjust()
	testutil.ErrorIf(t, !p.Submit(Form{Cost: "4.10", Reason: inventorymodels.ReasonCorrected}), "%v", "cost-only adjustment rejected")
	stock, err := fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	price, ok := stock.CostPerUnit.Unwrap()
	testutil.ErrorIf(t, !ok, "%v", "cost missing")
	cents, err := price.Cents()
	testutil.Ok(t, err)
	testutil.ErrorIf(t, stock.Amount.Value() != 12.5 || cents != 410, "stock=%#v", stock)
}

func TestValidateAdjustAcceptsCurrencyBearingPrice(t *testing.T) {
	validated, err := validate(Adjust, Form{Cost: "EUR 4.10", Reason: inventorymodels.ReasonCorrected}, measurement.UnitOz, optional.None[money.Price]())
	testutil.Ok(t, err)
	price, ok := validated.cost.Unwrap()
	testutil.ErrorIf(t, !ok || price.Currency != currency.EUR || price.String() != "4.10 €", "currency-bearing cost = %#v", validated.cost)
}

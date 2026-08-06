//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package gui

import (
	"context"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	frameworktest "fyne.io/fyne/v2/test"

	application "github.com/TheFellow/go-modular-monolith/app"
	drinkmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	orders "github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	pkglog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	appgui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestPresenterPagesFiltersResolvesDetailAndSnapshots(t *testing.T) {
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Comma, Collins")
	menu := testutil.CreateMenu(t, f, "Evening, Menu", testutil.WithDrink(drink), testutil.Published())
	first := testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 2, Notes: "neat"}}, Notes: "table seven"})
	testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
	p := newInlinePresenter(f)
	testutil.Equals(t, p.ApplyFilter(Filter{Status: models.OrderStatusPending, Expression: `notes == "table seven"`, Limit: 1}), true)
	state := p.State()
	testutil.Equals(t, len(state.Rows), 1)
	testutil.Equals(t, state.Rows[0].MenuName, "Evening, Menu")
	testutil.Equals(t, state.Rows[0].Lines[0].Name, "Comma, Collins")
	testutil.Equals(t, state.Rows[0].Lines[0].Quantity, 2)
	testutil.Equals(t, state.Rows[0].Lines[0].Total, "N/A")
	testutil.Equals(t, state.Rows[0].Total, "N/A")
	testutil.ErrorIf(t, formatTime(state.Rows[0].Order.CreatedAt) == "", "%v", "resolved detail omitted created time")
	testutil.Equals(t, state.Rows[0].Order.ID, first.ID)
	state.Rows[0].Order.Notes = "mutated"
	state.Rows[0].Lines[0].Name = "mutated"
	testutil.Equals(t, p.State().Rows[0].Order.Notes, "table seven")
	testutil.Equals(t, p.State().Rows[0].Lines[0].Name, "Comma, Collins")
	testutil.Equals(t, p.ApplyFilter(Filter{Expression: `status ==`, Limit: 25}), true)
	testutil.ErrorIsInvalid(t, p.State().Err)
	testutil.Equals(t, len(p.State().Rows), 1) // a failed refresh retains useful content
}

func TestPresenterTraversesForwardAndBackwardPages(t *testing.T) {
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Paged")
	menu := testutil.CreateMenu(t, f, "Paged", testutil.WithDrink(drink), testutil.Published())
	for range 3 {
		testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
	}
	p := newInlinePresenter(f)
	testutil.Equals(t, p.ApplyFilter(Filter{Limit: 1}), true)
	first := p.State()
	testutil.Equals(t, len(first.Rows), 1)
	testutil.ErrorIf(t, first.Next == "", "%v", "first page omitted next cursor")
	firstID := first.Rows[0].Order.ID
	p.NextPage()
	second := p.State()
	testutil.Equals(t, len(second.Rows), 2)
	testutil.ErrorIf(t, second.Rows[0].Order.ID != firstID || second.Rows[1].Order.ID == firstID, "%v", "next page was not appended")
	testutil.Equals(t, len(second.History), 1)
	p.PreviousPage()
	previous := p.State()
	testutil.Equals(t, previous.Rows[0].Order.ID, firstID)
	testutil.Equals(t, len(previous.History), 0)
}

func TestPresenterDetailBackPreservesListStateAndBreadcrumbResetClearsIt(t *testing.T) {
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Navigation")
	menu := testutil.CreateMenu(t, f, "Navigation", testutil.WithDrink(drink), testutil.Published())
	for range 2 {
		testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
	}
	p := newInlinePresenter(f)
	testutil.Equals(t, p.ApplyFilter(Filter{Expression: `status == "pending"`, Limit: 1}), true)
	p.NextPage()
	before := p.State()
	p.Select(0)
	testutil.Equals(t, p.State().Mode, Viewing)
	p.Back()
	after := p.State()
	testutil.Equals(t, after.Mode, Browsing)
	testutil.Equals(t, after.Filter, before.Filter)
	testutil.Equals(t, after.Cursor, before.Cursor)
	testutil.Equals(t, after.History, before.History)
	p.Select(0)
	p.ResetList()
	reset := p.State()
	testutil.Equals(t, reset.Mode, Browsing)
	testutil.Equals(t, reset.Filter.Expression, "")
	testutil.Equals(t, reset.Filter.Limit, appgui.PageLimit)
	testutil.Equals(t, len(reset.History), 0)
	testutil.Equals(t, reset.Selected == nil, true)
}

func TestPresenterExposesOnlyAuthorizedOrderActions(t *testing.T) {
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Authorization")
	menu := testutil.CreateMenu(t, f, "Authorization", testutil.WithDrink(drink), testutil.Published())
	testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})

	owner := newInlinePresenter(f)
	owner.Refresh()
	owner.Select(0)
	state := owner.State()
	testutil.ErrorIf(t, !state.CanPlace || !state.CanComplete || !state.CanCancel || !state.CanTag, "owner actions missing: %#v", state)

	reader := NewPresenter(application.NewSession(f.ActorContext("sommelier"), f.App.App), Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: &fynetest.Dialogs{}})
	reader.Refresh()
	reader.Select(0)
	state = reader.State()
	testutil.ErrorIf(t, state.CanPlace || state.CanComplete || state.CanCancel || state.CanTag, "read-only actor actions disclosed: %#v", state)
	testutil.ErrorIf(t, !state.CanList, "read-only actor could not browse orders: %#v", state)
}

func TestPresenterSurfacesActionProjectionEvaluatorFailure(t *testing.T) {
	f := testutil.NewFixture(t)
	want := errors.New("policy evaluator unavailable")
	projector := orders.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return want }}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}}, projector)
	testutil.ErrorIf(t, !errors.Is(p.State().Err, want), "projection error = %v", p.State().Err)
	testutil.ErrorIf(t, p.State().CanPlace, "failed projection exposed place")
}

func TestPresenterRecoversFromActionProjectionFailure(t *testing.T) {
	f := testutil.NewFixture(t)
	want := errors.New("policy evaluator unavailable")
	failing := true
	projector := orders.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error {
		if failing {
			return want
		}
		return nil
	}}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}}, projector)
	testutil.ErrorIf(t, !errors.Is(p.State().Err, want), "projection error = %v", p.State().Err)
	failing = false
	testutil.Ok(t, p.permissionsFor(nil))
	testutil.ErrorIf(t, p.State().Err != nil, "recovered projection retained error: %v", p.State().Err)
	testutil.ErrorIf(t, !p.State().CanList, "recovered projection did not expose list")
	p.actionErr = want
	businessErr := errors.New("order load failed")
	p.state.Err = businessErr
	testutil.Ok(t, p.permissionsFor(nil))
	testutil.ErrorIf(t, !errors.Is(p.State().Err, businessErr), "projection recovery cleared business error: %v", p.State().Err)
}

func TestDirtyPlaceBackAndResetRequireConfirmationAndRetainInput(t *testing.T) {
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Dirty placement")
	menu := testutil.CreateMenu(t, f, "Dirty placement", testutil.WithDrink(drink), testutil.Published())
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	p.ApplyFilter(Filter{Expression: `status == "pending"`, Limit: 1})
	p.StartPlace()
	p.ChooseMenu(menu.ID)
	testutil.Equals(t, p.AddItem(drink.ID, 2, "keep this note"), true)
	p.SetPlaceNotes("keep this order note")

	p.Back()
	testutil.Equals(t, len(dialogs.Confirmations()), 1)
	testutil.Equals(t, p.State().Confirming, true)
	dialogs.Confirmations()[0].Respond(false)
	state := p.State()
	testutil.Equals(t, state.Mode, Placing)
	testutil.Equals(t, state.Form.Notes, "keep this order note")
	testutil.Equals(t, state.Form.Items[0].Notes, "keep this note")
	testutil.Equals(t, state.Filter.Expression, `status == "pending"`)

	p.ResetList()
	testutil.Equals(t, len(dialogs.Confirmations()), 2)
	dialogs.Confirmations()[1].Respond(true)
	state = p.State()
	testutil.Equals(t, state.Mode, Browsing)
	testutil.Equals(t, state.Dirty, false)
	testutil.Equals(t, state.Filter.Expression, "")
	testutil.Equals(t, state.Filter.Limit, appgui.PageLimit)
}

func TestDirtyTagsBackAndResetRequireConfirmationAndRetainInput(t *testing.T) {
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Dirty tags")
	menu := testutil.CreateMenu(t, f, "Dirty tags", testutil.WithDrink(drink), testutil.Published())
	testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	p.ApplyFilter(Filter{Expression: `status == "pending"`, Limit: 1})
	p.Select(0)
	p.StartTags()
	p.SetTagForm("featured,region=west")

	p.ResetList()
	testutil.Equals(t, len(dialogs.Confirmations()), 1)
	dialogs.Confirmations()[0].Respond(false)
	state := p.State()
	testutil.Equals(t, state.Mode, Tagging)
	testutil.Equals(t, state.Form.Tags, "featured,region=west")
	testutil.Equals(t, state.Filter.Expression, `status == "pending"`)

	p.Back()
	testutil.Equals(t, len(dialogs.Confirmations()), 2)
	dialogs.Confirmations()[1].Respond(true)
	state = p.State()
	testutil.Equals(t, state.Mode, Browsing)
	testutil.Equals(t, state.Filter.Expression, `status == "pending"`)
	testutil.Equals(t, state.Filter.Limit, 1)
}

func TestPlaceCatalogPreservesDirtyFormRejectsStaleAndPlacesOnlyAvailableDrink(t *testing.T) {
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Gin, Fizz")
	menu := testutil.CreateMenu(t, f, "Published", testutil.WithDrink(drink), testutil.Published())
	executor, dispatcher := &fynetest.ManualExecutor{}, &fynetest.ManualDispatcher{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: dispatcher, Dialogs: &fynetest.Dialogs{}})
	p.StartPlace()
	dispatcher.Drain()
	p.state.Form.Notes = "keep me"
	p.SearchMenus("pub")
	dispatcher.Drain()
	testutil.Equals(t, executor.Run(1), true)
	dispatcher.Drain()
	testutil.Equals(t, p.State().Form.Notes, "keep me")
	testutil.Equals(t, len(p.State().Menus), 1)
	testutil.Equals(t, executor.RunNext(), true)
	dispatcher.Drain()
	testutil.Equals(t, len(p.State().Menus), 1) // stale first catalog did not overwrite the second
	p.ChooseMenu(menu.ID)
	testutil.ErrorIf(t, len(p.State().Drinks) == 0, "%v", "published menu drink was not available in catalog")
	testutil.Equals(t, p.AddItem(drink.ID, 2, " first "), true)
	p.SetPlaceNotes("  counter  ")
	testutil.Equals(t, p.SavePlace(), true)
	testutil.Equals(t, p.SavePlace(), false)
	testutil.Equals(t, p.State().Form.Notes, "  counter  ")
	testutil.Equals(t, executor.RunNext(), true)
	dispatcher.Drain()
	page, err := f.Orders.List(f.OwnerContext(), orders.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, len(page.Items), 1)
	created := page.Items[0]
	testutil.Equals(t, created.Notes, "counter")
	testutil.Equals(t, created.Items[0].Quantity, 2)
	testutil.Equals(t, created.Items[0].Notes, "first")
	stock, err := f.Inventory.Get(f.OwnerContext(), drink.Recipe.Ingredients[0].IngredientID)
	testutil.Ok(t, err)
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionPlace), created.EntityUID(), stock.EntityUID())
}

func TestPlaceValidationTagsAndTerminalConfirmationsPersist(t *testing.T) {
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Soda")
	menu := testutil.CreateMenu(t, f, "Lunch", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	p.Refresh()
	p.Select(0)
	p.StartTags()
	testutil.Equals(t, p.SaveTags(" region=west,featured "), true)
	stored, err := f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Tags.Canonical().String(), "featured,region=west")
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionTag), order.EntityUID())
	p.StartTags()
	testutil.Equals(t, p.SaveTags(""), true)
	stored, err = f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(stored.Tags), 0)
	p.ConfirmComplete()
	testutil.Equals(t, len(dialogs.Confirmations()), 1)
	dialogs.Confirmations()[0].Respond(false)
	stored, err = f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Status, models.OrderStatusPending)
	p.ConfirmComplete()
	dialogs.Confirmations()[1].Respond(true)
	stored, err = f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Status, models.OrderStatusCompleted)
	stock, err := f.Inventory.Get(f.OwnerContext(), drink.Recipe.Ingredients[0].IngredientID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Amount.Value(), 99.0)
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionComplete), order.EntityUID(), stock.EntityUID())
	p.ConfirmCancel()
	testutil.Equals(t, len(dialogs.Confirmations()), 2) // completed orders are guarded before dialog

	p.StartPlace()
	p.state.Form.MenuID = menu.ID
	testutil.Equals(t, p.AddItem(drink.ID, 0, ""), false)
	testutil.ErrorIsInvalid(t, p.State().Err)
	draft := testutil.CreateMenu(t, f, "Draft", testutil.WithDrink(drink))
	p.state.Form.MenuID = draft.ID
	testutil.Equals(t, p.AddItem(drink.ID, 1, ""), false) // not in the published/available catalog
}

func TestTaggedTerminalConfirmationsReplaceClearPreserveAndRejectInvalidAtomically(t *testing.T) {
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Tagged transitions")
	menu := testutil.CreateMenu(t, f, "Tagged transitions", testutil.WithDrink(drink), testutil.Published())
	newOrder := func() *models.Order {
		order := testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
		_, err := f.App.Tags.Replace(f.OwnerContext(), order.EntityUID(), tag.Tags{{Key: "old"}})
		testutil.Ok(t, err)
		return order
	}

	for _, tc := range []struct {
		name, action, values, wantTags string
		mode                           appgui.TagMutationMode
		wantStatus                     models.OrderStatus
	}{
		{"complete replace", "complete", "featured,region=west", "featured,region=west", appgui.ReplaceTags, models.OrderStatusCompleted},
		{"cancel clear", "cancel", "ignored", "", appgui.ClearTags, models.OrderStatusCancelled},
		{"cancel preserve", "cancel", "ignored", "old", appgui.PreserveTags, models.OrderStatusCancelled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			order := newOrder()
			dialogs := &fynetest.TaggedDialogs{}
			p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
			p.Refresh()
			for i, row := range p.State().Rows {
				if row.Order.ID == order.ID {
					p.Select(i)
					break
				}
			}
			if tc.action == "complete" {
				p.ConfirmComplete()
			} else {
				p.ConfirmCancel()
			}
			confirmation := dialogs.TaggedConfirmations()[0]
			testutil.Equals(t, confirmation.Current, "old")
			confirmation.Respond(true, tc.mode, tc.values)
			stored, err := f.Orders.Get(f.OwnerContext(), order.ID)
			testutil.Ok(t, err)
			testutil.Equals(t, stored.Status, tc.wantStatus)
			testutil.Equals(t, stored.Tags.Canonical().String(), tc.wantTags)
		})
	}

	order := newOrder()
	dialogs := &fynetest.TaggedDialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	p.Refresh()
	for i, row := range p.State().Rows {
		if row.Order.ID == order.ID {
			p.Select(i)
			break
		}
	}
	p.ConfirmComplete()
	dialogs.TaggedConfirmations()[0].Respond(true, appgui.ReplaceTags, "region=east,region=west")
	stored, err := f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Status, models.OrderStatusPending)
	testutil.Equals(t, stored.Tags.Canonical().String(), "old")
	testutil.ErrorIsInvalid(t, p.State().Err)
}

func TestTagValidationAndAuthorizationFailuresRetainCorrectableForms(t *testing.T) {
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Retained")
	menu := testutil.CreateMenu(t, f, "Retained", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
	p := newInlinePresenter(f)
	p.Refresh()
	p.Select(0)
	p.StartTags()
	testutil.Equals(t, p.SaveTags("region=west,region=east"), false)
	testutil.ErrorIsInvalid(t, p.State().Err)
	testutil.Equals(t, p.State().Form.Tags, "region=west,region=east")

	denied := application.NewSession(f.ActorContext("sommelier"), f.App.App)
	p = NewPresenter(denied, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: &fynetest.Dialogs{}})
	p.Refresh()
	p.Select(0)
	p.StartTags()
	testutil.Equals(t, p.State().Mode, Viewing)
	testutil.Equals(t, p.State().CanTag, false)
	testutil.Equals(t, p.SaveTags("featured"), false)
	stored, err := f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(stored.Tags), 0)

	p.StartPlace()
	testutil.Equals(t, p.State().Mode, Viewing)
	testutil.Equals(t, p.State().CanPlace, false)
	page, err := f.Orders.List(f.OwnerContext(), orders.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, len(page.Items), 1)
}

func TestCancelUsesStableTargetEvenWhenSelectionChanges(t *testing.T) {
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Stable")
	menu := testutil.CreateMenu(t, f, "Stable", testutil.WithDrink(drink), testutil.Published())
	first := testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
	second := testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	p.Refresh()
	var firstIndex int
	for i, row := range p.State().Rows {
		if row.Order.ID == first.ID {
			firstIndex = i
		}
	}
	p.Select(firstIndex)
	p.ConfirmCancel()
	p.state.Selected = findRow(p.state.Rows, second.ID)
	dialogs.Confirmations()[0].Respond(true)
	one, err := f.Orders.Get(f.OwnerContext(), first.ID)
	testutil.Ok(t, err)
	two, err := f.Orders.Get(f.OwnerContext(), second.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, one.Status, models.OrderStatusCancelled)
	testutil.Equals(t, two.Status, models.OrderStatusPending)
	stock, err := f.Inventory.Get(f.OwnerContext(), drink.Recipe.Ingredients[0].IngredientID)
	testutil.Ok(t, err)
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionCancel), first.EntityUID(), stock.EntityUID())
}

func TestHeadlessWidgetsPlaceOrderAndRetainCommaNames(t *testing.T) {
	gui := frameworktest.NewApp()
	defer gui.Quit()
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Widget, Sour")
	menu := testutil.CreateMenu(t, f, "Widget, Menu", testutil.WithDrink(drink), testutil.Published())
	p := newInlinePresenter(f)
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	p.Refresh()
	driver.Tap("orders-place")
	driver.Type("orders-place-menu-search", "widget")
	driver.Tap("orders-place-menu-search-apply")
	v.menus.SetSelected("Widget, Menu  [" + menu.ID.String() + "]")
	v.drinks.SetSelected(v.drinks.Options[0])
	driver.Type("orders-place-quantity", "3")
	driver.Type("orders-place-item-notes", "with, twist\nserve very cold")
	driver.Tap("orders-place-add-item")
	driver.Type("orders-place-notes", "headless")
	driver.Tap("orders-place-save")
	page, err := f.Orders.List(f.OwnerContext(), orders.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, len(page.Items), 1)
	testutil.Equals(t, page.Items[0].Items[0].Quantity, 3)
	testutil.Equals(t, page.Items[0].Items[0].Notes, "with, twist\nserve very cold")
}

func TestOrdersListRetainsTableInstanceAcrossRenders(t *testing.T) {
	gui := frameworktest.NewApp()
	defer gui.Quit()
	f := testutil.NewFixture(t)
	p := newInlinePresenter(f)
	v := NewView(p)
	v.browser(p.State())
	first := v.list
	v.browser(p.State())
	testutil.ErrorIf(t, v.list != first, "%v", "orders refresh recreated the table and discarded resized column widths")
}

func TestHeadlessWidgetsTagAndCompleteSelectedOrder(t *testing.T) {
	gui := frameworktest.NewApp()
	defer gui.Quit()
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Widget lifecycle")
	menu := testutil.CreateMenu(t, f, "Widget lifecycle", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	p.Refresh()
	p.Select(0)
	driver.Tap("orders-tags")
	driver.Type("orders-tags-value", "featured")
	driver.Submit("orders-tags-value")
	driver.Tap("orders-tags-save")
	stored, err := f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Tags.Canonical().String(), "featured")
	driver.Tap("orders-complete")
	testutil.Equals(t, len(dialogs.Confirmations()), 1)
	dialogs.Confirmations()[0].Respond(true)
	stored, err = f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Status, models.OrderStatusCompleted)
}

func TestCatalogRefreshDisablesEveryPlacementControl(t *testing.T) {
	gui := frameworktest.NewApp()
	defer gui.Quit()
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Busy")
	menu := testutil.CreateMenu(t, f, "Busy", testutil.WithDrink(drink), testutil.Published())
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appgui.InlineDispatcher{}, Dialogs: &fynetest.Dialogs{}})
	v := NewView(p)
	p.StartPlace()
	testutil.Equals(t, executor.RunNext(), true)
	p.ChooseMenu(menu.ID)
	testutil.Equals(t, p.AddItem(drink.ID, 1, "hold"), true)
	p.SearchMenus("busy")

	for name, disabled := range map[string]bool{
		"menu search": v.menuQuery.Disabled(), "drink search": v.drinkQuery.Disabled(),
		"quantity": v.quantity.Disabled(), "item notes": v.itemNotes.Disabled(),
		"order notes": v.orderNotes.Disabled(), "menus": v.menus.Disabled(), "drinks": v.drinks.Disabled(),
	} {
		testutil.ErrorIf(t, !disabled, "%s remained enabled during catalog refresh", name)
	}
	{
		remove := v.removeItems[0]
		testutil.ErrorIf(t, remove == nil || !remove.Disabled(), "%v", "remove item remained enabled during catalog refresh")
	}
}

//nolint:paralleltest // builds and executes a process-like CLI lifecycle against one database.
func TestCLIAndFyneShareOrderPersistenceContract(t *testing.T) {
	repo, err := filepath.Abs("../../../../../")
	testutil.Ok(t, err)
	dir := t.TempDir()
	binary := testutil.ExecutablePath(dir, "mixology")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./main/cli")
	build.Dir = repo
	output, err := build.CombinedOutput()
	testutil.ErrorIf(t, err != nil, "build CLI: %v\n%s", err, output)
	run := func(args ...string) string {
		cmd := exec.CommandContext(t.Context(), binary, args...)
		cmd.Dir = dir
		output, runErr := cmd.CombinedOutput()
		testutil.ErrorIf(t, runErr != nil, "CLI %v: %v\n%s", args, runErr, output)
		return string(output)
	}

	ctx := authn.ToContext(context.Background(), authn.Owner())
	ctx = pkglog.ToContext(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)))
	dbPath := filepath.Join(dir, "data", "mixology.db")
	database, err := store.Open(ctx, dbPath)
	testutil.Ok(t, err)
	core := application.New(ctx, application.Config{Store: database})
	mctx := middleware.NewContext(ctx)
	ingredient, err := core.Ingredients.Create(mctx, &ingredientmodels.Ingredient{Name: "Shared base", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.Ok(t, err)
	_, err = core.Inventory.Set(mctx, &inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	testutil.Ok(t, err)
	drink, err := core.Drinks.Create(mctx, &drinkmodels.Drink{Name: "Shared drink", Category: drinkmodels.DrinkCategoryMocktail, Glass: drinkmodels.GlassTypeHighball, Recipe: drinkmodels.Recipe{Ingredients: []drinkmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"Build"}}})
	testutil.Ok(t, err)
	menu, err := core.Menus.Create(mctx, &menumodels.Menu{Name: "Shared menu"})
	testutil.Ok(t, err)
	menu, err = core.Menus.AddDrink(mctx, &menumodels.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	menu, err = core.Menus.Publish(mctx, &menumodels.Menu{ID: menu.ID})
	testutil.Ok(t, err)
	testutil.Ok(t, core.Close())

	orderID := strings.TrimSpace(run("--log-level", "error", "orders", "place", "--menu-id", menu.ID.String(), drink.ID.String()+":2"))
	database, err = store.Open(ctx, dbPath)
	testutil.Ok(t, err)
	core = application.New(ctx, application.Config{Store: database})
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(application.NewSession(ctx, core), Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	p.Refresh()
	testutil.Equals(t, len(p.State().Rows), 1)
	testutil.Equals(t, p.State().Rows[0].Order.ID.String(), orderID)
	p.Select(0)
	p.ConfirmComplete()
	dialogs.Confirmations()[0].Respond(true)
	testutil.Ok(t, core.Close())

	got := run("--log-level", "error", "orders", "get", "--id", orderID, "--json")
	testutil.StringContains(t, got, `"status": "completed"`)
}

func newInlinePresenter(f *testutil.Fixture) *Presenter {
	return NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: &fynetest.Dialogs{}})
}

func availableDrink(t *testing.T, f *testutil.Fixture, name string) *drinkmodels.Drink {
	t.Helper()
	ingredient := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: name + " base", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(100, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	return testutil.CreateDrink(t, f, drinkmodels.Drink{Name: name, Category: drinkmodels.DrinkCategoryMocktail, Glass: drinkmodels.GlassTypeHighball, Recipe: drinkmodels.Recipe{Ingredients: []drinkmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"Build"}}})
}

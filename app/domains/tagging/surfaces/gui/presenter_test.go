//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package gui

import (
	"testing"

	application "github.com/TheFellow/go-modular-monolith/app"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestSummarySortingCoversCompleteInMemoryCatalogAndTogglesDirection(t *testing.T) {
	p := &Presenter{state: State{Mode: Results, Operation: Summary, VisibleSummaries: []tagging.Summary{{Tag: "z", Total: 1}, {Tag: "a", Total: 3}, {Tag: "m", Total: 2}}}}
	p.SortSummaries(1, ui.SortAscending)
	{
		got := p.state.VisibleSummaries
		testutil.ErrorIf(t, got[0].Total != 1 || got[2].Total != 3, "ascending totals = %#v", got)
	}
	p.SortSummaries(1, ui.SortDescending)
	{
		got := p.state.VisibleSummaries
		testutil.ErrorIf(t, got[0].Total != 3 || got[2].Total != 1, "descending totals = %#v", got)
	}
}

type tagFixtures struct {
	f       *testutil.Fixture
	targets map[cedar.EntityType]cedar.EntityUID
}

func fullFixture(t *testing.T) tagFixtures {
	t.Helper()
	f := testutil.NewFixture(t)
	ing := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Tagging Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{Name: "Tagging Highball", Category: drinksmodels.DrinkCategoryHighball, Glass: drinksmodels.GlassTypeHighball, Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ing.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"build"}}})
	stock := testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: ing.ID, Amount: measurement.MustAmount(10, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	menu := testutil.CreateMenu(t, f, "Tagging Menu", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{MenuID: menu.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
	return tagFixtures{f, map[cedar.EntityType]cedar.EntityUID{entity.TypeDrink: drink.EntityUID(), entity.TypeIngredient: ing.EntityUID(), entity.TypeInventory: stock.EntityUID(), entity.TypeMenu: menu.EntityUID(), entity.TypeOrder: order.EntityUID()}}
}
func presenter(session *application.Session) *Presenter {
	return NewPresenter(session, Dependencies{Executor: ui.InlineExecutor{}, Dispatcher: ui.InlineDispatcher{}})
}

func TestViewActivationPreservesTagWorkflowState(t *testing.T) {
	p := presenter(nil)
	v := NewView(p)
	p.Start(ShowKey)
	p.SetValue("seasonal")

	v.Activate()
	state := p.State()
	testutil.ErrorIf(t, state.Mode != EnteringValue || state.Operation != ShowKey || state.Value != "seasonal", "activation changed in-progress tag workflow: %#v", state)
}

func TestEntityPickerProvidesNamedSearchableActiveTargetsForEveryType(t *testing.T) {
	fx := fullFixture(t)
	p := presenter(fx.f.App)
	for _, kind := range []cedar.EntityType{entity.TypeDrink, entity.TypeIngredient, entity.TypeInventory, entity.TypeMenu, entity.TypeOrder} {
		p.Start(Add)
		p.SelectType(kind)
		s := p.State()
		testutil.ErrorIf(t, s.Mode != PickingEntity || len(s.Entities) != 1 || s.Entities[0].Name == "" || s.Entities[0].Name == string(s.Entities[0].UID.ID), "%s catalog = %#v", kind, s)
		p.Search("no such active entity")
		testutil.ErrorIf(t, len(p.State().Visible) != 0, "%s search did not filter", kind)
	}
}

func TestPresenterMutatesEveryOperationalEntityType(t *testing.T) {
	fx := fullFixture(t)
	p := presenter(fx.f.App)
	for _, kind := range []cedar.EntityType{entity.TypeDrink, entity.TypeIngredient, entity.TypeInventory, entity.TypeMenu, entity.TypeOrder} {
		p.Start(Add)
		p.SelectType(kind)
		p.SelectEntity(0)
		p.SetValue("surface=fyne")
		testutil.ErrorIf(t, !p.Submit() || !p.State().Result.Changed, "%s mutation = %#v", kind, p.State())
		values, err := fx.f.App.Tags.List(fx.f.OwnerContext(), fx.targets[kind])
		testutil.Ok(t, err)
		testutil.ErrorIf(t, values.Canonical().String() != "surface=fyne", "%s tags = %q", kind, values.Canonical().String())
	}
}

func TestRealWidgetsExerciseMutationInspectionDiscoveryAndSummary(t *testing.T) {
	fx := fullFixture(t)
	p := presenter(fx.f.App)
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	driver.Tap(ControlAdd)
	driver.Tap(typeControl(entity.TypeIngredient))
	driver.Tap(entityControl(0))
	driver.Type(ControlValue, "region=west")
	driver.Tap(ControlSubmit)
	{
		s := p.State()
		testutil.ErrorIf(t, s.Mode != Results || !s.Result.Changed || s.Result.Tags.Canonical().String() != "region=west", "add state = %#v", s)
	}
	p.Back()
	driver.Tap(ControlInspect)
	driver.Tap(typeControl(entity.TypeIngredient))
	driver.Tap(entityControl(0))
	{
		got := p.State().Result.Tags.Canonical().String()
		testutil.ErrorIf(t, got != "region=west", "inspect = %q", got)
	}
	p.Back()
	driver.Tap(ControlShowExact)
	driver.Type(ControlValue, "region=west")
	driver.Tap(ControlSubmit)
	testutil.ErrorIf(t, len(p.State().Result.References) != 1 || p.State().Result.References[0].EntityName != "Tagging Gin", "references = %#v", p.State().Result.References)
	p.Back()
	driver.Tap(ControlShowKey)
	driver.Type(ControlValue, "region")
	driver.Tap(ControlSubmit)
	testutil.ErrorIf(t, len(p.State().Result.References) != 1, "key references = %#v", p.State().Result.References)
	p.Back()
	driver.Tap(ControlSummary)
	testutil.ErrorIf(t, len(p.State().Result.Summaries) != 1 || p.State().Result.Summaries[0].Ingredients != 1, "summary = %#v", p.State().Result.Summaries)
	p.Back()
	driver.Tap(ControlRemove)
	driver.Tap(typeControl(entity.TypeIngredient))
	driver.Tap(entityControl(0))
	driver.Type(ControlValue, "region")
	driver.Tap(ControlSubmit)
	testutil.ErrorIf(t, !p.State().Result.Changed || len(p.State().Result.Tags) != 0, "remove = %#v", p.State())
}

func TestCanonicalValidationUnchangedAndDefensiveSnapshots(t *testing.T) {
	fx := fullFixture(t)
	target := fx.targets[entity.TypeIngredient]
	_, err := fx.f.App.Tags.Upsert(fx.f.OwnerContext(), target, tag.Tag{Key: "a", Value: "1"})
	testutil.Ok(t, err)
	p := presenter(fx.f.App)
	p.Start(Add)
	p.SelectType(entity.TypeIngredient)
	p.SelectEntity(0)
	p.SetValue("a=1")
	testutil.ErrorIf(t, !p.Submit() || p.State().Result.Changed, "expected unchanged result: %#v", p.State())
	p.Back()
	p.Start(ShowExact)
	p.SetValue("")
	testutil.ErrorIf(t, p.Submit() || p.State().Err == nil, "invalid tag accepted: %#v", p.State())
	s := p.State()
	s.Entities = append(s.Entities, EntityOption{Name: "poison"})
	testutil.ErrorIf(t, len(p.State().Entities) == len(s.Entities), "%v", "state snapshot aliases entity catalog")
	p.Back()
	p.Start(Inspect)
	p.SelectType(entity.TypeIngredient)
	p.SelectEntity(0)
	s = p.State()
	s.Result.Tags[0].Key = "poison"
	testutil.ErrorIf(t, p.State().Result.Tags[0].Key == "poison", "%v", "state snapshot aliases result tags")
}

func TestViewResetsSearchWithNewEntityCatalogAndIdentifiesResultTarget(t *testing.T) {
	fx := fullFixture(t)
	p := presenter(fx.f.App)
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	driver.Tap(ControlInspect)
	driver.Tap(typeControl(entity.TypeIngredient))
	driver.Type(ControlSearch, "no match")
	driver.Tap(ControlSearch + ".apply")
	p.Back()
	driver.Tap(typeControl(entity.TypeIngredient))
	testutil.ErrorIf(t, v.search.Text != "" || len(p.State().Visible) != 1, "new catalog retained stale search: text=%q state=%#v", v.search.Text, p.State())
	driver.Tap(entityControl(0))
	testutil.ErrorIf(t, p.State().Result.Target != fx.targets[entity.TypeIngredient] || p.State().Result.TargetName == "", "result lost target identity: %#v", p.State().Result)
}

func TestDeniedTargetMethodsDoNotDispatchOrPersist(t *testing.T) {
	fx := fullFixture(t)
	denied := application.NewSession(fx.f.ActorContext("bartender"), fx.f.App.App)
	p := presenter(denied)
	p.Start(Add)
	p.SelectType(entity.TypeIngredient)
	p.SelectEntity(0)
	testutil.ErrorIf(t, p.State().Mode != PickingEntity || !p.State().Target.IsZero(), "denied selection advanced workflow: %#v", p.State())

	// Even a stale or synthetic caller that reaches the editor cannot dispatch
	// while the projected target capability remains denied.
	p.state.Mode = EnteringValue
	p.state.Target = fx.targets[entity.TypeIngredient]
	p.state.Actions[tagging.ControlTag] = p.state.Visible[0].Actions[tagging.ControlTag]
	p.SetValue("denied=yes")
	testutil.ErrorIf(t, p.Submit(), "denied submit dispatched: %#v", p.State())
	values, err := fx.f.App.Tags.List(fx.f.OwnerContext(), fx.targets[entity.TypeIngredient])
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(values) != 0, "denied mutation persisted: %#v", values)
}

func TestDeniedDiscoveryMethodsDoNotDispatch(t *testing.T) {
	fx := fullFixture(t)
	denied := application.NewSession(fx.f.ActorContext("bartender"), fx.f.App.App)
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(denied, Dependencies{Executor: executor, Dispatcher: ui.InlineDispatcher{}})

	p.ResetList()
	p.Start(Summary)
	testutil.ErrorIf(t, executor.Pending() != 0 || p.State().Mode != Browsing, "denied summary dispatched: pending=%d state=%#v", executor.Pending(), p.State())

	p.state.Mode, p.state.Operation, p.state.Value = EnteringValue, ShowExact, "scope=private"
	testutil.ErrorIf(t, p.Submit() || executor.Pending() != 0, "denied show dispatched: pending=%d state=%#v", executor.Pending(), p.State())
}

func TestBackInvalidatesQueuedLoadAndSubmissionLocksNavigation(t *testing.T) {
	fx := fullFixture(t)
	executor := &fynetest.ManualExecutor{}
	dispatcher := &fynetest.ManualDispatcher{}
	p := NewPresenter(fx.f.App, Dependencies{Executor: executor, Dispatcher: dispatcher})
	p.Start(Summary)
	dispatcher.Drain()
	testutil.ErrorIf(t, !p.Back(), "%v", "back rejected while loading")
	executor.RunNext()
	dispatcher.Drain()
	testutil.ErrorIf(t, p.State().Mode != Browsing, "stale result published: %#v", p.State())
	p = NewPresenter(fx.f.App, Dependencies{Executor: executor, Dispatcher: ui.InlineDispatcher{}})
	p.Start(Add)
	p.SelectType(entity.TypeIngredient)
	executor.RunNext()
	p.SelectEntity(0)
	p.SetValue("locked=yes")
	testutil.ErrorIf(t, !p.Submit() || p.Back(), "%v", "accepted submission did not lock navigation")
	executor.RunNext()
	testutil.ErrorIf(t, p.State().Mode != Results, "submission = %#v", p.State())
}

func TestMutationAuditsTouchedEntity(t *testing.T) {
	fx := fullFixture(t)
	p := presenter(fx.f.App)
	p.Start(Add)
	p.SelectType(entity.TypeIngredient)
	p.SelectEntity(0)
	p.SetValue("audit=yes")
	p.Submit()
	entry := fx.f.LatestAuditEntry(ingredientauthz.ActionTag)
	testutil.AuditTouches(t, entry, fx.targets[entity.TypeIngredient])
}

func TestSummaryIsFilterableListAndBackRetainsDiscoveryState(t *testing.T) {
	fx := fullFixture(t)
	_, err := fx.f.App.Tags.Upsert(fx.f.OwnerContext(), fx.targets[entity.TypeIngredient], tag.Tag{Key: "region", Value: "west"})
	testutil.Ok(t, err)
	p := presenter(fx.f.App)
	p.ResetList()
	{
		s := p.State()
		testutil.ErrorIf(t, s.Mode != Results || s.Operation != Summary || len(s.VisibleSummaries) != 1, "summary list = %#v", s)
	}
	p.Search("region")
	p.SelectSummary(0)
	{
		s := p.State()
		testutil.ErrorIf(t, s.Operation != ShowExact || s.Value != "region=west" || len(s.Result.References) != 1, "tag detail = %#v", s)
	}
	p.Back()
	{
		s := p.State()
		testutil.ErrorIf(t, s.Operation != Summary || s.Query != "region" || len(s.VisibleSummaries) != 1, "back did not retain list state: %#v", s)
	}
	p.SelectSummary(0)
	p.ResetList()
	{
		s := p.State()
		testutil.ErrorIf(t, s.Operation != Summary || s.Query != "" || len(s.VisibleSummaries) != 1, "breadcrumb reset = %#v", s)
	}
}

func TestSummaryEmptyStateAndTypedDetailNavigationControls(t *testing.T) {
	fx := fullFixture(t)
	p := presenter(fx.f.App)
	v := NewView(p)
	v.Activate()
	testutil.ErrorIf(t, !v.list.Hidden || v.empty.Hidden, "empty summary visibility: list=%v empty=%v", v.list.Hidden, v.empty.Hidden)
	_, err := fx.f.App.Tags.Upsert(fx.f.OwnerContext(), fx.targets[entity.TypeDrink], tag.Tag{Key: "classic"})
	testutil.Ok(t, err)
	p.ResetList()
	testutil.ErrorIf(t, v.list.Hidden || !v.empty.Hidden, "populated summary visibility: list=%v empty=%v", v.list.Hidden, v.empty.Hidden)
	p.SelectSummary(0)
	driver := fynetest.NewDriver(t, v.Content())
	driver.Tap(ControlBack)
	testutil.ErrorIf(t, p.State().Operation != Summary, "typed Back did not return to summary: %#v", p.State())
}

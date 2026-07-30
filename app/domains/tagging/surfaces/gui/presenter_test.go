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
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	ui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	cedar "github.com/cedar-policy/cedar-go"
)

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
	if state.Mode != EnteringValue || state.Operation != ShowKey || state.Value != "seasonal" {
		t.Fatalf("activation changed in-progress tag workflow: %#v", state)
	}
}

func TestEntityPickerProvidesNamedSearchableActiveTargetsForEveryType(t *testing.T) {
	fx := fullFixture(t)
	p := presenter(fx.f.App)
	for _, kind := range []cedar.EntityType{entity.TypeDrink, entity.TypeIngredient, entity.TypeInventory, entity.TypeMenu, entity.TypeOrder} {
		p.Start(Add)
		p.SelectType(kind)
		s := p.State()
		if s.Mode != PickingEntity || len(s.Entities) != 1 || s.Entities[0].Name == "" || s.Entities[0].Name == string(s.Entities[0].UID.ID) {
			t.Fatalf("%s catalog = %#v", kind, s)
		}
		p.Search("no such active entity")
		if len(p.State().Visible) != 0 {
			t.Fatalf("%s search did not filter", kind)
		}
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
		if !p.Submit() || !p.State().Result.Changed {
			t.Fatalf("%s mutation = %#v", kind, p.State())
		}
		values, err := fx.f.App.Tags.List(fx.f.OwnerContext(), fx.targets[kind])
		testutil.Ok(t, err)
		if values.Canonical().String() != "surface=fyne" {
			t.Fatalf("%s tags = %q", kind, values.Canonical().String())
		}
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
	if s := p.State(); s.Mode != Results || !s.Result.Changed || s.Result.Tags.Canonical().String() != "region=west" {
		t.Fatalf("add state = %#v", s)
	}
	p.Back()
	driver.Tap(ControlInspect)
	driver.Tap(typeControl(entity.TypeIngredient))
	driver.Tap(entityControl(0))
	if got := p.State().Result.Tags.Canonical().String(); got != "region=west" {
		t.Fatalf("inspect = %q", got)
	}
	p.Back()
	driver.Tap(ControlShowExact)
	driver.Type(ControlValue, "region=west")
	driver.Tap(ControlSubmit)
	if len(p.State().Result.References) != 1 {
		t.Fatalf("references = %#v", p.State().Result.References)
	}
	p.Back()
	driver.Tap(ControlShowKey)
	driver.Type(ControlValue, "region")
	driver.Tap(ControlSubmit)
	if len(p.State().Result.References) != 1 {
		t.Fatalf("key references = %#v", p.State().Result.References)
	}
	p.Back()
	driver.Tap(ControlSummary)
	if len(p.State().Result.Summaries) != 1 || p.State().Result.Summaries[0].Ingredients != 1 {
		t.Fatalf("summary = %#v", p.State().Result.Summaries)
	}
	p.Back()
	driver.Tap(ControlRemove)
	driver.Tap(typeControl(entity.TypeIngredient))
	driver.Tap(entityControl(0))
	driver.Type(ControlValue, "region")
	driver.Tap(ControlSubmit)
	if !p.State().Result.Changed || len(p.State().Result.Tags) != 0 {
		t.Fatalf("remove = %#v", p.State())
	}
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
	if !p.Submit() || p.State().Result.Changed {
		t.Fatalf("expected unchanged result: %#v", p.State())
	}
	p.Back()
	p.Start(ShowExact)
	p.SetValue("")
	if p.Submit() || p.State().Err == nil {
		t.Fatalf("invalid tag accepted: %#v", p.State())
	}
	s := p.State()
	s.Entities = append(s.Entities, EntityOption{Name: "poison"})
	if len(p.State().Entities) == len(s.Entities) {
		t.Fatal("state snapshot aliases entity catalog")
	}
	p.Back()
	p.Start(Inspect)
	p.SelectType(entity.TypeIngredient)
	p.SelectEntity(0)
	s = p.State()
	s.Result.Tags[0].Key = "poison"
	if p.State().Result.Tags[0].Key == "poison" {
		t.Fatal("state snapshot aliases result tags")
	}
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
	if v.search.Text != "" || len(p.State().Visible) != 1 {
		t.Fatalf("new catalog retained stale search: text=%q state=%#v", v.search.Text, p.State())
	}
	driver.Tap(entityControl(0))
	if p.State().Result.Target != fx.targets[entity.TypeIngredient] || p.State().Result.TargetName == "" {
		t.Fatalf("result lost target identity: %#v", p.State().Result)
	}
}

func TestDeniedMutationRetainsEditorAndDoesNotPersist(t *testing.T) {
	fx := fullFixture(t)
	denied := application.NewSession(fx.f.ActorContext("bartender"), fx.f.App.App)
	p := presenter(denied)
	p.Start(Add)
	p.SelectType(entity.TypeIngredient)
	p.SelectEntity(0)
	p.SetValue("denied=yes")
	p.Submit()
	s := p.State()
	if s.Mode != EnteringValue || s.Err == nil {
		t.Fatalf("denied state = %#v", s)
	}
	testutil.ErrorIsPermission(t, s.Err)
	values, err := fx.f.App.Tags.List(fx.f.OwnerContext(), fx.targets[entity.TypeIngredient])
	testutil.Ok(t, err)
	if len(values) != 0 {
		t.Fatalf("denied mutation persisted: %#v", values)
	}
}

func TestBackInvalidatesQueuedLoadAndSubmissionLocksNavigation(t *testing.T) {
	fx := fullFixture(t)
	executor := &fynetest.ManualExecutor{}
	dispatcher := &fynetest.ManualDispatcher{}
	p := NewPresenter(fx.f.App, Dependencies{Executor: executor, Dispatcher: dispatcher})
	p.Start(Summary)
	dispatcher.Drain()
	if !p.Back() {
		t.Fatal("back rejected while loading")
	}
	executor.RunNext()
	dispatcher.Drain()
	if p.State().Mode != Browsing {
		t.Fatalf("stale result published: %#v", p.State())
	}
	p = NewPresenter(fx.f.App, Dependencies{Executor: executor, Dispatcher: ui.InlineDispatcher{}})
	p.Start(Add)
	p.SelectType(entity.TypeIngredient)
	executor.RunNext()
	p.SelectEntity(0)
	p.SetValue("locked=yes")
	if !p.Submit() || p.Back() {
		t.Fatal("accepted submission did not lock navigation")
	}
	executor.RunNext()
	if p.State().Mode != Results {
		t.Fatalf("submission = %#v", p.State())
	}
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

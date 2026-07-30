//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package gui

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	frameworktest "fyne.io/fyne/v2/test"

	appcore "github.com/TheFellow/go-modular-monolith/app"
	drinkmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menus "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	pkglog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	appgui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestDetailTextIncludesTimestampsItemIdentityAndSortedOrder(t *testing.T) {
	firstID, secondID := entity.NewDrinkID(), entity.NewDrinkID()
	menu := &models.Menu{
		Name: "Research", CreatedAt: time.Date(2025, 3, 1, 2, 3, 4, 0, time.UTC),
		PublishedAt: optional.Some(time.Date(2025, 3, 2, 3, 4, 5, 0, time.UTC)),
		Items:       []models.MenuItem{{DrinkID: firstID, SortOrder: 20}, {DrinkID: secondID, SortOrder: 10}},
	}
	names := map[entity.DrinkID]string{firstID: "First", secondID: "Second"}
	text := detailText(menu, func(id entity.DrinkID) string { return names[id] })
	for _, exact := range []string{"Created: 2025-03-01T02:03:04Z", "Published: 2025-03-02T03:04:05Z", "Drink ID: " + firstID.String(), "Sort order: 20"} {
		testutil.ErrorIf(t, !strings.Contains(text, exact), "expected %q in detail:\n%s", exact, text)
	}
	testutil.ErrorIf(t, strings.Index(text, "Second") > strings.Index(text, "First"), "items not sorted:\n%s", text)

	menu.PublishedAt = optional.None[time.Time]()
	testutil.ErrorIf(t, strings.Contains(detailText(menu, func(id entity.DrinkID) string { return names[id] }), "Published:"), "absent PublishedAt rendered")
}

func TestPresenterRefreshComposesPagingFiltersAndRejectsStaleResult(t *testing.T) {
	f := testutil.NewFixture(t)
	first, err := f.Menus.Create(f.OwnerContext(), &models.Menu{Name: "First"})
	testutil.Ok(t, err)
	_, err = f.Menus.Create(f.OwnerContext(), &models.Menu{Name: "Second"})
	testutil.Ok(t, err)
	executor, dispatcher := &fynetest.ManualExecutor{}, &fynetest.ManualDispatcher{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: dispatcher})
	p.Refresh()
	dispatcher.Drain()
	p.SetFilter(Filter{Status: models.MenuStatusDraft, Expression: `name == "Second"`})
	p.Refresh()
	dispatcher.Drain()
	testutil.Equals(t, executor.Run(1), true)
	dispatcher.Drain()
	testutil.Equals(t, len(p.State().Items), 1)
	testutil.Equals(t, p.State().Items[0].Name, "Second")
	testutil.Equals(t, executor.RunNext(), true)
	dispatcher.Drain()
	testutil.Equals(t, len(p.State().Items), 1)
	p.SetFilter(Filter{Expression: `name ==`})
	p.Refresh()
	dispatcher.Drain()
	testutil.Equals(t, executor.RunNext(), true)
	dispatcher.Drain()
	testutil.ErrorIsInvalid(t, p.State().Err)
	testutil.Equals(t, p.State().Items[0].Name, "Second")
	_ = first
}

func TestWidgetCRUDTagsCurationTransitionsAndDelete(t *testing.T) {
	gui := frameworktest.NewApp()
	defer gui.Quit()
	f := testutil.NewFixture(t)
	drink := menuDrink(t, f, "Comma, Collins")
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	p.Refresh()
	driver.Tap(ControlCreate)
	testutil.Equals(t, v.descriptionHelp.Hidden, true)
	driver.Type(ControlName, "  Dinner  ")
	driver.Type(ControlDescription, "Evening")
	driver.Tap(ControlSave)
	page, err := f.Menus.List(f.OwnerContext(), menus.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, len(page.Items), 1)
	menu := page.Items[0]
	testutil.Equals(t, menu.Name, "Dinner")
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionCreate), menu.EntityUID())
	driver.Tap(ControlRename)
	testutil.Equals(t, v.descriptionHelp.Hidden, false)
	driver.Type(ControlName, "Late Dinner")
	driver.Type(ControlDescription, "")
	driver.Tap(ControlSave)
	menu, err = f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, menu.Name, "Late Dinner")
	testutil.Equals(t, menu.Description, "Evening")
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionUpdate), menu.EntityUID())
	driver.Tap(ControlTags)
	testutil.Equals(t, v.descriptionHelp.Hidden, true)
	driver.Type(ControlTagValues, " region=west,featured ")
	driver.Tap(ControlSave)
	menu, err = f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, menu.Tags.Canonical().String(), "featured,region=west")
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionTag), menu.EntityUID())
	driver.Tap(ControlTags)
	driver.Type(ControlTagValues, "")
	driver.Tap(ControlSave)
	menu, err = f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(menu.Tags), 0)
	driver.Tap(ControlAddDrink)
	driver.Type(ControlDrinkSearch, "comma")
	driver.Tap(ControlDrinkSearch + ".apply")
	driver.Tap("menus.drink.choice." + drink.ID.String())
	menu, err = f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(menu.Items), 1)
	testutil.Equals(t, menu.Items[0].Availability, models.AvailabilityUnavailable)
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionAddDrink), menu.EntityUID())
	detail := detailText(menu, p.DrinkName)
	for _, want := range []string{"Comma, Collins", "Drink ID: " + drink.ID.String(), "unavailable", "N/A"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detail did not preserve %q: %s", want, detail)
		}
	}
	driver.Tap(ControlPublish)
	testutil.Equals(t, len(dialogs.Confirmations()), 1)
	dialogs.Confirmations()[0].Respond(true)
	menu, err = f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, menu.Status, models.MenuStatusPublished)
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionPublish), menu.EntityUID())
	driver.Tap(ControlDraft)
	dialogs.Confirmations()[1].Respond(true)
	menu, err = f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, menu.Status, models.MenuStatusDraft)
	testutil.NotNil(t, p.State().Selected)
	testutil.Equals(t, p.State().Selected.Status, models.MenuStatusDraft)
	testutil.Equals(t, len(p.State().Selected.Items), 1)
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionDraft), menu.EntityUID())
	driver.Tap("menus.drink.remove." + drink.ID.String())
	dialogs.Confirmations()[2].Respond(true)
	menu, err = f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(menu.Items), 0)
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionRemoveDrink), menu.EntityUID())
	driver.Tap(ControlDelete)
	dialogs.Confirmations()[3].Respond(true)
	_, err = f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.ErrorIsNotFound(t, err)
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionDelete), menu.EntityUID())
}

func TestTaggedMenuTransitionsReplaceClearPreserveAndRejectInvalidAtomically(t *testing.T) {
	f := testutil.NewFixture(t)
	drink := menuDrink(t, f, "Tagged lifecycle")
	menu := testutil.CreateMenu(t, f, "Tagged lifecycle", testutil.WithDrink(drink))
	_, err := f.App.Tags.Replace(f.OwnerContext(), menu.EntityUID(), tag.Tags{{Key: "old"}})
	testutil.Ok(t, err)
	dialogs := &fynetest.TaggedDialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	p.Refresh()
	p.Publish()
	dialogs.TaggedConfirmations()[0].Respond(true, appgui.ReplaceTags, "featured,region=west")
	stored, err := f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Status, models.MenuStatusPublished)
	testutil.Equals(t, stored.Tags.Canonical().String(), "featured,region=west")

	p.ReturnToDraft()
	dialogs.TaggedConfirmations()[1].Respond(true, appgui.ClearTags, "ignored")
	stored, err = f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Status, models.MenuStatusDraft)
	testutil.Equals(t, stored.Tags.Canonical().String(), "")

	_, err = f.App.Tags.Replace(f.OwnerContext(), menu.EntityUID(), tag.Tags{{Key: "keep"}})
	testutil.Ok(t, err)
	p.Refresh()
	p.RemoveDrink(drink.ID)
	dialogs.TaggedConfirmations()[2].Respond(true, appgui.PreserveTags, "ignored")
	stored, err = f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(stored.Items), 0)
	testutil.Equals(t, stored.Tags.Canonical().String(), "keep")

	// Invalid tags are parsed before publish, so neither state nor tags move.
	menu = testutil.CreateMenu(t, f, "Invalid lifecycle", testutil.WithDrink(drink))
	_, err = f.App.Tags.Replace(f.OwnerContext(), menu.EntityUID(), tag.Tags{{Key: "stable"}})
	testutil.Ok(t, err)
	p.Refresh()
	for i, candidate := range p.State().Items {
		if candidate.ID == menu.ID {
			p.Select(i)
			break
		}
	}
	p.Publish()
	dialogs.TaggedConfirmations()[3].Respond(true, appgui.ReplaceTags, "region=east,region=west")
	stored, err = f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Status, models.MenuStatusDraft)
	testutil.Equals(t, stored.Tags.Canonical().String(), "stable")
	testutil.ErrorIsInvalid(t, p.State().Err)
}

func TestWidgetAnalysisValidatesAndRendersCostCurrencyMarginAndAvailability(t *testing.T) {
	gui := frameworktest.NewApp()
	defer gui.Quit()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "Costed base", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(125, currency.USD)})
	drink := testutil.CreateDrink(t, f, drinkmodels.Drink{Name: "Costed Collins", Category: drinkmodels.DrinkCategoryMocktail, Glass: drinkmodels.GlassTypeHighball, Recipe: drinkmodels.Recipe{Ingredients: []drinkmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(2, measurement.UnitOz)}}, Steps: []string{"Build"}}})
	menu, err := f.Menus.Create(f.OwnerContext(), &models.Menu{Name: "Analysis"})
	testutil.Ok(t, err)
	_, err = f.Menus.AddDrink(f.OwnerContext(), &models.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	p.Refresh()
	driver.Tap(ControlAnalyze)
	driver.Type(ControlTargetMargin, "NaN")
	driver.Tap(ControlRunAnalysis)
	testutil.ErrorIsInvalid(t, p.State().Err)
	driver.Type(ControlTargetMargin, "+Inf")
	driver.Tap(ControlRunAnalysis)
	testutil.ErrorIsInvalid(t, p.State().Err)
	driver.Type(ControlTargetMargin, "1")
	driver.Tap(ControlRunAnalysis)
	testutil.ErrorIsInvalid(t, p.State().Err)
	// Correctable validation failures stay in the analysis form.
	testutil.Equals(t, len(dialogs.Errors()), 0)
	driver.Type(ControlTargetMargin, "0.75")
	driver.Tap(ControlRunAnalysis)
	state := p.State()
	testutil.NotNil(t, state.Analysis)
	testutil.Equals(t, state.Analysis.AvailableCount, 1)
	testutil.Equals(t, state.Analysis.TotalCount, 1)
	testutil.StringContains(t, v.analysisStatus.Text, "Costed Collins")
	testutil.StringContains(t, v.analysisStatus.Text, "$2.50")
	testutil.StringContains(t, v.analysisStatus.Text, "suggested $10.00")
	testutil.StringContains(t, v.analysisStatus.Text, "AVAILABLE")
	testutil.StringContains(t, v.analysisStatus.Text, "Margin: n/a")
	if state.Analysis.Items[0].SuggestedPrice == nil || state.Analysis.Items[0].SuggestedPrice.Currency != currency.USD {
		t.Fatal("analysis did not retain the calculated USD currency")
	}
}

func TestPresenterAnalysisRejectsNonFiniteTargetMargins(t *testing.T) {
	f := testutil.NewFixture(t)
	menu, err := f.Menus.Create(f.OwnerContext(), &models.Menu{Name: "Finite margins"})
	testutil.Ok(t, err)
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	p.state.Items = []*models.Menu{menu}
	p.Select(0)
	for _, target := range []string{"NaN", "+Inf", "-Inf"} {
		p.StartAnalysis()
		p.SetAnalysisForm(AnalysisForm{TargetMargin: target})
		testutil.Equals(t, p.Analyze(), false)
		testutil.ErrorIsInvalid(t, p.State().Err)
		testutil.Equals(t, p.State().Analysis == nil, true)
		p.Cancel()
	}
}

func TestSubmissionRetainsFormAndRejectsDuplicate(t *testing.T) {
	f := testutil.NewFixture(t)
	executor, dispatcher := &fynetest.ManualExecutor{}, &fynetest.ManualDispatcher{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: dispatcher})
	p.StartCreate()
	p.SetForm(Form{Name: "Queued", Description: "keep"})
	testutil.Equals(t, p.Save(), true)
	testutil.Equals(t, p.Save(), false)
	testutil.Equals(t, p.State().Submitting, true)
	testutil.Equals(t, p.State().Form.Description, "keep")
	snapshot := p.State()
	snapshot.Form.Name = "mutated"
	testutil.Equals(t, p.State().Form.Name, "Queued")
	testutil.Equals(t, executor.RunNext(), true)
	dispatcher.Drain()
	testutil.Equals(t, p.State().Submitting, false)
}

func TestStateSnapshotsAreDefensiveCopies(t *testing.T) {
	f := testutil.NewFixture(t)
	menu, err := f.Menus.Create(f.OwnerContext(), &models.Menu{Name: "Original"})
	testutil.Ok(t, err)
	drink := menuDrink(t, f, "Snapshot")
	menu.Items = []models.MenuItem{{DrinkID: drink.ID, Availability: models.AvailabilityAvailable}}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	p.state.Items = []*models.Menu{menu}
	p.state.Selected = menu
	p.state.Drinks = []DrinkOption{{ID: drink.ID, Name: drink.Name}}
	snapshot := p.State()
	snapshot.Items[0].Name = "Mutated"
	snapshot.Selected.Items[0].Availability = models.AvailabilityUnavailable
	snapshot.Drinks[0].Name = "Mutated"
	got := p.State()
	testutil.Equals(t, got.Items[0].Name, "Original")
	testutil.Equals(t, got.Selected.Items[0].Availability, models.AvailabilityAvailable)
	testutil.Equals(t, got.Drinks[0].Name, "Snapshot")
}

func TestWidgetPagesMoreThanOneHundredMenusAndValidatesPageSize(t *testing.T) {
	f := testutil.NewFixture(t)
	for i := 0; i <= 100; i++ {
		_, err := f.Menus.Create(f.OwnerContext(), &models.Menu{Name: fmt.Sprintf("Menu %03d", i)})
		testutil.Ok(t, err)
	}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	v.filterLimit.SetSelected("25")
	driver.Tap(ControlApplyFilter)
	testutil.Equals(t, len(p.State().Items), 25)
	first := p.State().Items[0].ID
	driver.Tap(ControlNext)
	testutil.Equals(t, len(p.State().Items), 25)
	if p.State().Items[0].ID == first {
		t.Fatal("next retained first page")
	}
	driver.Tap(ControlPrevious)
	testutil.Equals(t, p.State().Items[0].ID, first)
	if p.SetFilter(Filter{Limit: -1}) {
		t.Fatal("negative page size accepted")
	}
}

func TestDeniedMutationRetainsFormAndRollsBack(t *testing.T) {
	f := testutil.NewFixture(t)
	menu, err := f.Menus.Create(f.OwnerContext(), &models.Menu{Name: "Protected", Description: "original"})
	testutil.Ok(t, err)
	denied := appcore.NewSession(f.ActorContext("bartender"), f.App.App)
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(denied, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	p.state.Items = []*models.Menu{menu}
	p.Select(0)
	p.StartRename()
	p.SetForm(Form{Name: "Forbidden", Description: "retain me"})
	testutil.Equals(t, p.Save(), true)
	testutil.Equals(t, p.State().Mode, Renaming)
	testutil.Equals(t, p.State().Form.Description, "retain me")
	if p.State().Err == nil || len(dialogs.Errors()) != 1 {
		t.Fatal("authorization failure was not retained and presented")
	}
	stored, err := f.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Name, "Protected")
}

func TestAcceptedSubmissionAndConfirmationDisableEveryControl(t *testing.T) {
	gui := frameworktest.NewApp()
	defer gui.Quit()
	f := testutil.NewFixture(t)
	menu, err := f.Menus.Create(f.OwnerContext(), &models.Menu{Name: "Pending"})
	testutil.Ok(t, err)
	executor := &fynetest.ManualExecutor{}
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	p.state.Items = []*models.Menu{menu}
	p.Select(0)
	v := NewView(p)
	p.StartRename()
	p.SetForm(Form{Name: "Queued"})
	p.Save()
	for name, disabled := range map[string]bool{"save": v.save.Disabled(), "cancel": v.cancel.Disabled(), "name": v.name.Disabled(), "description": v.description.Disabled()} {
		if !disabled {
			t.Fatalf("%s remained enabled during submission", name)
		}
	}
	executor.RunNext()
	p.state.Items = []*models.Menu{menu}
	p.state.Selected = cloneMenu(menu)
	p.state.Mode = Browsing
	p.publish()
	p.Delete()
	for name, disabled := range map[string]bool{"refresh": v.refresh.Disabled(), "create": v.create.Disabled(), "filter status": v.filterStatus.Disabled(), "filter": v.filterExpression.Disabled(), "apply": v.applyFilter.Disabled(), "rename": v.rename.Disabled(), "tags": v.tagAction.Disabled(), "delete": v.delete.Disabled()} {
		if !disabled {
			t.Fatalf("%s remained enabled during confirmation", name)
		}
	}
	dialogs.Confirmations()[0].Respond(false)
	testutil.Equals(t, p.State().Confirming, false)
}

func TestDrinkChoiceLoadDoesNotWipeLiveSearchEdit(t *testing.T) {
	gui := frameworktest.NewApp()
	defer gui.Quit()
	f := testutil.NewFixture(t)
	menu, err := f.Menus.Create(f.OwnerContext(), &models.Menu{Name: "Curate"})
	testutil.Ok(t, err)
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appgui.InlineDispatcher{}})
	p.state.Items = []*models.Menu{menu}
	p.Select(0)
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	p.StartAddDrink()
	driver.Type(ControlDrinkSearch, "typed while loading")
	testutil.Equals(t, executor.RunNext(), true)
	testutil.Equals(t, v.drinkSearch.Text, "typed while loading")
}

func TestCLIWorkflowAndFyneShareMenuPersistenceContract(t *testing.T) {
	repo, err := filepath.Abs("../../../../../")
	testutil.Ok(t, err)
	dir := t.TempDir()
	binary := testutil.ExecutablePath(dir, "mixology")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./main/cli")
	build.Dir = repo
	output, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	run := func(stdin string, args ...string) string {
		cmd := exec.CommandContext(t.Context(), binary, args...)
		cmd.Dir = dir
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI %v: %v\n%s", args, err, output)
		}
		return string(output)
	}
	cliJSON := fmt.Sprintf(`{"name":%q,"description":"CLI description"}`, "Created through CLI")
	run(cliJSON, "--log-level", "error", "menus", "create", "--stdin")
	ctx := authn.ToContext(context.Background(), authn.Owner())
	ctx = pkglog.ToContext(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)))
	database, err := store.Open(ctx, filepath.Join(dir, "data", "mixology.db"))
	testutil.Ok(t, err)
	application := appcore.New(ctx, appcore.Config{Store: database})
	session := appcore.NewSession(ctx, application)
	p := NewPresenter(session, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	p.Refresh()
	foundCLI := false
	for _, item := range p.State().Items {
		foundCLI = foundCLI || item.Name == "Created through CLI"
	}
	testutil.Equals(t, foundCLI, true)
	p.StartCreate()
	p.SetForm(Form{Name: "Created in Fyne", Description: "desktop description"})
	p.Save()
	testutil.Ok(t, application.Close())
	cliOutput := run("", "--log-level", "error", "menus", "list", "--filter", `name == "Created in Fyne"`)
	testutil.StringContains(t, cliOutput, "Created in Fyne")
}

func menuDrink(t *testing.T, f *testutil.Fixture, name string) *drinkmodels.Drink {
	t.Helper()
	ingredient := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "Base", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	return testutil.CreateDrink(t, f, drinkmodels.Drink{Name: name, Category: drinkmodels.DrinkCategoryMocktail, Glass: drinkmodels.GlassTypeHighball, Recipe: drinkmodels.Recipe{Ingredients: []drinkmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"Build"}}})
}

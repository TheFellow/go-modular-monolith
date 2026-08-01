//nolint:paralleltest // desktop tests share Fyne process-global application and window state.
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	framework "fyne.io/fyne/v2"
	fynedesktop "fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	application "github.com/TheFellow/go-modular-monolith/app"
	auditgui "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/gui"
	drinksgui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/gui"
	ingredientsdomain "github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientsgui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/gui"
	inventorygui "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/gui"
	menusgui "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/gui"
	ordersgui "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/gui"
	tagginggui "github.com/TheFellow/go-modular-monolith/app/domains/tagging/surfaces/gui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func deterministicDesktopDependencies(onInformation func(string, string, framework.Window)) desktopDependencies {
	if onInformation == nil {
		onInformation = func(string, string, framework.Window) {}
	}
	return desktopDependencies{
		executor: toolkit.InlineExecutor{}, dispatcher: toolkit.InlineDispatcher{},
		dialogs:         func(window framework.Window) toolkit.Dialogs { return toolkit.WindowDialogs{Window: window} },
		showInformation: onInformation,
		openURL:         func(*url.URL) error { return nil },
		dashboardLoader: func(session *application.Session) dashboardLoader {
			return sessionDashboardLoader{session: session}
		},
	}
}

func TestSommelierDesktopShowsOnlyReadableWorkspaces(t *testing.T) {
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: t.TempDir(), actor: "sommelier",
	}, deterministicDesktopDependencies(nil))
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	t.Cleanup(func() { _ = desktop.Close() })

	want := []string{"dashboard", "drinks", "ingredients", "inventory", "menus", "orders"}
	if got := desktop.shell.RouteIDs(); !slices.Equal(got, want) {
		testutil.ErrorIf(t, true, "sommelier routes = %v, want %v", got, want)
	}
	for _, hidden := range []string{"audit", "tags"} {
		if err := desktop.shell.Navigate(hidden); err == nil {
			testutil.ErrorIf(t, true, "sommelier could navigate to hidden workspace %q", hidden)
		}
	}
}

func TestOpenDesktopBuildsHeadlessShellAndOwnsPersistenceLifecycle(t *testing.T) {
	dataDirectory := t.TempDir()
	gui := test.NewApp()
	t.Cleanup(gui.Quit)

	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: dataDirectory,
		actor:         "owner",
	}, deterministicDesktopDependencies(nil))
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if desktop.session == nil || desktop.window == nil || desktop.shell.Current() != "dashboard" {
		testutil.ErrorIf(t, true, "%v", "desktop composition did not produce a session, window, and dashboard shell")
	}
	if err := desktop.shell.Navigate("drinks"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if desktop.shell.Current() != "drinks" {
		testutil.ErrorIf(t, true, "current route = %q, want drinks", desktop.shell.Current())
	}
	if err := desktop.Close(); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if err := desktop.Close(); err != nil {
		testutil.ErrorIf(t, true, "second close was not safe: %v", err)
	}

	databasePath := filepath.Join(dataDirectory, databaseFilename)
	if _, err := os.Stat(databasePath); err != nil {
		testutil.ErrorIf(t, true, "desktop database was not created: %v", err)
	}
	reopened, err := store.Open(context.Background(), databasePath)
	if err != nil {
		testutil.ErrorIf(t, true, "desktop did not release database: %v", err)
	}
	if err := reopened.Close(); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
}

func TestDesktopDashboardWorkflowNavigatesAllWorkspacesAndPreservesState(t *testing.T) {
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: t.TempDir(), actor: "owner",
	}, deterministicDesktopDependencies(nil))
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	t.Cleanup(func() { _ = desktop.Close() })
	if got := desktop.dashboard.Snapshot(); got.Status != toolkit.Loaded || got.Err != nil {
		testutil.ErrorIf(t, true, "initial dashboard = %#v", got)
	}

	driver := fynetest.NewDriver(t, desktop.shell.Content())
	for _, route := range []string{"drinks", "ingredients", "inventory", "menus", "orders", "audit", "tags"} {
		if err := desktop.shell.Navigate("dashboard"); err != nil {
			testutil.ErrorIf(t, true, "%v", err)
		}
		driver.Tap("dashboard-open-" + route)
		if desktop.shell.Current() != route {
			testutil.ErrorIf(t, true, "current route = %q, want %q", desktop.shell.Current(), route)
		}
	}
	model := desktop.dashboard
	beforeAudit := model.Snapshot().Data.AuditCount
	_, err = desktop.session.Ingredients.Create(desktop.session.Context(), &ingredientsmodels.Ingredient{
		Name: "Fresh dashboard ingredient", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if err := desktop.shell.Navigate("dashboard"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if desktop.dashboard != model {
		testutil.ErrorIf(t, true, "%v", "dashboard presenter was rebuilt after navigation")
	}
	refreshed := model.Snapshot().Data
	if refreshed.IngredientCount != 1 || refreshed.AuditCount <= beforeAudit || len(refreshed.RecentActivity) == 0 {
		testutil.ErrorIf(t, true, "dashboard was not refreshed after reentry: %#v", refreshed)
	}
}

func TestDesktopRoutesBuildConcreteDomainViewsAndActivationReadsCurrentData(t *testing.T) {
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: t.TempDir(), actor: "owner",
	}, deterministicDesktopDependencies(nil))
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	t.Cleanup(func() { _ = desktop.Close() })

	wantTypes := map[string]any{
		"drinks": (*drinksgui.View)(nil), "ingredients": (*ingredientsgui.View)(nil),
		"inventory": (*inventorygui.View)(nil), "menus": (*menusgui.View)(nil),
		"orders": (*ordersgui.View)(nil), "audit": (*auditgui.View)(nil),
		"tags": (*tagginggui.View)(nil),
	}
	for route, want := range wantTypes {
		if err := desktop.shell.Navigate(route); err != nil {
			testutil.ErrorIf(t, true, "%v", err)
		}
		if got := desktop.views[route]; fmt.Sprintf("%T", got) != fmt.Sprintf("%T", want) {
			testutil.ErrorIf(t, true, "route %q built %T, want %T", route, got, want)
		}
	}

	first, err := desktop.session.Ingredients.Create(desktop.session.Context(), &ingredientsmodels.Ingredient{
		Name: "First activation", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if err := desktop.shell.Navigate("ingredients"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	presenter := desktop.presenters["ingredients"].(*ingredientsgui.Presenter)
	if got := presenter.Snapshot(); len(got.Items) != 1 || got.Items[0].ID != first.ID {
		testutil.ErrorIf(t, true, "initial activation did not read current ingredients: %#v", got)
	}

	second, err := desktop.session.Ingredients.Create(desktop.session.Context(), &ingredientsmodels.Ingredient{
		Name: "Reentry activation", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if err := desktop.shell.Navigate("dashboard"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if err := desktop.shell.Navigate("ingredients"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	got := presenter.Snapshot()
	foundSecond := false
	for _, item := range got.Items {
		foundSecond = foundSecond || item.ID == second.ID
	}
	if len(got.Items) != 2 || !foundSecond {
		testutil.ErrorIf(t, true, "reentry activation did not refresh ingredients: %#v", got)
	}
}

func TestDesktopConfirmsBeforeNavigatingAwayFromEditor(t *testing.T) {
	guiApp := test.NewApp()
	t.Cleanup(guiApp.Quit)
	confirmations := &fynetest.Dialogs{}
	deps := deterministicDesktopDependencies(nil)
	deps.dialogs = func(framework.Window) toolkit.Dialogs { return confirmations }
	desktop, err := openDesktopWithDependencies(context.Background(), guiApp, desktopConfig{dataDirectory: t.TempDir(), actor: "owner"}, deps)
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	t.Cleanup(func() { _ = desktop.Close() })
	if err := desktop.shell.Navigate("drinks"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	desktop.presenters["drinks"].(*drinksgui.Presenter).StartCreate()
	if err := desktop.shell.Navigate("ingredients"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if desktop.shell.Current() != "drinks" || len(confirmations.Confirmations()) != 1 {
		testutil.ErrorIf(t, true, "%v", "editor navigation was not held for confirmation")
	}
	confirmations.Confirmations()[0].Respond(true)
	if desktop.shell.Current() != "ingredients" {
		testutil.ErrorIf(t, true, "%v", "confirmed editor navigation did not continue")
	}
}

func TestDesktopSemanticControlsExerciseEveryWorkspaceAgainstOneSession(t *testing.T) {
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: t.TempDir(), actor: "owner",
	}, deterministicDesktopDependencies(nil))
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	t.Cleanup(func() { _ = desktop.Close() })

	created, err := desktop.session.Ingredients.Create(desktop.session.Context(), &ingredientsmodels.Ingredient{
		Name: "Composed desktop ingredient", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	driver := fynetest.NewDriver(t, desktop.shell.Content())
	refreshes := []struct{ route, control string }{
		{"drinks", drinksgui.ControlRefresh},
		{"ingredients", "ingredients-refresh"},
		{"inventory", "inventory-refresh"},
		{"menus", menusgui.ControlRefresh},
		{"orders", "orders-refresh"},
		{"audit", auditgui.ControlRefresh},
	}
	for _, item := range refreshes {
		t.Run(item.route, func(t *testing.T) {
			if err := desktop.shell.Navigate("dashboard"); err != nil {
				testutil.ErrorIf(t, true, "%v", err)
			}
			driver.Tap("dashboard-open-" + item.route)
			driver.Tap(item.control)
			if desktop.shell.Current() != item.route {
				testutil.ErrorIf(t, true, "current route = %q", desktop.shell.Current())
			}
			switch item.route {
			case "drinks":
				if state := desktop.presenters[item.route].(*drinksgui.Presenter).State(); state.Loading || state.Err != nil {
					testutil.ErrorIf(t, true, "drinks refresh = %#v", state)
				}
			case "ingredients":
				if state := desktop.presenters[item.route].(*ingredientsgui.Presenter).Snapshot(); state.Status != toolkit.Loaded || state.Err != nil {
					testutil.ErrorIf(t, true, "ingredients refresh = %#v", state)
				}
			case "inventory":
				if state := desktop.presenters[item.route].(*inventorygui.Presenter).Snapshot(); state.Status != toolkit.Loaded || state.Err != nil {
					testutil.ErrorIf(t, true, "inventory refresh = %#v", state)
				}
			case "menus":
				if state := desktop.presenters[item.route].(*menusgui.Presenter).State(); state.Loading || state.Err != nil {
					testutil.ErrorIf(t, true, "menus refresh = %#v", state)
				}
			case "orders":
				if state := desktop.presenters[item.route].(*ordersgui.Presenter).State(); state.Loading || state.Err != nil {
					testutil.ErrorIf(t, true, "orders refresh = %#v", state)
				}
			case "audit":
				if state := desktop.presenters[item.route].(*auditgui.Presenter).State(); state.Loading || state.Err != nil {
					testutil.ErrorIf(t, true, "audit refresh = %#v", state)
				}
			}
		})
	}

	ingredients := desktop.presenters["ingredients"].(*ingredientsgui.Presenter).Snapshot()
	if len(ingredients.Items) != 1 || ingredients.Items[0].ID != created.ID || ingredients.Err != nil {
		testutil.ErrorIf(t, true, "ingredients did not observe the composed session write: %#v", ingredients)
	}
	audit := desktop.presenters["audit"].(*auditgui.Presenter).State()
	if len(audit.Rows) == 0 || audit.Err != nil {
		testutil.ErrorIf(t, true, "audit did not observe the same-session mutation: %#v", audit)
	}

	if err := desktop.shell.Navigate("dashboard"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	driver.Tap("dashboard-open-tags")
	driver.Tap(tagginggui.ControlSummary)
	if state := desktop.presenters["tags"].(*tagginggui.Presenter).State(); state.Mode != tagginggui.Results || state.Err != nil {
		testutil.ErrorIf(t, true, "tags semantic workflow = %#v", state)
	}
}

func TestRestrictedDesktopHidesUnauthorizedTagWorkspaceAndDoesNotWrite(t *testing.T) {
	dataDirectory := t.TempDir()
	seedGUI := test.NewApp()
	seed, err := openDesktopWithDependencies(context.Background(), seedGUI, desktopConfig{
		dataDirectory: dataDirectory, actor: "owner",
	}, deterministicDesktopDependencies(nil))
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	ingredient, err := seed.session.Ingredients.Create(seed.session.Context(), &ingredientsmodels.Ingredient{
		Name: "Protected ingredient", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if err := seed.Close(); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	seedGUI.Quit()

	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	deps := deterministicDesktopDependencies(nil)
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: dataDirectory, actor: "bartender",
	}, deps)
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	t.Cleanup(func() { _ = desktop.Close() })
	if slices.Contains(desktop.shell.RouteIDs(), "tags") {
		testutil.ErrorIf(t, true, "%v", "restricted desktop exposed Tags navigation")
	}
	if err := desktop.shell.Navigate("tags"); err == nil {
		testutil.ErrorIf(t, true, "%v", "restricted desktop accepted hidden Tags route")
	}
	values, err := desktop.session.Tags.List(desktop.session.Context(), ingredient.EntityUID())
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if len(values) != 0 {
		testutil.ErrorIf(t, true, "denied mutation persisted: %#v", values)
	}
}

func TestDesktopProvidesNativeAboutHelpAndQuitMenus(t *testing.T) {
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	var informationTitle string
	var openedURL string
	deps := deterministicDesktopDependencies(func(title, _ string, _ framework.Window) { informationTitle = title })
	deps.openURL = func(target *url.URL) error { openedURL = target.String(); return nil }
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: t.TempDir(), actor: "owner",
	}, deps)
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	t.Cleanup(func() { _ = desktop.Close() })

	menu := desktop.window.MainMenu()
	if menu == nil || len(menu.Items) != 4 || menu.Items[0].Label != "Mixology" || menu.Items[1].Label != "File" || menu.Items[2].Label != "View" || menu.Items[3].Label != "Help" {
		testutil.ErrorIf(t, true, "main menu = %#v", menu)
	}
	menu.Items[0].Items[0].Action()
	if informationTitle != "About Mixology" {
		testutil.ErrorIf(t, true, "information title = %q", informationTitle)
	}
	if menu.Items[0].Items[len(menu.Items[0].Items)-1].Label != "Quit" {
		testutil.ErrorIf(t, true, "%v", "application menu does not expose Quit")
	}
	menu.Items[3].Items[0].Action()
	if openedURL != "https://thefellow.github.io/go-modular-monolith/" {
		testutil.ErrorIf(t, true, "help URL = %q", openedURL)
	}
}

func TestDesktopMenuShortcutsNavigateAndRespectWorkspaceMode(t *testing.T) {
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: t.TempDir(), actor: "owner",
	}, deterministicDesktopDependencies(nil))
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	t.Cleanup(func() { _ = desktop.Close() })
	menu := desktop.window.MainMenu()
	view := menu.Items[2]
	if shortcut, ok := view.Items[1].Shortcut.(*fynedesktop.CustomShortcut); !ok || shortcut.KeyName != framework.Key2 || shortcut.Modifier != framework.KeyModifierAlt {
		testutil.ErrorIf(t, true, "Drinks shortcut = %#v", view.Items[1].Shortcut)
	}
	view.Items[1].Action()
	if desktop.shell.Current() != "drinks" {
		testutil.ErrorIf(t, true, "Alt+2 route = %q", desktop.shell.Current())
	}
	presenter := desktop.presenters["drinks"].(*drinksgui.Presenter)
	file := menu.Items[1]
	file.Items[0].Action()
	if presenter.State().Mode != drinksgui.Creating {
		testutil.ErrorIf(t, true, "New mode = %v", presenter.State().Mode)
	}
	file.Items[0].Action()
	if presenter.State().Mode != drinksgui.Creating {
		testutil.ErrorIf(t, true, "%v", "New escaped its active-mode guard")
	}
	file.Items[4].Action()
	if presenter.State().Mode != drinksgui.Browsing {
		testutil.ErrorIf(t, true, "Escape mode = %v", presenter.State().Mode)
	}
}

func TestDesktopQuitMenuReleasesPersistence(t *testing.T) {
	dataDirectory := t.TempDir()
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: dataDirectory, actor: "owner",
	}, deterministicDesktopDependencies(nil))
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	menu := desktop.window.MainMenu()
	menu.Items[0].Items[len(menu.Items[0].Items)-1].Action()
	reopened, err := store.Open(context.Background(), filepath.Join(dataDirectory, databaseFilename))
	if err != nil {
		testutil.ErrorIf(t, true, "Quit did not release database: %v", err)
	}
	if err := reopened.Close(); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
}

type blockingDashboardLoader struct {
	delegate dashboardLoader
	started  chan struct{}
	release  chan struct{}
	once     sync.Once
}

func (l *blockingDashboardLoader) LoadDashboard() (application.Dashboard, error) {
	data, err := l.delegate.LoadDashboard()
	l.once.Do(func() { close(l.started) })
	<-l.release
	return data, err
}

func TestDesktopCloseWaitsForInFlightDashboardLoaderWorkBeforeClosingDatabase(t *testing.T) {
	dataDirectory := t.TempDir()
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	started := make(chan struct{})
	release := make(chan struct{})
	deps := deterministicDesktopDependencies(nil)
	deps.executor = toolkit.AsyncExecutor{}
	deps.dispatcher = toolkit.MainDispatcher{}
	deps.dashboardLoader = func(session *application.Session) dashboardLoader {
		return &blockingDashboardLoader{
			delegate: sessionDashboardLoader{session: session}, started: started, release: release,
		}
	}
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: dataDirectory, actor: "owner",
	}, deps)
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		testutil.ErrorIf(t, true, "%v", "dashboard loader work did not reach its controlled block")
	}
	closed := make(chan error, 1)
	waiting := make(chan struct{})
	desktop.dashboard.beforeCloseWait = func() { close(waiting) }
	go func() {
		closed <- desktop.Close()
	}()
	<-waiting
	select {
	case err := <-closed:
		testutil.ErrorIf(t, true, "Close returned before dashboard loader work completed: %v", err)
	default:
	}
	close(release)
	select {
	case err := <-closed:
		if err != nil {
			testutil.ErrorIf(t, true, "%v", err)
		}
	case <-time.After(5 * time.Second):
		testutil.ErrorIf(t, true, "%v", "Close deadlocked after dashboard loader work completed")
	}
	reopened, err := store.Open(context.Background(), filepath.Join(dataDirectory, databaseFilename))
	if err != nil {
		testutil.ErrorIf(t, true, "Close did not release database: %v", err)
	}
	if err := reopened.Close(); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
}

func TestDesktopCloseDrainsRealDomainLoadAndMutationBeforeStoreShutdown(t *testing.T) {
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	executor := toolkit.NewManagedExecutor()
	dispatcher := toolkit.NewGatedDispatcher(toolkit.InlineDispatcher{})
	deps := deterministicDesktopDependencies(nil)
	deps.executor = executor
	deps.dispatcher = dispatcher
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: t.TempDir(), actor: "owner",
	}, deps)
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	executor.Wait() // drain initial dashboard activation before controlling work

	started := make(chan struct{}, 2)
	release := make(chan struct{})
	results := make(chan error, 2)
	executor.Execute(func() {
		started <- struct{}{}
		<-release
		_, loadErr := desktop.session.Ingredients.Count(desktop.session.Context(), ingredientsdomain.ListRequest{})
		results <- loadErr
	})
	executor.Execute(func() {
		started <- struct{}{}
		<-release
		_, mutationErr := desktop.session.Ingredients.Create(desktop.session.Context(), &ingredientsmodels.Ingredient{
			Name: "Shutdown mutation", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
		})
		results <- mutationErr
	})
	<-started
	<-started
	closed := make(chan error, 1)
	go func() { closed <- desktop.Close() }()
	select {
	case closeErr := <-closed:
		testutil.ErrorIf(t, true, "Close returned while accepted domain work was blocked: %v", closeErr)
	default:
	}
	if _, err := desktop.session.Ingredients.Count(desktop.session.Context(), ingredientsdomain.ListRequest{}); err != nil {
		testutil.ErrorIf(t, true, "store closed before accepted work drained: %v", err)
	}
	close(release)
	for range 2 {
		if err := <-results; err != nil {
			testutil.ErrorIf(t, true, "accepted domain work raced store shutdown: %v", err)
		}
	}
	if err := <-closed; err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
}

func TestOpenDesktopRejectsMissingPresentationDependenciesAndReleasesDatabase(t *testing.T) {
	dataDirectory := t.TempDir()
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	_, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: dataDirectory, actor: "owner",
	}, desktopDependencies{})
	if err == nil {
		testutil.ErrorIf(t, true, "%v", "expected missing dependency error")
	}
	reopened, openErr := store.Open(context.Background(), filepath.Join(dataDirectory, databaseFilename))
	if openErr != nil {
		testutil.ErrorIf(t, true, "failed composition retained database: %v", openErr)
	}
	if err := reopened.Close(); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
}

func TestOpenDesktopRejectsInvalidActorBeforeOpeningDatabase(t *testing.T) {
	dataDirectory := t.TempDir()
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	_, err := openDesktop(context.Background(), gui, desktopConfig{
		dataDirectory: dataDirectory,
		actor:         "intruder",
	})
	if err == nil {
		testutil.ErrorIf(t, true, "%v", "expected invalid actor error")
	}
	if _, statErr := os.Stat(filepath.Join(dataDirectory, databaseFilename)); !os.IsNotExist(statErr) {
		testutil.ErrorIf(t, true, "database unexpectedly opened: %v", statErr)
	}
}

func TestCloseWindowReleasesPersistence(t *testing.T) {
	dataDirectory := t.TempDir()
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: dataDirectory,
		actor:         "owner",
	}, deterministicDesktopDependencies(nil))
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}

	desktop.closeWindow()
	reopened, err := store.Open(context.Background(), filepath.Join(dataDirectory, databaseFilename))
	if err != nil {
		testutil.ErrorIf(t, true, "close callback did not release database: %v", err)
	}
	if err := reopened.Close(); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
}

func TestPrepareDataDirectoryRejectsEmptyPath(t *testing.T) {
	if err := prepareDataDirectory(""); err == nil {
		testutil.ErrorIf(t, true, "%v", "expected empty data directory error")
	}
}

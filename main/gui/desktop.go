package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	fynedesktop "fyne.io/fyne/v2/driver/desktop"

	application "github.com/TheFellow/go-modular-monolith/app"
	auditauthz "github.com/TheFellow/go-modular-monolith/app/domains/audit/authz"
	auditgui "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/gui"
	drinksdomain "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinksgui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/gui"
	ingredientsdomain "github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientsgui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/gui"
	inventorydomain "github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorygui "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/gui"
	menusdomain "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menusgui "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/gui"
	ordersdomain "github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersgui "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/gui"
	taggingauthz "github.com/TheFellow/go-modular-monolith/app/domains/tagging/authz"
	tagginggui "github.com/TheFellow/go-modular-monolith/app/domains/tagging/surfaces/gui"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	pkg_authz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	pkglog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/runtimeconfig"
	"github.com/TheFellow/go-modular-monolith/pkg/set"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	gui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
	cedar "github.com/cedar-policy/cedar-go"
)

const (
	applicationID    = "com.thefellow.mixology"
	databaseFilename = "mixology.db"
	logFilename      = "mixology.log"
)

type desktopConfig struct {
	dataDirectory string
	databasePath  string
	actor         string
	logLevel      string
	logFormat     string
	logFile       string
	enableMetrics bool
}

type desktop struct {
	gui             framework.App
	window          framework.Window
	application     *application.App
	session         *application.Session
	shell           *gui.Shell
	logFile         *os.File
	metricsServer   *http.Server
	metricsShutdown func(context.Context) error
	closeOnce       sync.Once
	closeErr        error
	dashboard       *dashboardViewModel
	views           map[string]gui.View
	presenters      map[string]any
	executor        interface{ Close() }
	dispatcher      interface{ Close() }
	showInformation func(string, string, framework.Window)
	openURL         func(*url.URL) error
}

func (d *desktop) mainMenu() *framework.MainMenu {
	about := framework.NewMenuItem("About Mixology", func() {
		d.showInformation("About Mixology", "Mixology is a modular-monolith teaching application built entirely in Go.", d.window)
	})
	quit := framework.NewMenuItem("Quit", d.closeWindow)
	quit.Shortcut = &fynedesktop.CustomShortcut{KeyName: framework.KeyQ, Modifier: framework.KeyModifierShortcutDefault}
	guide := framework.NewMenuItem("Mixology Guide", func() {
		guideURL, err := url.Parse("https://thefellow.github.io/go-modular-monolith/")
		if err == nil {
			_ = d.openURL(guideURL)
		}
	})
	refresh := framework.NewMenuItem("Refresh", func() { d.shell.ExecuteCommand(gui.CommandRefresh) })
	refresh.Shortcut = commandShortcut(framework.KeyR)
	create := framework.NewMenuItem("New", func() { d.shell.ExecuteCommand(gui.CommandNew) })
	create.Shortcut = commandShortcut(framework.KeyN)
	save := framework.NewMenuItem("Save", func() { d.shell.ExecuteCommand(gui.CommandSave) })
	save.Shortcut = commandShortcut(framework.KeyS)
	cancel := framework.NewMenuItem("Cancel or Back", func() { d.shell.ExecuteCommand(gui.CommandCancel) })
	cancel.Shortcut = &fynedesktop.CustomShortcut{KeyName: framework.KeyEscape}
	viewItems := make([]*framework.MenuItem, 0, len(d.shell.RouteIDs()))
	for i, route := range d.shell.RouteIDs() {
		item := framework.NewMenuItem(d.shell.RouteLabel(route), func() { _ = d.shell.Navigate(route) })
		if i < len(routeKeys) {
			item.Shortcut = &fynedesktop.CustomShortcut{KeyName: routeKeys[i], Modifier: framework.KeyModifierAlt}
		}
		viewItems = append(viewItems, item)
	}
	return framework.NewMainMenu(
		framework.NewMenu("Mixology", about, framework.NewMenuItemSeparator(), quit),
		framework.NewMenu("File", create, save, framework.NewMenuItemSeparator(), refresh, cancel),
		framework.NewMenu("View", viewItems...),
		framework.NewMenu("Help", guide),
	)
}

var routeKeys = []framework.KeyName{framework.Key1, framework.Key2, framework.Key3, framework.Key4, framework.Key5, framework.Key6, framework.Key7, framework.Key8}

func commandShortcut(key framework.KeyName) *fynedesktop.CustomShortcut {
	return &fynedesktop.CustomShortcut{KeyName: key, Modifier: framework.KeyModifierShortcutDefault}
}

func (d *desktop) registerShortcuts() {
	canvas := d.window.Canvas()
	for _, binding := range []struct {
		shortcut *fynedesktop.CustomShortcut
		command  gui.Command
	}{
		{commandShortcut(framework.KeyR), gui.CommandRefresh},
		{commandShortcut(framework.KeyN), gui.CommandNew},
		{commandShortcut(framework.KeyS), gui.CommandSave},
		{&fynedesktop.CustomShortcut{KeyName: framework.KeyEscape}, gui.CommandCancel},
	} {
		command := binding.command
		canvas.AddShortcut(binding.shortcut, func(framework.Shortcut) { d.shell.ExecuteCommand(command) })
	}
	for i, route := range d.shell.RouteIDs() {
		if i >= len(routeKeys) {
			break
		}
		route := route
		canvas.AddShortcut(&fynedesktop.CustomShortcut{KeyName: routeKeys[i], Modifier: framework.KeyModifierAlt}, func(framework.Shortcut) { _ = d.shell.Navigate(route) })
	}
}

type desktopDependencies struct {
	executor        gui.Executor
	dispatcher      gui.Dispatcher
	dialogs         func(framework.Window) gui.Dialogs
	showInformation func(title, message string, window framework.Window)
	openURL         func(*url.URL) error
	dashboardLoader func(*application.Session) dashboardLoader
}

// visibleWorkspaces probes the same authorized read paths used by each
// workspace. Permission denials remove a workspace; operational failures leave
// it visible so its surface can report the underlying problem.
func visibleWorkspaces(session *application.Session) set.Set[workspace] {
	visible := set.New(workspaceDashboard)
	checks := []struct {
		id   workspace
		read func() error
	}{
		{workspaceDrinks, func() error {
			_, err := session.Drinks.Count(session.Context(), drinksdomain.ListRequest{})
			return err
		}},
		{workspaceIngredients, func() error {
			_, err := session.Ingredients.Count(session.Context(), ingredientsdomain.ListRequest{})
			return err
		}},
		{workspaceInventory, func() error {
			_, err := session.Inventory.Count(session.Context(), inventorydomain.ListRequest{})
			return err
		}},
		{workspaceMenus, func() error { _, err := session.Menus.Count(session.Context(), menusdomain.ListRequest{}); return err }},
		{workspaceOrders, func() error {
			_, err := session.Orders.Count(session.Context(), ordersdomain.ListRequest{})
			return err
		}},
		{workspaceAudit, func() error {
			resource := auditauthz.AuditEntry{UID: cedar.NewEntityUID(auditauthz.AuditEntryType, "workspace")}
			return pkg_authz.AuthorizeWithEntity(session.Context().Principal(), auditauthz.ActionList, resource.CedarEntity())
		}},
		{workspaceTags, func() error {
			resource := taggingauthz.TagDiscovery{UID: cedar.NewEntityUID(taggingauthz.TagDiscoveryType, "workspace")}
			return pkg_authz.AuthorizeWithEntity(session.Context().Principal(), taggingauthz.ActionSummary, resource.CedarEntity())
		}},
	}
	for _, check := range checks {
		if err := check.read(); err == nil || !apperrors.IsPermission(err) {
			visible.Add(check.id)
		}
	}
	return visible
}

func openDesktop(ctx context.Context, fyneApp framework.App, config desktopConfig) (*desktop, error) {
	executor := gui.NewManagedExecutor()
	dispatcher := gui.NewGatedDispatcher(gui.MainDispatcher{})
	return openDesktopWithDependencies(ctx, fyneApp, config, desktopDependencies{
		executor: executor, dispatcher: dispatcher,
		dialogs:         func(window framework.Window) gui.Dialogs { return gui.WindowDialogs{Window: window} },
		showInformation: dialog.ShowInformation,
		openURL:         fyneApp.OpenURL,
		dashboardLoader: func(session *application.Session) dashboardLoader {
			return sessionDashboardLoader{session: session}
		},
	})
}

func openDesktopWithDependencies(ctx context.Context, fyneApp framework.App, config desktopConfig, deps desktopDependencies) (*desktop, error) {
	if fyneApp == nil {
		return nil, fmt.Errorf("fyne application is required")
	}
	if err := prepareDataDirectory(config.dataDirectory); err != nil {
		return nil, err
	}
	principal, err := authn.ParseActor(config.actor)
	if err != nil {
		return nil, err
	}

	logPath := config.logFile
	if logPath == "" {
		logPath = filepath.Join(config.dataDirectory, logFilename)
	}
	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open desktop log: %w", err)
	}
	ctx = authn.ToContext(ctx, principal)
	logLevel, logFormat := config.logLevel, config.logFormat
	if logLevel == "" {
		logLevel = "info"
	}
	if logFormat == "" {
		logFormat = "text"
	}
	ctx = pkglog.ToContext(ctx, pkglog.Setup(logLevel, logFormat, logFile))
	metrics := telemetry.Nop()
	var metricsServer *http.Server
	var metricsShutdown func(context.Context) error
	if config.enableMetrics {
		prom, metricsErr := telemetry.NewPrometheus()
		if metricsErr != nil {
			_ = logFile.Close()
			return nil, metricsErr
		}
		metrics = prom.Metrics
		metricsShutdown = prom.Shutdown
		mux := http.NewServeMux()
		mux.Handle("/metrics", prom.Handler)
		metricsServer = &http.Server{Addr: runtimeconfig.DefaultMetricsAddr, Handler: mux}
		go func() { _ = metricsServer.ListenAndServe() }()
	}
	ctx = telemetry.WithMetrics(ctx, metrics)

	databasePath := config.databasePath
	if databasePath == "" {
		// Tests and embedders may supply only a data directory. The executable
		// supplies the shared CLI/TUI default explicitly.
		databasePath = filepath.Join(config.dataDirectory, databaseFilename)
	}
	s, err := store.Open(ctx, databasePath)
	if err != nil {
		if metricsServer != nil {
			_ = metricsServer.Shutdown(context.Background())
		}
		if metricsShutdown != nil {
			_ = metricsShutdown(context.Background())
		}
		_ = logFile.Close()
		return nil, err
	}
	app := application.New(ctx, application.Config{Store: s})
	d := &desktop{
		gui: fyneApp, application: app, session: application.NewSession(ctx, app), logFile: logFile,
		metricsServer: metricsServer, metricsShutdown: metricsShutdown,
		views: make(map[string]gui.View), presenters: make(map[string]any),
	}
	if closer, ok := deps.executor.(interface{ Close() }); ok {
		d.executor = closer
	}
	if closer, ok := deps.dispatcher.(interface{ Close() }); ok {
		d.dispatcher = closer
	}
	if deps.executor == nil || deps.dispatcher == nil || deps.dialogs == nil || deps.showInformation == nil ||
		deps.openURL == nil || deps.dashboardLoader == nil {
		_ = d.Close()
		return nil, fmt.Errorf("desktop presentation dependencies are required")
	}

	owned := func(id workspace, build func() gui.View) func() gui.View {
		return func() gui.View {
			view := build()
			d.views[id.routeID()] = view
			return view
		}
	}
	dialogs := func() gui.Dialogs { return deps.dialogs(d.window) }
	visible := visibleWorkspaces(d.session)
	routes := []gui.Route{
		{ID: workspaceDashboard.routeID(), Label: "Dashboard", Icon: gui.IconDashboard, Build: owned(workspaceDashboard, func() gui.View {
			d.dashboard = newDashboardViewModel(deps.dashboardLoader(d.session), deps.executor, deps.dispatcher)
			d.presenters[workspaceDashboard.routeID()] = d.dashboard
			return newDashboardView(d.dashboard, func(route string) error { return d.shell.Navigate(route) }, visible)
		})},
		{ID: workspaceDrinks.routeID(), Label: "Drinks", Icon: gui.IconDrinks, Build: owned(workspaceDrinks, func() gui.View {
			presenter := drinksgui.NewPresenter(d.session, drinksgui.Dependencies{Executor: deps.executor, Dispatcher: deps.dispatcher, Dialogs: dialogs()})
			d.presenters[workspaceDrinks.routeID()] = presenter
			return drinksgui.NewView(presenter)
		})},
		{ID: workspaceIngredients.routeID(), Label: "Ingredients", Icon: gui.IconIngredients, Build: owned(workspaceIngredients, func() gui.View {
			presenter := ingredientsgui.NewPresenter(d.session, deps.executor, deps.dispatcher, dialogs())
			d.presenters[workspaceIngredients.routeID()] = presenter
			return ingredientsgui.NewView(presenter)
		})},
		{ID: workspaceInventory.routeID(), Label: "Inventory", Icon: gui.IconInventory, Build: owned(workspaceInventory, func() gui.View {
			presenter := inventorygui.NewPresenter(d.session, deps.executor, deps.dispatcher, dialogs())
			d.presenters[workspaceInventory.routeID()] = presenter
			return inventorygui.NewView(presenter)
		})},
		{ID: workspaceMenus.routeID(), Label: "Menus", Icon: gui.IconMenus, Build: owned(workspaceMenus, func() gui.View {
			presenter := menusgui.NewPresenter(d.session, menusgui.Dependencies{Executor: deps.executor, Dispatcher: deps.dispatcher, Dialogs: dialogs()})
			d.presenters[workspaceMenus.routeID()] = presenter
			return menusgui.NewView(presenter)
		})},
		{ID: workspaceOrders.routeID(), Label: "Orders", Icon: gui.IconOrders, Build: owned(workspaceOrders, func() gui.View {
			presenter := ordersgui.NewPresenter(d.session, ordersgui.Dependencies{Executor: deps.executor, Dispatcher: deps.dispatcher, Dialogs: dialogs()})
			d.presenters[workspaceOrders.routeID()] = presenter
			return ordersgui.NewView(presenter)
		})},
		{ID: workspaceAudit.routeID(), Label: "Audit", Icon: gui.IconAudit, Build: owned(workspaceAudit, func() gui.View {
			presenter := auditgui.NewPresenter(d.session, auditgui.Dependencies{Executor: deps.executor, Dispatcher: deps.dispatcher, Dialogs: dialogs()})
			d.presenters[workspaceAudit.routeID()] = presenter
			return auditgui.NewView(presenter)
		})},
		{ID: workspaceTags.routeID(), Label: "Tags", Icon: gui.IconTags, Build: owned(workspaceTags, func() gui.View {
			presenter := tagginggui.NewPresenter(d.session, tagginggui.Dependencies{Executor: deps.executor, Dispatcher: deps.dispatcher, Dialogs: dialogs()})
			d.presenters[workspaceTags.routeID()] = presenter
			return tagginggui.NewView(presenter)
		})},
	}
	filtered := routes[:0]
	for _, route := range routes {
		if visible.Contains(workspace(route.ID)) {
			filtered = append(filtered, route)
		}
	}
	routes = filtered
	d.shell, err = gui.NewShell(routes, workspaceDashboard.routeID())
	if err != nil {
		_ = d.Close()
		return nil, err
	}
	d.shell.SetIdentity("Mixology", "Local user", config.actor)
	d.window = fyneApp.NewWindow("Mixology — " + config.actor)
	d.shell.SetAbandonConfirmation(func(respond func(bool)) {
		dialogs().Confirm("Discard unsaved changes?", "Your edits have not been saved. Discard them and leave this view?", respond)
	})
	d.window.SetContent(d.shell.Content())
	d.window.Resize(framework.NewSize(1100, 720))
	d.window.SetCloseIntercept(d.closeWindow)
	d.window.SetMainMenu(d.mainMenu())
	d.registerShortcuts()
	d.showInformation = deps.showInformation
	d.openURL = deps.openURL
	d.shell.ActivateCurrent()
	return d, nil
}

// closeWindow is kept separate from composition so lifecycle behavior can be
// exercised without requiring a native close event.
func (d *desktop) closeWindow() {
	d.shell.ConfirmAbandon(func() {
		_ = d.Close()
		d.window.Close()
	})
}

func (d *desktop) Close() error {
	d.closeOnce.Do(func() {
		var appErr, logErr error
		// Stop the separately owned dashboard lifecycle before closing executor
		// admission. Otherwise a concurrent activation can account work that the
		// executor rejects, leaving dashboard shutdown waiting forever.
		if d.dashboard != nil {
			d.dashboard.Close()
		}
		if d.executor != nil {
			d.executor.Close()
		}
		if d.dispatcher != nil {
			d.dispatcher.Close()
		}
		if d.application != nil {
			appErr = d.application.Close()
		}
		if d.logFile != nil {
			logErr = d.logFile.Close()
		}
		var metricsErr error
		if d.metricsServer != nil {
			metricsErr = d.metricsServer.Shutdown(context.Background())
		}
		if d.metricsShutdown != nil {
			metricsErr = errors.Join(metricsErr, d.metricsShutdown(context.Background()))
		}
		d.closeErr = errors.Join(appErr, logErr, metricsErr)
	})
	return d.closeErr
}

package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	fynedesktop "fyne.io/fyne/v2/driver/desktop"

	application "github.com/TheFellow/go-modular-monolith/app"
	auditgui "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/gui"
	drinksgui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/gui"
	ingredientsgui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/gui"
	inventorygui "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/gui"
	menusgui "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/gui"
	ordersgui "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/gui"
	tagginggui "github.com/TheFellow/go-modular-monolith/app/domains/tagging/surfaces/gui"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	fyneui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
	pkglog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

const (
	applicationID    = "com.thefellow.mixology"
	databaseFilename = "mixology.db"
	logFilename      = "mixology.log"
)

type desktopConfig struct {
	dataDirectory string
	actor         string
}

type desktop struct {
	gui             framework.App
	window          framework.Window
	application     *application.App
	session         *application.Session
	shell           *fyneui.Shell
	logFile         *os.File
	closeOnce       sync.Once
	closeErr        error
	dashboard       *dashboardViewModel
	views           map[string]fyneui.View
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
	refresh := framework.NewMenuItem("Refresh", func() { d.shell.ExecuteCommand(fyneui.CommandRefresh) })
	refresh.Shortcut = commandShortcut(framework.KeyR)
	create := framework.NewMenuItem("New", func() { d.shell.ExecuteCommand(fyneui.CommandNew) })
	create.Shortcut = commandShortcut(framework.KeyN)
	save := framework.NewMenuItem("Save", func() { d.shell.ExecuteCommand(fyneui.CommandSave) })
	save.Shortcut = commandShortcut(framework.KeyS)
	cancel := framework.NewMenuItem("Cancel or Back", func() { d.shell.ExecuteCommand(fyneui.CommandCancel) })
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
		command  fyneui.Command
	}{
		{commandShortcut(framework.KeyR), fyneui.CommandRefresh},
		{commandShortcut(framework.KeyN), fyneui.CommandNew},
		{commandShortcut(framework.KeyS), fyneui.CommandSave},
		{&fynedesktop.CustomShortcut{KeyName: framework.KeyEscape}, fyneui.CommandCancel},
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
	executor        fyneui.Executor
	dispatcher      fyneui.Dispatcher
	dialogs         func(framework.Window) fyneui.Dialogs
	showInformation func(title, message string, window framework.Window)
	openURL         func(*url.URL) error
	dashboardLoader func(*application.Session) dashboardLoader
}

func openDesktop(ctx context.Context, gui framework.App, config desktopConfig) (*desktop, error) {
	executor := fyneui.NewManagedExecutor()
	dispatcher := fyneui.NewGatedDispatcher(fyneui.MainDispatcher{})
	return openDesktopWithDependencies(ctx, gui, config, desktopDependencies{
		executor: executor, dispatcher: dispatcher,
		dialogs:         func(window framework.Window) fyneui.Dialogs { return fyneui.WindowDialogs{Window: window} },
		showInformation: dialog.ShowInformation,
		openURL:         gui.OpenURL,
		dashboardLoader: func(session *application.Session) dashboardLoader {
			return sessionDashboardLoader{session: session}
		},
	})
}

func openDesktopWithDependencies(ctx context.Context, gui framework.App, config desktopConfig, deps desktopDependencies) (*desktop, error) {
	if gui == nil {
		return nil, fmt.Errorf("fyne application is required")
	}
	if err := prepareDataDirectory(config.dataDirectory); err != nil {
		return nil, err
	}
	principal, err := authn.ParseActor(config.actor)
	if err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(filepath.Join(config.dataDirectory, logFilename), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open desktop log: %w", err)
	}
	ctx = authn.ToContext(ctx, principal)
	ctx = pkglog.ToContext(ctx, pkglog.Setup("info", "text", logFile))
	ctx = telemetry.WithMetrics(ctx, telemetry.Nop())

	s, err := store.Open(ctx, filepath.Join(config.dataDirectory, databaseFilename))
	if err != nil {
		_ = logFile.Close()
		return nil, err
	}
	app := application.New(ctx, application.Config{Store: s})
	d := &desktop{
		gui: gui, application: app, session: application.NewSession(ctx, app), logFile: logFile,
		views: make(map[string]fyneui.View), presenters: make(map[string]any),
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

	owned := func(id string, build func() fyneui.View) func() fyneui.View {
		return func() fyneui.View {
			view := build()
			d.views[id] = view
			return view
		}
	}
	dialogs := func() fyneui.Dialogs { return deps.dialogs(d.window) }
	routes := []fyneui.Route{
		{ID: "dashboard", Label: "Dashboard", Build: owned("dashboard", func() fyneui.View {
			d.dashboard = newDashboardViewModel(deps.dashboardLoader(d.session), deps.executor, deps.dispatcher)
			d.presenters["dashboard"] = d.dashboard
			return newDashboardView(d.dashboard, func(route string) error { return d.shell.Navigate(route) })
		})},
		{ID: "drinks", Label: "Drinks", Build: owned("drinks", func() fyneui.View {
			presenter := drinksgui.NewPresenter(d.session, drinksgui.Dependencies{Executor: deps.executor, Dispatcher: deps.dispatcher, Dialogs: dialogs()})
			d.presenters["drinks"] = presenter
			return drinksgui.NewView(presenter)
		})},
		{ID: "ingredients", Label: "Ingredients", Build: owned("ingredients", func() fyneui.View {
			presenter := ingredientsgui.NewPresenter(d.session, deps.executor, deps.dispatcher, dialogs())
			d.presenters["ingredients"] = presenter
			return ingredientsgui.NewView(presenter)
		})},
		{ID: "inventory", Label: "Inventory", Build: owned("inventory", func() fyneui.View {
			presenter := inventorygui.NewPresenter(d.session, deps.executor, deps.dispatcher, dialogs())
			d.presenters["inventory"] = presenter
			return inventorygui.NewView(presenter)
		})},
		{ID: "menus", Label: "Menus", Build: owned("menus", func() fyneui.View {
			presenter := menusgui.NewPresenter(d.session, menusgui.Dependencies{Executor: deps.executor, Dispatcher: deps.dispatcher, Dialogs: dialogs()})
			d.presenters["menus"] = presenter
			return menusgui.NewView(presenter)
		})},
		{ID: "orders", Label: "Orders", Build: owned("orders", func() fyneui.View {
			presenter := ordersgui.NewPresenter(d.session, ordersgui.Dependencies{Executor: deps.executor, Dispatcher: deps.dispatcher, Dialogs: dialogs()})
			d.presenters["orders"] = presenter
			return ordersgui.NewView(presenter)
		})},
		{ID: "audit", Label: "Audit", Build: owned("audit", func() fyneui.View {
			presenter := auditgui.NewPresenter(d.session, auditgui.Dependencies{Executor: deps.executor, Dispatcher: deps.dispatcher, Dialogs: dialogs()})
			d.presenters["audit"] = presenter
			return auditgui.NewView(presenter)
		})},
		{ID: "tags", Label: "Tags", Build: owned("tags", func() fyneui.View {
			presenter := tagginggui.NewPresenter(d.session, tagginggui.Dependencies{Executor: deps.executor, Dispatcher: deps.dispatcher, Dialogs: dialogs()})
			d.presenters["tags"] = presenter
			return tagginggui.NewView(presenter)
		})},
	}
	d.shell, err = fyneui.NewShell(routes, "dashboard")
	if err != nil {
		_ = d.Close()
		return nil, err
	}
	d.window = gui.NewWindow("Mixology")
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
	_ = d.Close()
	d.window.Close()
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
		d.closeErr = errors.Join(appErr, logErr)
	})
	return d.closeErr
}

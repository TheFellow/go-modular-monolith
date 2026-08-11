//nolint:paralleltest // Fyne's application driver and focus state are process-global.
package gui

import (
	"strings"
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

type testView struct {
	title   string
	content *widget.Label
}

type activatedTestView struct {
	testView
	activations int
}

type commandTestView struct {
	testView
	commands []Command
	enabled  bool
}

type dirtyTestView struct {
	testView
	dirty bool
}

type invalidationTestView struct {
	testView
	dirty       bool
	refreshes   int
	activations int
}

func (v *invalidationTestView) HasUnsavedChanges() bool { return v.dirty }
func (v *invalidationTestView) Activate()               { v.activations++ }
func (v *invalidationTestView) ExecuteCommand(command Command) bool {
	switch command {
	case CommandRefresh:
		if v.dirty {
			return false
		}
		v.refreshes++
		return true
	case CommandCancel:
		v.dirty = false
		return true
	case CommandNew, CommandSave:
		return false
	}
	return false
}

func (v *dirtyTestView) HasUnsavedChanges() bool { return v.dirty }

func (v *commandTestView) ExecuteCommand(command Command) bool {
	if !v.enabled {
		return false
	}
	v.commands = append(v.commands, command)
	return true
}

func (v *activatedTestView) Activate() { v.activations++ }

func (v *testView) Title() string                   { return v.title }
func (v *testView) Content() framework.CanvasObject { return v.content }

func TestShellNavigatesLazilyAndPreservesViews(t *testing.T) {
	startTestApp(t)
	builds := map[string]int{}
	build := func(id string) func() View {
		return func() View {
			builds[id]++
			return &testView{title: id + " title", content: widget.NewLabel(id)}
		}
	}
	shell, err := NewShell([]Route{
		{ID: "home", Label: "Home", Build: build("home")},
		{ID: "drinks", Label: "Drinks", Build: build("drinks")},
	}, "home")
	testutil.ErrorIf(t, err != nil, "%v", err)
	testutil.ErrorIf(t, shell.Current() != "home" || builds["home"] != 1 || builds["drinks"] != 0, "unexpected initial state: current=%q builds=%v", shell.Current(), builds)
	testutil.ErrorIf(t, shell.navigation["home"].Importance != widget.HighImportance || shell.navigation["drinks"].Importance != widget.LowImportance, "%v", "initial route is not distinguished in the navigation rail")
	{
		err := shell.Navigate("drinks")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	testutil.ErrorIf(t, shell.navigation["drinks"].Importance != widget.HighImportance || shell.navigation["home"].Importance != widget.LowImportance, "%v", "navigation rail did not track the selected route")
	{
		err := shell.Navigate("home")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	testutil.ErrorIf(t, builds["home"] != 1 || builds["drinks"] != 1, "views were not preserved: %v", builds)
}

func TestShellInvalidationRefreshesBrowsingAndDefersDirtyEditor(t *testing.T) {
	startTestApp(t)
	view := &invalidationTestView{testView: testView{title: "Editor", content: widget.NewLabel("editor")}}
	shell, err := NewShell([]Route{{ID: "editor", Label: "Editor", Build: func() View { return view }}}, "editor")
	testutil.ErrorIf(t, err != nil, "%v", err)

	shell.InvalidateCurrent()
	testutil.ErrorIf(t, view.refreshes != 1, "browsing refreshes = %d", view.refreshes)

	view.dirty = true
	shell.InvalidateCurrent()
	testutil.ErrorIf(t, view.refreshes != 1 || !shell.stale["editor"], "dirty invalidation was not deferred")
	shell.ExecuteCommand(CommandCancel)
	testutil.ErrorIf(t, view.dirty || shell.stale["editor"] || view.activations != 1, "deferred invalidation was not refreshed after cancel")
}

func TestShellShowsPersistentIdentityAndExplicitRouteIcon(t *testing.T) {
	startTestApp(t)
	shell, err := NewShell([]Route{{ID: "home", Label: "Home", Icon: IconDashboard, Build: func() View { return &testView{title: "Home", content: widget.NewLabel("home")} }}}, "home")
	testutil.ErrorIf(t, err != nil, "%v", err)
	shell.SetIdentity("Mixology", "Local user", "manager")
	testutil.ErrorIf(t, shell.identity.Text != "Mixology\nLocal user · manager", "identity = %q", shell.identity.Text)
	testutil.ErrorIf(t, shell.navigation["home"].Icon != IconResource(IconDashboard), "%v", "route did not use its enumerated icon")
	{
		err := shell.Navigate("home")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	testutil.ErrorIf(t, shell.identity.Text == "", "%v", "identity did not persist")
}

func TestLucideIconsPreserveStrokeOnlyRendering(t *testing.T) {
	startTestApp(t)
	content := string(IconResource(IconDashboard).Content())
	testutil.ErrorIf(t, !strings.Contains(content, `fill="none"`), "Lucide fill was not preserved: %s", content)
	testutil.ErrorIf(t, !strings.Contains(content, `stroke="#`), "Lucide stroke was not theme-colored: %s", content)
	testutil.ErrorIf(t, strings.Contains(content, "currentColor"), "Lucide currentColor was not resolved: %s", content)
}

func TestShellActivatesInitialViewAndEveryReentryWithoutRebuilding(t *testing.T) {
	startTestApp(t)
	home := &activatedTestView{testView: testView{title: "Home", content: widget.NewLabel("home")}}
	other := &testView{title: "Other", content: widget.NewLabel("other")}
	shell, err := NewShell([]Route{
		{ID: "home", Label: "Home", Build: func() View { return home }},
		{ID: "other", Label: "Other", Build: func() View { return other }},
	}, "home")
	testutil.ErrorIf(t, err != nil, "%v", err)
	shell.ActivateCurrent()
	shell.ActivateCurrent()
	testutil.ErrorIf(t, home.activations != 1, "initial activations = %d, want 1", home.activations)
	{
		err := shell.Navigate("other")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	{
		err := shell.Navigate("home")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	testutil.ErrorIf(t, home.activations != 2, "reentry activations = %d, want 2", home.activations)
}

func TestShellRejectsInvalidRoutesWithoutChangingSelection(t *testing.T) {
	startTestApp(t)
	shell, err := NewShell([]Route{{
		ID: "home", Label: "Home",
		Build: func() View { return &testView{title: "Home", content: widget.NewLabel("home")} },
	}}, "home")
	testutil.ErrorIf(t, err != nil, "%v", err)
	{
		err := shell.Navigate("missing")
		testutil.ErrorIf(t, err == nil, "%v", "expected unknown route error")
	}
	testutil.ErrorIf(t, shell.Current() != "home", "invalid navigation changed route to %q", shell.Current())
}

func TestShellRejectsDuplicateRoutes(t *testing.T) {
	startTestApp(t)
	build := func() View { return &testView{title: "Home", content: widget.NewLabel("home")} }
	_, err := NewShell([]Route{
		{ID: "home", Label: "Home", Build: build},
		{ID: "home", Label: "Again", Build: build},
	}, "home")
	testutil.ErrorIf(t, err == nil, "%v", "expected duplicate route error")
}

func TestShellOffersCommandsOnlyToCurrentConcreteView(t *testing.T) {
	startTestApp(t)
	home := &commandTestView{testView: testView{title: "Home", content: widget.NewLabel("home")}, enabled: true}
	other := &commandTestView{testView: testView{title: "Other", content: widget.NewLabel("other")}}
	shell, err := NewShell([]Route{
		{ID: "home", Label: "Home", Build: func() View { return home }},
		{ID: "other", Label: "Other", Build: func() View { return other }},
	}, "home")
	testutil.ErrorIf(t, err != nil, "%v", err)
	testutil.ErrorIf(t, !shell.ExecuteCommand(CommandRefresh) || len(home.commands) != 1, "%v", "current view did not handle command")
	{
		err := shell.Navigate("other")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	testutil.ErrorIf(t, shell.ExecuteCommand(CommandSave) || len(home.commands) != 1, "%v", "disabled view or inactive view handled command")
}

func startTestApp(t *testing.T) {
	t.Helper()
	app := test.NewApp()
	t.Cleanup(app.Quit)
}

func TestShellConfirmsBeforeLeavingUnsavedEditor(t *testing.T) {
	startTestApp(t)
	editor := &dirtyTestView{testView: testView{title: "Editor", content: widget.NewLabel("editor")}, dirty: true}
	shell, err := NewShell([]Route{
		{ID: "editor", Label: "Editor", Build: func() View { return editor }},
		{ID: "other", Label: "Other", Build: func() View { return &testView{title: "Other", content: widget.NewLabel("other")} }},
	}, "editor")
	testutil.ErrorIf(t, err != nil, "%v", err)
	var respond func(bool)
	shell.SetAbandonConfirmation(func(callback func(bool)) { respond = callback })
	{
		err := shell.Navigate("other")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	testutil.ErrorIf(t, shell.Current() != "editor" || respond == nil, "%v", "unsaved editor was replaced without confirmation")
	respond(false)
	testutil.ErrorIf(t, shell.Current() != "editor", "%v", "cancelled navigation changed route")
	{
		err := shell.Navigate("other")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	respond(true)
	testutil.ErrorIf(t, shell.Current() != "other", "%v", "confirmed navigation did not continue")
}

func TestShellConfirmsBeforeReactivatingDirtyCurrentRoute(t *testing.T) {
	startTestApp(t)
	editor := &dirtyTestView{testView: testView{title: "Editor", content: widget.NewLabel("editor")}, dirty: true}
	activated := &activatedTestView{testView: editor.testView}
	shell, err := NewShell([]Route{{ID: "editor", Label: "Editor", Build: func() View {
		// Combine the lifecycle and dirty contracts for this focused assertion.
		return structView{View: activated, dirty: editor}
	}}}, "editor")
	testutil.ErrorIf(t, err != nil, "%v", err)
	var respond func(bool)
	shell.SetAbandonConfirmation(func(callback func(bool)) { respond = callback })
	{
		err := shell.Navigate("editor")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	testutil.ErrorIf(t, respond == nil || activated.activations != 0, "%v", "same-route navigation bypassed confirmation")
	respond(false)
	testutil.ErrorIf(t, activated.activations != 0, "%v", "cancelled reactivation ran")
	{
		err := shell.Navigate("editor")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	respond(true)
	testutil.ErrorIf(t, activated.activations != 1, "activations = %d, want 1", activated.activations)
}

type structView struct {
	View
	dirty *dirtyTestView
}

func (v structView) HasUnsavedChanges() bool { return v.dirty.dirty }
func (v structView) Activate()               { v.View.(Activated).Activate() }

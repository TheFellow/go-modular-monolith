//nolint:paralleltest // Fyne's application driver and focus state are process-global.
package gui

import (
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
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if shell.Current() != "home" || builds["home"] != 1 || builds["drinks"] != 0 {
		testutil.ErrorIf(t, true, "unexpected initial state: current=%q builds=%v", shell.Current(), builds)
	}
	if shell.navigation["home"].Importance != widget.HighImportance || shell.navigation["drinks"].Importance != widget.LowImportance {
		testutil.ErrorIf(t, true, "%v", "initial route is not distinguished in the navigation rail")
	}
	if err := shell.Navigate("drinks"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if shell.navigation["drinks"].Importance != widget.HighImportance || shell.navigation["home"].Importance != widget.LowImportance {
		testutil.ErrorIf(t, true, "%v", "navigation rail did not track the selected route")
	}
	if err := shell.Navigate("home"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if builds["home"] != 1 || builds["drinks"] != 1 {
		testutil.ErrorIf(t, true, "views were not preserved: %v", builds)
	}
}

func TestShellShowsPersistentIdentityAndExplicitRouteIcon(t *testing.T) {
	startTestApp(t)
	shell, err := NewShell([]Route{{ID: "home", Label: "Home", Icon: IconDashboard, Build: func() View { return &testView{title: "Home", content: widget.NewLabel("home")} }}}, "home")
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	shell.SetIdentity("Mixology", "Local user", "manager")
	if shell.identity.Text != "Mixology\nLocal user · manager" {
		testutil.ErrorIf(t, true, "identity = %q", shell.identity.Text)
	}
	if shell.navigation["home"].Icon != IconResource(IconDashboard) {
		testutil.ErrorIf(t, true, "%v", "route did not use its enumerated icon")
	}
	if err := shell.Navigate("home"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if shell.identity.Text == "" {
		testutil.ErrorIf(t, true, "%v", "identity did not persist")
	}
}

func TestShellActivatesInitialViewAndEveryReentryWithoutRebuilding(t *testing.T) {
	startTestApp(t)
	home := &activatedTestView{testView: testView{title: "Home", content: widget.NewLabel("home")}}
	other := &testView{title: "Other", content: widget.NewLabel("other")}
	shell, err := NewShell([]Route{
		{ID: "home", Label: "Home", Build: func() View { return home }},
		{ID: "other", Label: "Other", Build: func() View { return other }},
	}, "home")
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	shell.ActivateCurrent()
	shell.ActivateCurrent()
	if home.activations != 1 {
		testutil.ErrorIf(t, true, "initial activations = %d, want 1", home.activations)
	}
	if err := shell.Navigate("other"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if err := shell.Navigate("home"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if home.activations != 2 {
		testutil.ErrorIf(t, true, "reentry activations = %d, want 2", home.activations)
	}
}

func TestShellRejectsInvalidRoutesWithoutChangingSelection(t *testing.T) {
	startTestApp(t)
	shell, err := NewShell([]Route{{
		ID: "home", Label: "Home",
		Build: func() View { return &testView{title: "Home", content: widget.NewLabel("home")} },
	}}, "home")
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if err := shell.Navigate("missing"); err == nil {
		testutil.ErrorIf(t, true, "%v", "expected unknown route error")
	}
	if shell.Current() != "home" {
		testutil.ErrorIf(t, true, "invalid navigation changed route to %q", shell.Current())
	}
}

func TestShellRejectsDuplicateRoutes(t *testing.T) {
	startTestApp(t)
	build := func() View { return &testView{title: "Home", content: widget.NewLabel("home")} }
	_, err := NewShell([]Route{
		{ID: "home", Label: "Home", Build: build},
		{ID: "home", Label: "Again", Build: build},
	}, "home")
	if err == nil {
		testutil.ErrorIf(t, true, "%v", "expected duplicate route error")
	}
}

func TestShellOffersCommandsOnlyToCurrentConcreteView(t *testing.T) {
	startTestApp(t)
	home := &commandTestView{testView: testView{title: "Home", content: widget.NewLabel("home")}, enabled: true}
	other := &commandTestView{testView: testView{title: "Other", content: widget.NewLabel("other")}}
	shell, err := NewShell([]Route{
		{ID: "home", Label: "Home", Build: func() View { return home }},
		{ID: "other", Label: "Other", Build: func() View { return other }},
	}, "home")
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if !shell.ExecuteCommand(CommandRefresh) || len(home.commands) != 1 {
		testutil.ErrorIf(t, true, "%v", "current view did not handle command")
	}
	if err := shell.Navigate("other"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if shell.ExecuteCommand(CommandSave) || len(home.commands) != 1 {
		testutil.ErrorIf(t, true, "%v", "disabled view or inactive view handled command")
	}
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
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	var respond func(bool)
	shell.SetAbandonConfirmation(func(callback func(bool)) { respond = callback })
	if err := shell.Navigate("other"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if shell.Current() != "editor" || respond == nil {
		testutil.ErrorIf(t, true, "%v", "unsaved editor was replaced without confirmation")
	}
	respond(false)
	if shell.Current() != "editor" {
		testutil.ErrorIf(t, true, "%v", "cancelled navigation changed route")
	}
	if err := shell.Navigate("other"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	respond(true)
	if shell.Current() != "other" {
		testutil.ErrorIf(t, true, "%v", "confirmed navigation did not continue")
	}
}

func TestShellConfirmsBeforeReactivatingDirtyCurrentRoute(t *testing.T) {
	startTestApp(t)
	editor := &dirtyTestView{testView: testView{title: "Editor", content: widget.NewLabel("editor")}, dirty: true}
	activated := &activatedTestView{testView: editor.testView}
	shell, err := NewShell([]Route{{ID: "editor", Label: "Editor", Build: func() View {
		// Combine the lifecycle and dirty contracts for this focused assertion.
		return structView{View: activated, dirty: editor}
	}}}, "editor")
	if err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	var respond func(bool)
	shell.SetAbandonConfirmation(func(callback func(bool)) { respond = callback })
	if err := shell.Navigate("editor"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	if respond == nil || activated.activations != 0 {
		testutil.ErrorIf(t, true, "%v", "same-route navigation bypassed confirmation")
	}
	respond(false)
	if activated.activations != 0 {
		testutil.ErrorIf(t, true, "%v", "cancelled reactivation ran")
	}
	if err := shell.Navigate("editor"); err != nil {
		testutil.ErrorIf(t, true, "%v", err)
	}
	respond(true)
	if activated.activations != 1 {
		testutil.ErrorIf(t, true, "activations = %d, want 1", activated.activations)
	}
}

type structView struct {
	View
	dirty *dirtyTestView
}

func (v structView) HasUnsavedChanges() bool { return v.dirty.dirty }
func (v structView) Activate()               { v.View.(Activated).Activate() }

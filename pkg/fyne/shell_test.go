//nolint:paralleltest // Fyne's application driver and focus state are process-global.
package fyne

import (
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
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
		t.Fatal(err)
	}
	if shell.Current() != "home" || builds["home"] != 1 || builds["drinks"] != 0 {
		t.Fatalf("unexpected initial state: current=%q builds=%v", shell.Current(), builds)
	}
	if err := shell.Navigate("drinks"); err != nil {
		t.Fatal(err)
	}
	if err := shell.Navigate("home"); err != nil {
		t.Fatal(err)
	}
	if builds["home"] != 1 || builds["drinks"] != 1 {
		t.Fatalf("views were not preserved: %v", builds)
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
		t.Fatal(err)
	}
	shell.ActivateCurrent()
	shell.ActivateCurrent()
	if home.activations != 1 {
		t.Fatalf("initial activations = %d, want 1", home.activations)
	}
	if err := shell.Navigate("other"); err != nil {
		t.Fatal(err)
	}
	if err := shell.Navigate("home"); err != nil {
		t.Fatal(err)
	}
	if home.activations != 2 {
		t.Fatalf("reentry activations = %d, want 2", home.activations)
	}
}

func TestShellRejectsInvalidRoutesWithoutChangingSelection(t *testing.T) {
	startTestApp(t)
	shell, err := NewShell([]Route{{
		ID: "home", Label: "Home",
		Build: func() View { return &testView{title: "Home", content: widget.NewLabel("home")} },
	}}, "home")
	if err != nil {
		t.Fatal(err)
	}
	if err := shell.Navigate("missing"); err == nil {
		t.Fatal("expected unknown route error")
	}
	if shell.Current() != "home" {
		t.Fatalf("invalid navigation changed route to %q", shell.Current())
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
		t.Fatal("expected duplicate route error")
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
		t.Fatal(err)
	}
	if !shell.ExecuteCommand(CommandRefresh) || len(home.commands) != 1 {
		t.Fatal("current view did not handle command")
	}
	if err := shell.Navigate("other"); err != nil {
		t.Fatal(err)
	}
	if shell.ExecuteCommand(CommandSave) || len(home.commands) != 1 {
		t.Fatal("disabled view or inactive view handled command")
	}
}

func startTestApp(t *testing.T) {
	t.Helper()
	app := test.NewApp()
	t.Cleanup(app.Quit)
}

//nolint:paralleltest // Fyne's focus and canvas state are process-global.
package gui

import (
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestTriggerRespectsVisibleEnabledState(t *testing.T) {
	startTestApp(t)
	called := 0
	button := NewButton("save", "Save", func() { called++ })
	triggered := Trigger(button)
	testutil.ErrorIf(t, !triggered || called != 1, "enabled trigger = %v calls=%d", triggered, called)
	button.Disable()
	testutil.ErrorIf(t, Trigger(button) || called != 1, "disabled button triggered: calls=%d", called)
	button.Enable()
	button.Hide()
	testutil.ErrorIf(t, Trigger(button) || called != 1, "hidden button triggered: calls=%d", called)
}

func TestTriggerRejectsButtonWithoutAction(t *testing.T) {
	startTestApp(t)
	testutil.ErrorIf(t, Trigger(NewButton("placeholder", "Placeholder", nil)), "%v", "button without a command reported a successful trigger")
}

func TestSubmitOnEnterUsesGuardedButtonActionExactlyOnce(t *testing.T) {
	startTestApp(t)
	entry := NewEntry("query")
	calls := 0
	button := NewButton("apply", "Apply", func() { calls++ })
	SubmitOnEnter(entry, button)

	entry.OnSubmitted("first")
	button.Disable()
	entry.OnSubmitted("disabled")
	button.Enable()
	button.Hide()
	entry.OnSubmitted("hidden")
	button.Show()
	entry.OnSubmitted("second")
	testutil.ErrorIf(t, calls != 2, "submitted callback %d times, want 2 enabled visible submissions", calls)
}

func TestKeyboardFocusTraversesSemanticControlsInBothDirections(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	first, second := NewEntry("first"), NewEntry("second")
	window := app.NewWindow("focus")
	window.SetContent(container.NewVBox(first, second, NewButton("submit", "Submit", func() {})))
	canvas := window.Canvas()
	canvas.Focus(first)
	canvas.FocusNext()
	testutil.ErrorIf(t, canvas.Focused() != second, "FocusNext focused %T, want second entry", canvas.Focused())
	canvas.FocusPrevious()
	testutil.ErrorIf(t, canvas.Focused() != first, "FocusPrevious focused %T, want first entry", canvas.Focused())
	canvas.Focus(first)
	canvas.Focused().TypedKey(&framework.KeyEvent{Name: framework.KeyTab})
}

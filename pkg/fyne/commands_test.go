//nolint:paralleltest // Fyne's focus and canvas state are process-global.
package fyne

import (
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

func TestTriggerRespectsVisibleEnabledState(t *testing.T) {
	startTestApp(t)
	called := 0
	button := NewButton("save", "Save", func() { called++ })
	if !Trigger(button) || called != 1 {
		t.Fatalf("enabled trigger = %v calls=%d", Trigger(button), called)
	}
	button.Disable()
	if Trigger(button) || called != 1 {
		t.Fatalf("disabled button triggered: calls=%d", called)
	}
	button.Enable()
	button.Hide()
	if Trigger(button) || called != 1 {
		t.Fatalf("hidden button triggered: calls=%d", called)
	}
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
	if canvas.Focused() != second {
		t.Fatalf("FocusNext focused %T, want second entry", canvas.Focused())
	}
	canvas.FocusPrevious()
	if canvas.Focused() != first {
		t.Fatalf("FocusPrevious focused %T, want first entry", canvas.Focused())
	}
	canvas.Focus(first)
	canvas.Focused().TypedKey(&framework.KeyEvent{Name: framework.KeyTab})
}

package gui

import (
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestFormScrollRelayScrollsPageAboveEntry(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	entry := NewEntry("description")
	entry.MultiLine = true
	entry.SetText("selectable text")
	fields := container.NewVBox(append([]framework.CanvasObject{entry}, spacer(30)...)...)
	scroll := newFormScroll(fields)
	scroll.Resize(framework.NewSize(400, 120))
	scroll.Content.Resize(framework.NewSize(400, fields.MinSize().Height))

	relay := scroll.Content.(*framework.Container).Objects[1].(*formScrollRelay)
	relay.Scrolled(&framework.ScrollEvent{Scrolled: framework.NewDelta(0, -80)})

	if scroll.Offset.Y == 0 {
		t.Fatal("wheel event over a form control did not scroll the page")
	}
	if entry.Text != "selectable text" || entry.Disabled() {
		t.Fatal("scroll relay changed the editable/selectable entry")
	}
}

func TestFormScrollRelayScrollsBackAcrossReadOnlyEntry(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	entry := NewEntry("audit-error")
	entry.SetText("copyable audit detail")
	entry.OnChanged = func(string) { entry.SetText("copyable audit detail") }
	fields := container.NewVBox(append([]framework.CanvasObject{entry}, spacer(30)...)...)
	scroll := newFormScroll(fields)
	scroll.Resize(framework.NewSize(400, 120))
	scroll.Content.Resize(framework.NewSize(400, fields.MinSize().Height))
	relay := scroll.Content.(*framework.Container).Objects[1].(*formScrollRelay)

	relay.Scrolled(&framework.ScrollEvent{Scrolled: framework.NewDelta(0, -200)})
	if scroll.Offset.Y == 0 {
		t.Fatal("page did not scroll down")
	}
	relay.Scrolled(&framework.ScrollEvent{Scrolled: framework.NewDelta(0, 200)})
	if scroll.Offset.Y != 0 {
		t.Fatalf("page offset after scrolling up = %v, want 0", scroll.Offset.Y)
	}
}

func spacer(count int) []framework.CanvasObject {
	objects := make([]framework.CanvasObject, count)
	for i := range objects {
		objects[i] = widget.NewLabel("long form field")
	}
	return objects
}

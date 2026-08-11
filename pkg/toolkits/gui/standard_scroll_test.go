package gui

import (
	"reflect"
	"strings"
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestSingleLineEntryPassesWheelToFormPage(t *testing.T) { //nolint:paralleltest // Fyne app and driver state is process-global.
	app := test.NewApp()
	t.Cleanup(app.Quit)

	entry := NewEntry("name")
	entry.SetText("selectable text")
	fields := container.NewVBox(append([]framework.CanvasObject{entry}, spacer(30)...)...)
	page, canvas := formCanvas(app, fields)

	test.Scroll(canvas, scrollPoint(app, entry), 0, -80)

	testutil.ErrorIf(t, page.Offset.Y == 0, "%v", "wheel event over a single-line field did not scroll the form page")
	testutil.ErrorIf(t, entry.Text != "selectable text" || entry.Disabled(), "%v", "scrolling changed the editable/selectable entry")
}

func TestMultiLineEntryKeepsWheelForItsOwnContent(t *testing.T) { //nolint:paralleltest // Fyne app and driver state is process-global.
	app := test.NewApp()
	t.Cleanup(app.Quit)

	entry := NewMultiLineEntry("steps")
	entry.SetText(strings.Repeat("A recipe step that must remain reachable\n", 20))
	fields := container.NewVBox(append([]framework.CanvasObject{entry}, spacer(30)...)...)
	page, canvas := formCanvas(app, fields)

	test.Scroll(canvas, scrollPoint(app, entry), 0, -80)

	testutil.ErrorIf(t, page.Offset.Y != 0, "multiline wheel event moved form page to %v instead of its field", page.Offset.Y)
	testutil.ErrorIf(t, entry.Scroll != framework.ScrollVerticalOnly, "multiline scroll direction = %v", entry.Scroll)
	entryScroll := test.WidgetRenderer(entry).Objects()[2]
	offsetY := reflect.ValueOf(entryScroll).Elem().FieldByName("Offset").FieldByName("Y").Float()
	testutil.ErrorIf(t, offsetY == 0, "%v", "wheel event over Steps did not scroll its multiline content")
}

func formCanvas(app framework.App, fields framework.CanvasObject) (*container.Scroll, framework.Canvas) {
	page := newFormScroll(fields)
	window := app.NewWindow("form scroll")
	window.SetContent(page)
	window.Resize(framework.NewSize(400, 120))
	window.Show()
	return page, window.Canvas()
}

func scrollPoint(app framework.App, object framework.CanvasObject) framework.Position {
	return app.Driver().AbsolutePositionForObject(object).Add(framework.NewPos(8, 8))
}

func spacer(count int) []framework.CanvasObject {
	objects := make([]framework.CanvasObject, count)
	for i := range objects {
		objects[i] = widget.NewLabel("long form field")
	}
	return objects
}

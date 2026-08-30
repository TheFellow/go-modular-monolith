package gui

import (
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

func TestReadonlyEntryPassesWheelToFormPage(t *testing.T) { //nolint:paralleltest // Fyne app and driver state is process-global.
	app := test.NewApp()
	t.Cleanup(app.Quit)

	entry := ReadonlyEntry("selectable detail")
	fields := container.NewVBox(append([]framework.CanvasObject{entry}, spacer(30)...)...)
	page, canvas := formCanvas(app, fields)

	test.Scroll(canvas, scrollPoint(app, entry), 0, -80)

	testutil.ErrorIf(t, page.Offset.Y == 0, "%v", "wheel event over a read-only detail field did not scroll the form page")
	testutil.Equals(t, entry.Text, "selectable detail")
}

func TestEditableMultiLineEntryPassesWheelToFormPage(t *testing.T) { //nolint:paralleltest // Fyne app and driver state is process-global.
	app := test.NewApp()
	t.Cleanup(app.Quit)

	entry := NewMultiLineEntry("steps")
	entry.SetText(strings.Repeat("A recipe step that must remain reachable\n", 20))
	fields := container.NewVBox(append([]framework.CanvasObject{entry}, spacer(30)...)...)
	page, canvas := formCanvas(app, fields)

	test.Scroll(canvas, scrollPoint(app, entry), 0, -80)

	testutil.ErrorIf(t, page.Offset.Y == 0, "%v", "wheel event over an editable multiline field did not scroll the form page")
	testutil.ErrorIf(t, entry.Scroll != framework.ScrollNone, "multiline scroll direction = %v", entry.Scroll)
	testutil.ErrorIf(t, entry.Text == "" || entry.Disabled(), "%v", "scrolling changed the editable multiline field")
}

func TestReadonlyMultiLineEntryPassesWheelToDetailPage(t *testing.T) { //nolint:paralleltest // Fyne app and driver state is process-global.
	app := test.NewApp()
	t.Cleanup(app.Quit)

	value := strings.Repeat("A long detail line\n", 20)
	entry := ReadonlyMultiLineEntry(value)
	fields := container.NewVBox(append([]framework.CanvasObject{entry}, spacer(30)...)...)
	page, canvas := formCanvas(app, fields)

	test.Scroll(canvas, scrollPoint(app, entry), 0, -80)

	testutil.ErrorIf(t, page.Offset.Y == 0, "%v", "wheel event over a multiline detail field did not scroll the detail page")
	testutil.Equals(t, entry.Scroll, framework.ScrollNone)
	testutil.Equals(t, entry.Text, value)
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
